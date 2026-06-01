package akp

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	accesscontrolv1 "github.com/akuity/api-client-go/pkg/api/gen/accesscontrol/v1"
	apikeyv1 "github.com/akuity/api-client-go/pkg/api/gen/apikey/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func NewAkpApiKeyResource() resource.Resource {
	return &GenericResource[types.ApiKey]{
		TypeNameSuffix: "api_key",
		SchemaFunc:     apiKeySchema,
		CreateFunc:     apiKeyCreate,
		ReadFunc:       apiKeyRead,
		UpdateFunc:     apiKeyUpdate,
		DeleteFunc:     apiKeyDelete,
		ImportStateFunc: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			// Import IDs:
			//   org-scoped:       <api_key_id>
			//   workspace-scoped: <workspace_name>/<api_key_id>
			parts := strings.Split(req.ID, "/")
			badID := func() {
				resp.Diagnostics.AddError(
					"Unexpected Import Identifier",
					fmt.Sprintf("Expected `api_key_id` or `workspace_name/api_key_id`. Got: %q", req.ID),
				)
			}
			switch len(parts) {
			case 1:
				if parts[0] == "" {
					badID()
					return
				}
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
			case 2:
				if parts[0] == "" || parts[1] == "" {
					badID()
					return
				}
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace"), parts[0])...)
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
			default:
				badID()
			}
		},
	}
}

func apiKeyCreate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.ApiKey) (*types.ApiKey, error) {
	if err := requireKnownWorkspace(plan.Workspace, "api_key"); err != nil {
		return nil, err
	}
	perms, err := buildApiKeyPermissions(plan.Permissions)
	if err != nil {
		return nil, err
	}

	// "0" is the established sentinel for "no expiry" — both str2duration on the
	// server and the service layer's `if Expiry > 0` short-circuit treat it as
	// no expiration. The portal UI uses the same convention. Omitted config
	// becomes "0" on the wire; a configured duration passes through verbatim.
	expireIn := "0"
	if !plan.ExpireInDuration.IsNull() && !plan.ExpireInDuration.IsUnknown() {
		if v := plan.ExpireInDuration.ValueString(); v != "" {
			expireIn = v
		}
	}

	if plan.Workspace.IsNull() || plan.Workspace.ValueString() == "" {
		resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.CreateOrganizationAPIKeyResponse, error) {
			return cli.OrgCli.CreateOrganizationAPIKey(ctx, &orgcv1.CreateOrganizationAPIKeyRequest{
				Id:               cli.OrgId,
				Description:      plan.Description.ValueString(),
				Permissions:      perms,
				ExpireInDuration: expireIn,
			})
		}, "CreateOrganizationAPIKey")
		if err != nil {
			return nil, fmt.Errorf("unable to create API key: %w", err)
		}
		applyApiKeyResponse(plan, resp.GetApiKey())
		return plan, nil
	}

	workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, plan.Workspace.ValueString())
	if err != nil {
		return nil, fmt.Errorf("unable to resolve workspace %q: %w", plan.Workspace.ValueString(), err)
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.CreateWorkspaceAPIKeyResponse, error) {
		return cli.OrgCli.CreateWorkspaceAPIKey(ctx, &orgcv1.CreateWorkspaceAPIKeyRequest{
			Id:               cli.OrgId,
			WorkspaceId:      workspace.GetId(),
			Description:      plan.Description.ValueString(),
			Permissions:      perms,
			ExpireInDuration: expireIn,
		})
	}, "CreateWorkspaceAPIKey")
	if err != nil {
		return nil, fmt.Errorf("unable to create workspace API key: %w", err)
	}
	applyApiKeyResponse(plan, resp.GetApiKey())
	return plan, nil
}

