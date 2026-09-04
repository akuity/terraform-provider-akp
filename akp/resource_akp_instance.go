package akp

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/akuity/api-client-go/pkg/api/argocdexport"
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

var argoResourceGroups = map[string]struct {
	appendFunc resourceGroupAppender[*argocdv1.ApplyInstanceRequest]
}{
	"Application": {
		appendFunc: func(req *argocdv1.ApplyInstanceRequest, item *structpb.Struct) {
			req.Applications = append(req.Applications, item)
		},
	},
	"ApplicationSet": {
		appendFunc: func(req *argocdv1.ApplyInstanceRequest, item *structpb.Struct) {
			req.ApplicationSets = append(req.ApplicationSets, item)
		},
	},
	"AppProject": {
		appendFunc: func(req *argocdv1.ApplyInstanceRequest, item *structpb.Struct) {
			req.AppProjects = append(req.AppProjects, item)
		},
	},
}

func NewAkpInstanceResource() resource.Resource {
	return &GenericResource[types.Instance]{
		TypeNameSuffix:      "instance",
		SchemaFunc:          instanceSchema,
		CreateFunc:          instanceCreate,
		ReadFunc:            instanceRead,
		UpdateWithStateFunc: instanceUpdate,
		DeleteFunc:          instanceDelete,
		CopyWriteOnlyFunc:   instanceCopyWriteOnly,
		ImportStateFunc: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
		},
	}
}

func instanceCopyWriteOnly(ctx context.Context, config tfsdk.Config, diags *diag.Diagnostics, plan *types.Instance) {
	if len(plan.ManagedSecrets) == 0 {
		return
	}
	var configSecrets map[string]*types.ManagedSecret
	diags.Append(config.GetAttribute(ctx, path.Root("managed_secrets"), &configSecrets)...)
	if diags.HasError() {
		return
	}
	for name, secret := range plan.ManagedSecrets {
		if secret == nil {
			continue
		}
		if configSecret, ok := configSecrets[name]; ok && configSecret != nil {
			secret.Data = configSecret.Data
		}
	}
}

func instanceCreateOrUpdate(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *types.Instance) (*types.Instance, error) {
	plannedCM := plan.ArgoCDConfigMap
	stateCanBeCommitted, err := instanceUpsert(ctx, cli, diags, plan)
	if stateCanBeCommitted {
		plan.ArgoCDConfigMap = types.FilterMapToPlannedKeys(ctx, diags, plan.ArgoCDConfigMap, plannedCM)
		return plan, err
	}
	return nil, err
}

func instanceCreate(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *types.Instance) (*types.Instance, error) {
	return applyInstanceWithManagedSecrets(ctx, cli, diags, nil, plan, nil)
}

func instanceUpdate(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, state, plan *types.Instance) (*types.Instance, error) {
	created, err := createAddedManagedSecrets(ctx, cli, diags, state, plan)
	if err != nil {
		return withCreatedManagedSecrets(state, created), err
	}
	result, err := applyInstanceWithManagedSecrets(ctx, cli, diags, state.ManagedSecrets, plan, created)
	if result == nil {
		return withCreatedManagedSecrets(state, created), err
	}
	return result, err
}

func withCreatedManagedSecrets(state *types.Instance, created map[string]*types.ManagedSecret) *types.Instance {
	if len(created) == 0 {
		return nil
	}
	tracked := *state
	tracked.ManagedSecrets = trackedManagedSecrets(state.ManagedSecrets, created)
	return &tracked
}

func trackedManagedSecrets(stateSecrets, created map[string]*types.ManagedSecret) map[string]*types.ManagedSecret {
	if len(created) == 0 {
		return stateSecrets
	}
	tracked := make(map[string]*types.ManagedSecret, len(stateSecrets)+len(created))
	maps.Copy(tracked, stateSecrets)
	maps.Copy(tracked, created)
	return tracked
}

func createAddedManagedSecrets(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, state, plan *types.Instance) (map[string]*types.ManagedSecret, error) {
	added := make(map[string]*types.ManagedSecret)
	for name, secret := range plan.ManagedSecrets {
		if secret == nil {
			continue
		}
		if tracked, ok := state.ManagedSecrets[name]; !ok || tracked == nil {
			added[name] = secret
		}
	}
	if len(added) == 0 {
		return nil, nil
	}
	workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, state.Workspace.ValueString())
	if err != nil {
		return nil, errors.Wrap(err, "Unable to get workspace for managed secret changes")
	}
	return syncManagedSecrets(ctx, cli, diags, state.ID.ValueString(), workspace.GetId(), nil, added)
}

