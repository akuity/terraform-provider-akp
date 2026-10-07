package akp

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"

	serviceaccountv1 "github.com/akuity/api-client-go/pkg/api/gen/serviceaccount/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func NewAkpServiceAccountResource() resource.Resource {
	return &GenericResource[types.ServiceAccount]{
		TypeNameSuffix:  "service_account",
		SchemaFunc:      serviceAccountSchema,
		CreateFunc:      serviceAccountCreate,
		ReadFunc:        serviceAccountRead,
		UpdateFunc:      serviceAccountUpdate,
		DeleteFunc:      serviceAccountDelete,
		ImportStateFunc: importScopedID,
	}
}

func serviceAccountCreate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.ServiceAccount) (*types.ServiceAccount, error) {
	if err := requireKnownWorkspace(plan.Workspace, "service_account"); err != nil {
		return nil, err
	}
	spec, err := serviceAccountSpec(ctx, plan)
	if err != nil {
		return nil, err
	}
	if !isWorkspaceScoped(plan.Workspace) {
		resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.CreateServiceAccountResponse, error) {
			return cli.ServiceAccountCli.CreateServiceAccount(ctx, &serviceaccountv1.CreateServiceAccountRequest{
				OrganizationId: cli.OrgId,
				Spec:           spec,
			})
		}, "CreateServiceAccount")
		if err != nil {
			return nil, fmt.Errorf("unable to create service account: %w", err)
		}
		return plan, applyServiceAccountResponse(ctx, plan, resp.GetServiceAccount())
	}
	workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, plan.Workspace.ValueString())
	if err != nil {
		return nil, fmt.Errorf("unable to resolve workspace %q: %w", plan.Workspace.ValueString(), err)
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.CreateWorkspaceServiceAccountResponse, error) {
		return cli.ServiceAccountCli.CreateWorkspaceServiceAccount(ctx, &serviceaccountv1.CreateWorkspaceServiceAccountRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspace.GetId(),
			Spec:           spec,
		})
	}, "CreateWorkspaceServiceAccount")
	if err != nil {
		return nil, fmt.Errorf("unable to create workspace service account: %w", err)
	}
	return plan, applyServiceAccountResponse(ctx, plan, resp.GetServiceAccount())
}

// serviceAccountRead routes to the scope's endpoint: an org-scoped lookup
// never finds a workspace account, and each scope enforces its own object.
func serviceAccountRead(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, data *types.ServiceAccount) error {
	if !isWorkspaceScoped(data.Workspace) {
		resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.GetServiceAccountResponse, error) {
			return cli.ServiceAccountCli.GetServiceAccount(ctx, &serviceaccountv1.GetServiceAccountRequest{
				OrganizationId: cli.OrgId,
				Id:             data.ID.ValueString(),
			})
		}, "GetServiceAccount")
		if err != nil {
			return err
		}
		return applyServiceAccountResponse(ctx, data, resp.GetServiceAccount())
	}
	workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, data.Workspace.ValueString())
	if err != nil {
		return err
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.GetWorkspaceServiceAccountResponse, error) {
		return cli.ServiceAccountCli.GetWorkspaceServiceAccount(ctx, &serviceaccountv1.GetWorkspaceServiceAccountRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspace.GetId(),
			Id:             data.ID.ValueString(),
		})
	}, "GetWorkspaceServiceAccount")
	if err != nil {
		return err
	}
	return applyServiceAccountResponse(ctx, data, resp.GetServiceAccount())
}

// serviceAccountUpdate sends the complete desired state: the server replaces
// the account wholesale, and workspace is RequiresReplace so it always
// matches state.
func serviceAccountUpdate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.ServiceAccount) (*types.ServiceAccount, error) {
	spec, err := serviceAccountSpec(ctx, plan)
	if err != nil {
		return nil, err
	}
	if !isWorkspaceScoped(plan.Workspace) {
		resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.UpdateServiceAccountResponse, error) {
			return cli.ServiceAccountCli.UpdateServiceAccount(ctx, &serviceaccountv1.UpdateServiceAccountRequest{
				OrganizationId: cli.OrgId,
				Id:             plan.ID.ValueString(),
				Spec:           spec,
			})
		}, "UpdateServiceAccount")
		if err != nil {
			return nil, fmt.Errorf("unable to update service account: %w", err)
		}
		return plan, applyServiceAccountResponse(ctx, plan, resp.GetServiceAccount())
	}
	workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, plan.Workspace.ValueString())
	if err != nil {
		return nil, fmt.Errorf("unable to resolve workspace %q: %w", plan.Workspace.ValueString(), err)
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.UpdateWorkspaceServiceAccountResponse, error) {
		return cli.ServiceAccountCli.UpdateWorkspaceServiceAccount(ctx, &serviceaccountv1.UpdateWorkspaceServiceAccountRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspace.GetId(),
			Id:             plan.ID.ValueString(),
			Spec:           spec,
		})
	}, "UpdateWorkspaceServiceAccount")
	if err != nil {
		return nil, fmt.Errorf("unable to update workspace service account: %w", err)
	}
	return plan, applyServiceAccountResponse(ctx, plan, resp.GetServiceAccount())
}

func serviceAccountDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, state *types.ServiceAccount) error {
	if isWorkspaceScoped(state.Workspace) {
		workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, state.Workspace.ValueString())
		if err != nil {
			// Workspace gone: its accounts went with it.
			if isGoneErr(err) {
				return nil
			}
			return fmt.Errorf("unable to resolve workspace %q: %w", state.Workspace.ValueString(), err)
		}
		_, err = retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.DeleteWorkspaceServiceAccountResponse, error) {
			resp, err := cli.ServiceAccountCli.DeleteWorkspaceServiceAccount(ctx, &serviceaccountv1.DeleteWorkspaceServiceAccountRequest{
				OrganizationId: cli.OrgId,
				WorkspaceId:    workspace.GetId(),
				Id:             state.ID.ValueString(),
			})
			if isGoneErr(err) {
				return resp, nil
			}
			return resp, err
		}, "DeleteWorkspaceServiceAccount")
		if err != nil {
			return fmt.Errorf("unable to delete workspace service account: %w", err)
		}
		return nil
	}
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*serviceaccountv1.DeleteServiceAccountResponse, error) {
		resp, err := cli.ServiceAccountCli.DeleteServiceAccount(ctx, &serviceaccountv1.DeleteServiceAccountRequest{
			OrganizationId: cli.OrgId,
			Id:             state.ID.ValueString(),
		})
		if isGoneErr(err) {
			return resp, nil
		}
		return resp, err
	}, "DeleteServiceAccount")
	if err != nil {
		return fmt.Errorf("unable to delete service account: %w", err)
	}
	return nil
}

func isWorkspaceScoped(workspace tftypes.String) bool {
	return !workspace.IsNull() && workspace.ValueString() != ""
}

func serviceAccountSpec(ctx context.Context, plan *types.ServiceAccount) (*serviceaccountv1.ServiceAccountSpec, error) {
	perms, err := buildApiKeyPermissions(plan.Permissions)
	if err != nil {
		return nil, err
	}
	if plan.OIDCBinding == nil {
		return nil, fmt.Errorf("oidc_binding is required")
	}
	match := map[string]string{}
	if diags := plan.OIDCBinding.Match.ElementsAs(ctx, &match, false); diags.HasError() {
		return nil, fmt.Errorf("oidc_binding.match must map claim names to patterns")
	}
	if _, ok := match["sub"]; !ok {
		return nil, fmt.Errorf("oidc_binding.match must include the sub claim")
	}
	return &serviceaccountv1.ServiceAccountSpec{
		Description: plan.Description.ValueString(),
		Permissions: perms,
		IpAllowlist: stringSliceFromTF(plan.IPAllowlist),
		OidcBinding: &serviceaccountv1.OIDCBinding{
			IssuerId: plan.OIDCBinding.IssuerID.ValueString(),
			Match:    match,
		},
		Disabled: plan.Disabled.ValueBool(),
	}, nil
}

func applyServiceAccountResponse(ctx context.Context, data *types.ServiceAccount, sa *serviceaccountv1.ServiceAccount) error {
	if sa == nil {
		return nil
	}
	data.ID = tftypes.StringValue(sa.GetId())
	// An omitted description stays null rather than becoming "".
	if sa.GetDescription() != "" || !data.Description.IsNull() {
		data.Description = tftypes.StringValue(sa.GetDescription())
	}
	data.Disabled = tftypes.BoolValue(sa.GetDisabled())
	data.WorkspaceID = tftypes.StringValue(sa.GetWorkspaceId())
	data.IPAllowlist = applyStringList(data.IPAllowlist, sa.GetIpAllowlist())
	if sa.GetPermissions() != nil {
		// Workspace accounts get organization/member appended server-side;
		// keep only the namespace the operator writes, as for API keys.
		wantNamespace := "organization"
		if isWorkspaceScoped(data.Workspace) {
			wantNamespace = "workspace"
		}
		var prior types.ApiKeyPermissions
		if data.Permissions != nil {
			prior = *data.Permissions
		}
		data.Permissions = &types.ApiKeyPermissions{
			Actions:     applyStringList(prior.Actions, sa.GetPermissions().GetActions()),
			Roles:       applyStringList(prior.Roles, stripRoleNamespace(filterRolesByNamespace(sa.GetPermissions().GetRoles(), wantNamespace))),
			CustomRoles: applyStringList(prior.CustomRoles, sa.GetPermissions().GetCustomRoles()),
		}
	}
	if binding := sa.GetOidcBinding(); binding != nil {
		elements := make(map[string]attr.Value, len(binding.GetMatch()))
		for claim, pattern := range binding.GetMatch() {
			elements[claim] = tftypes.StringValue(pattern)
		}
		match, diags := tftypes.MapValue(tftypes.StringType, elements)
		if diags.HasError() {
			return fmt.Errorf("decode oidc_binding.match: %v", diags)
		}
		data.OIDCBinding = &types.ServiceAccountOIDCBinding{
			IssuerID: tftypes.StringValue(binding.GetIssuerId()),
			Match:    match,
		}
	}
	if t := sa.GetCreationTimestamp(); t != nil {
		data.CreateTime = tftypes.StringValue(t.AsTime().Format(time.RFC3339))
	}
	if t := sa.GetLastUsedTimestamp(); t != nil {
		data.LastUsedTime = tftypes.StringValue(t.AsTime().Format(time.RFC3339))
	} else {
		data.LastUsedTime = tftypes.StringValue("")
	}
	return nil
}
