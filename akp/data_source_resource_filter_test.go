//go:build !acc

package akp

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	fwtypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/akuity/api-client-go/pkg/api/argocdexport"
	"github.com/akuity/api-client-go/pkg/api/gateway/accesscontrol"
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	"github.com/akuity/api-client-go/pkg/api/kargoexport"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

const rfInstanceName = "filter-test"

var rfSetType = tftypes.Set{ElementType: tftypes.String}

func rfStruct(m map[string]any) *structpb.Struct {
	s, err := structpb.NewStruct(m)
	if err != nil {
		panic(err)
	}
	return s
}

func rfObject(apiVersion, kind, namespace, name string) *structpb.Struct {
	metadata := map[string]any{"name": name}
	if namespace != "" {
		metadata["namespace"] = namespace
	}
	return rfStruct(map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   metadata,
		"spec":       map[string]any{"description": kind + " " + namespace + "/" + name},
	})
}

// rfListed is what a filtering portal reads of objs: the ones named in resources.
func rfListed(objs []*structpb.Struct, resources []string) []*structpb.Struct {
	return slices.DeleteFunc(slices.Clone(objs), func(o *structpb.Struct) bool {
		f := o.GetFields()
		group, _, found := strings.Cut(f["apiVersion"].GetStringValue(), "/")
		if !found {
			group = ""
		}
		meta := f["metadata"].GetStructValue().GetFields()
		entry := strings.Join([]string{group, f["kind"].GetStringValue(), meta["namespace"].GetStringValue(), meta["name"].GetStringValue()}, "/")
		return !slices.Contains(resources, entry)
	})
}

func rfArgoExport() *argocdv1.ExportInstanceResponse {
	app := func(ns, name string) *structpb.Struct {
		return rfObject("argoproj.io/v1alpha1", "Application", ns, name)
	}
	project := func(name string) *structpb.Struct {
		return rfObject("argoproj.io/v1alpha1", "AppProject", "argocd", name)
	}
	return &argocdv1.ExportInstanceResponse{
		Argocd:          rfStruct(map[string]any{"spec": map[string]any{"version": "v3.1.0", "instanceSpec": map[string]any{"subdomain": "filter-test"}}}),
		ArgocdConfigmap: rfStruct(map[string]any{"exec.enabled": "true"}),
		Applications:    []*structpb.Struct{app("argocd", "guestbook"), app("apps", "guestbook"), app("argocd", "billing")},
		ApplicationSets: []*structpb.Struct{rfObject("argoproj.io/v1alpha1", "ApplicationSet", "argocd", "fleet")},
		AppProjects:     []*structpb.Struct{project("default"), project("team-a"), project("team-b")},
		ValidatingAdmissionPolicies: []*structpb.Struct{
			rfObject("admissionregistration.k8s.io/v1", "ValidatingAdmissionPolicy", "", "require-labels"),
			rfObject("admissionregistration.k8s.io/v1", "ValidatingAdmissionPolicy", "", "deny-latest"),
		},
		ValidatingAdmissionPolicyBindings: []*structpb.Struct{
			rfObject("admissionregistration.k8s.io/v1", "ValidatingAdmissionPolicyBinding", "", "require-labels-binding"),
		},
	}
}

func rfKargoExport() *kargov1.ExportKargoInstanceResponse {
	kargo := func(kind, ns, name string) *structpb.Struct {
		return rfObject("kargo.akuity.io/v1alpha1", kind, ns, name)
	}
	return &kargov1.ExportKargoInstanceResponse{
		Kargo:          rfStruct(map[string]any{"spec": map[string]any{"version": "v1.7.0", "kargoInstanceSpec": map[string]any{}}}),
		KargoConfigmap: rfStruct(map[string]any{"adminAccountEnabled": "true"}),
		Projects:       []*structpb.Struct{kargo("Project", "", "proj-a"), kargo("Project", "", "proj-b")},
		Stages:         []*structpb.Struct{kargo("Stage", "proj-a", "dev"), kargo("Stage", "proj-b", "dev")},
		Warehouses:     []*structpb.Struct{kargo("Warehouse", "proj-a", "main")},
		Configmaps:     []*structpb.Struct{rfObject("v1", "ConfigMap", "proj-a", "shared")},
		ValidatingAdmissionPolicies: []*structpb.Struct{
			rfObject("admissionregistration.k8s.io/v1", "ValidatingAdmissionPolicy", "", "require-labels"),
		},
	}
}

