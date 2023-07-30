package akp

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
	var plan types.Instance

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upsert(ctx, &resp.Diagnostics, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data types.Instance
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.MaskLogStrings(ctx, data.GetSensitiveStrings()...)
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	refreshState(ctx, &resp.Diagnostics, r.akpCli.Cli, &data, r.akpCli.OrgId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AkpInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan types.Instance

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upsert(ctx, &resp.Diagnostics, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AkpInstanceResource) upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.Instance) {
	// Mark sensitive secret data
	tflog.MaskLogStrings(ctx, plan.GetSensitiveStrings()...)

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	apiReq := buildApplyRequest(ctx, diagnostics, plan, r.akpCli.OrgId)
	_, err := r.akpCli.Cli.ApplyInstance(ctx, apiReq)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to upsert Argo CD instance. %s", err))
		return
	}

	refreshState(ctx, diagnostics, r.akpCli.Cli, plan, r.akpCli.OrgId)
	if diagnostics.HasError() {
		return
	}
}

func buildApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, instance *types.Instance, orgID string) *argocdv1.ApplyInstanceRequest {
	applyReq := &argocdv1.ApplyInstanceRequest{
		OrganizationId:                orgID,
		IdType:                        idv1.Type_NAME,
		Id:                            instance.Name.ValueString(),
		Argocd:                        buildArgoCD(ctx, diagnostics, instance),
		ArgocdConfigmap:               buildConfigMap(ctx, diagnostics, instance.ArgoCDConfigMap, "argocd-cm"),
		ArgocdRbacConfigmap:           buildConfigMap(ctx, diagnostics, instance.ArgoCDRBACConfigMap, "argocd-rbac-cm"),
		ArgocdSecret:                  buildSecret(ctx, diagnostics, instance.ArgoCDSecret, "argocd-secret"),
		NotificationsConfigmap:        buildConfigMap(ctx, diagnostics, instance.NotificationsConfigMap, "argocd-notifications-cm"),
		NotificationsSecret:           buildSecret(ctx, diagnostics, instance.NotificationsSecret, "argocd-notifications-secret"),
		ImageUpdaterConfigmap:         buildConfigMap(ctx, diagnostics, instance.ImageUpdaterConfigMap, "argocd-image-updater-config"),
		ImageUpdaterSshConfigmap:      buildConfigMap(ctx, diagnostics, instance.ImageUpdaterSSHConfigMap, "argocd-image-updater-ssh-config"),
		ImageUpdaterSecret:            buildSecret(ctx, diagnostics, instance.ImageUpdaterSecret, "argocd-image-updater-secret"),
		ArgocdKnownHostsConfigmap:     buildConfigMap(ctx, diagnostics, instance.ArgoCDKnownHostsConfigMap, "argocd-ssh-known-hosts-cm"),
		ArgocdTlsCertsConfigmap:       buildConfigMap(ctx, diagnostics, instance.ArgoCDTLSCertsConfigMap, "argocd-tls-certs-cm"),
		RepoCredentialSecrets:         buildSecrets(ctx, diagnostics, instance.RepoCredentialSecrets),
		RepoTemplateCredentialSecrets: buildSecrets(ctx, diagnostics, instance.RepoTemplateCredentialSecrets),
		PruneRepoCredentialSecrets:    false,
		PruneClusters:                 false,
	}
	return applyReq
}

func buildArgoCD(ctx context.Context, diag *diag.Diagnostics, instance *types.Instance) *structpb.Struct {
	apiArgoCD := instance.ArgoCD.ToArgoCDAPIModel(ctx, diag, instance.Name.ValueString())
	argocdMap := map[string]interface{}{}
	if err := marshal.RemarshalTo(apiArgoCD, &argocdMap); err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to creat Argo CD instance. %s", err))
	}
	s, err := structpb.NewStruct(argocdMap)
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return nil
	}
	return s
}

func buildSecrets(ctx context.Context, diagnostics *diag.Diagnostics, secrets []types.Secret) []*structpb.Struct {
	var res []*structpb.Struct
	for _, secret := range secrets {
		res = append(res, buildSecret(ctx, diagnostics, &secret, secret.Name.ValueString()))
	}
	return res
}

func buildConfigMap(ctx context.Context, diagnostics *diag.Diagnostics, cmObj tftypes.Object, name string) *structpb.Struct {
	cm := &types.ConfigMap{}
	diagnostics.Append(cmObj.As(ctx, cm, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})...)
	if diagnostics.HasError() {
		return nil
	}
	if cm == nil || cm.Data.IsNull() {
		return nil
	}
	apiModel := cm.ToConfigMapAPIModel(ctx, diagnostics, name)
	m := map[string]interface{}{}
	if err := marshal.RemarshalTo(apiModel, &m); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ConfigMap. %s", err))
	}
	configMap, err := structpb.NewStruct(m)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ConfigMap. %s", err))
		return nil
	}
	return configMap
}

func buildSecret(ctx context.Context, diagnostics *diag.Diagnostics, secret *types.Secret, name string) *structpb.Struct {
	if secret == nil {
		return nil
	}
	apiModel := secret.ToSecretAPIModel(ctx, diagnostics, name)
	m := map[string]interface{}{}
	if err := marshal.RemarshalTo(apiModel, &m); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Secret. %s", err))
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Secret. %s", err))
		return nil
	}
	return s
}

func refreshState(ctx context.Context, diagnostics *diag.Diagnostics, client argocdv1.ArgoCDServiceGatewayClient, instance *types.Instance, orgID string) {
	getInstanceReq := &argocdv1.GetInstanceRequest{
		OrganizationId: orgID,
		IdType:         idv1.Type_NAME,
		Id:             instance.Name.ValueString(),
	}
	getInstanceResp, err := client.GetInstance(ctx, getInstanceReq)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD instance: %s", err))
		return
	}
	instance.ID = tftypes.StringValue(getInstanceResp.Instance.Id)
	exportResp, err := client.ExportInstance(ctx, &argocdv1.ExportInstanceRequest{
		OrganizationId: orgID,
		IdType:         idv1.Type_NAME,
		Id:             instance.Name.ValueString(),
	})
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to export Argo CD instance. %s", err))
		return
	}
	instance.Update(ctx, diagnostics, exportResp)
}