func applyInstanceWithManagedSecrets(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, stateSecrets map[string]*types.ManagedSecret, plan *types.Instance, created map[string]*types.ManagedSecret) (*types.Instance, error) {
	plannedSecrets := plan.ManagedSecrets
	result, err := instanceCreateOrUpdate(ctx, cli, diags, plan)
	if result == nil {
		return nil, err
	}
	if err != nil {
		result.ManagedSecrets = trackedManagedSecrets(stateSecrets, created)
		return result, err
	}
	return applyManagedSecretChanges(ctx, cli, diags, result, stateSecrets, plannedSecrets, created)
}

func applyManagedSecretChanges(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, result *types.Instance, stateSecrets, plannedSecrets, created map[string]*types.ManagedSecret) (*types.Instance, error) {
	if plannedSecrets == nil && len(stateSecrets) == 0 {
		result.ManagedSecrets = nil
		return result, nil
	}
	tracked := trackedManagedSecrets(stateSecrets, created)
	pending := plannedSecrets
	if len(created) > 0 {
		pending = maps.Clone(plannedSecrets)
		maps.DeleteFunc(pending, func(name string, _ *types.ManagedSecret) bool {
			_, ok := created[name]
			return ok
		})
	}
	workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, result.Workspace.ValueString())
	if err != nil {
		result.ManagedSecrets = tracked
		return result, errors.Wrap(err, "Unable to get workspace for managed secret changes")
	}
	synced, syncErr := syncManagedSecrets(ctx, cli, diags, result.ID.ValueString(), workspace.GetId(), stateSecrets, pending)
	maps.Copy(synced, created)
	if plannedSecrets == nil && len(synced) == 0 {
		result.ManagedSecrets = nil
	} else {
		result.ManagedSecrets = synced
	}
	return result, syncErr
}

func syncManagedSecrets(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, instanceID, workspaceID string, state, plan map[string]*types.ManagedSecret) (map[string]*types.ManagedSecret, error) {
	current := make(map[string]*types.ManagedSecret, len(state))
	for name, secret := range state {
		if secret != nil {
			current[name] = secret
		}
	}
	names := slices.Sorted(maps.Keys(plan))
	var existingKeys map[string][]string
	for _, name := range names {
		secret := plan[name]
		if secret == nil {
			continue
		}
		upsert := types.ToManagedSecretUpsertAPIModel(ctx, diags, name, secret)
		if diags.HasError() || upsert == nil {
			return current, errors.Errorf("Unable to build managed secret %q", name)
		}
		var err error
		created := false
		if stateSecret, tracked := current[name]; tracked {
			update := upsert
			if stateSecret.DataVersion.Equal(secret.DataVersion) {
				metadataOnly := *upsert
				metadataOnly.Data = nil
				metadataOnly.ClearData = false
				update = &metadataOnly
			}
			if update.ClearData {
				if existingKeys == nil {
					existingKeys, err = listManagedSecretKeys(ctx, cli, instanceID, workspaceID)
					if err != nil {
						return current, errors.Wrap(err, "Unable to list managed secrets")
					}
				}
				err = patchManagedSecretClearData(ctx, cli, instanceID, workspaceID, update, existingKeys[name])
			} else {
				err = updateManagedSecret(ctx, cli, instanceID, workspaceID, update)
			}
			if status.Code(err) == codes.NotFound {
				err = createManagedSecret(ctx, cli, instanceID, workspaceID, upsert)
				created = err == nil
			}
		} else {
			err = createManagedSecret(ctx, cli, instanceID, workspaceID, upsert)
			if status.Code(err) == codes.AlreadyExists {
				return current, errors.Wrapf(err, "secret %q already exists and is not tracked by this resource; existing secrets are not adopted — delete the existing secret or use a different name", name)
			}
			created = err == nil
		}
		if created && upsert.ClearData {
			existingKeys, err = listManagedSecretKeys(ctx, cli, instanceID, workspaceID)
			if err != nil {
				return current, errors.Wrap(err, "Unable to list managed secrets")
			}
			if len(existingKeys[name]) > 0 {
				err = patchManagedSecretClearData(ctx, cli, instanceID, workspaceID, upsert, existingKeys[name])
			}
		}
		if err != nil {
			return current, errors.Wrapf(err, "Unable to apply managed secret %q", name)
		}
		sanitized := *secret
		sanitized.Data = tftypes.MapNull(tftypes.StringType)
		current[name] = &sanitized
	}
	for _, name := range removedManagedSecretNames(state, plan) {
		if err := deleteManagedSecret(ctx, cli, instanceID, workspaceID, name); err != nil {
			return current, err
		}
		delete(current, name)
	}
	return current, nil
}

