package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/haqq-network/haqq/tests"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	"github.com/haqq-network/haqq/x/vesting/types"
)

func TestBalances(t *testing.T) {
	var (
		ctx    sdk.Context
		nw     *network.UnitTestNetwork
		req    *types.QueryBalancesRequest
		expRes *types.QueryBalancesResponse
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
					Address: "haqq1",
				}
			},
			expPass:     false,
			errContains: "decoding bech32 failed: invalid bech32 string length 5",
		},
		{
			name: "invalid account - not found",
			malleate: func() {
				req = &types.QueryBalancesRequest{
					Address: addr.String(),
				}
			},
			expPass:     false,
			errContains: fmt.Sprintf("rpc error: code = NotFound desc = account for address '%s'", addr.String()),
		},
		{
			name: "invalid account - not clawback vesting account",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(addr)
				acc := nw.App.AccountKeeper.NewAccount(ctx, baseAccount)
				nw.App.AccountKeeper.SetAccount(ctx, acc)

				req = &types.QueryBalancesRequest{
					Address: addr.String(),
				}
			},
			expPass:     false,
			errContains: fmt.Sprintf("account at address '%s' is not a vesting account", addr.String()),
		},
		{
			name: "valid",
			malleate: func() {
				vestingStart := ctx.BlockTime()

				funderAddr := sdk.AccAddress(tests.GenerateAddress().Bytes())
				baseFunderAccount := authtypes.NewBaseAccountWithAddress(funderAddr)
				funderAcc := nw.App.AccountKeeper.NewAccount(ctx, baseFunderAccount)
				nw.App.AccountKeeper.SetAccount(ctx, funderAcc)

				err := testutil.FundAccount(ctx, nw.App.BankKeeper, funderAddr, balances)
				require.NoError(t, err, "error while sending coins to the funder account")

				msg := types.NewMsgCreateClawbackVestingAccount(
					funderAddr,
					addr,
					vestingStart,
					lockupPeriods,
					vestingPeriods,
					false,
				)
				_, err = nw.App.VestingKeeper.CreateClawbackVestingAccount(ctx, msg)
				require.NoError(t, err, "error while creating the vesting account")

				req = &types.QueryBalancesRequest{
					Address: addr.String(),
				}
				expRes = &types.QueryBalancesResponse{
					Locked:   balances,
					Unvested: balances,
					Vested:   nil,
				}
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// reset
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()
			qc := nw.GetVestingClient()

			tc.malleate()

			res, err := qc.Balances(ctx, req)
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
