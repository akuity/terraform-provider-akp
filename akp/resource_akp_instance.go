package akp

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/akuity/terraform-provider-akp/akp/marshal"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &AkpInstanceResource{}
var _ resource.ResourceWithImportState = &AkpInstanceResource{}

func NewAkpInstanceResource() resource.Resource {
	return &AkpInstanceResource{}
}

type AkpInstanceResource struct {
	akpCli *AkpCli
}

func (r *AkpInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *AkpInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	akpCli, ok := req.ProviderData.(*AkpCli)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *AkpCli, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.akpCli = akpCli
}

func (r *AkpInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating an instance")
	var plan types.Instance

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.upsert(ctx, &resp.Diagnostics, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	} else {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *AkpInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading an instance")
	var data types.Instance
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.MaskLogStrings(ctx, data.GetSensitiveStrings(ctx, &resp.Diagnostics)...)
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	err := refreshState(ctx, &resp.Diagnostics, r.akpCli.Cli, &data, r.akpCli.OrgId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	} else {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *AkpInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating an instance")
	var plan types.Instance

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.upsert(ctx, &resp.Diagnostics, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	} else {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *AkpInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting an instance")
	var state types.Instance

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	_, err := r.akpCli.Cli.DeleteInstance(ctx, &argocdv1.DeleteInstanceRequest{
		Id:             state.ID.ValueString(),
		OrganizationId: r.akpCli.OrgId,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Argo CD instance, got error: %s", err))
		return
	}
	// Give it some time to remove the instance. This is useful when the terraform provider is performing a replace operation, to give it enough time to destroy the previous instance.
	time.Sleep(2 * time.Second)
}

func (r *AkpInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *AkpInstanceResource) upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.Instance) error {
	// Mark sensitive secret data
	tflog.MaskLogStrings(ctx, plan.GetSensitiveStrings(ctx, diagnostics)...)

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	apiReq := buildApplyRequest(ctx, diagnostics, plan, r.akpCli.OrgId)
	tflog.Debug(ctx, fmt.Sprintf("Apply instance request: %s", apiReq.Argocd))
	_, err := r.akpCli.Cli.ApplyInstance(ctx, apiReq)
	if err != nil {
		return errors.Wrap(err, "Unable to upsert Argo CD instance")
	}

	return refreshState(ctx, diagnostics, r.akpCli.Cli, plan, r.akpCli.OrgId)
}

func buildApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, instance *types.Instance, orgID string) *argocdv1.ApplyInstanceRequest {
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
	return applyReq
}

func buildArgoCD(ctx context.Context, diag *diag.Diagnostics, instance *types.Instance) *structpb.Struct {
	apiArgoCD := instance.ArgoCD.ToArgoCDAPIModel(ctx, diag, instance.Name.ValueString())
	s, err := marshal.ApiModelToPBStruct(apiArgoCD)
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
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

func buildCMPs(ctx context.Context, diagnostics *diag.Diagnostics, cmps map[string]*types.ConfigManagementPlugin) []*structpb.Struct {
	var res []*structpb.Struct
	for name, cmp := range cmps {
		apiModel := cmp.ToConfigManagementPluginAPIModel(ctx, diagnostics, name)
		s, err := marshal.ApiModelToPBStruct(apiModel)
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ConfigManagementPlugin. %s", err))
			return nil
		}
		res = append(res, s)
	}
	return res
}

func refreshState(ctx context.Context, diagnostics *diag.Diagnostics, client argocdv1.ArgoCDServiceGatewayClient, instance *types.Instance, orgID string) error {
	getInstanceReq := &argocdv1.GetInstanceRequest{
		OrganizationId: orgID,
		IdType:         idv1.Type_NAME,
		Id:             instance.Name.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("Get instance request: %s", getInstanceReq))
	getInstanceResp, err := client.GetInstance(ctx, getInstanceReq)
	if err != nil {
		return errors.Wrap(err, "Unable to read Argo CD instance")
	}
	tflog.Debug(ctx, fmt.Sprintf("Get instance response: %s", getInstanceResp))
	instance.ID = tftypes.StringValue(getInstanceResp.Instance.Id)
	exportReq := &argocdv1.ExportInstanceRequest{
		OrganizationId: orgID,
		IdType:         idv1.Type_NAME,
		Id:             instance.Name.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("Export instance request: %s", exportReq))
	exportResp, err := client.ExportInstance(ctx, exportReq)
	if err != nil {
		return errors.Wrap(err, "Unable to export Argo CD instance")
	}
	tflog.Debug(ctx, fmt.Sprintf("Export instance response: %s", exportResp))
	return instance.Update(ctx, diagnostics, exportResp)
}
