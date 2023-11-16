package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/coinomics/types"
)

func (suite *KeeperTestSuite) TestEra() {
	var (
		req    *types.QueryEraRequest
		expRes *types.QueryEraResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"default era",
			func() {
				req = &types.QueryEraRequest{}
				expRes = &types.QueryEraResponse{}
			},
			true,
		},
		{
			"set era",
			func() {
				era := uint64(2)
				suite.app.CoinomicsKeeper.SetEra(suite.ctx, era)

				req = &types.QueryEraRequest{}
				expRes = &types.QueryEraResponse{Era: era}
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			ctx := sdk.WrapSDKContext(suite.ctx)
			tc.malleate()

			res, err := suite.queryClient.Era(ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expRes, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestEraClosingSupply() {
	var (
		req    *types.QueryEraClosingSupplyRequest
		expRes *types.QueryEraClosingSupplyResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"default era closing supply",
			func() {
				req = &types.QueryEraClosingSupplyRequest{}
				expRes = &types.QueryEraClosingSupplyResponse{EraClosingSupply: sdk.NewCoin("aISLM", sdk.NewInt(0))}
			},
			true,
		},
		{
			"set era closing supply",
			func() {
				eraClosingSupply := sdk.NewCoin("aISLM", math.NewIntWithDecimal(1337, 18))
				suite.app.CoinomicsKeeper.SetEraClosingSupply(suite.ctx, eraClosingSupply)

				req = &types.QueryEraClosingSupplyRequest{}
				expRes = &types.QueryEraClosingSupplyResponse{EraClosingSupply: eraClosingSupply}
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			ctx := sdk.WrapSDKContext(suite.ctx)
			tc.malleate()

			res, err := suite.queryClient.EraClosingSupply(ctx, req)

			println("EraClosingSupply: ", res.EraClosingSupply.Amount.String())

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expRes, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestInflationRate() {
	var (
		req    *types.QueryInflationRateRequest
		expRes *types.QueryInflationRateResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"default inflation rate",
			func() {
				req = &types.QueryInflationRateRequest{}
				expRes = &types.QueryInflationRateResponse{InflationRate: sdk.ZeroDec()}
			},
			true,
		},
		{
			"set inflation rate",
			func() {
				rate := sdk.NewDec(10)
				suite.app.CoinomicsKeeper.SetInflation(suite.ctx, rate)

				req = &types.QueryInflationRateRequest{}
				expRes = &types.QueryInflationRateResponse{InflationRate: rate}
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			ctx := sdk.WrapSDKContext(suite.ctx)
			tc.malleate()

			res, err := suite.queryClient.InflationRate(ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expRes, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMaxSupply() {
	var (
		req    *types.QueryMaxSupplyRequest
		expRes *types.QueryMaxSupplyResponse
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
				expRes = &types.QueryMaxSupplyResponse{MaxSupply: sdk.Coin{Denom: "aISLM", Amount: math.NewIntWithDecimal(100_000_000_000, 18)}}
			},
			true,
		},
		{
			"set max supply",
			func() {
				maxSupply := sdk.Coin{Denom: "aISLM", Amount: math.NewIntWithDecimal(1337, 18)}
				suite.app.CoinomicsKeeper.SetMaxSupply(suite.ctx, maxSupply)

				req = &types.QueryMaxSupplyRequest{}
				expRes = &types.QueryMaxSupplyResponse{MaxSupply: maxSupply}
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			ctx := sdk.WrapSDKContext(suite.ctx)
			tc.malleate()

			res, err := suite.queryClient.MaxSupply(ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expRes, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryParams() {
	ctx := sdk.WrapSDKContext(suite.ctx)
	expParams := types.DefaultParams()

	res, err := suite.queryClient.Params(ctx, &types.QueryParamsRequest{})
	suite.Require().NoError(err)
	// due to mainnet chain id in tests setup
	suite.Require().NotEqual(expParams, res.Params)
}
