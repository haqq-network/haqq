package keeper_test

import (
	"github.com/haqq-network/haqq/x/coinomics/types"
)

func (suite *KeeperTestSuite) TestParams() {
	params := suite.network.App.CoinomicsKeeper.GetParams(suite.network.GetContext())
	expParams := types.DefaultParams()
	suite.Require().Equal(expParams, params)

	suite.network.App.CoinomicsKeeper.SetParams(suite.network.GetContext(), params)
	suite.Require().NoError(suite.network.NextBlock())

	newParams := suite.network.App.CoinomicsKeeper.GetParams(suite.network.GetContext())
	suite.Require().Equal(newParams, params)
}
