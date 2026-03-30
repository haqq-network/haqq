package keeper_test

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

func (suite *KeeperTestSuite) TestDenomQuery() {
	testCases := []struct {
		name        string
		malleate    func(ctx sdk.Context)
		req         *types.QueryDenomRequest
		expectPass  bool
		expectedErr codes.Code
	}{
		{
			name: "ok - get existing denom",
			malleate: func(ctx sdk.Context) {
				suite.network.App.LiquidVestingKeeper.SetDenom(ctx, types.Denom{
					BaseDenom:     "aLIQUID0",
					DisplayDenom:  "LIQUID0",
					OriginalDenom: utils.BaseDenom,
					LockupPeriods: sdkvesting.Periods{
						{Length: 100, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
					},
				})
			},
			req:        &types.QueryDenomRequest{Denom: "aLIQUID0"},
			expectPass: true,
		},
		{
			name:        "fail - denom not found",
			malleate:    func(_ sdk.Context) {},
			req:         &types.QueryDenomRequest{Denom: "aLIQUID99"},
			expectPass:  false,
			expectedErr: codes.NotFound,
		},
		{
			name:        "fail - nil request",
			malleate:    func(_ sdk.Context) {},
			req:         nil,
			expectPass:  false,
			expectedErr: codes.InvalidArgument,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := suite.network.GetContext()

			tc.malleate(ctx)

			var (
				resp *types.QueryDenomResponse
				err  error
			)
			// nil requests must be tested via the keeper directly because the gRPC
			// framework silently converts nil to an empty message.
			if tc.req == nil {
				resp, err = suite.network.App.LiquidVestingKeeper.Denom(ctx, nil)
			} else {
				resp, err = suite.queryClient.Denom(context.Background(), tc.req)
			}

			if tc.expectPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(resp)
				suite.Require().Equal(tc.req.Denom, resp.Denom.BaseDenom)
			} else {
				suite.Require().Error(err)
				st, ok := status.FromError(err)
				suite.Require().True(ok)
				suite.Require().Equal(tc.expectedErr, st.Code())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestDenomsQuery() {
	testCases := []struct {
		name          string
		malleate      func(ctx sdk.Context)
		req           *types.QueryDenomsRequest
		expectPass    bool
		expectedCount int
		expectedErr   codes.Code
	}{
		{
			name:          "ok - get empty list",
			malleate:      func(_ sdk.Context) {},
			req:           &types.QueryDenomsRequest{},
			expectPass:    true,
			expectedCount: 0,
		},
		{
			name: "ok - get multiple denoms",
			malleate: func(ctx sdk.Context) {
				for i := uint64(0); i < 3; i++ {
					suite.network.App.LiquidVestingKeeper.SetDenom(ctx, types.Denom{
						BaseDenom:     types.DenomBaseNameFromID(i),
						DisplayDenom:  types.DenomDisplayNameFromID(i),
						OriginalDenom: utils.BaseDenom,
						LockupPeriods: sdkvesting.Periods{
							{Length: 100, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
						},
					})
				}
			},
			req:           &types.QueryDenomsRequest{},
			expectPass:    true,
			expectedCount: 3,
		},
		{
			name: "ok - pagination",
			malleate: func(ctx sdk.Context) {
				for i := uint64(0); i < 3; i++ {
					suite.network.App.LiquidVestingKeeper.SetDenom(ctx, types.Denom{
						BaseDenom:     types.DenomBaseNameFromID(i),
						DisplayDenom:  types.DenomDisplayNameFromID(i),
						OriginalDenom: utils.BaseDenom,
						LockupPeriods: sdkvesting.Periods{
							{Length: 100, Amount: sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))},
						},
					})
				}
			},
			req: &types.QueryDenomsRequest{
				Pagination: &query.PageRequest{Limit: 2},
			},
			expectPass:    true,
			expectedCount: 2,
		},
		{
			name:        "fail - nil request",
			malleate:    func(_ sdk.Context) {},
			req:         nil,
			expectPass:  false,
			expectedErr: codes.InvalidArgument,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := suite.network.GetContext()

			tc.malleate(ctx)

			var (
				resp *types.QueryDenomsResponse
				err  error
			)
			// nil requests must be tested via the keeper directly because the gRPC
			// framework silently converts nil to an empty message.
			if tc.req == nil {
				resp, err = suite.network.App.LiquidVestingKeeper.Denoms(ctx, nil)
			} else {
				resp, err = suite.queryClient.Denoms(context.Background(), tc.req)
			}

			if tc.expectPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(resp)
				suite.Require().Len(resp.Denoms, tc.expectedCount)
			} else {
				suite.Require().Error(err)
				st, ok := status.FromError(err)
				suite.Require().True(ok)
				suite.Require().Equal(tc.expectedErr, st.Code())
			}
		})
	}
}
