package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func (r *AkpInstanceIPAllowListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the IP allow list for an Argo CD instance. This resource allows you to configure IP addresses that are allowed to access the Argo CD instance. This resource replaces the deprecated `ip_allow_list` field in the `akp_instance` resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource ID (same as instance_id)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the Argo CD instance",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"entries": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "List of IP allow list entries",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "IP address or CIDR block to allow (e.g., '192.168.1.0/24' or '2001:db8::/32')",
						},
						"description": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Description of the IP allow list entry",
						},
					},
				},
			},
		},
	}
}
