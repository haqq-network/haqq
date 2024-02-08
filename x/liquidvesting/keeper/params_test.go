package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

func (suite *KeeperTestSuite) TestParams() {
	params := suite.app.LiquidVestingKeeper.GetParams(suite.ctx)
	expParams := types.NewParams(math.NewInt(1_000_000))

	suite.Require().Equal(expParams, params)

	suite.app.LiquidVestingKeeper.SetParams(suite.ctx, params)
	newParams := suite.app.LiquidVestingKeeper.GetParams(suite.ctx)
	suite.Require().Equal(newParams, params)
}
