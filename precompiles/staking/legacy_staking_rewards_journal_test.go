package staking_test

import (
	"fmt"
	"math/big"
	"time"

	"cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/staking"
	"github.com/haqq-network/haqq/precompiles/testutil"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	commonfactory "github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/utils"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

var (
	stakingJournalDelegateAmt   = math.NewInt(9).Mul(math.NewInt(1e18))
	stakingJournalUndelegateAmt = math.NewInt(1).Mul(math.NewInt(1e18))
	stakingJournalRewardPool    = math.NewInt(1).Mul(math.NewInt(1e18))
)

// TestLegacyStakingRewardsBurnViaContract reproduces the staking journal gap:
// Cosmos auto-withdraws aISLM rewards onto the withdrawer on Undelegate, but the
// precompile used to journal nothing. A 1 wei touch dirties the withdrawer in
// StateDB; Commit SetBalance then burned the rewards.
//
// After the fix, the withdrawer keeps the auto-claimed aISLM (plus the 1 wei touch).
func (s *PrecompileTestSuite) TestLegacyStakingRewardsBurnViaContract() {
	delegator := s.keyring.GetKey(0)
	withdrawer := s.keyring.GetKey(1)
	validator := s.network.GetValidators()[0]
	stakingPrecompileAddr := common.HexToAddress(evmtypes.StakingPrecompileAddress)

	s.Require().NoError(
		s.factory.Delegate(delegator.Priv, validator.OperatorAddress, sdk.NewCoin(s.bondDenom, stakingJournalDelegateAmt)),
	)
	s.Require().NoError(s.network.NextBlock())

	setWithdraw := distrtypes.NewMsgSetWithdrawAddress(delegator.AccAddr, withdrawer.AccAddr)
	setRes, err := s.factory.CommitCosmosTx(delegator.Priv, commonfactory.CosmosTxArgs{
		Msgs: []sdk.Msg{setWithdraw},
	})
	s.Require().NoError(err)
	s.Require().True(setRes.IsOK(), "MsgSetWithdrawAddress failed: %s", setRes.Log)
	s.Require().NoError(s.network.NextBlock())

	s.Require().NoError(depositValidatorRewardsPoolAISLM(s, delegator.Priv, delegator.AccAddr, validator, stakingJournalRewardPool))
	s.Require().NoError(s.network.NextBlock())

	forwarderContract, err := contracts.LoadUcdaoForwarderContract()
	s.Require().NoError(err)
	contractAddr, err := s.factory.DeployContract(
		delegator.Priv,
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: forwarderContract},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
	s.Require().NoError(err)
	spendLimit := sdk.NewCoin(s.bondDenom, stakingJournalUndelegateAmt)
	undelegateAuthz, err := stakingtypes.NewStakeAuthorization(
		[]sdk.ValAddress{valAddr},
		nil,
		stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_UNDELEGATE,
		&spendLimit,
	)
	s.Require().NoError(err)
	expiration := time.Now().Add(time.Hour)
	s.Require().NoError(s.network.App.AuthzKeeper.SaveGrant(
		ctx,
		contractAddr.Bytes(),
		delegator.AccAddr,
		undelegateAuthz,
		&expiration,
	))

	bank := s.network.App.BankKeeper
	beforeWithdrawer := bank.GetBalance(ctx, withdrawer.AccAddr, utils.BaseDenom).Amount

	undelegateCalldata, err := s.precompile.ABI.Pack(
		staking.UndelegateMethod,
		delegator.Addr,
		validator.OperatorAddress,
		stakingJournalUndelegateAmt.BigInt(),
	)
	s.Require().NoError(err)

	callArgs := factory.CallArgs{
		ContractABI: forwarderContract.ABI,
		MethodName:  "touchAndForward",
		Args:        []interface{}{withdrawer.Addr, stakingPrecompileAddr, undelegateCalldata},
	}
	txArgs := evmtypes.EvmTxArgs{
		To:       &contractAddr,
		GasLimit: 1_500_000,
		Amount:   big.NewInt(1),
	}
	logCheck := testutil.LogCheckArgs{ABIEvents: s.precompile.Events}.
		WithExpPass(true).
		WithExpEvents(staking.EventTypeUnbond)

	res, ethRes, err := s.factory.CallContractAndCheckLogs(
		delegator.Priv,
		txArgs,
		callArgs,
		logCheck,
	)
	s.Require().NoError(err)
	s.Require().True(res.IsOK(), "touchAndForward failed: %s", res.Log)
	s.Require().False(ethRes.Failed(), "touchAndForward VM error: %s", ethRes.VmError)
	s.Require().NoError(s.network.NextBlock())

	ctx = s.network.GetContext()
	afterWithdrawer := bank.GetBalance(ctx, withdrawer.AccAddr, utils.BaseDenom).Amount
	delta := afterWithdrawer.Sub(beforeWithdrawer)
	touchDust := math.NewInt(1)

	// The withdrawer is not the tx signer, so its balance moves by exactly the
	// auto-claimed rewards plus the 1 wei touch. Asserting equality rather than a
	// lower bound covers both directions: the burn this test reproduces, and a
	// phantom credit minted at Commit.
	claimed := withdrawnRewards(res.Events, s.bondDenom)
	s.Require().True(claimed.IsPositive(), "need auto-claimed aISLM rewards to assert the journal gap")
	s.Require().Equal(
		claimed.Add(touchDust).String(),
		delta.String(),
		"withdrawer must keep exactly the auto-claimed aISLM rewards plus the 1 wei touch",
	)
}

func depositValidatorRewardsPoolAISLM(
	s *PrecompileTestSuite,
	depositorPriv cryptotypes.PrivKey,
	depositor sdk.AccAddress,
	validator stakingtypes.Validator,
	amount math.Int,
) error {
	coins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, amount))
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
