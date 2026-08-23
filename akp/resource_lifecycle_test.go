//go:build !acc

package akp

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type partialApplyPlan struct {
	refreshed bool
}

func TestResourceLifecycleRefreshesStateAfterPostApplyFailure(t *testing.T) {
	applyErr := errors.New("post-apply step failed")
	plan := &partialApplyPlan{}
	regularRefreshCalls := 0
	recoveryRefreshCalls := 0
	lifecycle := &ResourceLifecycle[partialApplyPlan, struct{}, int]{
		Apply: func(context.Context, *diag.Diagnostics, *partialApplyPlan) (bool, error) { return true, applyErr },
		Refresh: func(_ context.Context, _ *diag.Diagnostics, plan *partialApplyPlan) error {
			regularRefreshCalls++
			plan.refreshed = true
			return nil
		},
		RefreshAfterApplyError: func(_ context.Context, _ *diag.Diagnostics, plan *partialApplyPlan) error {
			recoveryRefreshCalls++
			plan.refreshed = true
			return nil
		},
		ResourceName: func(*partialApplyPlan) string { return "test resource" },
	}
	stateCanBeCommitted, err := lifecycle.Upsert(context.Background(), &diag.Diagnostics{}, plan)
	assert.True(t, stateCanBeCommitted)
	require.ErrorIs(t, err, applyErr)
	assert.True(t, plan.refreshed)
	assert.Zero(t, regularRefreshCalls)
	assert.Equal(t, 1, recoveryRefreshCalls)
}

func TestResourceLifecycleFallsBackToRefreshAfterPostApplyFailure(t *testing.T) {
	applyErr := errors.New("post-apply step failed")
	plan := &partialApplyPlan{}
	refreshCalls := 0
	lifecycle := &ResourceLifecycle[partialApplyPlan, struct{}, int]{
		Apply: func(context.Context, *diag.Diagnostics, *partialApplyPlan) (bool, error) { return true, applyErr },
		Refresh: func(_ context.Context, _ *diag.Diagnostics, plan *partialApplyPlan) error {
			refreshCalls++
			plan.refreshed = true
			return nil
		},
		ResourceName: func(*partialApplyPlan) string { return "test resource" },
	}
	stateCanBeCommitted, err := lifecycle.Upsert(context.Background(), &diag.Diagnostics{}, plan)
	assert.True(t, stateCanBeCommitted)
	require.ErrorIs(t, err, applyErr)
	assert.True(t, plan.refreshed)
	assert.Equal(t, 1, refreshCalls)
}

func TestResourceLifecycleDoesNotRefreshFailedApply(t *testing.T) {
	applyErr := errors.New("apply failed")
	refreshCalls := 0
	lifecycle := &ResourceLifecycle[partialApplyPlan, struct{}, int]{
		Apply: func(context.Context, *diag.Diagnostics, *partialApplyPlan) (bool, error) { return false, applyErr },
		Refresh: func(context.Context, *diag.Diagnostics, *partialApplyPlan) error {
			refreshCalls++
			return nil
		},
		ResourceName: func(*partialApplyPlan) string { return "test resource" },
	}
	stateCanBeCommitted, err := lifecycle.Upsert(context.Background(), &diag.Diagnostics{}, &partialApplyPlan{})
	assert.False(t, stateCanBeCommitted)
	require.ErrorIs(t, err, applyErr)
	assert.Zero(t, refreshCalls)
}

func TestResourceLifecyclePreservesApplyAndRefreshErrors(t *testing.T) {
	applyErr := errors.New("post-apply step failed")
	refreshErr := errors.New("refresh failed")
	lifecycle := &ResourceLifecycle[partialApplyPlan, struct{}, int]{
		Apply:                  func(context.Context, *diag.Diagnostics, *partialApplyPlan) (bool, error) { return true, applyErr },
		Refresh:                func(context.Context, *diag.Diagnostics, *partialApplyPlan) error { return nil },
		RefreshAfterApplyError: func(context.Context, *diag.Diagnostics, *partialApplyPlan) error { return refreshErr },
		ResourceName:           func(*partialApplyPlan) string { return "test resource" },
	}
	stateCanBeCommitted, err := lifecycle.Upsert(context.Background(), &diag.Diagnostics{}, &partialApplyPlan{})
	assert.False(t, stateCanBeCommitted)
	require.ErrorIs(t, err, applyErr)
	require.ErrorIs(t, err, refreshErr)
}
