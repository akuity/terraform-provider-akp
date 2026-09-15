//go:build !acc

package akp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"sigs.k8s.io/yaml"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	model "github.com/akuity/terraform-provider-akp/akp/types"
)

type ClusterUpgradeSuite struct{ suite.Suite }

func TestClusterUpgradeSuite(t *testing.T) { suite.Run(t, new(ClusterUpgradeSuite)) }

func (s *ClusterUpgradeSuite) TestLegacyStateRefreshThenUpdate() {
	s.T().Setenv("TF_CLI_CONFIG_FILE", os.DevNull)
	for _, user := range []string{"", "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\ncommonAnnotations:\n  example.com/owner: user\n"} {
		name := "generated patches"
		if user != "" {
			name = "user kustomization"
		}
		s.Run(name, func() {
			backend := &upgradeClusterBackend{}
			legacy, current := upgradeFactories(backend, true), upgradeFactories(backend, false)
			config := upgradeClusterConfig("2Gi", user)
			check := resource.TestCheckNoResourceAttr("akp_cluster.test", "spec.data.kustomization")
			if user != "" {
				check = resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.kustomization", user)
			}
			resource.UnitTest(s.T(), resource.TestCase{Steps: []resource.TestStep{
				{ProtoV6ProviderFactories: legacy, Config: config, Check: resource.TestCheckResourceAttrSet("akp_cluster.test", "spec.data.kustomization")},
				{ProtoV6ProviderFactories: current, Config: config, PlanOnly: true},
				{ProtoV6ProviderFactories: current, Config: config, Check: check},
				{ProtoV6ProviderFactories: current, Config: upgradeClusterConfig("3Gi", user), Check: resource.ComposeAggregateTestCheckFunc(check, resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.memory", "3Gi"))},
				{ProtoV6ProviderFactories: current, Config: upgradeClusterConfig("3Gi", user), PlanOnly: true},
			}})
			backend.Lock()
			s.Equal(2, backend.writes, "refresh migration must not write to the API; only create and the size update may write")
			backend.Unlock()
		})
	}
}

func (s *ClusterUpgradeSuite) TestRedundantConfiguredPatchesRequireRemoval() {
	s.T().Setenv("TF_CLI_CONFIG_FILE", os.DevNull)
	custom := &model.CustomAgentSizeConfig{ApplicationController: &model.AppControllerCustomAgentSizeConfig{Cpu: tftypes.StringValue("1000m"), Memory: tftypes.StringValue("2Gi")}}
	generated, err := model.GenerateExpectedKustomization(custom, "")
	s.Require().NoError(err)
	backend := &upgradeClusterBackend{}
	legacy, current := upgradeFactories(backend, true), upgradeFactories(backend, false)
	resource.UnitTest(s.T(), resource.TestCase{Steps: []resource.TestStep{
		{ProtoV6ProviderFactories: legacy, Config: upgradeClusterConfig("2Gi", "")},
		// Old state makes this redundant configuration appear stable until an update.
		{ProtoV6ProviderFactories: legacy, Config: upgradeClusterConfig("2Gi", generated), PlanOnly: true},
		{ProtoV6ProviderFactories: current, Config: upgradeClusterConfig("2Gi", generated), PlanOnly: true, ExpectNonEmptyPlan: true},
		{ProtoV6ProviderFactories: current, Config: upgradeClusterConfig("2Gi", generated), ExpectError: regexp.MustCompile("conflicts with custom_agent_size_config")},
		{ProtoV6ProviderFactories: current, Config: upgradeClusterConfig("3Gi", ""), Check: resource.TestCheckNoResourceAttr("akp_cluster.test", "spec.data.kustomization")},
		{ProtoV6ProviderFactories: current, Config: upgradeClusterConfig("3Gi", ""), PlanOnly: true},
	}})
}

func upgradeClusterConfig(memory, user string) string {
	kustomization := ""
	if user != "" {
		kustomization = fmt.Sprintf("kustomization = %q", user)
	}
	return fmt.Sprintf(`
provider "akp" { org_name = "test" }
resource "akp_cluster" "test" {
 instance_id = "instance"
 name = "custom-cluster"
 namespace = "agent"
 spec = {
  data = {
   size = "custom"
   custom_agent_size_config = {
    application_controller = { cpu = "1000m", memory = %q }
   }
   %s
  }
 }
}
`, memory, kustomization)
}

// Use Terraform Core and the production schema, request builder, and state
// conversion. Test CRUD methods replace the remote lifecycle; legacy mode
// reproduces generated-patches-in-state behavior without running an old binary.
type upgradeClusterBackend struct {
	sync.Mutex
	cluster *argocdv1.Cluster
	writes  int
}

type upgradeProvider struct {
	provider.Provider
	backend *upgradeClusterBackend
	legacy  bool
}

func upgradeFactories(backend *upgradeClusterBackend, legacy bool) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){"akp": providerserver.NewProtocol6WithError(&upgradeProvider{Provider: New("test")(), backend: backend, legacy: legacy})}
}

