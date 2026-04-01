package akp

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	waitCtx := ctx
	if _, deadlineSet := ctx.Deadline(); !deadlineSet {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	startTime := time.Now()
	lastStatusLog := time.Now()
	var lastStatus StatusCodeType

	for {
		select {
		case <-waitCtx.Done():
			elapsed := time.Since(startTime)
			if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("timed out after %v waiting for %s %s status (expected: %v, last seen: %v)", timeout, resourceName, statusName, targetStatuses, lastStatus)
			}
			return fmt.Errorf("context cancelled/done while waiting for %s %s status after %v: %w", resourceName, statusName, elapsed, waitCtx.Err())
		default:
		}

		resource, err := getResourceFunc(waitCtx)
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.NotFound {
				tflog.Debug(ctx, fmt.Sprintf("%s not found yet, retrying...", resourceName))
			} else {
				elapsed := time.Since(startTime)
				tflog.Error(ctx, fmt.Sprintf("Failed to get %s during status wait after %v: %v", resourceName, elapsed, err))
				return fmt.Errorf("failed to get %s during status wait: %w", resourceName, err)
			}
		} else {
			currentStatus := getStatusFunc(resource)
			lastStatus = currentStatus

			if time.Since(lastStatusLog) >= 30*time.Second {
				elapsed := time.Since(startTime)
				tflog.Info(ctx, fmt.Sprintf("%s %s status: %v (elapsed: %v, target: %v)", resourceName, statusName, currentStatus, elapsed, targetStatuses))
				lastStatusLog = time.Now()
			} else {
				tflog.Debug(ctx, fmt.Sprintf("%s %s status: %v", resourceName, statusName, currentStatus))
			}

			if slices.Contains(targetStatuses, currentStatus) {
				elapsed := time.Since(startTime)
				tflog.Info(ctx, fmt.Sprintf("%s %s status reached target state (%v) after %v", resourceName, statusName, currentStatus, elapsed))
				return nil
			}
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-timer.C:
		case <-waitCtx.Done():
			timer.Stop()
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
			codes.Canceled,
			codes.Internal:
			return true
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

func isConnectedKargoAgentsDeleteError(err error) bool {
	if err == nil {
		return false
	}

	type grpcStatus interface {
		GRPCStatus() *status.Status
	}

	var statusErr grpcStatus
	if !errors.As(err, &statusErr) {
		return false
	}

	st := statusErr.GRPCStatus()
	msg := st.Message()
	if !strings.Contains(msg, "instance has some connected kargo agents") &&
		!strings.Contains(msg, "delete them before deleting instance") {
		return false
	}

	return st.Code() == codes.InvalidArgument
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
