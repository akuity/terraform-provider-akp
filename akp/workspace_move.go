package akp

import (
	"context"

	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
)

// ensureInstanceWorkspace moves the Argo CD instance into the given workspace
// when it currently lives in a different one. Each retry re-reads the instance
// so a successful move with a lost response is observed instead of repeated.
func ensureInstanceWorkspace(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, orgID, instanceID, instanceName string, workspace *orgcv1.Workspace) (moveAttempted bool, err error) {
	_, err = retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.UpdateInstanceWorkspaceResponse, error) {
		getResp, getErr := client.GetInstance(ctx, instanceGetRequest(orgID, instanceID, instanceName))
		if getErr != nil {
			return nil, getErr
		}
		instance := getResp.GetInstance()
		if instance == nil {
			return nil, errors.New("GetInstance response did not include an instance")
		}
		current := instance.GetWorkspaceId()
		if workspaceMoveNotNeeded(current, workspace) {
			return &argocdv1.UpdateInstanceWorkspaceResponse{}, nil
		}
		moveAttempted = true
		return client.UpdateInstanceWorkspace(ctx, &argocdv1.UpdateInstanceWorkspaceRequest{
			OrganizationId: orgID,
			Id:             instance.GetId(),
			WorkspaceId:    current,
			NewWorkspaceId: workspace.GetId(),
		})
	}, "EnsureInstanceWorkspace")
	if err != nil {
		return moveAttempted, errors.Wrapf(err, "Unable to move Argo CD instance to workspace %q", workspace.GetName())
	}
	return moveAttempted, nil
}

// ensureKargoInstanceWorkspace is the Kargo counterpart of
// ensureInstanceWorkspace: ApplyKargoInstance also ignores workspace_id for
// existing instances. Unlike the Argo CD variant it must run BEFORE
// ApplyKargoInstance: that RPC's access check validates the request's
// workspace_id against the instance's actual workspace, so applying with the
// new workspace while the instance is still in the old one is rejected.
// A not-yet-existing instance (create path) is not an error — there is
// nothing to move and the apply will create it in the target workspace.
func ensureKargoInstanceWorkspace(ctx context.Context, client kargov1.KargoServiceGatewayClient, orgID, instanceID, instanceName string, workspace *orgcv1.Workspace) (moveAttempted bool, err error) {
	_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.UpdateKargoInstanceWorkspaceResponse, error) {
		getResp, getErr := getKargoInstanceByIdentity(ctx, client, orgID, instanceID, instanceName)
		if getErr != nil {
			if instanceID == "" && status.Code(getErr) == codes.NotFound {
				return &kargov1.UpdateKargoInstanceWorkspaceResponse{}, nil
			}
			return nil, getErr
		}
		instance := getResp.GetInstance()
		if instance == nil {
			return nil, errors.New("Kargo instance lookup did not include an instance")
		}
		current := instance.GetWorkspaceId()
		if workspaceMoveNotNeeded(current, workspace) {
			return &kargov1.UpdateKargoInstanceWorkspaceResponse{}, nil
		}
		moveAttempted = true
		return client.UpdateKargoInstanceWorkspace(ctx, &kargov1.UpdateKargoInstanceWorkspaceRequest{
			OrganizationId: orgID,
			Id:             instance.GetId(),
			WorkspaceId:    current,
			NewWorkspaceId: workspace.GetId(),
		})
	}, "EnsureKargoInstanceWorkspace")
	if err != nil {
		return moveAttempted, errors.Wrapf(err, "Unable to move Kargo instance to workspace %q", workspace.GetName())
	}
	return moveAttempted, nil
}

// getKargoInstanceByIdentity uses the immutable instance ID for managed
// resources. Kargo's Get API only supports names, so ID lookups list the
// caller-visible instances and require an exact match without a name fallback.
func getKargoInstanceByIdentity(ctx context.Context, client kargov1.KargoServiceGatewayClient, orgID, instanceID, instanceName string) (*kargov1.GetKargoInstanceResponse, error) {
	if instanceID == "" {
		return client.GetKargoInstance(ctx, &kargov1.GetKargoInstanceRequest{
			OrganizationId: orgID,
			Name:           instanceName,
		})
	}

	listResp, err := client.ListKargoInstances(ctx, &kargov1.ListKargoInstancesRequest{
		OrganizationId: orgID,
	})
	if err != nil {
		return nil, err
	}
	for _, instance := range listResp.GetInstances() {
		if instance.GetId() == instanceID {
			return &kargov1.GetKargoInstanceResponse{Instance: instance}, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "Kargo instance with ID %q not found", instanceID)
}

// workspaceStateFromAPI resolves an API workspace ID to the Terraform state
// value. Ordinary reads preserve the provider's existing best-effort behavior;
// post-apply recovery can require a successful lookup before committing state.
func workspaceStateFromAPI(
	ctx context.Context,
	client orgcv1.OrganizationServiceGatewayClient,
	orgID string,
	workspaceID string,
	current tftypes.String,
	strict bool,
) (tftypes.String, error) {
	if workspaceID == "" {
		if strict {
			return current, errors.New("instance response has an empty workspace ID")
		}
		return current, nil
	}
	workspace, err := getWorkspaceByID(ctx, client, orgID, workspaceID)
	if err != nil {
		if strict {
			return current, errors.Wrapf(err, "unable to resolve workspace ID %q after apply", workspaceID)
		}
		return current, nil
	}
	if workspace == nil {
		if strict {
			return current, errors.Errorf("workspace lookup for ID %q returned no workspace", workspaceID)
		}
		return current, nil
	}
	return workspaceStateValue(current, workspace), nil
}

// workspaceStateValue preserves an explicitly configured empty string as the
// Terraform spelling of the default workspace. Replacing it with the default
// workspace's name would violate Terraform's planned-value contract and cause
// a perpetual diff. Null/unknown values and API drift still hydrate to the
// workspace's actual name.
func workspaceStateValue(current tftypes.String, workspace *orgcv1.Workspace) tftypes.String {
	if workspace.GetIsDefault() && !current.IsNull() && !current.IsUnknown() && current.ValueString() == "" {
		return current
	}
	return tftypes.StringValue(workspace.GetName())
}

// workspaceMoveNotNeeded reports whether the instance is already in the target
// workspace. An instance with no workspace assigned (created before workspaces
// were enabled for the org) implicitly belongs to the default workspace, so
// targeting the default workspace is not a move.
func workspaceMoveNotNeeded(currentWorkspaceID string, target *orgcv1.Workspace) bool {
	if currentWorkspaceID == target.GetId() {
		return true
	}
	return currentWorkspaceID == "" && target.GetIsDefault()
}
