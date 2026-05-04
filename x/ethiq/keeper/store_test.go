package keeper_test

import (
	"math"
	"slices"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
)

func (suite *KeeperTestSuite) TestTotalBurntAmount() {
	suite.SetupTest()
	ctx := s.network.GetContext()
	oneIslm := sdk.NewCoin(utils.BaseDenom, sdkmath.OneInt())
	oneHaqq := sdk.NewCoin(ethiqtypes.BaseDenom, sdkmath.OneInt())

	suite.Require().Equal(
		sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt()),
		s.network.App.EthiqKeeper.GetTotalBurnedAmount(ctx),
	)
	suite.Require().PanicsWithValue("the total burned amount must be aISLM", func() {
		s.network.App.EthiqKeeper.SetTotalBurnedAmount(ctx, oneHaqq)
	})
	suite.Require().NotPanics(func() {
		s.network.App.EthiqKeeper.SetTotalBurnedAmount(ctx, oneIslm)
	})
	suite.Require().Equal(oneIslm, s.network.App.EthiqKeeper.GetTotalBurnedAmount(ctx))
	suite.Require().NotPanics(func() {
		s.network.App.EthiqKeeper.AddToTotalBurnedAmount(ctx, sdkmath.OneInt())
	})
	suite.Require().Equal(oneIslm.Add(oneIslm), s.network.App.EthiqKeeper.GetTotalBurnedAmount(ctx))
}

func (suite *KeeperTestSuite) TestTotalBurntFromApplicationsAmount() {
	suite.SetupTest()
	ctx := s.network.GetContext()
	oneIslm := sdk.NewCoin(utils.BaseDenom, sdkmath.OneInt())
	oneHaqq := sdk.NewCoin(ethiqtypes.BaseDenom, sdkmath.OneInt())

	suite.Require().Equal(
		sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt()),
		s.network.App.EthiqKeeper.GetTotalBurnedFromApplicationsAmount(ctx),
	)
	suite.Require().PanicsWithValue("the total burned from applications amount must be aISLM", func() {
		s.network.App.EthiqKeeper.SetTotalBurnedFromApplicationsAmount(ctx, oneHaqq)
	})
	suite.Require().NotPanics(func() {
		s.network.App.EthiqKeeper.SetTotalBurnedFromApplicationsAmount(ctx, oneIslm)
	})
	suite.Require().Equal(oneIslm, s.network.App.EthiqKeeper.GetTotalBurnedFromApplicationsAmount(ctx))
	suite.Require().NotPanics(func() {
		s.network.App.EthiqKeeper.AddToTotalBurnedFromApplicationsAmount(ctx, sdkmath.OneInt())
	})
	suite.Require().Equal(oneIslm.Add(oneIslm), s.network.App.EthiqKeeper.GetTotalBurnedFromApplicationsAmount(ctx))
}

func (suite *KeeperTestSuite) TestExecutedApplications() {
	suite.SetupTest()
	ctx := s.network.GetContext()

	suite.Require().False(s.network.App.EthiqKeeper.IsApplicationExecuted(ctx, 5))
	s.network.App.EthiqKeeper.SetApplicationAsExecuted(ctx, 5)
	suite.Require().True(s.network.App.EthiqKeeper.IsApplicationExecuted(ctx, 5))

	s.network.App.EthiqKeeper.SetApplicationAsExecuted(ctx, 9)
	suite.Require().True(s.network.App.EthiqKeeper.IsApplicationExecuted(ctx, 9))
	s.network.App.EthiqKeeper.SetApplicationAsExecuted(ctx, 7)
	suite.Require().True(s.network.App.EthiqKeeper.IsApplicationExecuted(ctx, 7))

	suite.Require().Equal([]uint64{5, 7, 9}, s.network.App.EthiqKeeper.GetAllExecutedApplicationsIDs(ctx))

	s.network.App.EthiqKeeper.ResetApplicationByID(ctx, 7)
	suite.Require().False(s.network.App.EthiqKeeper.IsApplicationExecuted(ctx, 7))

	suite.Require().Equal([]uint64{5, 9}, s.network.App.EthiqKeeper.GetAllExecutedApplicationsIDs(ctx))
}

// TestExecutedApplicationIDZero ensures application ID 0 can be marked executed in the KV store.
// uint64(0) encodes to an empty big.Int.Bytes() slice without normalization, which must not be passed to store.Set.
func (suite *KeeperTestSuite) TestExecutedApplicationIDZero() {
	suite.SetupTest()
	ctx := s.network.GetContext()

	suite.Require().False(s.network.App.EthiqKeeper.IsApplicationExecuted(ctx, 0))
	suite.Require().NotPanics(func() {
		s.network.App.EthiqKeeper.SetApplicationAsExecuted(ctx, 0)
	})
	suite.Require().True(s.network.App.EthiqKeeper.IsApplicationExecuted(ctx, 0))
	suite.Require().Equal([]uint64{0}, s.network.App.EthiqKeeper.GetAllExecutedApplicationsIDs(ctx))

	s.network.App.EthiqKeeper.ResetApplicationByID(ctx, 0)
	suite.Require().False(s.network.App.EthiqKeeper.IsApplicationExecuted(ctx, 0))
	suite.Require().Empty(s.network.App.EthiqKeeper.GetAllExecutedApplicationsIDs(ctx))
}

