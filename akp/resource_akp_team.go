package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func NewAkpTeamResource() resource.Resource {
	return &GenericResource[types.Team]{
		TypeNameSuffix: "team",
		SchemaFunc:     teamSchema,
		CreateFunc:     teamCreate,
		ReadFunc:       teamRead,
		UpdateFunc:     teamUpdate,
		DeleteFunc:     teamDelete,
		ImportStateFunc: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			// Teams are identified by name.
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
		},
	}
}

func teamCreate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.Team) (*types.Team, error) {
	createReq := &orgcv1.CreateTeamRequest{
		OrganizationId: cli.OrgId,
		Name:           plan.Name.ValueString(),
		CustomRoles:    stringSliceFromTF(plan.CustomRoles),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		createReq.Description = &desc
	}

	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.CreateTeamResponse, error) {
		return cli.OrgCli.CreateTeam(ctx, createReq)
	}, "CreateTeam")
	if err != nil {
		return nil, fmt.Errorf("unable to create team: %w", err)
	}
	applyTeamResponse(plan, resp.GetUserTeam())
	return plan, nil
}

func teamRead(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, data *types.Team) error {
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.GetTeamResponse, error) {
		return cli.OrgCli.GetTeam(ctx, &orgcv1.GetTeamRequest{
			OrganizationId: cli.OrgId,
			Name:           data.Name.ValueString(),
		})
	}, "GetTeam")
	if err != nil {
		return err
	}
	if resp.GetUserTeam() == nil || resp.GetUserTeam().GetTeam() == nil {
		return status.Error(codes.NotFound, "team not found")
	}
	applyTeamResponse(data, resp.GetUserTeam())
	return nil
}

func teamUpdate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.Team) (*types.Team, error) {
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.UpdateTeamResponse, error) {
		return cli.OrgCli.UpdateTeam(ctx, &orgcv1.UpdateTeamRequest{
			OrganizationId: cli.OrgId,
			Name:           plan.Name.ValueString(),
			Description:    plan.Description.ValueString(),
			CustomRoles:    stringSliceFromTF(plan.CustomRoles),
		})
	}, "UpdateTeam")
	if err != nil {
		return nil, fmt.Errorf("unable to update team: %w", err)
	}
	applyTeamResponse(plan, resp.GetUserTeam())
	return plan, nil
}

func teamDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, state *types.Team) error {
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.DeleteTeamResponse, error) {
		resp, err := cli.OrgCli.DeleteTeam(ctx, &orgcv1.DeleteTeamRequest{
			OrganizationId: cli.OrgId,
			Name:           state.Name.ValueString(),
		})
		if isGoneErr(err) {
			return resp, nil
		}
		return resp, err
	}, "DeleteTeam")
	if err != nil {
		return fmt.Errorf("unable to delete team: %w", err)
	}
	return nil
}

func applyTeamResponse(data *types.Team, userTeam *orgcv1.UserTeam) {
	if userTeam == nil || userTeam.GetTeam() == nil {
		return
	}
	team := userTeam.GetTeam()
	data.Name = tftypes.StringValue(team.GetName())
	data.Description = tftypes.StringValue(team.GetDescription())
	data.CustomRoles = applyStringList(data.CustomRoles, userTeam.GetCustomRoles())
	if t := team.GetCreateTime(); t != nil {
		data.CreateTime = tftypes.StringValue(t.AsTime().Format("2006-01-02T15:04:05Z07:00"))
	}
	data.MemberCount = tftypes.Int64Value(team.GetMemberCount())
}
