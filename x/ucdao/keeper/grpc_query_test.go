package keeper_test

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"

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
			expPass: true,
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

type testAccountsList []sdk.AccAddress

func (ta testAccountsList) Len() int {
	return len(ta)
}

func (ta testAccountsList) Less(i, j int) bool {
	return string(ta[i]) < string(ta[j])
}

func (ta testAccountsList) Swap(i, j int) {
	ta[i], ta[j] = ta[j], ta[i]
}

func (suite *KeeperTestSuite) TestHolders() {
	var (
		req    *types.QueryHoldersRequest
		expRes *types.QueryHoldersResponse
	)

	testAccountsCount := 5
	testAccounts := make(testAccountsList, testAccountsCount)
	for i := 0; i < testAccountsCount; i++ {
		testAccounts[i] = tests.GenerateAddress().Bytes()
	}
	sort.Sort(testAccounts)

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
				expRes = &types.QueryHoldersResponse{
					Pagination: &query.PageResponse{
						NextKey: nil,
						Total:   0,
					},
				}
			},
			expPass:     true, // should be false
			errContains: "empty request",
		},
		{
			name: "valid - no holders",
			malleate: func() {
				req = &types.QueryHoldersRequest{}
				expRes = &types.QueryHoldersResponse{
					Pagination: &query.PageResponse{
						NextKey: nil,
						Total:   0,
					},
				}
			},
			expPass: true,
		},
		{
			name: "valid - 1 holder",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(testAccounts[0])
				acc := suite.app.AccountKeeper.NewAccount(suite.ctx, baseAccount)
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, acc.GetAddress(), balances)
				suite.Require().NoError(err, "error while funding the bank account")

				msg := types.NewMsgFund(balances, acc.GetAddress())
				ctx := sdk.WrapSDKContext(suite.ctx)
				msgSrv := keeper.NewMsgServerImpl(suite.app.DaoKeeper)
				_, err = msgSrv.Fund(ctx, msg)
				suite.Require().NoError(err, "error while funding the UC DAO account")

				req = &types.QueryHoldersRequest{}
				expRes = &types.QueryHoldersResponse{
					Balances: []types.Balance{
						{
							Address: acc.GetAddress().String(),
							Coins:   balances,
						},
					},
					Pagination: &query.PageResponse{
						NextKey: nil,
						Total:   1,
					},
				}
			},
			expPass: true,
		},
		{
			name: "valid - 5 holders, all",
			malleate: func() {
				holders := make([]types.Balance, testAccountsCount)
				for i := 0; i < testAccountsCount; i++ {
					baseAccount := authtypes.NewBaseAccountWithAddress(testAccounts[i])
					acc := suite.app.AccountKeeper.NewAccount(suite.ctx, baseAccount)
					suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

					coins := balances.MulInt(sdk.NewInt(int64(i + 1)))
					err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, acc.GetAddress(), coins)
					suite.Require().NoError(err, "error while funding the bank account")

					msg := types.NewMsgFund(coins, acc.GetAddress())
					ctx := sdk.WrapSDKContext(suite.ctx)
					msgSrv := keeper.NewMsgServerImpl(suite.app.DaoKeeper)
					_, err = msgSrv.Fund(ctx, msg)
					suite.Require().NoError(err, "error while funding the UC DAO account")

					holders[i] = types.Balance{
						Address: acc.GetAddress().String(),
						Coins:   coins,
					}
				}

				req = &types.QueryHoldersRequest{}
				expRes = &types.QueryHoldersResponse{
					Balances: holders,
					Pagination: &query.PageResponse{
						NextKey: nil,
						Total:   5,
					},
				}
			},
			expPass: true,
		},
		{
			name: "valid - 5 holders, paginated by 2, page 1",
			malleate: func() {
				holders := make([]types.Balance, 2)
				var nextKey []byte
				for i := 0; i < testAccountsCount; i++ {
					baseAccount := authtypes.NewBaseAccountWithAddress(testAccounts[i])
					acc := suite.app.AccountKeeper.NewAccount(suite.ctx, baseAccount)
					suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

					coins := balances.MulInt(sdk.NewInt(int64(i + 1)))
					err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, acc.GetAddress(), coins)
					suite.Require().NoError(err, "error while funding the bank account")

					msg := types.NewMsgFund(coins, acc.GetAddress())
					ctx := sdk.WrapSDKContext(suite.ctx)
					msgSrv := keeper.NewMsgServerImpl(suite.app.DaoKeeper)
					_, err = msgSrv.Fund(ctx, msg)
					suite.Require().NoError(err, "error while funding the UC DAO account")

					if i < 2 {
						holders[i] = types.Balance{
							Address: acc.GetAddress().String(),
							Coins:   coins,
						}
					}
					if i == 2 {
						nextKey = address.MustLengthPrefix(acc.GetAddress())
					}
				}

				req = &types.QueryHoldersRequest{
					Pagination: &query.PageRequest{
						Limit:      2,
						CountTotal: true,
					},
				}
				expRes = &types.QueryHoldersResponse{
					Balances: holders,
					Pagination: &query.PageResponse{
						NextKey: nextKey,
						Total:   5,
					},
				}
			},
			expPass: true,
		},
		{
			name: "valid - 5 holders, paginated by 2, page 2",
			malleate: func() {
				holders := make([]types.Balance, 2)
				var nextKey []byte
				for i := 0; i < testAccountsCount; i++ {
					baseAccount := authtypes.NewBaseAccountWithAddress(testAccounts[i])
					acc := suite.app.AccountKeeper.NewAccount(suite.ctx, baseAccount)
					suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

					coins := balances.MulInt(sdk.NewInt(int64(i + 1)))
					err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, acc.GetAddress(), coins)
					suite.Require().NoError(err, "error while funding the bank account")

					msg := types.NewMsgFund(coins, acc.GetAddress())
					ctx := sdk.WrapSDKContext(suite.ctx)
					msgSrv := keeper.NewMsgServerImpl(suite.app.DaoKeeper)
					_, err = msgSrv.Fund(ctx, msg)
					suite.Require().NoError(err, "error while funding the UC DAO account")

					if i > 1 && i < 4 {
						holders[i-2] = types.Balance{
							Address: acc.GetAddress().String(),
							Coins:   coins,
						}
					}
					if i == 4 {
						nextKey = address.MustLengthPrefix(acc.GetAddress())
					}
				}

				req = &types.QueryHoldersRequest{
					Pagination: &query.PageRequest{
						Limit:      2,
						Offset:     2,
						CountTotal: true,
					},
				}
				expRes = &types.QueryHoldersResponse{
					Balances: holders,
					Pagination: &query.PageResponse{
						NextKey: nextKey,
						Total:   5,
					},
				}
			},
			expPass: true,
		},
		{
			name: "valid - holders changed after full transfer ownership",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(testAccounts[0])
				acc := suite.app.AccountKeeper.NewAccount(suite.ctx, baseAccount)
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				baseAccount2 := authtypes.NewBaseAccountWithAddress(testAccounts[1])
				acc2 := suite.app.AccountKeeper.NewAccount(suite.ctx, baseAccount2)
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc2)

				err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, acc.GetAddress(), balances)
				suite.Require().NoError(err, "error while funding the bank account")

				msg := types.NewMsgFund(balances, acc.GetAddress())
				msg2 := types.NewMsgTransferOwnership(acc.GetAddress(), acc2.GetAddress())
				ctx := sdk.WrapSDKContext(suite.ctx)
				msgSrv := keeper.NewMsgServerImpl(suite.app.DaoKeeper)
				_, err = msgSrv.Fund(ctx, msg)
				suite.Require().NoError(err, "error while funding the UC DAO account")
				_, err = msgSrv.TransferOwnership(ctx, msg2)
				suite.Require().NoError(err, "error while transfer ownership to new account")

				req = &types.QueryHoldersRequest{}
				expRes = &types.QueryHoldersResponse{
					Balances: []types.Balance{
						{
							Address: acc2.GetAddress().String(),
							Coins:   balances,
						},
					},
					Pagination: &query.PageResponse{
						NextKey: nil,
						Total:   1,
					},
				}
			},
			expPass: true,
		},
		{
			name: "valid - holders changed after partial transfer ownership",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(testAccounts[0])
				acc := suite.app.AccountKeeper.NewAccount(suite.ctx, baseAccount)
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				baseAccount2 := authtypes.NewBaseAccountWithAddress(testAccounts[1])
				acc2 := suite.app.AccountKeeper.NewAccount(suite.ctx, baseAccount2)
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc2)

				err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, acc.GetAddress(), balances)
				suite.Require().NoError(err, "error while funding the bank account")

				msg := types.NewMsgFund(balances, acc.GetAddress())
				msg2 := types.NewMsgTransferOwnershipWithAmount(acc.GetAddress(), acc2.GetAddress(), sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, int64(300))))
				ctx := sdk.WrapSDKContext(suite.ctx)
				msgSrv := keeper.NewMsgServerImpl(suite.app.DaoKeeper)
				_, err = msgSrv.Fund(ctx, msg)
				suite.Require().NoError(err, "error while funding the UC DAO account")
				_, err = msgSrv.TransferOwnershipWithAmount(ctx, msg2)
				suite.Require().NoError(err, "error while transfer ownership to new account")

				req = &types.QueryHoldersRequest{}
				expRes = &types.QueryHoldersResponse{
					Balances: []types.Balance{
						{
							Address: acc.GetAddress().String(),
							Coins:   sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, int64(700))),
						},
						{
							Address: acc2.GetAddress().String(),
							Coins:   sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, int64(300))),
						},
					},
					Pagination: &query.PageResponse{
						NextKey: nil,
						Total:   2,
					},
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

			res, err := suite.queryClient.Holders(ctx, req)
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
