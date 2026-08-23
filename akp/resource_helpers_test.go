//go:build !acc

package akp

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpc_testing "google.golang.org/grpc/interop/grpc_testing"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type oversizedResponseServer struct {
	grpc_testing.UnimplementedTestServiceServer
	payloadSize int
}

func (s *oversizedResponseServer) UnaryCall(context.Context, *grpc_testing.SimpleRequest) (*grpc_testing.SimpleResponse, error) {
	return &grpc_testing.SimpleResponse{
		Payload: &grpc_testing.Payload{Body: make([]byte, s.payloadSize)},
	}, nil
}

func TestIsRetryableError_ResourceExhausted(t *testing.T) {
	testCases := map[string]struct {
		err      error
		expected bool
	}{
		// Deterministic message-size overflow: every retry rebuilds the same
		// oversized payload, so it must fail fast (#11935).
		"message larger than max": {
			err:      status.Error(codes.ResourceExhausted, "grpc: received message larger than max (60382962 vs. 52428800)"),
			expected: false,
		},
		// Genuine transient exhaustion stays retryable.
		"rate limited": {
			err:      status.Error(codes.ResourceExhausted, "rate limit exceeded, try again later"),
			expected: true,
		},
		"unavailable still retryable": {
			err:      status.Error(codes.Unavailable, "connection refused"),
			expected: true,
		},
		"deadline exceeded still retryable": {
			err:      status.Error(codes.DeadlineExceeded, "timeout while getting manifests"),
			expected: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isRetryableError(tc.err))
		})
	}
}

// TestIsRetryableError_ActualOversizedGRPCResponse guards the grpc-go error
// contract used by isRetryableError. If grpc-go changes the status code or
// message for receive-size overflows, this test fails so the classifier can be
// updated intentionally instead of silently restoring the full retry schedule.
func TestIsRetryableError_ActualOversizedGRPCResponse(t *testing.T) {
	const (
		maxReceiveSize = 1 << 10
		payloadSize    = 2 * maxReceiveSize
	)

	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	grpc_testing.RegisterTestServiceServer(server, &oversizedResponseServer{payloadSize: payloadSize})
	go func() { _ = server.Serve(listener) }()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
		server.Stop()
		_ = listener.Close()
	})

	_, err = grpc_testing.NewTestServiceClient(conn).UnaryCall(
		t.Context(),
		&grpc_testing.SimpleRequest{},
		grpc.MaxCallRecvMsgSize(maxReceiveSize),
	)
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.False(t, isRetryableError(err))
}

func TestIsLastWorkspaceMemberErr(t *testing.T) {
	lastMember := status.Error(codes.InvalidArgument, "cannot remove this member as it is the last member of the workspace")

	testCases := map[string]struct {
		err      error
		expected bool
	}{
		"nil":                       {err: nil, expected: false},
		"last member":               {err: lastMember, expected: true},
		"last member wrapped":       {err: fmt.Errorf("unable to remove workspace member: %w", lastMember), expected: true},
		"invalid argument, other":   {err: status.Error(codes.InvalidArgument, "some other validation error"), expected: false},
		"right message, wrong code": {err: status.Error(codes.NotFound, "last member of the workspace"), expected: false},
		"non-grpc error":            {err: fmt.Errorf("plain error"), expected: false},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isLastWorkspaceMemberErr(tc.err))
		})
	}
}