const managedSecretOwner = "terraform"

func createManagedSecret(ctx context.Context, cli *AkpCli, instanceID, workspaceID string, upsert *types.ManagedSecretUpsert) error {
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.CreateManagedSecretResponse, error) {
		return cli.Cli.CreateManagedSecret(ctx, &argocdv1.CreateManagedSecretRequest{
			OrganizationId:    cli.OrgId,
			WorkspaceId:       workspaceID,
			InstanceId:        instanceID,
			ManagedSecret:     upsert.Secret,
			ManagedSecretData: upsert.Data,
			Owner:             managedSecretOwner,
		})
	}, "CreateManagedSecret")
	return err
}

func updateManagedSecret(ctx context.Context, cli *AkpCli, instanceID, workspaceID string, upsert *types.ManagedSecretUpsert) error {
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.UpdateManagedSecretResponse, error) {
		return cli.Cli.UpdateManagedSecret(ctx, &argocdv1.UpdateManagedSecretRequest{
			OrganizationId:    cli.OrgId,
			WorkspaceId:       workspaceID,
			InstanceId:        instanceID,
			Name:              upsert.Secret.GetName(),
			ManagedSecret:     upsert.Secret,
			ManagedSecretData: upsert.Data,
		})
	}, "UpdateManagedSecret")
	return err
}

func patchManagedSecretClearData(ctx context.Context, cli *AkpCli, instanceID, workspaceID string, upsert *types.ManagedSecretUpsert, keys []string) error {
	data := make(map[string]string, len(keys))
	for _, key := range keys {
		data[key] = ""
	}
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.PatchManagedSecretResponse, error) {
		return cli.Cli.PatchManagedSecret(ctx, &argocdv1.PatchManagedSecretRequest{
			OrganizationId:    cli.OrgId,
			WorkspaceId:       workspaceID,
			InstanceId:        instanceID,
			Name:              upsert.Secret.GetName(),
			ManagedSecret:     upsert.Secret,
			ManagedSecretData: data,
		})
	}, "PatchManagedSecret")
	return err
}

func listManagedSecretKeys(ctx context.Context, cli *AkpCli, instanceID, workspaceID string) (map[string][]string, error) {
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.ListInstanceManagedSecretsResponse, error) {
		return cli.Cli.ListInstanceManagedSecrets(ctx, &argocdv1.ListInstanceManagedSecretsRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspaceID,
			InstanceId:     instanceID,
		})
	}, "ListInstanceManagedSecrets")
	if err != nil {
		return nil, err
	}
	keys := make(map[string][]string, len(resp.GetManagedSecrets()))
	for _, secret := range resp.GetManagedSecrets() {
		if secret != nil {
			keys[secret.GetName()] = secret.GetSecretKeys()
		}
	}
	return keys, nil
}

func removedManagedSecretNames(state, plan map[string]*types.ManagedSecret) []string {
	var names []string
	for name, secret := range state {
		if secret == nil {
			continue
		}
		plannedSecret, ok := plan[name]
		if !ok || plannedSecret == nil {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

func deleteManagedSecret(ctx context.Context, cli *AkpCli, instanceID, workspaceID, name string) error {
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.DeleteManagedSecretResponse, error) {
		return cli.Cli.DeleteManagedSecret(ctx, &argocdv1.DeleteManagedSecretRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspaceID,
			InstanceId:     instanceID,
			Name:           name,
		})
	}, "DeleteManagedSecret")
	if err != nil {
		return errors.Wrapf(err, "Unable to delete managed secret %q", name)
	}
	return nil
}

