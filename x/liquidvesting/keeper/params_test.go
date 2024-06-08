package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"

	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

func (suite *KeeperTestSuite) TestParams() {
	params := suite.app.LiquidVestingKeeper.GetParams(suite.ctx)
	expParams := types.NewParams(math.NewInt(1_000_000), true)

	suite.Require().Equal(expParams, params)

	params.EnableLiquidVesting = false

	err := suite.app.LiquidVestingKeeper.SetParams(suite.ctx, params)
	if err != nil {
		panic(fmt.Errorf("error setting params %s", err))
	}

	newParams := suite.app.LiquidVestingKeeper.GetParams(suite.ctx)
	suite.Require().Equal(newParams, params)
}
