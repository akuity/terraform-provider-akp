package akp

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ResourceLifecycle encapsulates the common Apply → Wait → Refresh pipeline
// used by resources that follow the pattern: apply changes, wait for health status,
// then refresh state from the API.
type ResourceLifecycle[Plan any, APIResponse any, StatusCode comparable] struct {
	// Apply executes the create/update API call.
	Apply func(ctx context.Context, diagnostics *diag.Diagnostics, plan *Plan) error
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
// Returns true if Apply succeeded (state should be committed even if Wait/Refresh fails),
// and any error encountered.
func (lc *ResourceLifecycle[Plan, APIResponse, StatusCode]) Upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *Plan) (applied bool, err error) {
	var preApplyGeneration uint32
	if lc.GetGeneration != nil {
		resp, err := lc.Get(ctx, plan)
		if err == nil {
			preApplyGeneration = lc.GetGeneration(resp)
			tflog.Debug(ctx, fmt.Sprintf("%s pre-apply generation: %d", lc.ResourceName(plan), preApplyGeneration))
		}
	}

	if err := lc.Apply(ctx, diagnostics, plan); err != nil {
		return false, err
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

const (
	reconGracePeriod       = 15 * time.Second
	reconGracePollInterval = 2 * time.Second
)

func (lc *ResourceLifecycle[Plan, APIResponse, StatusCode]) waitForReconciliation(
	ctx context.Context,
	plan *Plan,
	preApplyGeneration uint32,
	pollInterval time.Duration,
	timeout time.Duration,
	resourceName string,
) error {
	waitCtx := ctx
	if _, deadlineSet := ctx.Deadline(); !deadlineSet {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	startTime := time.Now()
	lastStatusLog := time.Now()
	sawReconciling := false

	tflog.Info(ctx, fmt.Sprintf("%s waiting for reconciliation (pre-apply generation: %d)", resourceName, preApplyGeneration))

	for {
		select {
		case <-waitCtx.Done():
			elapsed := time.Since(startTime)
			if ctx.Err() != nil {
				return fmt.Errorf("context cancelled while waiting for %s after %v", resourceName, elapsed)
			}
			return fmt.Errorf("timed out after %v waiting for %s reconciliation", timeout, resourceName)
		default:
		}

		resp, err := lc.Get(waitCtx, plan)
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.NotFound {
				tflog.Debug(ctx, fmt.Sprintf("%s not found yet, retrying...", resourceName))
			} else {
				return fmt.Errorf("failed to get %s during reconciliation wait: %w", resourceName, err)
			}
		} else {
			currentStatus := lc.GetStatus(resp)
			currentGen := lc.GetGeneration(resp)
			isTarget := slices.Contains(lc.TargetStatuses, currentStatus)
			reconDone := lc.GetReconciliationDone(resp)
			genAdvanced := currentGen > preApplyGeneration

			if !reconDone {
				sawReconciling = true
			}

			gracePeriodExpired := time.Since(startTime) >= reconGracePeriod
			reconcileConfirmed := genAdvanced || sawReconciling || gracePeriodExpired

			if time.Since(lastStatusLog) >= 30*time.Second {
				tflog.Info(ctx, fmt.Sprintf(
					"%s status: %v, generation: %d (pre: %d), recon_done: %v, health_target: %v, confirmed: %v (elapsed: %v)",
					resourceName, currentStatus, currentGen, preApplyGeneration, reconDone, isTarget, reconcileConfirmed, time.Since(startTime)))
				lastStatusLog = time.Now()
			} else {
				tflog.Debug(ctx, fmt.Sprintf(
					"%s status: %v, generation: %d (pre: %d), recon_done: %v, health_target: %v, confirmed: %v",
					resourceName, currentStatus, currentGen, preApplyGeneration, reconDone, isTarget, reconcileConfirmed))
			}

			if lc.GetReconciliationFailed != nil && lc.GetReconciliationFailed(resp) {
				return fmt.Errorf("%s reconciliation failed (health status: %v)", resourceName, currentStatus)
			}

			if reconcileConfirmed && reconDone && isTarget {
				tflog.Info(ctx, fmt.Sprintf("%s reconciliation complete: status=%v, generation=%d (elapsed: %v)", resourceName, currentStatus, currentGen, time.Since(startTime)))
				return nil
			}
		}

		currentPollInterval := pollInterval
		if !sawReconciling && time.Since(startTime) < reconGracePeriod {
			currentPollInterval = reconGracePollInterval
		}

		timer := time.NewTimer(currentPollInterval)
		select {
		case <-timer.C:
		case <-waitCtx.Done():
			timer.Stop()
		}
	}
}