func apiKeyRead(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, data *types.ApiKey) error {
	// Route to the scope-specific endpoint: server-side permission checks
	// differ — top-level GetAPIKey always enforces `organization/apikeys`,
	// but workspace-scoped keys need `workspace/apikeys` instead.
	key, err := fetchAPIKey(ctx, cli, data)
	if err != nil {
		return err
	}
	if key == nil {
		return status.Error(codes.NotFound, "API key not found")
	}
	// Server does not echo Secret or ExpireInDuration on Get. Both are
	// preserved across reads as side effects: applyApiKeyResponse keeps a
	// known Secret untouched (and normalizes a null/unknown one to "" so
	// imports don't leave the Computed attribute null forever), and it never
	// touches ExpireInDuration.
	applyApiKeyResponse(data, key)
	return nil
}

// fetchAPIKey picks the right Get endpoint for the resource's scope. The
// workspace path needs the workspace ID resolved from the name kept in state.
func fetchAPIKey(ctx context.Context, cli *AkpCli, data *types.ApiKey) (*apikeyv1.APIKey, error) {
	if !data.Workspace.IsNull() && data.Workspace.ValueString() != "" {
		ws, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, data.Workspace.ValueString())
		if err != nil {
			return nil, err
		}
		resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*apikeyv1.GetWorkspaceAPIKeyResponse, error) {
			return cli.ApiKeyCli.GetWorkspaceAPIKey(ctx, &apikeyv1.GetWorkspaceAPIKeyRequest{
				OrganizationId: cli.OrgId,
				WorkspaceId:    ws.GetId(),
				Id:             data.ID.ValueString(),
			})
		}, "GetWorkspaceAPIKey")
		if err != nil {
			return nil, err
		}
		return resp.GetApiKey(), nil
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*apikeyv1.GetAPIKeyResponse, error) {
		return cli.ApiKeyCli.GetAPIKey(ctx, &apikeyv1.GetAPIKeyRequest{
			Id: data.ID.ValueString(),
		})
	}, "GetAPIKey")
	if err != nil {
		return nil, err
	}
	return resp.GetApiKey(), nil
}

func apiKeyUpdate(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *types.ApiKey) (*types.ApiKey, error) {
	// All mutable fields are marked RequiresReplace, so Update should only
	// fire when Terraform refreshes a value that doesn't actually change the
	// remote (e.g. computed attributes). Re-read to keep state in sync.
	return plan, apiKeyRead(ctx, cli, diags, plan)
}

func apiKeyDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, state *types.ApiKey) error {
	if !state.Workspace.IsNull() && state.Workspace.ValueString() != "" {
		workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, state.Workspace.ValueString())
		if err != nil {
			// Workspace gone: fall back to the org-level delete by ID.
			if !isGoneErr(err) {
				return fmt.Errorf("unable to resolve workspace %q: %w", state.Workspace.ValueString(), err)
			}
		} else {
			_, err := retryWithBackoff(ctx, func(ctx context.Context) (*apikeyv1.DeleteWorkspaceAPIKeyResponse, error) {
				resp, err := cli.ApiKeyCli.DeleteWorkspaceAPIKey(ctx, &apikeyv1.DeleteWorkspaceAPIKeyRequest{
					OrganizationId: cli.OrgId,
					WorkspaceId:    workspace.GetId(),
					Id:             state.ID.ValueString(),
				})
				if isGoneErr(err) {
					return resp, nil
				}
				return resp, err
			}, "DeleteWorkspaceAPIKey")
			if err != nil {
				return fmt.Errorf("unable to delete workspace API key: %w", err)
			}
			return nil
		}
	}

	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*apikeyv1.DeleteAPIKeyResponse, error) {
		resp, err := cli.ApiKeyCli.DeleteAPIKey(ctx, &apikeyv1.DeleteAPIKeyRequest{
			Id: state.ID.ValueString(),
		})
		if isGoneErr(err) {
			return resp, nil
		}
		return resp, err
	}, "DeleteAPIKey")
	if err != nil {
		return fmt.Errorf("unable to delete API key: %w", err)
	}
	return nil
}