func instanceRead(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, data *types.Instance) error {
	if data.ArgoCD == nil || data.ID.IsNull() || data.ID.ValueString() == "" {
		ctx = types.WithReadContext(ctx)
	}
	tflog.MaskLogStrings(ctx, data.GetSensitiveStrings(ctx, diags)...)
	return refreshState(ctx, diags, cli, data, instanceGetRequest(cli.OrgId, data.ID.ValueString(), data.Name.ValueString()), false)
}

func instanceDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, state *types.Instance) error {
	deleteCtx := ctx
	if _, deadlineSet := ctx.Deadline(); !deadlineSet {
		var cancel context.CancelFunc
		deleteCtx, cancel = context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
	}

	for {
		err := deleteWithCooldown(deleteCtx, func(ctx context.Context) (*argocdv1.DeleteInstanceResponse, error) {
			return cli.Cli.DeleteInstance(ctx, &argocdv1.DeleteInstanceRequest{
				Id:             state.ID.ValueString(),
				OrganizationId: cli.OrgId,
			})
		}, "DeleteInstance", 2*time.Second)
		if err == nil {
			return nil
		}
		if !isConnectedKargoAgentsDeleteError(err) {
			return fmt.Errorf("unable to delete Argo CD instance, got error: %s", err)
		}

		tflog.Warn(ctx, fmt.Sprintf("DeleteInstance blocked by connected Kargo agents for instance %s; retrying until detach completes", state.Name.ValueString()))

		select {
		case <-time.After(5 * time.Second):
		case <-deleteCtx.Done():
			return fmt.Errorf("unable to delete Argo CD instance, got error: %s", err)
		}
	}
}

func validateInstanceAIFeatures(ctx context.Context, plan *types.Instance) error {
	if plan.ArgoCD == nil || plan.ArgoCD.Spec.InstanceSpec.AkuityIntelligenceExtension == nil {
		return nil
	}
	aiExt := plan.ArgoCD.Spec.InstanceSpec.AkuityIntelligenceExtension
	if aiExt.Enabled.IsNull() || aiExt.Enabled.IsUnknown() {
		return nil
	}

	if !aiExt.Enabled.ValueBool() {
		if aiExt.AiSupportEngineerEnabled.ValueBool() ||
			len(aiExt.AllowedUsernames) > 0 ||
			len(aiExt.AllowedGroups) > 0 {
			return fmt.Errorf("AI configs are specified but AI Intelligence is disabled")
		}
	} else {
		if len(aiExt.AllowedUsernames) == 0 && len(aiExt.AllowedGroups) == 0 {
			tflog.Warn(ctx, "AI Intelligence is enabled but no allowed usernames or groups are specified")
		}
	}
	return nil
}

