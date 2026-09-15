package akp

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/wait"
)

func waitForStatus[ResourceType any, StatusCodeType comparable](
	ctx context.Context,
	getResourceFunc func(ctx context.Context) (ResourceType, error),
	getStatusFunc func(resource ResourceType) StatusCodeType,
	targetStatuses []StatusCodeType,
	pollInterval time.Duration,
	timeout time.Duration,
	resourceName string,
	statusName string,
) error {
	tflog.Debug(ctx, fmt.Sprintf("Waiting for %s %s status to reach one of %v", resourceName, statusName, targetStatuses))
	start := time.Now()
	lastLog := start
	var lastStatus StatusCodeType
	err := wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		resource, err := getResourceFunc(ctx)
		if status.Code(err) == codes.NotFound {
			tflog.Debug(ctx, fmt.Sprintf("%s not found yet, retrying...", resourceName))
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("failed to get %s during status wait: %w", resourceName, err)
		}
		lastStatus = getStatusFunc(resource)
		if time.Since(lastLog) >= 30*time.Second {
			tflog.Info(ctx, fmt.Sprintf("%s %s status: %v (elapsed: %v, target: %v)", resourceName, statusName, lastStatus, time.Since(start), targetStatuses))
			lastLog = time.Now()
		}
		return slices.Contains(targetStatuses, lastStatus), nil
	})
	if wait.Interrupted(err) && ctx.Err() == nil {
		return fmt.Errorf("timed out after %v waiting for %s %s status (expected: %v, last seen: %v)", timeout, resourceName, statusName, targetStatuses, lastStatus)
	}
	return err
}

// isGoneErr reports whether err indicates the resource is no longer accessible
// to the caller — either because it was deleted (NotFound) or because the
// server revokes ACL access for deleted records and surfaces it as
// PermissionDenied. Tests and Delete handlers treat both as "successfully
// gone" since a caller already authorized to write the resource has no other
// reason to see PermissionDenied immediately after a delete.
func isGoneErr(err error) bool {
	if err == nil {
		return false
	}
	switch status.Code(err) {
	case codes.NotFound, codes.PermissionDenied:
		return true
	default:
		return false
	}
}

// Six attempts with 1s, 2s, 4s, 8s and 16s between them, about 31s of retrying in total.
var retryBackoff = wait.Backoff{Duration: time.Second, Factor: 2, Jitter: 0.1, Steps: 6, Cap: 30 * time.Second}

// retryWithBackoff retries operation on retryable errors with exponential backoff.
func retryWithBackoff[T any](
	ctx context.Context,
	operation func(ctx context.Context) (T, error),
	operationName string,
) (T, error) {
	var result T
	var lastErr error
	err := wait.ExponentialBackoffWithContext(ctx, retryBackoff, func(ctx context.Context) (bool, error) {
		result, lastErr = operation(ctx)
		if lastErr != nil && isRetryableError(lastErr) {
			tflog.Debug(ctx, fmt.Sprintf("%s failed with retryable error: %v", operationName, lastErr))
			return false, nil
		}
		return true, nil
	})
	switch {
	case err == nil:
		return result, lastErr
	case ctx.Err() != nil:
		return result, ctx.Err()
	default:
		return result, fmt.Errorf("%s failed after %d attempts: %w", operationName, retryBackoff.Steps, lastErr)
	}
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Extract gRPC status code
	st, ok := status.FromError(err)
	if ok {
		switch st.Code() {
		case codes.Unavailable,
			codes.DeadlineExceeded,
			codes.Aborted,
			codes.Canceled,
			codes.Internal:
			return true
		case codes.ResourceExhausted:
			// A "received message larger than max" overflow is deterministic:
			// every retry rebuilds the same oversized payload and burns the
			// whole backoff schedule (#11935). Fail fast on it, but keep
			// retrying genuine transient resource exhaustion (rate limits).
			return !strings.Contains(st.Message(), "larger than max")
		case codes.InvalidArgument:
			// Retry InvalidArgument only if it's due to provisioning delays
			if strings.Contains(st.Message(), "still being provisioned") {
				return true
			}
			return false
		default:
			return false
		}
	}

	// For non-gRPC errors, be conservative and retry only on clearly temporary issues
	errStr := err.Error()

	// Network-related errors that are usually temporary
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "no route to host") ||
		strings.Contains(errStr, "service unavailable") {
		return true
	}

	return false
}

// isLastWorkspaceMemberErr reports whether err is the backend's refusal to
// remove the final member of a workspace (RemoveWorkspaceMember rejects when
// memberCount == 1). DeleteWorkspace cascade-removes its members and the backend
// forbids a memberless workspace, so removing the last member is only meaningful
// as part of deleting the workspace — which Terraform does immediately after
// (the member resource depends on the workspace, so it is destroyed first).
// Delete treats this as success so `terraform destroy` completes; the workspace
// delete that follows cleans up the member. The trade-off is that removing the
// last member while keeping the workspace reports success without removing it —
// an operation the backend disallows as a standalone end-state anyway.
func isLastWorkspaceMemberErr(err error) bool {
	st := status.Convert(err)
	return st.Code() == codes.InvalidArgument &&
		strings.Contains(st.Message(), "last member of the workspace")
}

func isConnectedKargoAgentsDeleteError(err error) bool {
	st := status.Convert(err)
	return st.Code() == codes.InvalidArgument &&
		(strings.Contains(st.Message(), "instance has some connected kargo agents") ||
			strings.Contains(st.Message(), "delete them before deleting instance"))
}

func deleteWithCooldown[Resp any](
	ctx context.Context,
	deleteFunc func(ctx context.Context) (Resp, error),
	operationName string,
	cooldown time.Duration,
) error {
	_, err := retryWithBackoff(ctx, deleteFunc, operationName)
	if err != nil {
		return err
	}
	select {
	case <-time.After(cooldown):
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}
