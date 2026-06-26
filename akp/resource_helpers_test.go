//go:build !acc

package akp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