func instanceUpsert(ctx context.Context, cli *AkpCli, diagnostics *diag.Diagnostics, plan *types.Instance) (stateCanBeCommitted bool, err error) {
	lc := &ResourceLifecycle[types.Instance, *argocdv1.GetInstanceResponse, healthv1.StatusCode]{
		Apply: func(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.Instance) (bool, error) {
			tflog.MaskLogStrings(ctx, plan.GetSensitiveStrings(ctx, diagnostics)...)

			if err := validateInstanceAIFeatures(ctx, plan); err != nil {
				return false, err
			}

			workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, plan.Workspace.ValueString())
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get workspace. %s", err))
				return false, errors.New("Unable to get workspace")
			}

			apiReq := buildApplyRequest(ctx, diagnostics, plan, cli.OrgId, workspace.GetId())
			if diagnostics.HasError() {
				return false, errors.New("Unable to build Argo CD instance request")
			}
			tflog.Debug(ctx, fmt.Sprintf("Apply instance request: %s", apiReq.Argocd))
			_, err = retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.ApplyInstanceResponse, error) {
				return cli.Cli.ApplyInstance(ctx, apiReq)
			}, "ApplyInstance")
			if err != nil {
				// Export cannot reconstruct write-only fields such as secrets. Do
				// not commit the planned state unless the full apply succeeded.
				return false, errors.Wrap(err, "Unable to upsert Argo CD instance")
			}

			// ApplyInstance only honors workspace_id when creating an instance;
			// moving an existing one requires the dedicated workspace API.
			if _, err := ensureInstanceWorkspace(ctx, cli.Cli, cli.OrgId, plan.ID.ValueString(), plan.Name.ValueString(), workspace); err != nil {
				return true, err
			}

			plan.Workspace = workspaceStateValue(plan.Workspace, workspace)
			return true, nil
		},
		Get: func(ctx context.Context, plan *types.Instance) (*argocdv1.GetInstanceResponse, error) {
			return retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
				return cli.Cli.GetInstance(ctx, instanceGetRequest(cli.OrgId, plan.ID.ValueString(), plan.Name.ValueString()))
			}, "GetInstance")
		},
		GetStatus: func(resp *argocdv1.GetInstanceResponse) healthv1.StatusCode {
			if resp == nil || resp.Instance == nil {
				return healthv1.StatusCode_STATUS_CODE_UNKNOWN
			}
			return resp.Instance.GetHealthStatus().GetCode()
		},
		GetGeneration: func(resp *argocdv1.GetInstanceResponse) uint32 {
			if resp == nil || resp.Instance == nil {
				return 0
			}
			return resp.Instance.GetGeneration()
		},
		GetReconciliationDone: func(resp *argocdv1.GetInstanceResponse) bool {
			if resp == nil || resp.Instance == nil {
				return false
			}
			code := resp.Instance.GetReconciliationStatus().GetCode()
			return code == reconv1.StatusCode_STATUS_CODE_SUCCESSFUL
		},
		GetReconciliationFailed: func(resp *argocdv1.GetInstanceResponse) bool {
			if resp == nil || resp.Instance == nil {
				return false
			}
			return resp.Instance.GetReconciliationStatus().GetCode() == reconv1.StatusCode_STATUS_CODE_FAILED
		},
		TargetStatuses: []healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY},
		Refresh: func(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.Instance) error {
			return refreshState(ctx, diagnostics, cli, plan, instanceGetRequest(cli.OrgId, plan.ID.ValueString(), plan.Name.ValueString()), false)
		},
		RefreshAfterApplyError: func(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.Instance) error {
			return refreshStateAfterWorkspaceMoveError(ctx, diagnostics, cli, plan, instanceGetRequest(cli.OrgId, plan.ID.ValueString(), plan.Name.ValueString()))
		},
		ResourceName: func(plan *types.Instance) string {
			return fmt.Sprintf("Instance %s", plan.Name.ValueString())
		},
		StatusName:   "health",
		PollInterval: 10 * time.Second,
		Timeout:      10 * time.Minute,
	}

	return lc.Upsert(ctx, diagnostics, plan)
}

func instanceGetRequest(orgID, instanceID, instanceName string) *argocdv1.GetInstanceRequest {
	idType := idv1.Type_NAME
	id := instanceName
	if instanceID != "" {
		idType = idv1.Type_ID
		id = instanceID
	}
	return &argocdv1.GetInstanceRequest{
		OrganizationId: orgID,
		IdType:         idType,
		Id:             id,
	}
}

func buildApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, instance *types.Instance, orgID, workspaceID string) *argocdv1.ApplyInstanceRequest {
	idType := idv1.Type_NAME
	id := instance.Name.ValueString()

	if !instance.ID.IsNull() && instance.ID.ValueString() != "" {
		idType = idv1.Type_ID
		id = instance.ID.ValueString()
	}

	applyReq := &argocdv1.ApplyInstanceRequest{
		OrganizationId:                orgID,
		IdType:                        idType,
		Id:                            id,
		WorkspaceId:                   workspaceID,
		Argocd:                        buildArgoCD(ctx, diagnostics, instance),
		ArgocdConfigmap:               buildConfigMap(ctx, diagnostics, instance.ArgoCDConfigMap, "argocd-cm"),
		ArgocdRbacConfigmap:           buildConfigMap(ctx, diagnostics, instance.ArgoCDRBACConfigMap, "argocd-rbac-cm"),
		ArgocdSecret:                  buildSecret(ctx, diagnostics, instance.ArgoCDSecret, "argocd-secret", nil),
		ApplicationSetSecret:          buildSecret(ctx, diagnostics, instance.ApplicationSetSecret, "argocd-application-set-secret", nil),
		NotificationsConfigmap:        buildConfigMap(ctx, diagnostics, instance.NotificationsConfigMap, "argocd-notifications-cm"),
		NotificationsSecret:           buildSecret(ctx, diagnostics, instance.NotificationsSecret, "argocd-notifications-secret", nil),
		ImageUpdaterConfigmap:         buildConfigMap(ctx, diagnostics, instance.ImageUpdaterConfigMap, "argocd-image-updater-config"),
		ImageUpdaterSshConfigmap:      buildConfigMap(ctx, diagnostics, instance.ImageUpdaterSSHConfigMap, "argocd-image-updater-ssh-config"),
		ImageUpdaterSecret:            buildSecret(ctx, diagnostics, instance.ImageUpdaterSecret, "argocd-image-updater-secret", nil),
		ArgocdKnownHostsConfigmap:     buildConfigMap(ctx, diagnostics, instance.ArgoCDKnownHostsConfigMap, "argocd-ssh-known-hosts-cm"),
		ArgocdTlsCertsConfigmap:       buildConfigMap(ctx, diagnostics, instance.ArgoCDTLSCertsConfigMap, "argocd-tls-certs-cm"),
		RepoCredentialSecrets:         buildSecrets(ctx, diagnostics, instance.RepoCredentialSecrets, map[string]string{"argocd.argoproj.io/secret-type": "repository"}),
		RepoTemplateCredentialSecrets: buildSecrets(ctx, diagnostics, instance.RepoTemplateCredentialSecrets, map[string]string{"argocd.argoproj.io/secret-type": "repo-creds"}),
		ConfigManagementPlugins:       buildCMPs(ctx, diagnostics, instance.ConfigManagementPlugins),
		PruneResourceTypes:            []argocdv1.PruneResourceType{argocdv1.PruneResourceType_PRUNE_RESOURCE_TYPE_CONFIG_MANAGEMENT_PLUGINS},
	}

	if !instance.ArgoCDResources.IsUnknown() {
		processResources(
			ctx,
			diagnostics,
			instance.ArgoCDResources,
			argoResourceGroups,
			isArgoResourceValid,
			applyReq,
			"ArgoCD",
		)
	}
	return applyReq
}

func buildArgoCD(_ context.Context, diag *diag.Diagnostics, instance *types.Instance) *structpb.Struct {
	rawMap := types.TFToMapWithOverrides(instance.ArgoCD, types.OverridesMap, nil)
	if rawMap == nil {
		diag.AddError("Client Error", "Unable to convert Argo CD instance to map")
		return nil
	}
	rawMap["metadata"] = map[string]any{
		"name": instance.Name.ValueString(),
	}

	s, err := structpb.NewStruct(rawMap)
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance struct. %s", err))
		return nil
	}
	return s
}

func buildSecrets(ctx context.Context, diagnostics *diag.Diagnostics, secrets tftypes.Map, labels map[string]string) []*structpb.Struct {
	var res []*structpb.Struct
	var sMap map[string]tftypes.Map
	if secrets.IsNull() {
		return res
	}
	diagnostics.Append(secrets.ElementsAs(ctx, &sMap, true)...)
	for name, secret := range sMap {
		res = append(res, buildSecret(ctx, diagnostics, secret, name, labels))
	}
	return res
}

func buildConfigMap(ctx context.Context, diagnostics *diag.Diagnostics, cm tftypes.Map, name string) *structpb.Struct {
	if cm.IsNull() {
		return nil
	}
	apiModel := types.ToConfigMapAPIModel(ctx, diagnostics, name, cm)
	configMap, err := marshal.ApiModelToPBStruct(apiModel)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ConfigMap. %s", err))
		return nil
	}
	return configMap
}

func buildSecret(ctx context.Context, diagnostics *diag.Diagnostics, secret tftypes.Map, name string, labels map[string]string) *structpb.Struct {
	if secret.IsNull() {
		return nil
	}
	apiModel := types.ToSecretAPIModel(ctx, diagnostics, name, labels, secret)
	s, err := marshal.ApiModelToPBStruct(apiModel)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Secret. %s", err))
		return nil
	}
	return s
}