// rfArgoCli serves rfArgoExport. A current portal streams it, reading only the listed
// Applications, ApplicationSets, and AppProjects but every admission policy. An older portal
// has only the unary export and ignores the filter.
type rfArgoCli struct {
	argocdv1.ArgoCDServiceGatewayClient
	current  bool
	calls    int
	requests []*argocdv1.ExportInstanceRequest
}

func (f *rfArgoCli) GetInstance(_ context.Context, req *argocdv1.GetInstanceRequest) (*argocdv1.GetInstanceResponse, error) {
	f.calls++
	return &argocdv1.GetInstanceResponse{Instance: &argocdv1.Instance{Id: "argocd-id", Name: req.GetId(), WorkspaceId: "workspace-id"}}, nil
}

func (f *rfArgoCli) ListInstanceManagedSecrets(context.Context, *argocdv1.ListInstanceManagedSecretsRequest) (*argocdv1.ListInstanceManagedSecretsResponse, error) {
	return &argocdv1.ListInstanceManagedSecretsResponse{ManagedSecrets: []*argocdv1.ManagedSecret{{Name: "registry", SecretKeys: []string{"token"}}}}, nil
}

func (f *rfArgoCli) ExportInstanceStream(_ context.Context, streamReq *argocdv1.ExportInstanceStreamRequest) (<-chan *argocdv1.ExportInstanceStreamResponse, <-chan error, error) {
	if !f.current {
		return nil, nil, status.Error(codes.Unimplemented, "unknown method ExportInstanceStream")
	}
	req := argocdexport.FromStreamRequest(streamReq)
	f.requests = append(f.requests, req)
	resp := rfArgoExport()
	if req.GetOnlyListedResources() {
		resp.Applications = rfListed(resp.Applications, req.GetResources())
		resp.ApplicationSets = rfListed(resp.ApplicationSets, req.GetResources())
		resp.AppProjects = rfListed(resp.AppProjects, req.GetResources())
	}
	msgs := []*argocdv1.ExportInstanceStreamResponse{
		{Resource: &argocdv1.ExportInstanceStreamResponse_Argocd{Argocd: resp.Argocd}},
		{Resource: &argocdv1.ExportInstanceStreamResponse_ArgocdConfigmap{ArgocdConfigmap: resp.ArgocdConfigmap}},
	}
	for _, o := range resp.Applications {
		msgs = append(msgs, &argocdv1.ExportInstanceStreamResponse{Resource: &argocdv1.ExportInstanceStreamResponse_Application{Application: o}})
	}
	for _, o := range resp.ApplicationSets {
		msgs = append(msgs, &argocdv1.ExportInstanceStreamResponse{Resource: &argocdv1.ExportInstanceStreamResponse_ApplicationSet{ApplicationSet: o}})
	}
	for _, o := range resp.AppProjects {
		msgs = append(msgs, &argocdv1.ExportInstanceStreamResponse{Resource: &argocdv1.ExportInstanceStreamResponse_AppProject{AppProject: o}})
	}
	for _, o := range resp.ValidatingAdmissionPolicies {
		msgs = append(msgs, &argocdv1.ExportInstanceStreamResponse{Resource: &argocdv1.ExportInstanceStreamResponse_ValidatingAdmissionPolicy{ValidatingAdmissionPolicy: o}})
	}
	for _, o := range resp.ValidatingAdmissionPolicyBindings {
		msgs = append(msgs, &argocdv1.ExportInstanceStreamResponse{Resource: &argocdv1.ExportInstanceStreamResponse_ValidatingAdmissionPolicyBinding{ValidatingAdmissionPolicyBinding: o}})
	}
	ch := make(chan *argocdv1.ExportInstanceStreamResponse, len(msgs))
	for _, m := range msgs {
		ch <- m
	}
	close(ch)
	return ch, make(chan error), nil
}

func (f *rfArgoCli) ExportInstance(_ context.Context, req *argocdv1.ExportInstanceRequest) (*argocdv1.ExportInstanceResponse, error) {
	if f.current {
		return nil, status.Error(codes.Internal, "a current portal is read through the stream")
	}
	f.requests = append(f.requests, req)
	return rfArgoExport(), nil
}

