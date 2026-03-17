package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func (r *AkpKargoDefaultShardAgentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the default shard agent for a Kargo instance. This resource binds a Kargo instance to a specific agent that will be used as the default shard agent.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource ID (same as kargo_instance_id)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"kargo_instance_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the Kargo instance",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"agent_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the Kargo agent to set as the default shard agent",
			},
		},
	}
}
