package cosmos_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	cosmosante "github.com/haqq-network/haqq/app/ante/cosmos"
	"github.com/haqq-network/haqq/testutil"
	testutiltx "github.com/haqq-network/haqq/testutil/tx"
)

// This tests setup contains expensive operations.
// Make sure to run this benchmark tests with a limited number of iterations
// To do so, specify the iteration num with the -benchtime flag
// e.g.: go test -bench=DeductFeeDecorator -benchtime=1000x
func BenchmarkDeductFeeDecorator(b *testing.B) {
	var (
		err error
		ctx sdk.Context
	)
	s := new(AnteTestSuite)
	s.SetT(&testing.T{})
	s.SetupTest()

	delegationsRewards := make([]sdkmath.Int, 110)
	for i := 0; i < len(delegationsRewards); i++ {
		delegationsRewards[i] = sdkmath.NewInt(1e14)
	}

	testCases := []deductFeeDecoratorTestCase{
		{
			name:     "sufficient balance to pay fees",
			balance:  sdkmath.NewInt(1e18),
			rewards:  []sdkmath.Int{sdkmath.ZeroInt()},
			simulate: true,
		},
		{
			name:    "insufficient funds but sufficient staking rewards",
			balance: sdkmath.ZeroInt(),
			rewards: []sdkmath.Int{sdkmath.NewInt(1e18)},
			gas:     10_000_000,
		},
		{
			name:     "sufficient balance to pay fees with 10.000 users staking",
			balance:  sdkmath.NewInt(1e18),
			rewards:  []sdkmath.Int{sdkmath.ZeroInt()},
			simulate: true,
			setup: func() {
				usersCount := 10_000
				// setup other users rewards
				for i := 0; i < usersCount; i++ {
					userAddr, _ := testutiltx.NewAccAddressAndKey()
					ctx, err = testutil.PrepareAccountsForDelegationRewards(s.T(), ctx, s.GetNetwork().App, userAddr, sdkmath.ZeroInt(), sdkmath.NewInt(1e18))
					s.Require().NoError(err, "failed to prepare accounts for delegation rewards")
				}
				s.Require().NoError(s.GetNetwork().NextBlock())
			},
		},
		{
			name:    "insufficient funds but sufficient staking rewards with 10.000 users staking",
			balance: sdkmath.ZeroInt(),
			rewards: []sdkmath.Int{sdkmath.NewInt(1e18)},
			gas:     10_000_000,
			setup: func() {
				var err error
				usersCount := 10_000
				// setup other users rewards
				for i := 0; i < usersCount; i++ {
					userAddr, _ := testutiltx.NewAccAddressAndKey()
					ctx, err = testutil.PrepareAccountsForDelegationRewards(s.T(), ctx, s.GetNetwork().App, userAddr, sdkmath.ZeroInt(), sdkmath.NewInt(1e18))
					s.Require().NoError(err, "failed to prepare accounts for delegation rewards")
				}
				s.Require().NoError(s.GetNetwork().NextBlock())
			},
		},
		{
			name:    "insufficient funds but sufficient staking rewards - 110 delegations",
			balance: sdkmath.ZeroInt(),
			rewards: delegationsRewards,
			gas:     10_000_000,
		},
	}

	b.ResetTimer()

	for _, tc := range testCases {
		if tc.setup != nil {
			tc.setup()
		}
		b.Run(fmt.Sprintf("Case: %s", tc.name), func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				// Stop the timer to perform expensive test setup
				b.StopTimer()
				s.SetupTest()
				ctx = s.GetNetwork().GetContext()
				addr, priv := testutiltx.NewAccAddressAndKey()

				// prepare the testcase
				ctx, err = testutil.PrepareAccountsForDelegationRewards(s.T(), ctx, s.GetNetwork().App, addr, tc.balance, tc.rewards...)
				s.Require().NoError(err, "failed to prepare accounts for delegation rewards")
				s.Require().NoError(s.GetNetwork().NextBlock())

				dfd := cosmosante.NewDeductFeeDecorator(
					s.GetNetwork().App.AccountKeeper, s.GetNetwork().App.BankKeeper, s.GetNetwork().App.DistrKeeper, s.GetNetwork().App.FeeGrantKeeper, s.GetNetwork().App.StakingKeeper, nil,
				)
				msg := sdktestutil.NewTestMsg(addr)
				args := testutiltx.CosmosTxArgs{
					TxCfg:      s.GetClientCtx().TxConfig,
					Priv:       priv,
					Gas:        tc.gas,
					GasPrice:   tc.gasPrice,
					FeeGranter: tc.feeGranter(),
					Msgs:       []sdk.Msg{msg},
				}

				ctx = ctx.WithIsCheckTx(tc.checkTx)

				// Create a transaction out of the message
				tx, _ := testutiltx.PrepareCosmosTx(ctx, s.GetNetwork().App, args, signing.SignMode_SIGN_MODE_DIRECT)

				// Benchmark only the ante handler logic - start the timer
				b.StartTimer()
				_, err = dfd.AnteHandle(ctx, tx, tc.simulate, testutil.NextFn)
				s.Require().NoError(err)
			}
		})
	}
}
