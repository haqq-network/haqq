package distribution_test

import (
	"fmt"

	"cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/haqq-network/haqq/precompiles/distribution"
	"github.com/haqq-network/haqq/precompiles/testutil"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	commonfactory "github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/utils"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var (
	legacyHaqqRewardPool, _ = math.NewIntFromString("1000000000000000000") // 1 aHAQQ via MsgDepositValidatorRewardsPool
	legacyHaqqExpWithdraw   = legacyHaqqRewardPool.MulRaw(95).QuoRaw(100)  // 5% validator commission
	// Enough for contract deploy + withdraw tx; keeps logs readable vs 100k genesis prefund.
	legacyHaqqDelegatorIslmReserve = math.NewInt(10).Mul(math.NewInt(1e18)) // 10 ISLM
)

// TestLegacyHaqqRewardsDoubleCreditViaContract reproduces the mainnet issue:
// operator seeds validator rewards with MsgDepositValidatorRewardsPool (aHAQQ), then
// withdrawing via DistributionCaller credits aHAQQ from x/distribution AND mints a
// similar aISLM amount through the EVM balance mirror.
//
// After a fix, the second assertion (spurious aISLM mint) should fail and be replaced
// with an expectation that aISLM changes only by gas fees.
func (s *PrecompileTestSuite) TestLegacyHaqqRewardsDoubleCreditViaContract() {
	delegator := s.keyring.GetAccAddr(0)
	validator := s.network.GetValidators()[0]

	s.Require().NoError(
		s.factory.Delegate(s.keyring.GetPrivKey(0), validator.OperatorAddress, sdk.NewCoin(s.bondDenom, math.NewInt(9e18))),
	)

	s.Require().NoError(depositValidatorRewardsPool(
		s, s.keyring.GetPrivKey(0), delegator, validator, legacyHaqqRewardPool,
	))
	s.Require().NoError(trimDelegatorIslmBalance(s, delegator, legacyHaqqDelegatorIslmReserve))
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	bank := s.network.App.BankKeeper
	beforeHaqq := bank.GetBalance(ctx, delegator, ethiqtypes.BaseDenom).Amount
	beforeIslm := bank.GetBalance(ctx, delegator, utils.BaseDenom).Amount
	s.T().Logf("before withdraw: aHAQQ=%s aISLM=%s", beforeHaqq, beforeIslm)

	distrCallerContract, err := contracts.LoadDistributionCallerContract()
	s.Require().NoError(err)

	contractAddr, err := s.factory.DeployContract(
		s.keyring.GetPrivKey(0),
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: distrCallerContract},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	callArgs := factory.CallArgs{
		ContractABI: distrCallerContract.ABI,
		MethodName:  "testWithdrawDelegatorRewards",
		Args:        []interface{}{s.keyring.GetAddr(0), validator.OperatorAddress},
	}
	txArgs := evmtypes.EvmTxArgs{
		To:       &contractAddr,
		GasLimit: 300_000,
	}
	logCheck := testutil.LogCheckArgs{ABIEvents: s.precompile.Events}.
		WithExpPass(true).
		WithExpEvents(distribution.EventTypeWithdrawDelegatorRewards)

	res, _, err := s.factory.CallContractAndCheckLogs(
		s.keyring.GetPrivKey(0),
		txArgs,
		callArgs,
		logCheck,
	)
	s.Require().NoError(err)
	s.Require().True(res.IsOK(), "tx failed: %s", res.Log)
	s.Require().NoError(s.network.NextBlock())

	ctx = s.network.GetContext()
	afterHaqq := bank.GetBalance(ctx, delegator, ethiqtypes.BaseDenom).Amount
	afterIslm := bank.GetBalance(ctx, delegator, utils.BaseDenom).Amount
	deltaHaqq := bank.GetBalance(ctx, delegator, ethiqtypes.BaseDenom).Amount.Sub(beforeHaqq)
	deltaIslm := bank.GetBalance(ctx, delegator, utils.BaseDenom).Amount.Sub(beforeIslm)
	s.T().Logf("after withdraw: aHAQQ=%s aISLM=%s", afterHaqq, afterIslm)
	s.T().Logf("after withdraw:  delta aHAQQ=%s delta aISLM=%s (gas used %d)", deltaHaqq, deltaIslm, res.GasUsed)

	s.Require().Equal(legacyHaqqExpWithdraw, deltaHaqq,
		"delegator should receive legacy aHAQQ rewards from x/distribution")

	s.T().Logf("withdraw succeeded: aHAQQ reward credited as expected (delta=%s)", deltaHaqq)

	// Bug: precompile mirrors coins[0] (aHAQQ) into native EVM balance → spurious aISLM mint.
	spuriousIslmMint := deltaIslm.GT(legacyHaqqExpWithdraw.Quo(math.NewInt(2)))
	if spuriousIslmMint {
		s.T().Logf(
			"BUG reproduced: spurious aISLM mint delta=%s (expected gas-only change, reward aHAQQ was %s)",
			deltaIslm, legacyHaqqExpWithdraw,
		)
	}
	s.Require().False(
		spuriousIslmMint,
		"bug: contract withdraw should not mint aISLM via StateDB mirror (delta aISLM=%s, gas used %d)",
		deltaIslm, res.GasUsed,
	)
}

// depositValidatorRewardsPool mirrors mainnet: depositor sends aHAQQ into a validator's
// reward pool via MsgDepositValidatorRewardsPool (pending rewards accrue to delegators).
func depositValidatorRewardsPool(
	s *PrecompileTestSuite,
	depositorPriv cryptotypes.PrivKey,
	depositor sdk.AccAddress,
	validator stakingtypes.Validator,
	amount math.Int,
) error {
	coins := sdk.NewCoins(sdk.NewCoin(ethiqtypes.BaseDenom, amount))
	ctx := s.network.GetContext()
	if err := s.network.App.BankKeeper.MintCoins(ctx, coinomicstypes.ModuleName, coins); err != nil {
		return err
	}
	if err := s.network.App.BankKeeper.SendCoinsFromModuleToAccount(
		ctx, coinomicstypes.ModuleName, depositor, coins,
	); err != nil {
		return err
	}

	msg := distrtypes.NewMsgDepositValidatorRewardsPool(
		depositor.String(),
		validator.OperatorAddress,
		coins,
	)
	resp, err := s.factory.CommitCosmosTx(depositorPriv, commonfactory.CosmosTxArgs{
		Msgs: []sdk.Msg{msg},
	})
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("MsgDepositValidatorRewardsPool failed: code=%d log=%s", resp.Code, resp.Log)
	}
	return nil
}

// trimDelegatorIslmBalance sends excess aISLM to another prefunded account.
func trimDelegatorIslmBalance(s *PrecompileTestSuite, delegator sdk.AccAddress, reserve math.Int) error {
	ctx := s.network.GetContext()
	balance := s.network.App.BankKeeper.GetBalance(ctx, delegator, utils.BaseDenom).Amount
	if !balance.GT(reserve) {
		return nil
	}
	return s.factory.FundAccount(
		s.keyring.GetKey(0),
		s.keyring.GetAccAddr(1),
		sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, balance.Sub(reserve))),
	)
}