func (*upgradeProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}

func (p *upgradeProvider) Resources(context.Context) []func() fwresource.Resource {
	return []func() fwresource.Resource{func() fwresource.Resource {
		return &upgradeClusterResource{Resource: NewAkpClusterResource(), backend: p.backend, legacy: p.legacy}
	}}
}

type upgradeClusterResource struct {
	fwresource.Resource
	backend *upgradeClusterBackend
	legacy  bool
}

func (r *upgradeClusterResource) Create(ctx context.Context, req fwresource.CreateRequest, resp *fwresource.CreateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *upgradeClusterResource) Update(ctx context.Context, req fwresource.UpdateRequest, resp *fwresource.UpdateResponse) {
	r.apply(ctx, req.Plan, &resp.State, &resp.Diagnostics)
}

func (r *upgradeClusterResource) Delete(context.Context, fwresource.DeleteRequest, *fwresource.DeleteResponse) {
	r.backend.Lock()
	defer r.backend.Unlock()
	r.backend.cluster = nil
}

func (r *upgradeClusterResource) Read(ctx context.Context, req fwresource.ReadRequest, resp *fwresource.ReadResponse) {
	var state model.Cluster
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.backend.Lock()
	defer r.backend.Unlock()
	if r.backend.cluster == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.refresh(ctx, &state, &resp.State, &resp.Diagnostics)
}

func (r *upgradeClusterResource) apply(ctx context.Context, plan tfsdk.Plan, state *tfsdk.State, diags *diag.Diagnostics) {
	var data model.Cluster
	diags.Append(plan.Get(ctx, &data)...)
	if diags.HasError() {
		return
	}
	request := buildClusterApplyRequest(ctx, diags, &data, "org")
	if diags.HasError() {
		return
	}
	spec := request.Clusters[0].AsMap()["spec"].(map[string]any)
	// The declarative API accepts lowercase sizes; its read response uses protobuf enums.
	spec["data"].(map[string]any)["size"] = "CLUSTER_SIZE_LARGE"
	if connectivity, ok := spec["data"].(map[string]any)["connectivity"].(string); ok {
		spec["data"].(map[string]any)["connectivity"] = "CONNECTIVITY_" + strings.ToUpper(connectivity)
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		diags.AddError("Test API encode", err.Error())
		return
	}
	api := &argocdv1.Cluster{}
	if err = protojson.Unmarshal(raw, api); err != nil {
		diags.AddError("Test API decode", err.Error())
		return
	}
	api.Id, api.Name = "cluster-id", data.Name.ValueString()
	api.Data.Namespace = data.Namespace.ValueString()
	r.backend.Lock()
	defer r.backend.Unlock()
	r.backend.cluster = api
	r.backend.writes++
	r.refresh(ctx, &data, state, diags)
}

func (r *upgradeClusterResource) refresh(ctx context.Context, data *model.Cluster, state *tfsdk.State, diags *diag.Diagnostics) {
	api := proto.Clone(r.backend.cluster).(*argocdv1.Cluster)
	prior := model.DeepCopy(data)
	data.Update(ctx, diags, api, prior)
	if r.legacy && prior.Spec.Data.Kustomization.ValueString() == "" {
		raw, err := yaml.Marshal(api.Data.Kustomization.AsMap())
		if err != nil {
			diags.AddError("Legacy state encode", err.Error())
			return
		}
		data.Spec.Data.Kustomization = tftypes.StringValue(string(raw))
	} else if r.legacy {
		data.Spec.Data.Kustomization = prior.Spec.Data.Kustomization
	}
	diags.Append(state.Set(ctx, data)...)
}
