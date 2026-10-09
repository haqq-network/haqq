package distribution_test

import (
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	commonfactory "github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// TestNonEVMWithdrawAddressIsNotJournaled guards getWithdrawerHexAddr against
// address truncation: a withdraw address longer than 20 bytes has no EVM
// representation, so it must not produce a balance change entry. Truncating it
// would credit an unrelated EVM account - here, by construction, the delegator's
// own - and Commit would mint the rewards a second time.
func (s *PrecompileTestSuite) TestNonEVMWithdrawAddressIsNotJournaled() {
	delegator := s.keyring.GetKey(0)
	bank := s.network.App.BankKeeper

	valAddr := sdk.ValAddress(s.validatorsKeys[0].Addr.Bytes())
	ctx := s.network.GetContext()
	val, err := s.network.App.StakingKeeper.GetValidator(ctx, valAddr)
	s.Require().NoError(err)

	s.Require().NoError(
		s.factory.Delegate(delegator.Priv, val.OperatorAddress, sdk.NewCoin(s.bondDenom, math.NewInt(1e18))),
	)
	s.Require().NoError(s.network.NextBlock())

	wideBytes := make([]byte, 32)
	copy(wideBytes[:12], []byte("nonevmwithdr"))
	copy(wideBytes[12:], delegator.Addr.Bytes())
	wideAddr := sdk.AccAddress(wideBytes)
	s.Require().Equal(
		delegator.Addr,
		common.BytesToAddress(wideAddr.Bytes()),
		"the wide address must truncate onto the delegator for this test to be meaningful",
	)

	setRes, err := s.factory.CommitCosmosTx(delegator.Priv, commonfactory.CosmosTxArgs{
		Msgs: []sdk.Msg{distrtypes.NewMsgSetWithdrawAddress(delegator.AccAddr, wideAddr)},
	})
	s.Require().NoError(err)
	s.Require().True(setRes.IsOK(), "MsgSetWithdrawAddress failed: %s", setRes.Log)
	s.Require().NoError(s.network.NextBlock())

	ctx = s.network.GetContext()
	val, err = s.network.App.StakingKeeper.GetValidator(ctx, valAddr)
	s.Require().NoError(err)
	_, err = s.prepareStakingRewards(ctx, stakingRewards{
		Delegator: delegator.AccAddr,
		Validator: val,
		RewardAmt: testRewardsAmt,
	})
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	callerContract, err := contracts.LoadDistributionCallerContract()
	s.Require().NoError(err)
	contractAddr, err := s.factory.DeployContract(
		delegator.Priv,
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: callerContract},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	ctx = s.network.GetContext()
	delBefore := bank.GetBalance(ctx, delegator.AccAddr, utils.BaseDenom).Amount
	wideBefore := bank.GetBalance(ctx, wideAddr, utils.BaseDenom).Amount

	gasPrice := big.NewInt(1e9)
	res, err := s.factory.ExecuteContractCall(delegator.Priv, evmtypes.EvmTxArgs{
		To:       &contractAddr,
		GasLimit: 2_000_000,
		GasPrice: gasPrice,
	}, factory.CallArgs{
		ContractABI: callerContract.ABI,
		MethodName:  "testWithdrawDelegatorRewards",
		Args:        []interface{}{delegator.Addr, val.OperatorAddress},
	})
	s.Require().NoError(err)
	s.Require().True(res.IsOK(), "withdraw via contract failed: %s", res.Log)

	ctx = s.network.GetContext()
	delAfter := bank.GetBalance(ctx, delegator.AccAddr, utils.BaseDenom).Amount
	wideAfter := bank.GetBalance(ctx, wideAddr, utils.BaseDenom).Amount

	rewardsToWide := wideAfter.Sub(wideBefore)
	s.Require().True(
		rewardsToWide.IsPositive(),
		"the wide withdrawer must actually be paid rewards, otherwise this test proves nothing",
	)

	fee := math.NewInt(res.GasUsed).Mul(math.NewIntFromBigInt(gasPrice))
	expectedDel := delBefore.Sub(fee)
	s.Require().Equal(
		expectedDel.String(),
		delAfter.String(),
		"delegator was credited %s aISLM that bank paid to the non-EVM withdraw address",
		delAfter.Sub(expectedDel),
	)
}
