package evm_test

import (
	"math/big"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	ethante "github.com/haqq-network/haqq/app/ante/evm"
	"github.com/haqq-network/haqq/testutil"
	testutiltx "github.com/haqq-network/haqq/testutil/tx"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// global variables used for testing the eth vesting ante handler
var (
	balances       = sdk.NewCoins(sdk.NewInt64Coin("test", 1000))
	quarter        = sdk.NewCoins(sdk.NewInt64Coin("test", 250))
	lockupPeriods  = sdkvesting.Periods{{Length: 5000, Amount: balances}}
	vestingPeriods = sdkvesting.Periods{
		{Length: 2000, Amount: quarter},
		{Length: 2000, Amount: quarter},
		{Length: 2000, Amount: quarter},
		{Length: 2000, Amount: quarter},
	}
	vestingCoins = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1000000000)))
)

// TestEthVestingTransactionDecorator tests the EthVestingTransactionDecorator ante handler.
func (suite *AnteTestSuite) TestEthVestingTransactionDecorator() {
	addr := testutiltx.GenerateAddress()

	ethTxParams := &evmtypes.EvmTxArgs{
		ChainID:  suite.GetNetwork().App.EvmKeeper.ChainID(),
		Nonce:    1,
		To:       &addr,
		Amount:   big.NewInt(1000000000),
		GasLimit: 100000,
		GasPrice: big.NewInt(1000000000),
	}
	tx := evmtypes.NewTx(ethTxParams)
	tx.From = addr.Hex()

	testcases := []struct {
		name        string
		tx          sdk.Tx
		malleate    func()
		expPass     bool
		errContains string
	}{
		{
			"pass - valid transaction, no vesting account",
			tx,
			func() {
				acc := suite.GetNetwork().App.AccountKeeper.NewAccountWithAddress(suite.GetNetwork().GetContext(), addr.Bytes())
				suite.GetNetwork().App.AccountKeeper.SetAccount(suite.GetNetwork().GetContext(), acc)
			},
			true,
			"",
		},
		{
			"fail - invalid transaction",
			&testutiltx.InvalidTx{},
			func() {},
			false,
			"invalid transaction",
		},
		{
			"fail - from address not found",
			tx,
			func() {},
			false,
			"does not exist: unknown address",
		},
		{
			"pass - valid transaction, vesting account",
			tx,
			func() {
				baseAcc := authtypes.NewBaseAccountWithAddress(addr.Bytes())
				codeHash := common.BytesToHash(crypto.Keccak256(nil))
				vestingAcc := vestingtypes.NewClawbackVestingAccount(
					baseAcc, addr.Bytes(), vestingCoins, time.Now(), lockupPeriods, vestingPeriods, &codeHash,
				)
				acc := suite.GetNetwork().App.AccountKeeper.NewAccount(suite.GetNetwork().GetContext(), vestingAcc)
				suite.GetNetwork().App.AccountKeeper.SetAccount(suite.GetNetwork().GetContext(), acc)

				denom := suite.GetNetwork().App.EvmKeeper.GetParams(suite.GetNetwork().GetContext()).EvmDenom
				coins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(1000000000)))
				err := testutil.FundAccount(suite.GetNetwork().GetContext(), suite.GetNetwork().App.BankKeeper, addr.Bytes(), coins)
				suite.Require().NoError(err, "failed to fund account")
			},
			true,
			"",
		},
		{
			"fail - valid transaction, vesting account, no balance",
			tx,
			func() {
				baseAcc := authtypes.NewBaseAccountWithAddress(addr.Bytes())
				codeHash := common.BytesToHash(crypto.Keccak256(nil))
				vestingAcc := vestingtypes.NewClawbackVestingAccount(
					baseAcc, addr.Bytes(), vestingCoins, time.Now(), lockupPeriods, vestingPeriods, &codeHash,
				)
				acc := suite.GetNetwork().App.AccountKeeper.NewAccount(suite.GetNetwork().GetContext(), vestingAcc)
				suite.GetNetwork().App.AccountKeeper.SetAccount(suite.GetNetwork().GetContext(), acc)
			},
			false,
			"account has no balance to execute transaction",
		},
	}

	for _, tc := range testcases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.malleate()

			dec := ethante.NewEthVestingTransactionDecorator(
				suite.GetNetwork().App.AccountKeeper,
				suite.GetNetwork().App.BankKeeper,
				suite.GetNetwork().App.EvmKeeper,
			)
			_, err := dec.AnteHandle(suite.GetNetwork().GetContext(), tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().ErrorContains(err, tc.errContains, tc.name)
			}
		})
	}
}
