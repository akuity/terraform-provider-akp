package akp

import (
	"context"
	"fmt"
	"testing"

	hashitype "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func TestAccClusterResource(t *testing.T) {
	name := fmt.Sprintf("cluster-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccClusterResourceConfig("small", name, "test one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "namespace", "test"),
					resource.TestCheckResourceAttr("akp_cluster.test", "labels.test-label", "true"),
					resource.TestCheckResourceAttr("akp_cluster.test", "annotations.test-annotation", "false"),
					// spec
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.description", "test one"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.namespace_scoped", "true"),
					// spec.data
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "small"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.kustomization", `  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
  patches:
    - patch: |-
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: argocd-repo-server
        spec:
          template:
            spec:
              containers:
              - name: argocd-repo-server
                resources:
                  limits:
                    memory: 2Gi
                  requests:
                    cpu: 750m
                    memory: 1Gi
      target:
        kind: Deployment
        name: argocd-repo-server
`),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.app_replication", "false"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.redis_tunneling", "false"),
					resource.TestCheckResourceAttr("akp_cluster.test", "remove_agent_resources_on_destroy", "true"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccClusterResourceConfig("medium", name, "test two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.description", "test two"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "medium"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccClusterResourceConfig(size string, name string, description string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = "6pzhawvy4echbd8x"
  name      = %q
  namespace = "test"
  labels = {
    test-label = "true"
  }
  annotations = {
    test-annotation = "false"
  }
  spec = {
    namespace_scoped = true
    description      = %q
    data = {
      size                  = %q
      auto_upgrade_disabled = true
      kustomization         = <<EOF
  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
  patches:
    - patch: |-
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: argocd-repo-server
        spec:
          template:
            spec:
              containers:
              - name: argocd-repo-server
                resources:
                  limits:
                    memory: 2Gi
                  requests:
                    cpu: 750m
                    memory: 1Gi
      target:
        kind: Deployment
        name: argocd-repo-server
EOF
    }
  }
}
`, name, description, size)
}

func TestAkpClusterResource_applyInstance(t *testing.T) {
	type args struct {
		plan             *types.Cluster
		apiReq           *argocdv1.ApplyInstanceRequest
		isCreate         bool
		applyInstance    func(context.Context, *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error)
		upsertKubeConfig func(ctx context.Context, plan *types.Cluster) error
	}
	tests := []struct {
		name  string
		args  args
		want  *types.Cluster
		error error
	}{
		{
			name: "happy path, no kubeconfig",
			args: args{
				plan: &types.Cluster{},
				applyInstance: func(ctx context.Context, request *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error) {
					return &argocdv1.ApplyInstanceResponse{}, nil
				},
				upsertKubeConfig: func(ctx context.Context, plan *types.Cluster) error {
					return errors.New("this should not be called")
				},
			},
			want:  &types.Cluster{},
			error: nil,
		},
		{
			name: "error path, no kubeconfig",
			args: args{
				plan: &types.Cluster{},
				applyInstance: func(ctx context.Context, request *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error) {
					return &argocdv1.ApplyInstanceResponse{}, errors.New("some error")
				},
				upsertKubeConfig: func(ctx context.Context, plan *types.Cluster) error {
					return errors.New("this should not be called")
				},
			},
			want:  nil,
			error: fmt.Errorf("unable to create Argo CD instance: some error"),
		},
		{
			name: "happy path, with kubeconfig",
			args: args{
				plan: &types.Cluster{
					Kubeconfig: &types.Kubeconfig{
						Host: hashitype.StringValue("some-host"),
					},
				},
				applyInstance: func(ctx context.Context, request *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error) {
					return &argocdv1.ApplyInstanceResponse{}, nil
				},
				upsertKubeConfig: func(ctx context.Context, plan *types.Cluster) error {
					assert.Equal(t, &types.Cluster{
						Kubeconfig: &types.Kubeconfig{
							Host: hashitype.StringValue("some-host"),
						},
					}, plan)
					return nil
				},
			},
			want: &types.Cluster{
				Kubeconfig: &types.Kubeconfig{
					Host: hashitype.StringValue("some-host"),
				},
			},
			error: nil,
		},
		{
			name: "error path, with kubeconfig",
			args: args{
				plan: &types.Cluster{
					Kubeconfig: &types.Kubeconfig{
						Host: hashitype.StringValue("some-host"),
					},
				},
				applyInstance: func(ctx context.Context, request *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error) {
					return &argocdv1.ApplyInstanceResponse{}, nil
				},
				upsertKubeConfig: func(ctx context.Context, plan *types.Cluster) error {
					return errors.New("some kube apply error")
				},
			},
			want:  &types.Cluster{},
			error: fmt.Errorf("unable to apply manifests: some kube apply error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &AkpClusterResource{}
			ctx := context.Background()
			got, err := r.applyInstance(ctx, tt.args.plan, tt.args.apiReq, tt.args.isCreate, tt.args.applyInstance, tt.args.upsertKubeConfig)
			assert.Equal(t, tt.error, err)
			assert.Equalf(t, tt.want, got, "applyInstance(%v, %v, %v)", tt.args.plan, tt.args.apiReq, tt.args.isCreate)
		})
	}
}
