package akp

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/slices"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/conversion"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &AkpInstanceDataSource{}

func NewAkpInstanceDataSource() datasource.DataSource {
	return &AkpInstanceDataSource{}
}

// AkpInstanceDataSource defines the data source implementation.
type AkpInstanceDataSource struct {
	akpCli *AkpCli
}

func (r *AkpInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *AkpInstanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *AkpInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Info(ctx, "Reading an Argo CD Instance")

	var state types.Instance
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	// Save data into Terraform state
	r.refresh(ctx, resp.Diagnostics, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AkpInstanceDataSource) refresh(ctx context.Context, diagnostics diag.Diagnostics, x *types.Instance) {

	exportResp, err := r.akpCli.Cli.ExportInstance(ctx, &argocdv1.ExportInstanceRequest{
		OrganizationId: r.akpCli.OrgId,
		IdType:         idv1.Type_NAME,
		Id:             x.ID.ValueString(),
	})
	tflog.Debug(ctx, fmt.Sprintf("Export Resp: %s", exportResp.String()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
		return
	}
	var argoCD *v1alpha1.ArgoCD
	err = conversion.RemarshalTo(exportResp.Argocd.AsMap(), &argoCD)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
		return
	}

	planClusters := map[string]types.Cluster{}
	for _, c := range x.Clusters {
		planClusters[c.Name.ValueString()] = c
	}
	var clusters []types.Cluster
	for _, cluster := range exportResp.Clusters {
		var c *v1alpha1.Cluster
		err = conversion.RemarshalTo(cluster.AsMap(), &c)
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Argo CD instance. %s", err))
			return
		}
		cl := *conversion.ToClusterTFModel(ctx, diagnostics, c)
		cl.Manifests = r.getManifests(ctx, diagnostics, r.akpCli.OrgId, x.ID.ValueString(), cl.Name.ValueString())
		cl.KubeConfig = planClusters[cl.Name.ValueString()].KubeConfig

		clusters = append(clusters, cl)
	}

	x.ArgoCD = *conversion.ToArgoCDTFModel(ctx, diagnostics, argoCD)
	x.Clusters = clusters

	x.ArgoCDConfigMap = buildTFConfigMap(ctx, diagnostics, x.ArgoCDConfigMap, exportResp.ArgocdConfigmap, "argocd-cm")
	x.ArgoCDRBACConfigMap = buildTFConfigMap(ctx, diagnostics, x.ArgoCDRBACConfigMap, exportResp.ArgocdRbacConfigmap, "argocd-rbac-cm")
	x.NotificationsConfigMap = buildTFConfigMap(ctx, diagnostics, x.NotificationsConfigMap, exportResp.NotificationsConfigmap, "argocd-notifications-cm")
	x.ImageUpdaterConfigMap = buildTFConfigMap(ctx, diagnostics, x.ImageUpdaterConfigMap, exportResp.ImageUpdaterConfigmap, "argocd-image-updater-config")
	x.ImageUpdaterSSHConfigMap = buildTFConfigMap(ctx, diagnostics, x.ImageUpdaterSSHConfigMap, exportResp.ImageUpdaterSshConfigmap, "argocd-image-updater-ssh-config")
	x.ArgoCDTLSCertsConfigMap = buildTFConfigMap(ctx, diagnostics, x.ArgoCDTLSCertsConfigMap, exportResp.ArgocdTlsCertsConfigmap, "argocd-tls-certs-cm")
	tflog.Debug(ctx, fmt.Sprintf("refreash---------%+v", x))

}

func (d *AkpInstanceDataSource) getManifests(ctx context.Context, diags diag.Diagnostics, orgId, instanceId, Id string) tftypes.String {
	clusterReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: orgId,
		InstanceId:     instanceId,
		Id:             Id,
		IdType:         idv1.Type_NAME,
	}
	tflog.Debug(ctx, fmt.Sprintf("Api Request: %s", clusterReq))
	clusterResp, err := d.akpCli.Cli.GetInstanceCluster(ctx, clusterReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Response: %s", clusterResp))
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read instance clusters, got error: %s", err))
		return tftypes.StringNull()
	}
	cluster, err := d.waitClusterReconStatus(ctx, clusterResp.GetCluster(), instanceId)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to check cluster reconciliation status. %s", err))
		return tftypes.StringNull()
	}
	apiReq := &argocdv1.GetInstanceClusterManifestsRequest{
		OrganizationId: orgId,
		InstanceId:     instanceId,
		Id:             cluster.Id,
	}
	tflog.Debug(ctx, fmt.Sprintf("------manifest apiReq: %s", apiReq))
	apiResp, err := d.akpCli.Cli.GetInstanceClusterManifests(ctx, apiReq)
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("------manifest apiReq err: %s", err.Error()))
		diags.AddError("Akuity API error", fmt.Sprintf("Unable to download manifests: %s", err))
		return tftypes.StringNull()
	}
	tflog.Debug(ctx, fmt.Sprintf("-------manifest apiResp: %s", apiResp))
	return tftypes.StringValue(string(apiResp.GetData()))
}

func (d *AkpInstanceDataSource) waitClusterReconStatus(ctx context.Context, cluster *argocdv1.Cluster, instanceId string) (*argocdv1.Cluster, error) {
	reconStatus := cluster.GetReconciliationStatus()
	breakStatusesRecon := []reconv1.StatusCode{reconv1.StatusCode_STATUS_CODE_SUCCESSFUL, reconv1.StatusCode_STATUS_CODE_FAILED}

	for !slices.Contains(breakStatusesRecon, reconStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiResp, err := d.akpCli.Cli.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
			OrganizationId: d.akpCli.OrgId,
			InstanceId:     instanceId,
			Id:             cluster.Id,
			IdType:         idv1.Type_ID,
		})
		if err != nil {
			return nil, err
		}
		cluster = apiResp.GetCluster()
		reconStatus = cluster.GetReconciliationStatus()
		tflog.Debug(ctx, fmt.Sprintf("Cluster recon status: %s", reconStatus.String()))
	}
	return cluster, nil
}