// rfKargoCli is rfArgoCli for Kargo: a current portal reads only the listed control-plane
// objects, but every admission policy.
type rfKargoCli struct {
	kargov1.KargoServiceGatewayClient
	current  bool
	calls    int
	requests []*kargov1.ExportKargoInstanceRequest
}

func (f *rfKargoCli) GetKargoInstance(_ context.Context, req *kargov1.GetKargoInstanceRequest) (*kargov1.GetKargoInstanceResponse, error) {
	f.calls++
	return &kargov1.GetKargoInstanceResponse{Instance: &kargov1.KargoInstance{Id: "kargo-id", Name: req.GetName(), WorkspaceId: "workspace-id"}}, nil
}

func (f *rfKargoCli) ExportKargoInstanceStream(_ context.Context, streamReq *kargov1.ExportKargoInstanceStreamRequest) (<-chan *kargov1.ExportKargoInstanceStreamResponse, <-chan error, error) {
	if !f.current {
		return nil, nil, status.Error(codes.Unimplemented, "unknown method ExportKargoInstanceStream")
	}
	req := kargoexport.FromStreamRequest(streamReq)
	f.requests = append(f.requests, req)
	resp := rfKargoExport()
	if req.GetOnlyListedResources() {
		resp.Projects = rfListed(resp.Projects, req.GetResources())
		resp.Stages = rfListed(resp.Stages, req.GetResources())
		resp.Warehouses = rfListed(resp.Warehouses, req.GetResources())
		resp.Configmaps = rfListed(resp.Configmaps, req.GetResources())
	}
	msgs := []*kargov1.ExportKargoInstanceStreamResponse{
		{Resource: &kargov1.ExportKargoInstanceStreamResponse_Kargo{Kargo: resp.Kargo}},
		{Resource: &kargov1.ExportKargoInstanceStreamResponse_KargoConfigmap{KargoConfigmap: resp.KargoConfigmap}},
	}
	for _, o := range resp.Projects {
		msgs = append(msgs, &kargov1.ExportKargoInstanceStreamResponse{Resource: &kargov1.ExportKargoInstanceStreamResponse_Project{Project: o}})
	}
	for _, o := range resp.Stages {
		msgs = append(msgs, &kargov1.ExportKargoInstanceStreamResponse{Resource: &kargov1.ExportKargoInstanceStreamResponse_Stage{Stage: o}})
	}
	for _, o := range resp.Warehouses {
		msgs = append(msgs, &kargov1.ExportKargoInstanceStreamResponse{Resource: &kargov1.ExportKargoInstanceStreamResponse_Warehouse{Warehouse: o}})
	}
	for _, o := range resp.Configmaps {
		msgs = append(msgs, &kargov1.ExportKargoInstanceStreamResponse{Resource: &kargov1.ExportKargoInstanceStreamResponse_Configmap{Configmap: o}})
	}
	for _, o := range resp.ValidatingAdmissionPolicies {
		msgs = append(msgs, &kargov1.ExportKargoInstanceStreamResponse{Resource: &kargov1.ExportKargoInstanceStreamResponse_ValidatingAdmissionPolicy{ValidatingAdmissionPolicy: o}})
	}
	ch := make(chan *kargov1.ExportKargoInstanceStreamResponse, len(msgs))
	for _, m := range msgs {
		ch <- m
	}
	close(ch)
	return ch, make(chan error), nil
}

func (f *rfKargoCli) ExportKargoInstance(_ context.Context, req *kargov1.ExportKargoInstanceRequest) (*kargov1.ExportKargoInstanceResponse, error) {
	if f.current {
		return nil, status.Error(codes.Internal, "a current portal is read through the stream")
	}
	f.requests = append(f.requests, req)
	return rfKargoExport(), nil
}

func rfCli(argo *rfArgoCli, kargo *rfKargoCli) *AkpCli {
	return &AkpCli{
		Cli:      argo,
		KargoCli: kargo,
		OrgCli:   &fakeOrgCli{workspaces: []*orgcv1.Workspace{{Id: "workspace-id", Name: "default", IsDefault: true}}},
		Cred:     accesscontrol.NewAPIKeyCredential("key-id", "key-secret"),
		OrgId:    "org-id",
	}
}

func rfFilter(entries ...string) tftypes.Value {
	elems := make([]tftypes.Value, len(entries))
	for i, e := range entries {
		elems[i] = tftypes.NewValue(tftypes.String, e)
	}
	return tftypes.NewValue(rfSetType, elems)
}