// boundaryExecutedApplicationIDs are representative uint64 values (encoding length / edge cases).
// Exhaustive iteration over all uint64 IDs is not feasible; this set covers 0, single-byte edges,
// uint16/uint32 boundaries, and max uint64.
func boundaryExecutedApplicationIDs() []uint64 {
	return []uint64{
		0, 1,
		127, 128, 255, 256,
		65535, 65536,
		1<<32 - 1, 1 << 32,
		math.MaxUint64 - 1, math.MaxUint64,
	}
}

// TestExecutedApplicationsBoundaryIDs exercises Set / Has / Reset / listing for many application IDs.
func (suite *KeeperTestSuite) TestExecutedApplicationsBoundaryIDs() {
	suite.SetupTest()
	ctx := s.network.GetContext()
	k := s.network.App.EthiqKeeper

	for _, appID := range boundaryExecutedApplicationIDs() {
		suite.Require().Falsef(k.IsApplicationExecuted(ctx, appID), "id=%d should not be executed yet", appID)
		suite.Require().NotPanicsf(func() {
			k.SetApplicationAsExecuted(ctx, appID)
		}, "SetApplicationAsExecuted id=%d", appID)
		suite.Require().Truef(k.IsApplicationExecuted(ctx, appID), "id=%d should be executed", appID)
		suite.Require().NotPanicsf(func() {
			k.ResetApplicationByID(ctx, appID)
		}, "ResetApplicationByID id=%d", appID)
		suite.Require().Falsef(k.IsApplicationExecuted(ctx, appID), "id=%d should be cleared", appID)
	}

	// All boundaries at once: listing must contain every ID (order is lexicographic by key bytes).
	for _, appID := range boundaryExecutedApplicationIDs() {
		k.SetApplicationAsExecuted(ctx, appID)
	}
	got := k.GetAllExecutedApplicationsIDs(ctx)
	suite.Require().Len(got, len(boundaryExecutedApplicationIDs()))
	wantSet := make(map[uint64]struct{}, len(got))
	for _, id := range boundaryExecutedApplicationIDs() {
		wantSet[id] = struct{}{}
	}
	for _, id := range got {
		_, ok := wantSet[id]
		suite.Require().Truef(ok, "unexpected id in listing: %d", id)
		delete(wantSet, id)
	}
	suite.Require().Empty(wantSet, "missing ids in listing")
	slices.Sort(got)
	suite.Require().Equal(boundaryExecutedApplicationIDs(), got)
}

// TestExecutedApplicationsAllRegisteredIDs marks executed-store round-trips for every application ID
// present in the bundled registeredApplications slice (0 .. TotalNumberOfApplications()-1).
// This complements boundaryExecutedApplicationIDs with real in-repo waitlist coverage.
func (suite *KeeperTestSuite) TestExecutedApplicationsAllRegisteredIDs() {
	suite.SetupTest()
	ctx := s.network.GetContext()
	k := s.network.App.EthiqKeeper

	n := ethiqtypes.TotalNumberOfApplications()
	suite.Require().Positive(n, "registeredApplications must be non-empty")

	for id := uint64(0); id < n; id++ {
		suite.Require().Falsef(k.IsApplicationExecuted(ctx, id), "id=%d should not be executed yet", id)
		suite.Require().NotPanicsf(func() {
			k.SetApplicationAsExecuted(ctx, id)
		}, "SetApplicationAsExecuted id=%d", id)
		suite.Require().Truef(k.IsApplicationExecuted(ctx, id), "id=%d should be executed", id)
		suite.Require().NotPanicsf(func() {
			k.ResetApplicationByID(ctx, id)
		}, "ResetApplicationByID id=%d", id)
		suite.Require().Falsef(k.IsApplicationExecuted(ctx, id), "id=%d should be cleared", id)
	}

	for id := uint64(0); id < n; id++ {
		k.SetApplicationAsExecuted(ctx, id)
	}
	got := k.GetAllExecutedApplicationsIDs(ctx)
	suite.Require().Len(got, int(n))
	wantSet := make(map[uint64]struct{}, n)
	for id := uint64(0); id < n; id++ {
		wantSet[id] = struct{}{}
	}
	for _, id := range got {
		_, ok := wantSet[id]
		suite.Require().Truef(ok, "unexpected id in listing: %d", id)
		delete(wantSet, id)
	}
	suite.Require().Empty(wantSet, "missing ids in listing")

	wantSorted := make([]uint64, n)
	for i := range wantSorted {
		wantSorted[i] = uint64(i)
	}
	slices.Sort(got)
	suite.Require().Equal(wantSorted, got)
}
