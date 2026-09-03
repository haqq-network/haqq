package staking_test

import (
	"math/big"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/staking"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	commonfactory "github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// TestNonEVMWithdrawAddressIsNotJournaled guards the balance mirror against
// address truncation.
//
// A delegator may point their distribution withdraw address at a Cosmos account
// longer than 20 bytes. common.BytesToAddress keeps only the trailing 20 bytes,
// so a withdrawer shaped as <12 arbitrary bytes || attacker EVM address> used to
// make the mirror credit the attacker's own EVM account with rewards that bank
// had paid to the wide account - and Commit minted the difference.
func (s *PrecompileTestSuite) TestNonEVMWithdrawAddressIsNotJournaled() {
	delegator := s.keyring.GetKey(0)
	validator := s.network.GetValidators()[0]
	precompileAddr := common.HexToAddress(evmtypes.StakingPrecompileAddress)
	bank := s.network.App.BankKeeper

	firstDelAmt := math.NewInt(9).Mul(math.NewInt(1e18))
	secondDelAmt := math.NewInt(1).Mul(math.NewInt(1e18))
	rewardPool := math.NewInt(5).Mul(math.NewInt(1e18))

	// 32-byte withdraw address whose trailing 20 bytes are the delegator's own
	// EVM address, so truncation would land exactly on the delegator.
	wideBytes := make([]byte, 32)
	copy(wideBytes[:12], []byte("nonevmwithdr"))
	copy(wideBytes[12:], delegator.Addr.Bytes())
	wideAddr := sdk.AccAddress(wideBytes)
	s.Require().Equal(
		delegator.Addr,
		common.BytesToAddress(wideAddr.Bytes()),
		"the wide address must truncate onto the delegator for this test to be meaningful",
	)

	s.Require().NoError(
		s.factory.Delegate(delegator.Priv, validator.OperatorAddress, sdk.NewCoin(s.bondDenom, firstDelAmt)),
	)
	s.Require().NoError(s.network.NextBlock())

	setRes, err := s.factory.CommitCosmosTx(delegator.Priv, commonfactory.CosmosTxArgs{
		Msgs: []sdk.Msg{distrtypes.NewMsgSetWithdrawAddress(delegator.AccAddr, wideAddr)},
	})
	s.Require().NoError(err)
	s.Require().True(setRes.IsOK(), "MsgSetWithdrawAddress failed: %s", setRes.Log)
	s.Require().NoError(s.network.NextBlock())

	s.Require().NoError(depositValidatorRewardsPoolAISLM(s, delegator.Priv, delegator.AccAddr, validator, rewardPool))
	s.Require().NoError(s.network.NextBlock())

	forwarder, err := contracts.LoadUcdaoForwarderContract()
	s.Require().NoError(err)
	contractAddr, err := s.factory.DeployContract(
		delegator.Priv,
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: forwarder},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
	s.Require().NoError(err)
	spendLimit := sdk.NewCoin(s.bondDenom, secondDelAmt)
	grant, err := stakingtypes.NewStakeAuthorization(
		[]sdk.ValAddress{valAddr}, nil,
		stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
		&spendLimit,
	)
	s.Require().NoError(err)
	expiration := time.Now().Add(time.Hour)
	s.Require().NoError(s.network.App.AuthzKeeper.SaveGrant(
		ctx, contractAddr.Bytes(), delegator.AccAddr, grant, &expiration,
	))

	calldata, err := s.precompile.ABI.Pack(
		staking.DelegateMethod,
		delegator.Addr,
		validator.OperatorAddress,
		secondDelAmt.BigInt(),
	)
	s.Require().NoError(err)

	delBefore := bank.GetBalance(ctx, delegator.AccAddr, utils.BaseDenom).Amount
	wideBefore := bank.GetBalance(ctx, wideAddr, utils.BaseDenom).Amount

	gasPrice := big.NewInt(1e9)
	res, err := s.factory.ExecuteContractCall(delegator.Priv, evmtypes.EvmTxArgs{
		To:       &contractAddr,
		GasLimit: 2_000_000,
		GasPrice: gasPrice,
	}, factory.CallArgs{
		ContractABI: forwarder.ABI,
		MethodName:  "forward",
		Args:        []interface{}{precompileAddr, calldata},
	})
	s.Require().NoError(err)
	s.Require().True(res.IsOK(), "forward failed: %s", res.Log)

	ctx = s.network.GetContext()
	delAfter := bank.GetBalance(ctx, delegator.AccAddr, utils.BaseDenom).Amount
	wideAfter := bank.GetBalance(ctx, wideAddr, utils.BaseDenom).Amount

	rewardsToWide := wideAfter.Sub(wideBefore)
	s.Require().True(
		rewardsToWide.IsPositive(),
		"the wide withdrawer must actually be paid rewards, otherwise this test proves nothing",
	)

	fee := math.NewInt(res.GasUsed).Mul(math.NewIntFromBigInt(gasPrice))
	expectedDel := delBefore.Sub(fee).Sub(secondDelAmt)
	s.Require().Equal(
		expectedDel.String(),
		delAfter.String(),
		"delegator was credited %s aISLM that bank paid to the non-EVM withdraw address",
		delAfter.Sub(expectedDel),
	)
}