// rfLongKeys returns n resource_filter entries with 240-character names. Each is sent as the
// matching rfLongRefs entry, which takes 288 bytes in the URL.
func rfLongKeys(n int) []string {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("argoproj.io/v1alpha1/Application/argocd/%0240d", i)
	}
	return keys
}

func rfLongRefs(n int) []string {
	refs := make([]string, n)
	for i := range refs {
		refs[i] = fmt.Sprintf("argoproj.io/Application/argocd/%0240d", i)
	}
	return refs
}

type DataSourceResourceFilterSuite struct {
	suite.Suite
}

func TestDataSourceResourceFilterSuite(t *testing.T) {
	suite.Run(t, new(DataSourceResourceFilterSuite))
}

// read calls ds.Read with a configuration naming the test instance and the given filter.
func (s *DataSourceResourceFilterSuite) read(ds datasource.DataSource, filter tftypes.Value) (tfsdk.State, diag.Diagnostics) {
	ctx := context.Background()
	var schemaResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	s.Require().False(schemaResp.Diagnostics.HasError(), schemaResp.Diagnostics)
	typ := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(typ.AttributeTypes))
	for name, attrType := range typ.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	values["name"] = tftypes.NewValue(tftypes.String, rfInstanceName)
	values["resource_filter"] = filter
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(typ, nil)}}
	ds.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(typ, values)}}, &resp)
	return resp.State, resp.Diagnostics
}

// requireFilterKept checks that state holds the configured filter unchanged.
func (s *DataSourceResourceFilterSuite) requireFilterKept(state tfsdk.State, filter tftypes.Value) {
	var raw map[string]tftypes.Value
	s.Require().NoError(state.Raw.As(&raw))
	s.Require().True(raw["resource_filter"].Equal(filter), "resource_filter in state: %s", raw["resource_filter"])
}

// requireResources checks the resource map's keys and that each value is serialized exactly
// as an unfiltered read serializes it. A nil want means a null map.
func (s *DataSourceResourceFilterSuite) requireResources(got, unfiltered fwtypes.Map, want []string) {
	if want == nil {
		s.Require().True(got.IsNull(), "want a null map, got %s", got)
		return
	}
	s.Require().False(got.IsNull() || got.IsUnknown(), "want a known map")
	s.Require().ElementsMatch(want, slices.Collect(maps.Keys(got.Elements())))
	for key, value := range got.Elements() {
		s.Require().Equal(unfiltered.Elements()[key], value, key)
	}
}

type rfCase struct {
	filter        tftypes.Value
	wantListed    bool
	wantResources []string
	wantKeys      []string
}

