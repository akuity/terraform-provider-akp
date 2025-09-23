package akp

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// waitForStatus polls a resource until its status reaches one of the target statuses or a timeout occurs.
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

	waitCtx := ctx
	if _, deadlineSet := ctx.Deadline(); !deadlineSet {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	for {
		select {
		case <-waitCtx.Done():
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return errors.Errorf("timed out after %v waiting for %s %s status", timeout, resourceName, statusName)
			}
			return errors.Wrapf(waitCtx.Err(), "context cancelled/done while waiting for %s %s status", resourceName, statusName)
		default:
		}

		resource, err := getResourceFunc(waitCtx)
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.NotFound {
				tflog.Debug(ctx, fmt.Sprintf("%s not found yet, retrying...", resourceName))
			} else {
				return errors.Wrapf(err, "failed to get %s during status wait", resourceName)
			}
		} else {
			currentStatus := getStatusFunc(resource)
			tflog.Debug(ctx, fmt.Sprintf("%s %s status: %v", resourceName, statusName, currentStatus))

			if slices.Contains(targetStatuses, currentStatus) {
				tflog.Info(ctx, fmt.Sprintf("%s %s status reached target state (%v).", resourceName, statusName, currentStatus))
				return nil
			}
		}

		select {
		case <-time.After(pollInterval):
		case <-waitCtx.Done():
		}
	}
}

// retryWithBackoff executes a function with exponential backoff retry logic
func retryWithBackoff[T any](
	ctx context.Context,
	operation func(ctx context.Context) (T, error),
	operationName string,
) (T, error) {
	const (
		maxRetries    = 5
		initialDelay  = 500 * time.Millisecond
		maxDelay      = 30 * time.Second
		backoffFactor = 2.0
	)

	var result T
	var lastErr error
	delay := initialDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Apply jitter to prevent thundering herd (10% jitter)
			jitterRange := time.Duration(float64(delay) * 0.1)
			jitter := time.Duration(rand.Int64N(int64(jitterRange*2))) - jitterRange
			actualDelay := delay + jitter
			if actualDelay < 0 {
				actualDelay = delay
			}

			tflog.Debug(ctx, fmt.Sprintf("Retrying %s (attempt %d/%d) after %v", operationName, attempt, maxRetries, actualDelay))

			select {
			case <-time.After(actualDelay):
				// Continue with retry
			case <-ctx.Done():
				return result, ctx.Err()
			}
		}

		result, lastErr = operation(ctx)
		if lastErr == nil {
			if attempt > 0 {
				tflog.Info(ctx, fmt.Sprintf("%s succeeded after %d retries", operationName, attempt))
			}
			return result, nil
		}

		// Check if the error is retryable
		if !isRetryableError(lastErr) {
			tflog.Debug(ctx, fmt.Sprintf("%s failed with non-retryable error: %v", operationName, lastErr))
			return result, lastErr
		}

		tflog.Debug(ctx, fmt.Sprintf("%s failed with retryable error (attempt %d/%d): %v", operationName, attempt+1, maxRetries+1, lastErr))

		// Exponential backoff with cap
		delay = time.Duration(float64(delay) * backoffFactor)
		if delay > maxDelay {
			delay = maxDelay
		}
	}

	tflog.Error(ctx, fmt.Sprintf("%s failed after %d retries, last error: %v", operationName, maxRetries+1, lastErr))
	return result, fmt.Errorf("%s failed after %d retries: %w", operationName, maxRetries+1, lastErr)
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
			codes.ResourceExhausted,
			codes.Canceled:
			return true
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
