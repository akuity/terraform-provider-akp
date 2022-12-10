package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *AkpClusterResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Create a cluster attached to an Argo CD instance. Use `.manifests` attribute to get agent installation manifests",

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "Cluster ID",
				Type:                types.StringType,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"manifests": {
				MarkdownDescription: "Agent Installation Manifests",
				Type:                types.StringType,
				Computed:            true,
				Sensitive:           true,
			},
			"instance_id": {
				MarkdownDescription: "Argo CD Instance ID",
				Type:                types.StringType,
				Required:            true,
			},
			"name": {
				MarkdownDescription: "Cluster Name",
				Type:                types.StringType,
				Required:            true,
			},
			"description": {
				MarkdownDescription: "Cluster Description",
				Type:                types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"namespace": {
				MarkdownDescription: "Agent Installation Namespace",
				Type:                types.StringType,
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"namespace_scoped": {
				MarkdownDescription: "Agent Namespace Scoped",
				Type:                types.BoolType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
		},
	}, nil
}