func (s *DataSourceResourceFilterSuite) TestInstanceDataSource() {
	ctx := context.Background()
	const (
		guestbook     = "argoproj.io/v1alpha1/Application//guestbook"
		appsGuestbook = "argoproj.io/v1alpha1/Application/apps/guestbook"
		billing       = "argoproj.io/v1alpha1/Application//billing"
		fleet         = "argoproj.io/v1alpha1/ApplicationSet//fleet"
		teamA         = "argoproj.io/v1alpha1/AppProject//team-a"
		teamB         = "argoproj.io/v1alpha1/AppProject//team-b"
		requireLabels = "admissionregistration.k8s.io/v1/ValidatingAdmissionPolicy//require-labels"
		denyLatest    = "admissionregistration.k8s.io/v1/ValidatingAdmissionPolicy//deny-latest"
		binding       = "admissionregistration.k8s.io/v1/ValidatingAdmissionPolicyBinding//require-labels-binding"
	)
	testCases := map[string]rfCase{
		"omitted filter reads everything": {
			filter:   tftypes.NewValue(rfSetType, nil),
			wantKeys: []string{guestbook, appsGuestbook, billing, fleet, teamA, teamB, requireLabels, denyLatest, binding},
		},
		"empty filter gives an empty map": {
			filter:        rfFilter(),
			wantListed:    true,
			wantResources: []string{},
			wantKeys:      []string{},
		},
		"resource keys select namespaces and admission policies": {
			filter: rfFilter(
				"argoproj.io/v1alpha1/AppProject/argocd/team-a",
				"argoproj.io/v1alpha1/Application/apps/guestbook",
				"admissionregistration.k8s.io/v1/ValidatingAdmissionPolicy//require-labels",
			),
			wantListed: true,
			wantResources: []string{
				"admissionregistration.k8s.io/ValidatingAdmissionPolicy//require-labels",
				"argoproj.io/AppProject/argocd/team-a",
				"argoproj.io/Application/apps/guestbook",
			},
			wantKeys: []string{teamA, appsGuestbook, requireLabels},
		},
		"data source keys select objects in argocd": {
			filter:        rfFilter(teamB, guestbook),
			wantListed:    true,
			wantResources: []string{"argoproj.io/AppProject/argocd/team-b", "argoproj.io/Application/argocd/guestbook"},
			wantKeys:      []string{teamB, guestbook},
		},
		"entries match real identity only": {
			filter: rfFilter(
				"argoproj.io/v1alpha1/AppProject/argocd/missing",
				"v1/AppProject/argocd/team-b", // core group
				"argoproj.io/v1alpha1/AppProject//default",
				"argoproj.io/v1beta1/Application/argocd/billing", // the version is not compared
				"argoproj.io/v1alpha1/Application/default/guestbook",
			),
			wantListed: true,
			wantResources: []string{
				"/AppProject/argocd/team-b",
				"argoproj.io/AppProject/argocd/default",
				"argoproj.io/AppProject/argocd/missing",
				"argoproj.io/Application/argocd/billing",
				"argoproj.io/Application/default/guestbook",
			},
			wantKeys: []string{billing},
		},
		"filter at the URL budget is sent": {
			filter:        rfFilter(rfLongKeys(14)...),
			wantListed:    true,
			wantResources: rfLongRefs(14),
			wantKeys:      []string{},
		},
	}
	for name, tc := range testCases {
		for _, current := range []bool{true, false} {
			s.Run(fmt.Sprintf("%s/current portal=%t", name, current), func() {
				read := func(filter tftypes.Value) (*rfArgoCli, types.InstanceDataSource, tfsdk.State) {
					argo := &rfArgoCli{current: current}
					state, diags := s.read(&AkpInstanceDataSource{BaseDataSource{akpCli: rfCli(argo, nil)}}, filter)
					s.Require().False(diags.HasError(), diags)
					var data types.InstanceDataSource
					s.Require().False(state.Get(ctx, &data).HasError())
					return argo, data, state
				}
				_, unfiltered, _ := read(tftypes.NewValue(rfSetType, nil))
				argo, got, state := read(tc.filter)

				s.Require().Len(argo.requests, 1)
				s.Require().Equal(tc.wantListed, argo.requests[0].GetOnlyListedResources())
				s.Require().Equal(tc.wantResources, argo.requests[0].GetResources())
				s.requireFilterKept(state, tc.filter)
				s.requireResources(got.ArgoCDResources, unfiltered.ArgoCDResources, tc.wantKeys)

				// Nothing else in the data source depends on the filter.
				got.ArgoCDResources, unfiltered.ArgoCDResources = fwtypes.Map{}, fwtypes.Map{}
				got.ResourceFilter, unfiltered.ResourceFilter = fwtypes.Set{}, fwtypes.Set{}
				s.Require().Equal(unfiltered, got)
				s.Require().Contains(got.ManagedSecrets, "registry")
			})
		}
	}
}

