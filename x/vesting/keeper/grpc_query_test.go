package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	"github.com/haqq-network/haqq/x/vesting/types"
)

func TestTotalLocked(t *testing.T) {
	var (
		ctx sdk.Context
		nw  *network.UnitTestNetwork
		req *types.QueryTotalLockedRequest
	)

	testCases := []struct {
		name        string
		malleate    func()
		expPass     bool
		errContains string
		checkRes    func(res *types.QueryTotalLockedResponse)
		// useKeeper indicates to call the keeper directly instead of the gRPC client
		// (gRPC framework converts nil to empty proto message before reaching the handler)
		useKeeper bool
	}{
		{
			name: "nil req",
			malleate: func() {
				req = nil
			},
			expPass:     false,
			errContains: "empty request",
			useKeeper:   true,
		},
		{
			name: "disabled - no env var",
			malleate: func() {
				req = &types.QueryTotalLockedRequest{}
			},
			expPass:     false,
			errContains: "vesting stats is disabled",
		},
		{
			name: "enabled - empty result",
			malleate: func() {
				t.Setenv("HAQQ_ENABLE_VESTING_STATS", "true")
				req = &types.QueryTotalLockedRequest{}
			},
			expPass: true,
			checkRes: func(res *types.QueryTotalLockedResponse) {
				require.NotNil(t, res)
				require.Empty(t, res.Locked)
				require.Empty(t, res.Unvested)
				require.Empty(t, res.Vested)
			},
		},
		{
			name: "enabled - with clawback account",
			malleate: func() {
				t.Setenv("HAQQ_ENABLE_VESTING_STATS", "true")

				vestingStart := ctx.BlockTime()
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, funder, balances)
				require.NoError(t, err, "error while funding the funder account")

				msg := types.NewMsgCreateClawbackVestingAccount(
					funder,
					vestingAddr,
					vestingStart,
					lockupPeriods,
					vestingPeriods,
					false,
				)
				_, err = nw.App.VestingKeeper.CreateClawbackVestingAccount(ctx, msg)
				require.NoError(t, err, "error while creating the vesting account")

				req = &types.QueryTotalLockedRequest{}
			},
			expPass: true,
			checkRes: func(res *types.QueryTotalLockedResponse) {
				require.NotNil(t, res)
				require.NotEmpty(t, res.Locked)
				require.NotEmpty(t, res.Unvested)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			// reset
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()
			qc := nw.GetVestingClient()

			tc.malleate()

			var (
				res *types.QueryTotalLockedResponse
				err error
			)
			if tc.useKeeper {
				res, err = nw.App.VestingKeeper.TotalLocked(ctx, req)
			} else {
				res, err = qc.TotalLocked(ctx, req)
			}

			if tc.expPass {
				require.NoError(t, err)
				if tc.checkRes != nil {
					tc.checkRes(res)
				}
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errContains)
			}
		})
	}
}

func TestKeeperLogger(t *testing.T) {
	nw := network.NewUnitTestNetwork()
	ctx := nw.GetContext()
	logger := nw.App.VestingKeeper.Logger(ctx)
	require.NotNil(t, logger)
}

func TestBalances(t *testing.T) {
	var (
		ctx    sdk.Context
		nw     *network.UnitTestNetwork
		req    *types.QueryBalancesRequest
		expRes *types.QueryBalancesResponse
	)

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
					Address: vestingAddr.String(),
				}
			},
			expPass:     false,
			errContains: "either does not exist or is not a vesting account",
		},
		{
			name: "invalid account - not clawback vesting account",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				acc := nw.App.AccountKeeper.NewAccount(ctx, baseAccount)
				nw.App.AccountKeeper.SetAccount(ctx, acc)

				req = &types.QueryBalancesRequest{
					Address: vestingAddr.String(),
				}
			},
			expPass:     false,
			errContains: "either does not exist or is not a vesting account",
		},
		{
			name: "valid",
			malleate: func() {
				vestingStart := ctx.BlockTime()
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, funder, balances)
				require.NoError(t, err, "error while funding the funder account")

				msg := types.NewMsgCreateClawbackVestingAccount(
					funder,
					vestingAddr,
					vestingStart,
					lockupPeriods,
					vestingPeriods,
					false,
				)
				_, err = nw.App.VestingKeeper.CreateClawbackVestingAccount(ctx, msg)
				require.NoError(t, err, "error while creating the vesting account")

				req = &types.QueryBalancesRequest{
					Address: vestingAddr.String(),
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
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
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
