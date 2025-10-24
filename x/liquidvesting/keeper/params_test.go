package keeper_test

import (
	"reflect"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

func (suite *KeeperTestSuite) TestParams() {
	var ctx sdk.Context

	testCases := []struct {
		name      string
		paramsFun func() interface{}
		getFun    func() interface{}
		expected  bool
	}{
		{
			"fail - Params are not default in test setup",
			func() interface{} {
				return types.DefaultParams()
			},
			func() interface{} {
				return suite.network.App.LiquidVestingKeeper.GetParams(ctx)
			},
			false,
		},
		{
			"success - Checks if the params are updated properly",
			func() interface{} {
				params := types.DefaultParams()
				params.MinimumLiquidationAmount = math.NewInt(2_000_000)
				err := suite.network.App.LiquidVestingKeeper.SetParams(ctx, params)
				suite.Require().NoError(err)
				return params
			},
			func() interface{} {
				return suite.network.App.LiquidVestingKeeper.GetParams(ctx)
			},
			true,
		},
		{
			"success - Reset params to default",
			func() interface{} {
				defParams := types.DefaultParams()
				params := suite.network.App.LiquidVestingKeeper.GetParams(ctx)
				outcome := reflect.DeepEqual(defParams, params)
				suite.Require().Equal(false, outcome)

				suite.network.App.LiquidVestingKeeper.ResetParamsToDefault(ctx)
				return defParams
			},
			func() interface{} {
				return suite.network.App.LiquidVestingKeeper.GetParams(ctx)
			},
			true,
		},
		{
			"success - Is Liquid Vesting enabled by default",
			func() interface{} {
				return true
			},
			func() interface{} {
				return suite.network.App.LiquidVestingKeeper.IsLiquidVestingEnabled(ctx)
			},
			true,
		},
		{
			"success - Disable Liquid Vesting",
			func() interface{} {
				isEnabled := suite.network.App.LiquidVestingKeeper.IsLiquidVestingEnabled(ctx)
				suite.Require().Equal(true, isEnabled)

				suite.network.App.LiquidVestingKeeper.SetLiquidVestingEnabled(ctx, false)
				return false
			},
			func() interface{} {
				return suite.network.App.LiquidVestingKeeper.IsLiquidVestingEnabled(ctx)
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()

			outcome := reflect.DeepEqual(tc.paramsFun(), tc.getFun())
			suite.Require().Equal(tc.expected, outcome)
		})
	}
}