func (s *DataSourceResourceFilterSuite) TestKargoDataSource() {
	ctx := context.Background()
	const (
		projA         = "kargo.akuity.io/v1alpha1/Project//proj-a"
		projB         = "kargo.akuity.io/v1alpha1/Project//proj-b"
		devA          = "kargo.akuity.io/v1alpha1/Stage/proj-a/dev"
		devB          = "kargo.akuity.io/v1alpha1/Stage/proj-b/dev"
		warehouse     = "kargo.akuity.io/v1alpha1/Warehouse/proj-a/main"
		configMap     = "v1/ConfigMap/proj-a/shared"
		requireLabels = "admissionregistration.k8s.io/v1/ValidatingAdmissionPolicy//require-labels"
	)
	testCases := map[string]rfCase{
		"omitted filter reads everything": {
			filter:   tftypes.NewValue(rfSetType, nil),
			wantKeys: []string{projA, projB, devA, devB, warehouse, configMap, requireLabels},
		},
		"empty filter gives an empty map": {
			filter:        rfFilter(),
			wantListed:    true,
			wantResources: []string{},
			wantKeys:      []string{},
		},
		"keys select core group and admission policies": {
			filter:     rfFilter(devB, configMap, requireLabels),
			wantListed: true,
			wantResources: []string{
				"/ConfigMap/proj-a/shared",
				"admissionregistration.k8s.io/ValidatingAdmissionPolicy//require-labels",
				"kargo.akuity.io/Stage/proj-b/dev",
			},
			wantKeys: []string{devB, configMap, requireLabels},
		},
		"entries match real identity only": {
			filter: rfFilter(
				"kargo.akuity.io/v1alpha1/Stage/proj-a/missing",
				"kargo.akuity.io/v1alpha1/Stage//dev", // only Argo CD keys leave out a namespace
				"apps/v1/ConfigMap/proj-a/shared",     // wrong group
				projA,
			),
			wantListed: true,
			wantResources: []string{
				"apps/ConfigMap/proj-a/shared",
				"kargo.akuity.io/Project//proj-a",
				"kargo.akuity.io/Stage//dev",
				"kargo.akuity.io/Stage/proj-a/missing",
			},
			wantKeys: []string{projA},
		},
	}
	for name, tc := range testCases {
		for _, current := range []bool{true, false} {
			s.Run(fmt.Sprintf("%s/current portal=%t", name, current), func() {
				read := func(filter tftypes.Value) (*rfKargoCli, types.KargoInstanceDataSource, tfsdk.State) {
					kargo := &rfKargoCli{current: current}
					state, diags := s.read(&AkpKargoDataSource{BaseDataSource{akpCli: rfCli(nil, kargo)}}, filter)
					s.Require().False(diags.HasError(), diags)
					var data types.KargoInstanceDataSource
					s.Require().False(state.Get(ctx, &data).HasError())
					return kargo, data, state
				}
				_, unfiltered, _ := read(tftypes.NewValue(rfSetType, nil))
				kargo, got, state := read(tc.filter)

				s.Require().Len(kargo.requests, 1)
				s.Require().Equal(tc.wantListed, kargo.requests[0].GetOnlyListedResources())
				s.Require().Equal(tc.wantResources, kargo.requests[0].GetResources())
				s.requireFilterKept(state, tc.filter)
				s.requireResources(got.KargoResources, unfiltered.KargoResources, tc.wantKeys)

				got.KargoResources, unfiltered.KargoResources = fwtypes.Map{}, fwtypes.Map{}
				got.ResourceFilter, unfiltered.ResourceFilter = fwtypes.Set{}, fwtypes.Set{}
				s.Require().Equal(unfiltered, got)
			})
		}
	}
}

// A filter that can't be sent as written fails the read before anything is fetched; it is
// never widened to a full export.
func (s *DataSourceResourceFilterSuite) TestUnusableFilterFailsRead() {
	shortEntries := make([]string, 1001)
	for i := range shortEntries {
		shortEntries[i] = fmt.Sprintf("v1/ConfigMap/ns/c%d", i)
	}
	testCases := map[string]struct {
		filter  tftypes.Value
		wantErr string
	}{
		"entry without an API version": {rfFilter("argoproj.io/AppProject/argocd/team-a"), "Invalid resource_filter entry"},
		"entry with an empty group":    {rfFilter("/v1alpha1/AppProject/argocd/team-a"), "Invalid resource_filter entry"},
		"entry with a bad version":     {rfFilter("argoproj.io/latest/AppProject/argocd/team-a"), "Invalid resource_filter entry"},
		"entry with too many parts":    {rfFilter("argoproj.io/v1alpha1/AppProject/argocd/team/a"), "Invalid resource_filter entry"},
		"entry the API would refuse":   {rfFilter("argoproj.io/v1alpha1/Application/argocd/100%"), "Invalid resource_filter entry"},
		"null entry": {
			tftypes.NewValue(rfSetType, []tftypes.Value{tftypes.NewValue(tftypes.String, nil)}),
			"Invalid resource_filter entry",
		},
		"unknown filter": {tftypes.NewValue(rfSetType, tftypes.UnknownValue), "resource_filter is not known"},
		"unknown entry": {
			tftypes.NewValue(rfSetType, []tftypes.Value{
				tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				tftypes.NewValue(tftypes.String, "argoproj.io/v1alpha1/AppProject/argocd/team-a"),
			}),
			"resource_filter is not known",
		},
		"over the URL budget":           {rfFilter(rfLongKeys(15)...), "resource_filter is too long"},
		"over the server's entry limit": {rfFilter(shortEntries...), "resource_filter is too long"},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			argo, kargo := &rfArgoCli{current: true}, &rfKargoCli{current: true}
			cli := rfCli(argo, kargo)
			for _, ds := range []datasource.DataSource{
				&AkpInstanceDataSource{BaseDataSource{akpCli: cli}},
				&AkpKargoDataSource{BaseDataSource{akpCli: cli}},
			} {
				_, diags := s.read(ds, tc.filter)
				s.Require().True(diags.HasError(), "%T", ds)
				s.Require().Contains(diags.Errors()[0].Summary(), tc.wantErr, "%T", ds)
			}
			s.Require().Zero(argo.calls + kargo.calls)
		})
	}
}

