package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/haqq-network/haqq/x/epochs/types"
)

func TestDeleteEpochInfo(t *testing.T) {
	suite := SetupTest([]types.EpochInfo{})
	ctx := suite.network.GetContext()

	// Verify day epoch exists
	_, found := suite.network.App.EpochsKeeper.GetEpochInfo(ctx, types.DayEpochID)
	require.True(t, found)

	// Delete it and verify it's gone
	suite.network.App.EpochsKeeper.DeleteEpochInfo(ctx, types.DayEpochID)
	_, found = suite.network.App.EpochsKeeper.GetEpochInfo(ctx, types.DayEpochID)
	require.False(t, found)

	// Other epochs are unaffected
	_, found = suite.network.App.EpochsKeeper.GetEpochInfo(ctx, types.WeekEpochID)
	require.True(t, found)
}

func TestIterateEpochInfoEarlyStop(t *testing.T) {
	suite := SetupTest([]types.EpochInfo{})
	ctx := suite.network.GetContext()

	// Iterate but stop after the first epoch
	var visited []string
	suite.network.App.EpochsKeeper.IterateEpochInfo(ctx, func(_ int64, epochInfo types.EpochInfo) bool {
		visited = append(visited, epochInfo.Identifier)
		return true // stop immediately
	})

	require.Len(t, visited, 1)
}

func TestEpochLifeCycle(t *testing.T) {
	// The default genesis includes day and week epochs.
	suite := SetupTest([]types.EpochInfo{})

	epochInfo := types.EpochInfo{
		Identifier:            monthIdentifier,
		StartTime:             time.Time{},
		Duration:              time.Hour * 24 * 30,
		CurrentEpoch:          0,
		CurrentEpochStartTime: time.Time{},
		EpochCountingStarted:  false,
	}
	ctx := suite.network.GetContext()
	suite.network.App.EpochsKeeper.SetEpochInfo(ctx, epochInfo)
	epochInfoSaved, found := suite.network.App.EpochsKeeper.GetEpochInfo(ctx, monthIdentifier)
	require.True(t, found)
	require.Equal(t, epochInfo, epochInfoSaved)

	allEpochs := suite.network.App.EpochsKeeper.AllEpochInfos(ctx)
	require.Len(t, allEpochs, 3)
	require.Equal(t, allEpochs[0].Identifier, types.DayEpochID) // alphabetical order
	require.Equal(t, allEpochs[1].Identifier, monthIdentifier)
	require.Equal(t, allEpochs[2].Identifier, types.WeekEpochID)
}
