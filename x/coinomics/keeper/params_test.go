package keeper_test

import (
	"github.com/haqq-network/haqq/x/coinomics/types"
)

func (suite *KeeperTestSuite) TestParams() {
	params := suite.app.CoinomicsKeeper.GetParams(suite.ctx)
	expParams := types.DefaultParams()

	suite.Require().NotEqual(expParams, params)

	suite.app.CoinomicsKeeper.SetParams(suite.ctx, params)
	newParams := suite.app.CoinomicsKeeper.GetParams(suite.ctx)
	suite.Require().Equal(newParams, params)
}
