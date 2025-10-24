package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *KeeperTestSuite) TestSetGetPrevBlockTs() {
	var ctx sdk.Context

	expEra := math.NewInt(100)

	testCases := []struct {
		name     string
		malleate func()
		ok       bool
	}{
		{
			"default prevblockts",
			func() {},
			false,
		},
		{
			"prevblockts set",
			func() {
				suite.network.App.CoinomicsKeeper.SetPrevBlockTS(ctx, expEra)
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()

			tc.malleate()

			prevBlockTS := suite.network.App.CoinomicsKeeper.GetPrevBlockTS(ctx)
			if tc.ok {
				suite.Require().Equal(expEra.String(), prevBlockTS.String(), tc.name)
			} else {
				// start block time from setup test, as we've already committed this first block
				suite.Require().Equal(math.NewInt(1640995200000).String(), prevBlockTS.String(), tc.name)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSetGetMaxSupply() {
	var ctx sdk.Context

	defaultMaxSupply := sdk.Coin{Denom: denomMint, Amount: math.NewIntWithDecimal(100_000_000_000, 18)}
	expMaxSupply := sdk.Coin{Denom: denomMint, Amount: math.NewIntWithDecimal(1337, 18)}

	testCases := []struct {
		name     string
		malleate func()
		ok       bool
	}{
		{
			"default MaxSupply",
			func() {},
			false,
		},
		{
			"MaxSupply set",
			func() {
				suite.network.App.CoinomicsKeeper.SetMaxSupply(ctx, expMaxSupply)
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()

			tc.malleate()

			maxSupply := suite.network.App.CoinomicsKeeper.GetMaxSupply(ctx)

			if tc.ok {
				suite.Require().Equal(expMaxSupply.String(), maxSupply.String(), tc.name)
			} else {
				suite.Require().Equal(defaultMaxSupply.String(), maxSupply.String(), tc.name)
			}
		})
	}
}
