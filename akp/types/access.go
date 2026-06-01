package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ApiKey struct {
	ID               types.String       `tfsdk:"id"`
	Workspace        types.String       `tfsdk:"workspace"`
	Description      types.String       `tfsdk:"description"`
	ExpireInDuration types.String       `tfsdk:"expire_in_duration"`
	Permissions      *ApiKeyPermissions `tfsdk:"permissions"`
	Secret           types.String       `tfsdk:"secret"`
	OrganizationID   types.String       `tfsdk:"organization_id"`
	CreateTime       types.String       `tfsdk:"create_time"`
	ExpireTime       types.String       `tfsdk:"expire_time"`
}

type ApiKeyPermissions struct {
	Actions     []types.String `tfsdk:"actions"`
	Roles       []types.String `tfsdk:"roles"`
	CustomRoles []types.String `tfsdk:"custom_roles"`
}

type CustomRole struct {
	ID          types.String `tfsdk:"id"`
	Workspace   types.String `tfsdk:"workspace"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Policy      types.String `tfsdk:"policy"`
}

type Workspace struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreateTime  types.String `tfsdk:"create_time"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
}
