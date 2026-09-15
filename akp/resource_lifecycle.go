package akp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ResourceLifecycle encapsulates the common Apply → Wait → Refresh pipeline
// used by resources that follow the pattern: apply changes, wait for health status,
// then refresh state from the API.
type ResourceLifecycle[Plan any, APIResponse any, StatusCode comparable] struct {
	// Apply executes the create/update flow. It returns true once the primary
	// apply request has succeeded, even if a later operation
	// (such as a workspace move) returns an error.
	Apply func(ctx context.Context, diagnostics *diag.Diagnostics, plan *Plan) (applyRequestSucceeded bool, err error)
	// Get fetches the current resource from the API (used during wait polling).
	Get func(ctx context.Context, plan *Plan) (APIResponse, error)
	// GetStatus extracts the status code from the API response.
	GetStatus func(resp APIResponse) StatusCode
	// GetGeneration extracts the generation or version number from the API response, if available.
	GetGeneration func(resp APIResponse) uint32
	// GetReconciliationDone returns true when the controller has successfully
	// finished processing (SUCCESSFUL — not PROGRESSING or FAILED).
	GetReconciliationDone func(resp APIResponse) bool
	// GetReconciliationFailed returns true when reconciliation has failed.
	// Used to fast-fail instead of waiting for the full timeout.
	GetReconciliationFailed func(resp APIResponse) bool
	// TargetStatuses is the set of statuses that indicate the resource is ready.
	TargetStatuses []StatusCode
	// Refresh updates the plan with the latest state from the API after the wait completes.
	Refresh func(ctx context.Context, diagnostics *diag.Diagnostics, plan *Plan) error
	// RefreshAfterApplyError optionally performs stricter state hydration after
	// Apply reports that the primary apply request succeeded but a later step
	// failed. When unset, Refresh is used.
	RefreshAfterApplyError func(ctx context.Context, diagnostics *diag.Diagnostics, plan *Plan) error
	// ResourceName returns a human-readable name for logging and error messages.
	ResourceName func(plan *Plan) string
	// StatusName is the status type being waited on (e.g. "health", "reconciliation").
	StatusName string
	// PollInterval is how often to poll during the wait. Default: 10s.
	PollInterval time.Duration
	// Timeout is the maximum time to wait for the target status. Default: 5m.
	Timeout time.Duration
}

// Upsert executes the full Apply → Wait → Refresh pipeline.
// Returns whether the caller should commit the current plan/state, together with
// any error encountered.
func (lc *ResourceLifecycle[Plan, APIResponse, StatusCode]) Upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *Plan) (stateCanBeCommitted bool, err error) {
	var preApplyGeneration uint32
	if lc.GetGeneration != nil {
		resp, err := lc.Get(ctx, plan)
		if err == nil {
			preApplyGeneration = lc.GetGeneration(resp)
			tflog.Debug(ctx, fmt.Sprintf("%s pre-apply generation: %d", lc.ResourceName(plan), preApplyGeneration))
		}
	}

	applyRequestSucceeded, applyErr := lc.Apply(ctx, diagnostics, plan)
	if applyErr != nil {
		if !applyRequestSucceeded {
			return false, applyErr
		}
		// A post-apply operation (currently the Argo CD workspace move) may
		// fail after the primary apply request was accepted. Refresh with
		// isolated diagnostics so only authoritative state is committed.
		var refreshDiagnostics diag.Diagnostics
		refresh := lc.Refresh
		if lc.RefreshAfterApplyError != nil {
			refresh = lc.RefreshAfterApplyError
		}
		refreshErr := refresh(ctx, &refreshDiagnostics, plan)
		refreshHasError := refreshDiagnostics.HasError()
		diagnostics.Append(refreshDiagnostics...)
		if refreshErr != nil || refreshHasError {
			if refreshErr == nil {
				refreshErr = fmt.Errorf("refresh for %s produced diagnostics errors", lc.ResourceName(plan))
			}
			return false, errors.Join(
				applyErr,
				fmt.Errorf("unable to refresh %s after an applied change: %w", lc.ResourceName(plan), refreshErr),
			)
		}
		return true, applyErr
	}
	if diagnostics.HasError() {
		return false, fmt.Errorf("diagnostics errors during apply for %s", lc.ResourceName(plan))
	}

	pollInterval := lc.PollInterval
	if pollInterval == 0 {
		pollInterval = 10 * time.Second
	}
	timeout := lc.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	resourceName := lc.ResourceName(plan)

	var waitErr error
	useGenerationWait := lc.GetGeneration != nil && lc.GetReconciliationDone != nil && preApplyGeneration > 0
	if useGenerationWait {
		waitErr = lc.waitForReconciliation(ctx, plan, preApplyGeneration, pollInterval, timeout, resourceName)
	} else {
		waitErr = waitForStatus(
			ctx,
			func(ctx context.Context) (APIResponse, error) { return lc.Get(ctx, plan) },
			lc.GetStatus,
			lc.TargetStatuses,
			pollInterval,
			timeout,
			resourceName,
			lc.StatusName,
		)
	}

	if waitErr != nil {
		diagnostics.AddError(
			"Instance Wait Error",
			fmt.Sprintf("%s did not reach target status: %s", resourceName, waitErr.Error()),
		)
		tflog.Error(ctx, fmt.Sprintf("%s wait failed: %s", resourceName, waitErr.Error()))
		return true, waitErr
	}

	return true, lc.Refresh(ctx, diagnostics, plan)
}

const reconGracePeriod = 15 * time.Second

func (lc *ResourceLifecycle[Plan, APIResponse, StatusCode]) waitForReconciliation(
	ctx context.Context,
	plan *Plan,
	preApplyGeneration uint32,
	pollInterval time.Duration,
	timeout time.Duration,
	resourceName string,
) error {
	start := time.Now()
	lastLog := start
	sawReconciling := false
	tflog.Info(ctx, fmt.Sprintf("%s waiting for reconciliation (pre-apply generation: %d)", resourceName, preApplyGeneration))
	err := wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		resp, err := lc.Get(ctx, plan)
		if status.Code(err) == codes.NotFound {
			tflog.Debug(ctx, fmt.Sprintf("%s not found yet, retrying...", resourceName))
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("failed to get %s during reconciliation wait: %w", resourceName, err)
		}
		currentStatus := lc.GetStatus(resp)
		currentGen := lc.GetGeneration(resp)
		reconDone := lc.GetReconciliationDone(resp)
		if !reconDone {
			sawReconciling = true
		}
		// A no-op apply never advances the generation, so accept the current state once the grace period passes.
		reconcileConfirmed := currentGen > preApplyGeneration || sawReconciling || time.Since(start) >= reconGracePeriod
		if time.Since(lastLog) >= 30*time.Second {
			tflog.Info(ctx, fmt.Sprintf("%s status: %v, generation: %d (pre: %d), recon_done: %v, confirmed: %v (elapsed: %v)",
				resourceName, currentStatus, currentGen, preApplyGeneration, reconDone, reconcileConfirmed, time.Since(start)))
			lastLog = time.Now()
		}
		if lc.GetReconciliationFailed != nil && lc.GetReconciliationFailed(resp) {
			return false, fmt.Errorf("%s reconciliation failed (health status: %v)", resourceName, currentStatus)
		}
		return reconcileConfirmed && reconDone && slices.Contains(lc.TargetStatuses, currentStatus), nil
	})
	if wait.Interrupted(err) && ctx.Err() == nil {
		return fmt.Errorf("timed out after %v waiting for %s reconciliation", timeout, resourceName)
	}
	return err
}