func buildCMPs(_ context.Context, diagnostics *diag.Diagnostics, cmps map[string]*types.ConfigManagementPlugin) []*structpb.Struct {
	var res []*structpb.Struct
	for name, cmp := range cmps {
		rawMap := types.BuildCMPMap(cmp, name)
		s, err := structpb.NewStruct(rawMap)
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ConfigManagementPlugin struct. %s", err))
			return nil
		}
		res = append(res, s)
	}
	return res
}

func refreshState(ctx context.Context, diagnostics *diag.Diagnostics, cli *AkpCli, instance *types.Instance, getInstanceReq *argocdv1.GetInstanceRequest, isDataSource bool) error {
	return refreshStateWithWorkspaceMode(ctx, diagnostics, cli, instance, getInstanceReq, isDataSource, false)
}

func refreshStateAfterWorkspaceMoveError(ctx context.Context, diagnostics *diag.Diagnostics, cli *AkpCli, instance *types.Instance, getInstanceReq *argocdv1.GetInstanceRequest) error {
	return refreshStateWithWorkspaceMode(ctx, diagnostics, cli, instance, getInstanceReq, false, true)
}

func refreshStateWithWorkspaceMode(ctx context.Context, diagnostics *diag.Diagnostics, cli *AkpCli, instance *types.Instance, getInstanceReq *argocdv1.GetInstanceRequest, isDataSource, strictWorkspace bool) error {
	tflog.Debug(ctx, fmt.Sprintf("Get instance request: %s", getInstanceReq))
	getInstanceResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
		return cli.Cli.GetInstance(ctx, getInstanceReq)
	}, "GetInstance")
	if err != nil {
		return errors.Wrap(err, "Unable to read Argo CD instance")
	}
	tflog.Debug(ctx, fmt.Sprintf("Get instance response: %s", getInstanceResp))
	instance.ID = tftypes.StringValue(getInstanceResp.Instance.Id)
	instance.Name = tftypes.StringValue(getInstanceResp.Instance.Name)
	// Always resolve the workspace from the API so out-of-band moves (e.g. via
	// the UI) surface as drift instead of being masked by the stored state.
	workspaceState, err := workspaceStateFromAPI(
		ctx,
		cli.OrgCli,
		cli.OrgId,
		getInstanceResp.Instance.GetWorkspaceId(),
		instance.Workspace,
		strictWorkspace,
	)
	if err != nil {
		return err
	}
	instance.Workspace = workspaceState
	exportReq := &argocdv1.ExportInstanceRequest{
		OrganizationId: getInstanceReq.OrganizationId,
		IdType:         idv1.Type_ID,
		Id:             instance.ID.ValueString(),
		WorkspaceId:    getInstanceResp.Instance.WorkspaceId,
	}
	tflog.Debug(ctx, fmt.Sprintf("Export instance request: %s", exportReq))
	exportResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.ExportInstanceResponse, error) {
		return argocdexport.ExportInstance(ctx, cli.Cli, exportReq)
	}, "ExportInstance")
	if err != nil {
		return errors.Wrap(err, "Unable to export Argo CD instance")
	}
	if err := instance.Update(ctx, diagnostics, exportResp, isDataSource); err != nil {
		return err
	}
	if !isDataSource {
		return refreshManagedSecrets(ctx, diagnostics, cli, instance, getInstanceResp.Instance.GetWorkspaceId())
	}
	return nil
}

func refreshManagedSecrets(ctx context.Context, diagnostics *diag.Diagnostics, cli *AkpCli, instance *types.Instance, workspaceID string) error {
	if instance.ManagedSecrets == nil {
		return nil
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.ListInstanceManagedSecretsResponse, error) {
		return cli.Cli.ListInstanceManagedSecrets(ctx, &argocdv1.ListInstanceManagedSecretsRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspaceID,
			InstanceId:     instance.ID.ValueString(),
		})
	}, "ListInstanceManagedSecrets")
	if err != nil {
		return errors.Wrap(err, "Unable to list managed secrets")
	}
	instance.ManagedSecrets = types.ToManagedSecretsTFModel(ctx, diagnostics, instance.ManagedSecrets, resp.GetManagedSecrets())
	return nil
}

func isArgoResourceValid(un *unstructured.Unstructured) error {
	return validateResource(un, "argoproj.io/v1alpha1", argoResourceGroups)
}
