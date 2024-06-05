package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/coinomics/types"
)

func (s *KeeperTestSuite) TestRewardCoefficient() {
	var (
		req    *types.QueryRewardCoefficientRequest
		expRes *types.QueryRewardCoefficientResponse
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
				expRes = &types.QueryRewardCoefficientResponse{RewardCoefficient: math.LegacyNewDecWithPrec(78, 1)}
			},
			true,
		},
		{
			"set reward coefficient",
			func() {
				coinomicsParams := s.network.App.CoinomicsKeeper.GetParams(s.network.GetContext())
				coinomicsParams.RewardCoefficient = math.LegacyNewDecWithPrec(10, 0)
				s.network.App.CoinomicsKeeper.SetParams(s.network.GetContext(), coinomicsParams)

				req = &types.QueryRewardCoefficientRequest{}
				expRes = &types.QueryRewardCoefficientResponse{RewardCoefficient: math.LegacyNewDecWithPrec(10, 0)}
			},
			true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset

			ctx := sdk.WrapSDKContext(s.network.GetContext())
			tc.malleate()

			res, err := s.network.GetCoinomicsClient().RewardCoefficient(ctx, req)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(expRes, res)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMaxSupply() {
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
				s.network.App.CoinomicsKeeper.SetMaxSupply(s.network.GetContext(), maxSupply)

				req = &types.QueryMaxSupplyRequest{}
				expRes = &types.QueryMaxSupplyResponse{MaxSupply: maxSupply}
			},
			true,
		},
	}
	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset

			ctx := sdk.WrapSDKContext(s.network.GetContext())
			tc.malleate()

			res, err := s.network.GetCoinomicsClient().MaxSupply(ctx, req)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(expRes, res)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryParams() {
	ctx := sdk.WrapSDKContext(s.network.GetContext())
	expParams := types.DefaultParams()

	res, err := s.network.GetCoinomicsClient().Params(ctx, &types.QueryParamsRequest{})
	s.Require().NoError(err)
	// due to mainnet chain id in tests setup
	s.Require().Equal(expParams, res.Params)
}
