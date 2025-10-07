package keeper_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ucdao/keeper"
	"github.com/haqq-network/haqq/x/ucdao/types"
)

func TestBalances(t *testing.T) {
	var (
		ctx    sdk.Context
		nw     *network.UnitTestNetwork
		kr     keyring.Keyring
		qc     types.QueryClient
		req    *types.QueryBalanceRequest
		expRes *types.QueryBalanceResponse
	)

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
					Address: kr.GetAddr(0).String(),
					Denom:   utils.BaseDenom,
				}

				zeroCoin := sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt())
				expRes = &types.QueryBalanceResponse{
					Balance: &zeroCoin,
				}
			},
			expPass: true,
		},
		{
			name: "valid bech32",
			malleate: func() {
				msg := types.NewMsgFund(balances, kr.GetAccAddr(0))
				msgSrv := keeper.NewMsgServerImpl(nw.App.DaoKeeper)
				_, err := msgSrv.Fund(ctx, msg)
				require.NoError(t, err, "error while funding the UC DAO account")

				req = &types.QueryBalanceRequest{
					Address: kr.GetAccAddr(0).String(),
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
				msg := types.NewMsgFund(balances, kr.GetAccAddr(0))
				msgSrv := keeper.NewMsgServerImpl(nw.App.DaoKeeper)
				_, err := msgSrv.Fund(ctx, msg)
				require.NoError(t, err, "error while funding the UC DAO account")

				req = &types.QueryBalanceRequest{
					Address: kr.GetAddr(0).String(),
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
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			kr = keyring.New(2)
			nw = network.NewUnitTestNetwork(
				network.WithPreFundedAccounts(kr.GetAllAccAddrs()...),
			)
			ctx = nw.GetContext()
			qc = nw.GetUCDAOClient()

			tc.malleate()

			require.NoError(t, nw.NextBlock())

			res, err := qc.Balance(ctx, req)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, expRes, res)
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errContains)
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

func TestHolders(t *testing.T) {
	var (
		ctx    sdk.Context
		nw     *network.UnitTestNetwork
		kr     keyring.Keyring
		qc     types.QueryClient
		req    *types.QueryHoldersRequest
		expRes *types.QueryHoldersResponse
	)

	testAccountsCount := 5
	testAccounts := make(testAccountsList, testAccountsCount)
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
				msg := types.NewMsgFund(balances, kr.GetAccAddr(0))
				msgSrv := keeper.NewMsgServerImpl(nw.App.DaoKeeper)
				_, err := msgSrv.Fund(ctx, msg)
				require.NoError(t, err, "error while funding the UC DAO account")

				req = &types.QueryHoldersRequest{}
				expRes = &types.QueryHoldersResponse{
					Balances: []types.Balance{
						{
							Address: kr.GetAccAddr(0).String(),
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
					coins := balances.MulInt(sdkmath.NewInt(int64(i + 1)))

					msg := types.NewMsgFund(coins, testAccounts[i])
					msgSrv := keeper.NewMsgServerImpl(nw.App.DaoKeeper)
					_, err := msgSrv.Fund(ctx, msg)
					require.NoError(t, err, "error while funding the UC DAO account")

					holders[i] = types.Balance{
						Address: testAccounts[i].String(),
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
					coins := balances.MulInt(sdkmath.NewInt(int64(i + 1)))

					msg := types.NewMsgFund(coins, testAccounts[i])
					msgSrv := keeper.NewMsgServerImpl(nw.App.DaoKeeper)
					_, err := msgSrv.Fund(ctx, msg)
					require.NoError(t, err, "error while funding the UC DAO account")

					if i < 2 {
						holders[i] = types.Balance{
							Address: testAccounts[i].String(),
							Coins:   coins,
						}
					}
					if i == 2 {
						nextKey = address.MustLengthPrefix(testAccounts[i])
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
					coins := balances.MulInt(sdkmath.NewInt(int64(i + 1)))

					msg := types.NewMsgFund(coins, testAccounts[i])
					msgSrv := keeper.NewMsgServerImpl(nw.App.DaoKeeper)
					_, err := msgSrv.Fund(ctx, msg)
					require.NoError(t, err, "error while funding the UC DAO account")

					if i > 1 && i < 4 {
						holders[i-2] = types.Balance{
							Address: testAccounts[i].String(),
							Coins:   coins,
						}
					}
					if i == 4 {
						nextKey = address.MustLengthPrefix(testAccounts[i])
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
				msg := types.NewMsgFund(balances, testAccounts[0])
				msg2 := types.NewMsgTransferOwnership(testAccounts[0], testAccounts[1])
				msgSrv := keeper.NewMsgServerImpl(nw.App.DaoKeeper)
				_, err := msgSrv.Fund(ctx, msg)
				require.NoError(t, err, "error while funding the UC DAO account")
				_, err = msgSrv.TransferOwnership(ctx, msg2)
				require.NoError(t, err, "error while transfer ownership to new account")

				req = &types.QueryHoldersRequest{}
				expRes = &types.QueryHoldersResponse{
					Balances: []types.Balance{
						{
							Address: testAccounts[1].String(),
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
				msg := types.NewMsgFund(balances, testAccounts[0])
				msg2 := types.NewMsgTransferOwnershipWithAmount(testAccounts[0], testAccounts[1], sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, int64(300))))
				msgSrv := keeper.NewMsgServerImpl(nw.App.DaoKeeper)
				_, err := msgSrv.Fund(ctx, msg)
				require.NoError(t, err, "error while funding the UC DAO account")
				_, err = msgSrv.TransferOwnershipWithAmount(ctx, msg2)
				require.NoError(t, err, "error while transfer ownership to new account")

				req = &types.QueryHoldersRequest{}
				expRes = &types.QueryHoldersResponse{
					Balances: []types.Balance{
						{
							Address: testAccounts[0].String(),
							Coins:   sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, int64(700))),
						},
						{
							Address: testAccounts[1].String(),
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
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			kr = keyring.New(testAccountsCount)
			nw = network.NewUnitTestNetwork(
				network.WithPreFundedAccounts(kr.GetAllAccAddrs()...),
			)
			ctx = nw.GetContext()
			qc = nw.GetUCDAOClient()

			for i := 0; i < testAccountsCount; i++ {
				testAccounts[i] = kr.GetAccAddr(i).Bytes()
			}
			sort.Sort(testAccounts)

			tc.malleate()

			require.NoError(t, nw.NextBlock())

			res, err := qc.Holders(ctx, req)
			if tc.expPass {
				require.NoError(t, err)
				require.Equal(t, expRes, res)
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errContains)
			}
		})
	}
}
