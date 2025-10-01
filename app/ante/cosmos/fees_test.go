package cosmos_test

import (
	"fmt"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	cosmosante "github.com/haqq-network/haqq/app/ante/cosmos"
	"github.com/haqq-network/haqq/testutil"
	testutiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
)

type deductFeeDecoratorTestCase struct {
	name        string
	balance     math.Int
	rewards     []math.Int
	gas         uint64
	gasPrice    *math.Int
	feeGranter  func() sdk.AccAddress
	checkTx     bool
	simulate    bool
	expPass     bool
	errContains string
	postCheck   func()
	setup       func()
	malleate    func()
}

func (suite *AnteTestSuite) TestDeductFeeDecorator() {
	var (
		err error
		ctx sdk.Context
		dfd cosmosante.DeductFeeDecorator
		// General setup
		addr sdk.AccAddress
		priv cryptotypes.PrivKey
		// fee granter
		fgAddr      sdk.AccAddress
		initBalance = math.NewInt(1e18)
		lowGasPrice = math.NewInt(1)
		zero        = math.ZeroInt()
	)

	// Testcase definitions
	testcases := []deductFeeDecoratorTestCase{
		{
			name:    "pass - sufficient balance to pay fees",
			balance: initBalance,
			rewards: []math.Int{zero},
			gas:     0,
			feeGranter: func() sdk.AccAddress {
				return nil
			},
			checkTx:     false,
			simulate:    true,
			expPass:     true,
			errContains: "",
		},
		{
			name:    "fail - zero gas limit in check tx mode",
			balance: initBalance,
			rewards: []math.Int{zero},
			gas:     0,
			feeGranter: func() sdk.AccAddress {
				return nil
			},
			checkTx:     true,
			simulate:    false,
			expPass:     false,
			errContains: "must provide positive gas",
		},
		{
			name:    "fail - checkTx - insufficient funds and no staking rewards",
			balance: zero,
			rewards: []math.Int{zero},
			gas:     10_000_000,
			feeGranter: func() sdk.AccAddress {
				return nil
			},
			checkTx:     true,
			simulate:    false,
			expPass:     false,
			errContains: "insufficient funds and failed to claim sufficient staking rewards",
			postCheck: func() {
				// the balance should not have changed
				balance := suite.GetNetwork().App.BankKeeper.GetBalance(ctx, addr, utils.BaseDenom)
				suite.Require().Equal(zero, balance.Amount, "expected balance to be zero")

				// there should be no rewards
				rewards, err := testutil.GetTotalDelegationRewards(ctx, suite.GetNetwork().App.DistrKeeper, addr)
				suite.Require().NoError(err, "failed to get total delegation rewards")
				suite.Require().Empty(rewards, "expected rewards to be zero")
			},
		},
		{
			name:    "pass - insufficient funds but sufficient staking rewards",
			balance: zero,
			rewards: []math.Int{initBalance},
			gas:     10_000_000,
			feeGranter: func() sdk.AccAddress {
				return nil
			},
			checkTx:     false,
			simulate:    false,
			expPass:     true,
			errContains: "",
			postCheck: func() {
				// the balance should have increased
				balance := suite.GetNetwork().App.BankKeeper.GetBalance(ctx, addr, utils.BaseDenom)
				suite.Require().False(
					balance.Amount.IsZero(),
					"expected balance to have increased after withdrawing a surplus amount of staking rewards",
				)

				// the rewards should all have been withdrawn
				rewards, err := testutil.GetTotalDelegationRewards(ctx, suite.GetNetwork().App.DistrKeeper, addr)
				suite.Require().NoError(err, "failed to get total delegation rewards")
				suite.Require().Empty(rewards, "expected all rewards to be withdrawn")
			},
		},
		{
			name:    "fail - insufficient funds and insufficient staking rewards",
			balance: math.NewInt(1e5),
			rewards: []math.Int{math.NewInt(1e5)},
			gas:     10_000_000,
			feeGranter: func() sdk.AccAddress {
				return nil
			},
			checkTx:     false,
			simulate:    false,
			expPass:     false,
			errContains: "insufficient funds and failed to claim sufficient staking rewards",
			postCheck: func() {
				// the balance should not have changed
				balance := suite.GetNetwork().App.BankKeeper.GetBalance(ctx, addr, utils.BaseDenom)
				suite.Require().Equal(math.NewInt(1e5), balance.Amount, "expected balance to be unchanged")

				// the rewards should not have changed
				rewards, err := testutil.GetTotalDelegationRewards(ctx, suite.GetNetwork().App.DistrKeeper, addr)
				suite.Require().NoError(err, "failed to get total delegation rewards")
				suite.Require().Equal(
					sdk.NewDecCoins(sdk.NewDecCoin(utils.BaseDenom, math.NewInt(1e5))),
					rewards,
					"expected rewards to be unchanged")
			},
		},
		{
			name:    "fail - sufficient balance to pay fees but provided fees < required fees",
			balance: initBalance,
			rewards: []math.Int{zero},
			gas:     10_000_000,
			feeGranter: func() sdk.AccAddress {
				return nil
			},
			gasPrice:    &lowGasPrice,
			checkTx:     true,
			simulate:    false,
			expPass:     false,
			errContains: "insufficient fees",
			malleate: func() {
				ctx = ctx.WithMinGasPrices(
					sdk.NewDecCoins(
						sdk.NewDecCoin(utils.BaseDenom, math.NewInt(10_000)),
					),
				)
			},
		},
		{
			name:    "success - sufficient balance to pay fees & min gas prices is zero",
			balance: initBalance,
			rewards: []math.Int{zero},
			gas:     10_000_000,
			feeGranter: func() sdk.AccAddress {
				return nil
			},
			gasPrice:    &lowGasPrice,
			checkTx:     true,
			simulate:    false,
			expPass:     true,
			errContains: "",
			malleate: func() {
				ctx = ctx.WithMinGasPrices(
					sdk.NewDecCoins(
						sdk.NewDecCoin(utils.BaseDenom, zero),
					),
				)
			},
		},
		{
			name:    "success - sufficient balance to pay fees (fees > required fees)",
			balance: initBalance,
			rewards: []math.Int{zero},
			gas:     10_000_000,
			feeGranter: func() sdk.AccAddress {
				return nil
			},
			checkTx:     true,
			simulate:    false,
			expPass:     true,
			errContains: "",
			malleate: func() {
				ctx = ctx.WithMinGasPrices(
					sdk.NewDecCoins(
						sdk.NewDecCoin(utils.BaseDenom, math.NewInt(100)),
					),
				)
			},
		},
		{
			name:    "success - zero fees",
			balance: initBalance,
			rewards: []math.Int{zero},
			gas:     100,
			feeGranter: func() sdk.AccAddress {
				return nil
			},
			gasPrice:    &zero,
			checkTx:     true,
			simulate:    false,
			expPass:     true,
			errContains: "",
			malleate: func() {
				ctx = ctx.WithMinGasPrices(
					sdk.NewDecCoins(
						sdk.NewDecCoin(utils.BaseDenom, zero),
					),
				)
			},
			postCheck: func() {
				// the tx sender balance should not have changed
				balance := suite.GetNetwork().App.BankKeeper.GetBalance(ctx, addr, utils.BaseDenom)
				suite.Require().Equal(initBalance, balance.Amount, "expected balance to be unchanged")
			},
		},
		{
			name:    "fail - with not authorized fee granter",
			balance: initBalance,
			rewards: []math.Int{zero},
			gas:     10_000_000,
			feeGranter: func() sdk.AccAddress {
				return fgAddr
			},
			checkTx:     true,
			simulate:    false,
			expPass:     false,
			errContains: fmt.Sprintf("%s does not allow to pay fees for %s", fgAddr, addr),
		},
		{
			name:    "success - with authorized fee granter",
			balance: initBalance,
			rewards: []math.Int{zero},
			gas:     10_000_000,
			feeGranter: func() sdk.AccAddress {
				return fgAddr
			},
			checkTx:     true,
			simulate:    false,
			expPass:     true,
			errContains: "",
			malleate: func() {
				// Fund the fee granter
				err := testutil.FundAccountWithBaseDenom(ctx, suite.GetNetwork().App.BankKeeper, fgAddr, initBalance.Int64())
				suite.Require().NoError(err)
				// grant the fees
				grant := sdk.NewCoins(sdk.NewCoin(
					utils.BaseDenom, initBalance,
				))
				err = suite.GetNetwork().App.FeeGrantKeeper.GrantAllowance(ctx, fgAddr, addr, &feegrant.BasicAllowance{
					SpendLimit: grant,
				})
				suite.Require().NoError(err)
			},
			postCheck: func() {
				// the tx sender balance should not have changed
				balance := suite.GetNetwork().App.BankKeeper.GetBalance(ctx, addr, utils.BaseDenom)
				suite.Require().Equal(initBalance, balance.Amount, "expected balance to be unchanged")
			},
		},
		{
			name:    "fail - authorized fee granter but no feegrant keeper on decorator",
			balance: initBalance,
			rewards: []math.Int{zero},
			gas:     10_000_000,
			feeGranter: func() sdk.AccAddress {
				return fgAddr
			},
			checkTx:     true,
			simulate:    false,
			expPass:     false,
			errContains: "fee grants are not enabled",
			malleate: func() {
				// Fund the fee granter
				err := testutil.FundAccountWithBaseDenom(ctx, suite.GetNetwork().App.BankKeeper, fgAddr, initBalance.Int64())
				suite.Require().NoError(err)
				// grant the fees
				grant := sdk.NewCoins(sdk.NewCoin(
					utils.BaseDenom, initBalance,
				))
				err = suite.GetNetwork().App.FeeGrantKeeper.GrantAllowance(ctx, fgAddr, addr, &feegrant.BasicAllowance{
					SpendLimit: grant,
				})
				suite.Require().NoError(err)

				// remove the feegrant keeper from the decorator
				dfd = cosmosante.NewDeductFeeDecorator(
					suite.GetNetwork().App.AccountKeeper, suite.GetNetwork().App.BankKeeper, suite.GetNetwork().App.DistrKeeper, nil, suite.GetNetwork().App.StakingKeeper, nil,
				)
			},
		},
	}

	// Test execution
	for _, tc := range testcases {
		suite.Run(tc.name, func() {
			var args testutiltx.CosmosTxArgs
			suite.SetupTest()
			ctx = suite.GetNetwork().GetContext()

			// create empty accounts
			addrAccNumber := suite.GetKeyring().AddKey()
			addr = suite.GetKeyring().GetAccAddr(addrAccNumber)
			priv = suite.GetKeyring().GetPrivKey(addrAccNumber)
			fgAddrAccNumber := suite.GetKeyring().AddKey()
			fgAddr = suite.GetKeyring().GetAccAddr(fgAddrAccNumber)

			// make the setup for the test case
			// prepare the testcase
			ctx, err = testutil.PrepareAccountsForDelegationRewards(suite.T(), ctx, suite.GetNetwork().App, addr, tc.balance, tc.rewards...)
			suite.Require().NoError(err, "failed to prepare accounts for delegation rewards")
			suite.Require().NoError(suite.GetNetwork().NextBlock())

			dfd = cosmosante.NewDeductFeeDecorator(
				suite.GetNetwork().App.AccountKeeper, suite.GetNetwork().App.BankKeeper, suite.GetNetwork().App.DistrKeeper, suite.GetNetwork().App.FeeGrantKeeper, suite.GetNetwork().App.StakingKeeper, nil,
			)
			msg := sdktestutil.NewTestMsg(addr)
			args = testutiltx.CosmosTxArgs{
				TxCfg:      suite.GetClientCtx().TxConfig,
				Priv:       priv,
				Gas:        tc.gas,
				GasPrice:   tc.gasPrice,
				FeeGranter: tc.feeGranter(),
				Msgs:       []sdk.Msg{msg},
			}

			if tc.malleate != nil {
				tc.malleate()
			}
			ctx = ctx.WithIsCheckTx(tc.checkTx)

			// Create a transaction out of the message
			tx, err := testutiltx.PrepareCosmosTx(ctx, suite.GetNetwork().App, args, signing.SignMode_SIGN_MODE_DIRECT)
			suite.Require().NoError(err, "failed to create transaction")

			// run the ante handler
			_, err = dfd.AnteHandle(ctx, tx, tc.simulate, testutil.NextFn)

			// assert the resulting error
			if tc.expPass {
				suite.Require().NoError(err, "expected no error")
			} else {
				suite.Require().Error(err, "expected error")
				suite.Require().ErrorContains(err, tc.errContains)
			}

			// run the post check
			if tc.postCheck != nil {
				tc.postCheck()
			}
		})
	}
}
