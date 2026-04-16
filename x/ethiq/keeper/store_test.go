package keeper_test

import (
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
