package akp

import (
	"context"
	"fmt"
	"io"
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
			} else if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return errors.Wrapf(err, "context cancelled/timeout while getting %s during status wait", resourceName)
			} else if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "EOF") {
				tflog.Debug(ctx, fmt.Sprintf("EOF or connection error getting %s, retrying... (error: %v)", resourceName, err))
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
