package akp

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

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
		TypeNameSuffix: "instance",
		SchemaFunc:     instanceSchema,
		CreateFunc:     instanceCreateOrUpdate,
		ReadFunc:       instanceRead,
		UpdateFunc:     instanceCreateOrUpdate,
		DeleteFunc:     instanceDelete,
		ImportStateFunc: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
		},
	}
}

func instanceCreateOrUpdate(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *types.Instance) (*types.Instance, error) {
	plannedCM := plan.ArgoCDConfigMap
	applied, err := instanceUpsert(ctx, cli, diags, plan)
	if applied {
		plan.ArgoCDConfigMap = types.FilterMapToPlannedKeys(ctx, diags, plan.ArgoCDConfigMap, plannedCM)
		return plan, err
	}
	return nil, err
}

func instanceRead(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, data *types.Instance) error {
	if data.ArgoCD == nil || data.ID.IsNull() || data.ID.ValueString() == "" {
		ctx = types.WithReadContext(ctx)
	}
	tflog.MaskLogStrings(ctx, data.GetSensitiveStrings(ctx, diags)...)
	return refreshState(ctx, diags, cli, data, &argocdv1.GetInstanceRequest{
		OrganizationId: cli.OrgId,
		IdType:         idv1.Type_NAME,
		Id:             data.Name.ValueString(),
	}, false)
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

func instanceUpsert(ctx context.Context, cli *AkpCli, diagnostics *diag.Diagnostics, plan *types.Instance) (applied bool, err error) {
	lc := &ResourceLifecycle[types.Instance, *argocdv1.GetInstanceResponse, healthv1.StatusCode]{
		Apply: func(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.Instance) error {
			tflog.MaskLogStrings(ctx, plan.GetSensitiveStrings(ctx, diagnostics)...)

			if err := validateInstanceAIFeatures(ctx, plan); err != nil {
				return err
			}

			workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, plan.Workspace.ValueString())
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get workspace. %s", err))
				return errors.New("Unable to get workspace")
			}

			apiReq := buildApplyRequest(ctx, diagnostics, plan, cli.OrgId, workspace.GetId())
			tflog.Debug(ctx, fmt.Sprintf("Apply instance request: %s", apiReq.Argocd))
			_, err = retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.ApplyInstanceResponse, error) {
				return cli.Cli.ApplyInstance(ctx, apiReq)
			}, "ApplyInstance")
			if err != nil {
				return errors.Wrap(err, "Unable to upsert Argo CD instance")
			}

			if plan.Workspace.ValueString() == "" {
				plan.Workspace = tftypes.StringValue(workspace.GetName())
			}
			return nil
		},
		Get: func(ctx context.Context, plan *types.Instance) (*argocdv1.GetInstanceResponse, error) {
			return retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
				return cli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
					OrganizationId: cli.OrgId,
					Id:             plan.Name.ValueString(),
					IdType:         idv1.Type_NAME,
				})
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
			return refreshState(ctx, diagnostics, cli, plan, &argocdv1.GetInstanceRequest{
				OrganizationId: cli.OrgId,
				IdType:         idv1.Type_NAME,
				Id:             plan.Name.ValueString(),
			}, false)
		},
		ResourceName: func(plan *types.Instance) string {
			return fmt.Sprintf("Instance %s", plan.Name.ValueString())
		},
		StatusName:   "health",
		PollInterval: 10 * time.Second,
		Timeout:      5 * time.Minute,
	}

	return lc.Upsert(ctx, diagnostics, plan)
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
	tflog.Debug(ctx, fmt.Sprintf("Get instance request: %s", getInstanceReq))
	getInstanceResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
		return cli.Cli.GetInstance(ctx, getInstanceReq)
	}, "GetInstance")
	if err != nil {
		return errors.Wrap(err, "Unable to read Argo CD instance")
	}
	tflog.Debug(ctx, fmt.Sprintf("Get instance response: %s", getInstanceResp))
	instance.ID = tftypes.StringValue(getInstanceResp.Instance.Id)
	if instance.Workspace.IsNull() || instance.Workspace.ValueString() == "" {
		workspace, wErr := getWorkspaceByID(ctx, cli.OrgCli, cli.OrgId, getInstanceResp.Instance.WorkspaceId)
		if wErr == nil && workspace != nil {
			instance.Workspace = tftypes.StringValue(workspace.GetName())
		}
	}
	exportReq := &argocdv1.ExportInstanceRequest{
		OrganizationId: getInstanceReq.OrganizationId,
		IdType:         idv1.Type_ID,
		Id:             instance.ID.ValueString(),
		WorkspaceId:    getInstanceResp.Instance.WorkspaceId,
	}
	tflog.Debug(ctx, fmt.Sprintf("Export instance request: %s", exportReq))
	exportResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.ExportInstanceResponse, error) {
		return cli.Cli.ExportInstance(ctx, exportReq)
	}, "ExportInstance")
	if err != nil {
		return errors.Wrap(err, "Unable to export Argo CD instance")
	}
	err = instance.Update(ctx, diagnostics, exportResp, isDataSource)
	return err
}

func isArgoResourceValid(un *unstructured.Unstructured) error {
	return validateResource(un, "argoproj.io/v1alpha1", argoResourceGroups)
}