func (s *DataSourceResourceFilterSuite) TestValidator() {
	ctx := context.Background()
	str := fwtypes.StringValue
	unknown := fwtypes.StringUnknown()
	set := func(elems ...attr.Value) fwtypes.Set { return fwtypes.SetValueMust(fwtypes.StringType, elems) }
	testCases := map[string]struct {
		value    fwtypes.Set
		wantErrs int
	}{
		"null":    {fwtypes.SetNull(fwtypes.StringType), 0},
		"unknown": {fwtypes.SetUnknown(fwtypes.StringType), 0},
		"empty":   {set(), 0},
		"valid": {set(
			str("argoproj.io/v1alpha1/AppProject/argocd/team-a"),
			str("argoproj.io/v1alpha1/AppProject//team-b"),
			str("v1/ConfigMap/ns/shared"),
		), 0},
		"unknown beside valid":     {set(unknown, str("argoproj.io/v1alpha1/AppProject/argocd/team-a")), 0},
		"unknown beside invalid":   {set(unknown, str("argoproj.io/AppProject/argocd/team-a")), 1},
		"each invalid entry":       {set(str("a/b"), str("a/b/c"), str("a/v1/b/c")), 3},
		"null entry":               {set(fwtypes.StringNull()), 1},
		"over the URL budget":      {set(rfStrings(rfLongKeys(15))...), 1},
		"at the URL budget":        {set(rfStrings(rfLongKeys(14))...), 0},
		"unknown beside many long": {set(append(rfStrings(rfLongKeys(15)), unknown)...), 0},
	}
	for name, tc := range testCases {
		s.Run(name, func() {
			resp := &validator.SetResponse{}
			resourceFilterValidator{resourceType: "ArgoCD"}.ValidateSet(ctx, validator.SetRequest{Path: resourceFilterPath, ConfigValue: tc.value}, resp)
			s.Require().Equal(tc.wantErrs, resp.Diagnostics.ErrorsCount(), resp.Diagnostics)
		})
	}
}

func rfStrings(entries []string) []attr.Value {
	values := make([]attr.Value, len(entries))
	for i, e := range entries {
		values[i] = fwtypes.StringValue(e)
	}
	return values
}

