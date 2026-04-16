package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/epochs/keeper"
	"github.com/haqq-network/haqq/x/epochs/types"
)

// noopHook is a minimal EpochHooks implementation for testing.
type noopHook struct{}

func (noopHook) AfterEpochEnd(_ sdk.Context, _ string, _ int64) {}

func (noopHook) BeforeEpochStart(_ sdk.Context, _ string, _ int64) {}

func TestMultiEpochHooks(t *testing.T) {
	suite := SetupTest([]types.EpochInfo{})
	ctx := suite.network.GetContext()

	hooks := keeper.NewMultiEpochHooks(noopHook{})

	// Verify neither AfterEpochEnd nor BeforeEpochStart panic
	require.NotPanics(t, func() {
		hooks.AfterEpochEnd(ctx, types.WeekEpochID, 1)
	})
	require.NotPanics(t, func() {
		hooks.BeforeEpochStart(ctx, types.WeekEpochID, 1)
	})
}

func TestSetHooksPanic(t *testing.T) {
	suite := SetupTest([]types.EpochInfo{})
	ctx := suite.network.GetContext()
	_ = ctx

	// The app keeper already has hooks set; setting them again must panic.
	require.Panics(t, func() {
		suite.network.App.EpochsKeeper.SetHooks(noopHook{})
	})
}