func buildApiKeyPermissions(p *types.ApiKeyPermissions) (*accesscontrolv1.Permissions, error) {
	if p == nil {
		return nil, fmt.Errorf("permissions are required")
	}
	out := &accesscontrolv1.Permissions{
		Actions:     stringSliceFromTF(p.Actions),
		Roles:       stringSliceFromTF(p.Roles),
		CustomRoles: stringSliceFromTF(p.CustomRoles),
	}
	if len(out.Roles) == 0 && len(out.CustomRoles) == 0 {
		return nil, fmt.Errorf("permissions must include at least one entry in `roles` or `custom_roles`")
	}
	return out, nil
}

func stringSliceFromTF(in []tftypes.String) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v.IsNull() || v.IsUnknown() {
			continue
		}
		out = append(out, v.ValueString())
	}
	return out
}

func stringSliceToTF(in []string) []tftypes.String {
	if len(in) == 0 {
		return nil
	}
	out := make([]tftypes.String, 0, len(in))
	for _, v := range in {
		out = append(out, tftypes.StringValue(v))
	}
	return out
}

func applyApiKeyResponse(data *types.ApiKey, key *apikeyv1.APIKey) {
	if key == nil {
		return
	}
	data.ID = tftypes.StringValue(key.GetId())
	data.Description = tftypes.StringValue(key.GetDescription())
	data.OrganizationID = tftypes.StringValue(key.GetOrganizationId())
	if key.Secret != nil {
		data.Secret = tftypes.StringValue(*key.Secret)
	} else if data.Secret.IsUnknown() || data.Secret.IsNull() {
		data.Secret = tftypes.StringValue("")
	}
	if t := key.GetCreateTime(); t != nil {
		data.CreateTime = tftypes.StringValue(t.AsTime().Format("2006-01-02T15:04:05Z07:00"))
	}
	if t := key.GetExpireTime(); t != nil && t.AsTime().Unix() > 0 {
		data.ExpireTime = tftypes.StringValue(t.AsTime().Format("2006-01-02T15:04:05Z07:00"))
	} else {
		data.ExpireTime = tftypes.StringValue("")
	}

	if key.GetPermissions() != nil {
		// Workspace-scoped keys get `organization/member` force-appended server-side
		// (see internal/portalapi/organization/create_workspace_api_key_v1.go).
		// Filter to the namespace matching this resource's scope so state only
		// reflects what the operator actually wrote.
		wantNamespace := "organization"
		if !data.Workspace.IsNull() && data.Workspace.ValueString() != "" {
			wantNamespace = "workspace"
		}
		data.Permissions = &types.ApiKeyPermissions{
			Actions:     stringSliceToTF(key.GetPermissions().GetActions()),
			Roles:       stringSliceToTF(stripRoleNamespace(filterRolesByNamespace(key.GetPermissions().GetRoles(), wantNamespace))),
			CustomRoles: stringSliceToTF(key.GetPermissions().GetCustomRoles()),
		}
	}
}

// stripRoleNamespace strips the server's `<scope>/` prefix (e.g.
// `organization/member`, `workspace/admin`) back to the short name the UI
// and operators write in config. The server normalizes the inverse on write
// via `Role(name).ToAccessControlRole()`, so the round-trip is lossless.
func stripRoleNamespace(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	for i, r := range in {
		if idx := strings.IndexByte(r, '/'); idx >= 0 && idx+1 < len(r) {
			out[i] = r[idx+1:]
		} else {
			out[i] = r
		}
	}
	return out
}

// filterRolesByNamespace keeps only roles whose `<namespace>/...` prefix
// matches namespace. Roles without a slash are passed through as-is — those
// represent already-stripped or namespace-less roles, neither of which we
// want to silently drop.
func filterRolesByNamespace(in []string, namespace string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	prefix := namespace + "/"
	for _, r := range in {
		if !strings.Contains(r, "/") || strings.HasPrefix(r, prefix) {
			out = append(out, r)
		}
	}
	return out
}
