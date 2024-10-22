package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/haqq-network/haqq/tests"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/x/vesting/types"
)

func (suite *KeeperTestSuite) TestBalances() {
	var (
		req       *types.QueryBalancesRequest
		expResult *types.QueryBalancesResponse
	)
	addr := sdk.AccAddress(tests.GenerateAddress().Bytes())

	testCases := []struct {
		name        string
		malleate    func()
		expPass     bool
		errContains string
	}{
		{
			name: "nil req",
			malleate: func() {
				req = nil
			},
			expPass:     false,
			errContains: "empty address string is not allowed",
		},
		{
			name: "empty req",
			malleate: func() {
				req = &types.QueryBalancesRequest{}
			},
			expPass:     false,
			errContains: "empty address string is not allowed",
		},
		{
			name: "invalid address",
			malleate: func() {
				req = &types.QueryBalancesRequest{
					Address: "haqq11",
				}
			},
			expPass:     false,
			errContains: "decoding bech32 failed: invalid bech32 string length 6",
		},
		{
			name: "invalid account - not found",
			malleate: func() {
				req = &types.QueryBalancesRequest{
					Address: addr.String(),
				}
			},
			expPass:     false,
			errContains: "either does not exist or is not a vesting account",
		},
		{
			name: "invalid account - not clawback vesting account",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(addr)
				acc := suite.app.AccountKeeper.NewAccount(suite.ctx, baseAccount)
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				req = &types.QueryBalancesRequest{
					Address: addr.String(),
				}
			},
			expPass:     false,
			errContains: "either does not exist or is not a vesting account",
		},
		{
			name: "valid",
			malleate: func() {
				vestingStart := s.ctx.BlockTime()
				funder := sdk.AccAddress(types.ModuleName)
				err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, funder, balances)
				suite.Require().NoError(err, "error while funding the target account")

				msg := types.NewMsgCreateClawbackVestingAccount(
					funder,
					addr,
					vestingStart,
					lockupPeriods,
					vestingPeriods,
					false,
				)
				ctx := sdk.WrapSDKContext(suite.ctx)
				_, err = suite.app.VestingKeeper.CreateClawbackVestingAccount(ctx, msg)
				suite.Require().NoError(err, "error while creating the vesting account")

				req = &types.QueryBalancesRequest{
					Address: addr.String(),
				}
				expResult = &types.QueryBalancesResponse{
					Locked:   balances,
					Unvested: balances,
					Vested:   nil,
				}
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset
			ctx := sdk.WrapSDKContext(suite.ctx)
			tc.malleate()
			suite.Commit()

			res, err := suite.queryClient.Balances(ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expResult, res)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
			}
		})
	}
}
