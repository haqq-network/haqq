package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	erc20types "github.com/evmos/evmos/v14/x/erc20/types"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
)

func (suite *KeeperTestSuite) TestTokenPairs() {
	var (
		req    *erc20types.QueryTokenPairsRequest
		expRes *erc20types.QueryTokenPairsResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"no pairs registered",
			func() {
				req = &erc20types.QueryTokenPairsRequest{}
				expRes = &erc20types.QueryTokenPairsResponse{Pagination: &query.PageResponse{}}
			},
			true,
		},
		{
			"1 pair registered w/pagination",
			func() {
				req = &erc20types.QueryTokenPairsRequest{
					Pagination: &query.PageRequest{Limit: 10, CountTotal: true},
				}
				pair := erc20types.NewTokenPair(utiltx.GenerateAddress(), "coin", erc20types.OWNER_MODULE)
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)

				expRes = &erc20types.QueryTokenPairsResponse{
					Pagination: &query.PageResponse{Total: 1},
					TokenPairs: []erc20types.TokenPair{pair},
				}
			},
			true,
		},
		{
			"2 pairs registered wo/pagination",
			func() {
				req = &erc20types.QueryTokenPairsRequest{}
				pair := erc20types.NewTokenPair(utiltx.GenerateAddress(), "coin", erc20types.OWNER_MODULE)
				pair2 := erc20types.NewTokenPair(utiltx.GenerateAddress(), "coin2", erc20types.OWNER_MODULE)
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair2)

				expRes = &erc20types.QueryTokenPairsResponse{
					Pagination: &query.PageResponse{Total: 2},
					TokenPairs: []erc20types.TokenPair{pair, pair2},
				}
			},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			ctx := sdk.WrapSDKContext(suite.ctx)
			tc.malleate()

			res, err := suite.queryClient.TokenPairs(ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expRes.Pagination, res.Pagination)
				suite.Require().ElementsMatch(expRes.TokenPairs, res.TokenPairs)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestTokenPair() {
	var (
		req    *erc20types.QueryTokenPairRequest
		expRes *erc20types.QueryTokenPairResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"invalid token address",
			func() {
				req = &erc20types.QueryTokenPairRequest{}
				expRes = &erc20types.QueryTokenPairResponse{}
			},
			false,
		},
		{
			"token pair not found",
			func() {
				req = &erc20types.QueryTokenPairRequest{
					Token: utiltx.GenerateAddress().Hex(),
				}
				expRes = &erc20types.QueryTokenPairResponse{}
			},
			false,
		},
		{
			"token pair found",
			func() {
				addr := utiltx.GenerateAddress()
				pair := erc20types.NewTokenPair(addr, "coin", erc20types.OWNER_MODULE)
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, addr, pair.GetID())
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, pair.Denom, pair.GetID())

				req = &erc20types.QueryTokenPairRequest{
					Token: pair.Erc20Address,
				}
				expRes = &erc20types.QueryTokenPairResponse{TokenPair: pair}
			},
			true,
		},
		{
			"token pair not found - with erc20 existent",
			func() {
				addr := utiltx.GenerateAddress()
				pair := erc20types.NewTokenPair(addr, "coin", erc20types.OWNER_MODULE)
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, addr, pair.GetID())
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, pair.Denom, pair.GetID())

				req = &erc20types.QueryTokenPairRequest{
					Token: pair.Erc20Address,
				}
				expRes = &erc20types.QueryTokenPairResponse{TokenPair: pair}
			},
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			ctx := sdk.WrapSDKContext(suite.ctx)
			tc.malleate()

			res, err := suite.queryClient.TokenPair(ctx, req)
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
	expParams := erc20types.DefaultParams()

	res, err := suite.queryClient.Params(ctx, &erc20types.QueryParamsRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(expParams, res.Params)
}
