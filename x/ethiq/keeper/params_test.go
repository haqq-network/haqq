package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
)

func (suite *KeeperTestSuite) TestParams() {
	suite.SetupTest()
	ctx := s.network.GetContext()

	suite.Require().Equal(ethiqtypes.DefaultParams(), s.network.App.EthiqKeeper.GetParams(ctx))
	suite.Require().True(s.network.App.EthiqKeeper.IsModuleEnabled(ctx))

	np := ethiqtypes.Params{
		Enabled:      false,
		MinMintPerTx: sdkmath.OneInt().MulRaw(1e10),
		MaxMintPerTx: sdkmath.OneInt().MulRaw(1e18).MulRaw(1e5),
		MaxSupply:    sdkmath.OneInt().MulRaw(1e18).MulRaw(1e5),
	}

	s.network.App.EthiqKeeper.SetParams(ctx, np)
	suite.Require().False(s.network.App.EthiqKeeper.IsModuleEnabled(ctx))
	suite.Require().Equal(np, s.network.App.EthiqKeeper.GetParams(ctx))
}
