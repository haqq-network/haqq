package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/tests"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ucdao/keeper"
	"github.com/haqq-network/haqq/x/ucdao/types"
)

func (suite *KeeperTestSuite) TestBalances() {
	var (
		req    *types.QueryBalanceRequest
		expRes *types.QueryBalanceResponse
	)
	addr := sdk.AccAddress(tests.GenerateAddress().Bytes())
	daoAmount := int64(1000)
	balances := sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, daoAmount))

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
			errContains: "invalid denom", // should be "empty request"
		},
		{
			name: "empty req - invalid denom",
			malleate: func() {
				req = &types.QueryBalanceRequest{}
			},
			expPass:     false,
			errContains: "invalid denom",
		},
		{
			name: "invalid address",
			malleate: func() {
				req = &types.QueryBalanceRequest{
					Address: "haqq11",
					Denom:   utils.BaseDenom,
				}
			},
			expPass:     false,
			errContains: "decoding bech32 failed: invalid bech32 string length 6",
		},
		{
			name: "valid - zero balance",
			malleate: func() {
				req = &types.QueryBalanceRequest{
					Address: addr.String(),
					Denom:   utils.BaseDenom,
				}

				zeroCoin := sdk.NewCoin(utils.BaseDenom, sdk.ZeroInt())
				expRes = &types.QueryBalanceResponse{
					Balance: &zeroCoin,
				}
			},
			expPass:     true,
			errContains: "either does not exist or is not a vesting account",
		},
		{
			name: "valid bech32",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(addr)
				acc := suite.app.AccountKeeper.NewAccount(suite.ctx, baseAccount)
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, acc.GetAddress(), balances)
				suite.Require().NoError(err, "error while funding the bank account")

				msg := types.NewMsgFund(balances, acc.GetAddress())
				ctx := sdk.WrapSDKContext(suite.ctx)
				msgSrv := keeper.NewMsgServerImpl(suite.app.DaoKeeper)
				_, err = msgSrv.Fund(ctx, msg)
				suite.Require().NoError(err, "error while funding the UC DAO account")

				req = &types.QueryBalanceRequest{
					Address: addr.String(),
					Denom:   utils.BaseDenom,
				}
				_, islmBal := balances.Find(utils.BaseDenom)
				expRes = &types.QueryBalanceResponse{
					Balance: &islmBal,
				}
			},
			expPass: true,
		},
		{
			name: "valid hex",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(addr)
				acc := suite.app.AccountKeeper.NewAccount(suite.ctx, baseAccount)
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, acc.GetAddress(), balances)
				suite.Require().NoError(err, "error while funding the bank account")

				msg := types.NewMsgFund(balances, acc.GetAddress())
				ctx := sdk.WrapSDKContext(suite.ctx)
				msgSrv := keeper.NewMsgServerImpl(suite.app.DaoKeeper)
				_, err = msgSrv.Fund(ctx, msg)
				suite.Require().NoError(err, "error while funding the UC DAO account")

				hexAddr := common.Bytes2Hex(addr.Bytes())
				req = &types.QueryBalanceRequest{
					Address: hexAddr,
					Denom:   utils.BaseDenom,
				}
				_, islmBal := balances.Find(utils.BaseDenom)
				expRes = &types.QueryBalanceResponse{
					Balance: &islmBal,
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

			res, err := suite.queryClient.Balance(ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expRes, res)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
			}
		})
	}
}
