package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *KeeperTestSuite) TestSetGetPrevBlockTs() {
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
				suite.app.CoinomicsKeeper.SetPrevBlockTS(suite.ctx, expEra)
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			tc.malleate()

			prevBlockTS := suite.app.CoinomicsKeeper.GetPrevBlockTS(suite.ctx)
			if tc.ok {
				suite.Require().Equal(expEra, prevBlockTS, tc.name)
			} else {
				suite.Require().Equal(sdk.ZeroInt(), prevBlockTS, tc.name)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSetGetMaxSupply() {
	defaultMaxSupply := sdk.Coin{Denom: "aISLM", Amount: math.NewIntWithDecimal(100_000_000_000, 18)}
	expMaxSupply := sdk.Coin{Denom: "aISLM", Amount: math.NewIntWithDecimal(1337, 18)}

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
				suite.app.CoinomicsKeeper.SetMaxSupply(suite.ctx, expMaxSupply)
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			tc.malleate()

			maxSupply := suite.app.CoinomicsKeeper.GetMaxSupply(suite.ctx)

			if tc.ok {
				suite.Require().Equal(expMaxSupply, maxSupply, tc.name)
			} else {
				suite.Require().Equal(defaultMaxSupply, maxSupply, tc.name)
			}
		})
	}
}
