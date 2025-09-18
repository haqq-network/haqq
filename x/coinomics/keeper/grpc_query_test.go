package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/coinomics/types"
)

func (suite *KeeperTestSuite) TestRewardCoefficient() {
	var (
		req         *types.QueryRewardCoefficientRequest
		expResponse *types.QueryRewardCoefficientResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"default reward coefficient",
			func() {
				req = &types.QueryRewardCoefficientRequest{}
				expResponse = &types.QueryRewardCoefficientResponse{RewardCoefficient: math.LegacyNewDecWithPrec(78, 1)}
			},
			true,
		},
		{
			"set reward coefficient",
			func() {
				coinomicsParams := suite.app.CoinomicsKeeper.GetParams(s.ctx)
				coinomicsParams.RewardCoefficient = math.LegacyNewDecWithPrec(10, 0)
				s.app.CoinomicsKeeper.SetParams(s.ctx, coinomicsParams)

				req = &types.QueryRewardCoefficientRequest{}
				expResponse = &types.QueryRewardCoefficientResponse{RewardCoefficient: math.LegacyNewDecWithPrec(10, 0)}
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			tc.malleate()

			res, err := suite.queryClient.RewardCoefficient(suite.ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expResponse, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMaxSupply() {
	var (
		req         *types.QueryMaxSupplyRequest
		expResponse *types.QueryMaxSupplyResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"default max supply",
			func() {
				req = &types.QueryMaxSupplyRequest{}
				expResponse = &types.QueryMaxSupplyResponse{MaxSupply: sdk.Coin{Denom: "aISLM", Amount: math.NewIntWithDecimal(100_000_000_000, 18)}}
			},
			true,
		},
		{
			"set max supply",
			func() {
				maxSupply := sdk.Coin{Denom: "aISLM", Amount: math.NewIntWithDecimal(1337, 18)}
				suite.app.CoinomicsKeeper.SetMaxSupply(suite.ctx, maxSupply)

				req = &types.QueryMaxSupplyRequest{}
				expResponse = &types.QueryMaxSupplyResponse{MaxSupply: maxSupply}
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			tc.malleate()

			res, err := suite.queryClient.MaxSupply(suite.ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expResponse, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryParams() {
	expParams := types.DefaultParams()

	res, err := suite.queryClient.Params(suite.ctx, &types.QueryParamsRequest{})
	suite.Require().NoError(err)
	// due to mainnet chain id in tests setup
	suite.Require().Equal(expParams, res.Params)
}