// The managed resources still ask only for the objects in their state, keep only those, and
// ignore the extra objects an older portal returns.
func (s *DataSourceResourceFilterSuite) TestResourceRefreshStillFiltersByState() {
	ctx := context.Background()
	stateMap := func(keys ...string) fwtypes.Map {
		elems := make(map[string]attr.Value, len(keys))
		for _, k := range keys {
			elems[k] = fwtypes.StringValue("{}")
		}
		return fwtypes.MapValueMust(fwtypes.StringType, elems)
	}
	for _, current := range []bool{true, false} {
		s.Run(fmt.Sprintf("current portal=%t", current), func() {
			argo, kargo := &rfArgoCli{current: current}, &rfKargoCli{current: current}
			cli := rfCli(argo, kargo)
			var diags diag.Diagnostics

			argoKeys := []string{"argoproj.io/v1alpha1/AppProject/argocd/team-a", "argoproj.io/v1alpha1/Application/apps/guestbook"}
			instance := &types.Instance{Name: fwtypes.StringValue(rfInstanceName), ArgoCDResources: stateMap(argoKeys...)}
			s.Require().NoError(refreshState(ctx, &diags, cli, instance, &argocdv1.GetInstanceRequest{
				OrganizationId: cli.OrgId, IdType: idv1.Type_NAME, Id: rfInstanceName,
			}, false, nil))
			s.Require().False(diags.HasError(), diags)
			s.Require().Len(argo.requests, 1)
			s.Require().True(argo.requests[0].GetOnlyListedResources())
			s.Require().Equal([]string{"argoproj.io/AppProject/argocd/team-a", "argoproj.io/Application/apps/guestbook"}, argo.requests[0].GetResources())
			s.Require().ElementsMatch(argoKeys, slices.Collect(maps.Keys(instance.ArgoCDResources.Elements())))

			kargoKeys := []string{"kargo.akuity.io/v1alpha1/Stage/proj-b/dev", "v1/ConfigMap/proj-a/shared"}
			kargoInstance := &types.KargoInstance{Name: fwtypes.StringValue(rfInstanceName), KargoResources: stateMap(kargoKeys...)}
			s.Require().NoError(refreshKargoState(ctx, &diags, cli, kargoInstance, cli.OrgId, false, nil))
			s.Require().False(diags.HasError(), diags)
			s.Require().Len(kargo.requests, 1)
			s.Require().True(kargo.requests[0].GetOnlyListedResources())
			s.Require().Equal([]string{"/ConfigMap/proj-a/shared", "kargo.akuity.io/Stage/proj-b/dev"}, kargo.requests[0].GetResources())
			s.Require().ElementsMatch(kargoKeys, slices.Collect(maps.Keys(kargoInstance.KargoResources.Elements())))
		})
	}
}

// rfProvider is the real provider with fake API clients.
type rfProvider struct {
	provider.Provider
	cli *AkpCli
}

func (p *rfProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.DataSourceData = p.cli
}

// Terraform defers a data source whose resource_filter isn't known at plan time, so the read
// happens at apply with the real entries and is never sent as an empty filter.
func (s *DataSourceResourceFilterSuite) TestUnknownFilterDefersRead() {
	s.T().Setenv("TF_CLI_CONFIG_FILE", os.DevNull)
	argo, kargo := &rfArgoCli{current: true}, &rfKargoCli{current: true}
	resource.UnitTest(s.T(), resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"akp": providerserver.NewProtocol6WithError(&rfProvider{Provider: New("test")(), cli: rfCli(argo, kargo)}),
		},
		Steps: []resource.TestStep{{
			Config: `
provider "akp" { org_name = "test" }
resource "terraform_data" "entries" {
  input = ["argoproj.io/v1alpha1/AppProject//team-a", "kargo.akuity.io/v1alpha1/Stage/proj-b/dev"]
}
data "akp_instance" "test" {
  name            = "filter-test"
  resource_filter = [terraform_data.entries.output[0]]
}
data "akp_kargo_instance" "test" {
  name            = "filter-test"
  resource_filter = [terraform_data.entries.output[1]]
}
`,
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectUnknownValue("data.akp_instance.test", tfjsonpath.New("argocd_resources")),
				plancheck.ExpectUnknownValue("data.akp_kargo_instance.test", tfjsonpath.New("kargo_resources")),
			}},
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.akp_instance.test", "argocd_resources.%", "1"),
				resource.TestCheckResourceAttrSet("data.akp_instance.test", "argocd_resources.argoproj.io/v1alpha1/AppProject//team-a"),
				resource.TestCheckResourceAttr("data.akp_kargo_instance.test", "kargo_resources.%", "1"),
				resource.TestCheckResourceAttrSet("data.akp_kargo_instance.test", "kargo_resources.kargo.akuity.io/v1alpha1/Stage/proj-b/dev"),
				func(*terraform.State) error {
					if len(argo.requests) == 0 || len(kargo.requests) == 0 {
						return fmt.Errorf("want reads of both data sources, got %d and %d", len(argo.requests), len(kargo.requests))
					}
					for _, r := range argo.requests {
						if !r.GetOnlyListedResources() || !slices.Equal(r.GetResources(), []string{"argoproj.io/AppProject/argocd/team-a"}) {
							return fmt.Errorf("unexpected Argo CD export request: %v", r)
						}
					}
					for _, r := range kargo.requests {
						if !r.GetOnlyListedResources() || !slices.Equal(r.GetResources(), []string{"kargo.akuity.io/Stage/proj-b/dev"}) {
							return fmt.Errorf("unexpected Kargo export request: %v", r)
						}
					}
					return nil
				},
			),
		}},
	})
}
