package ethiq_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/ethiq"
	"github.com/haqq-network/haqq/precompiles/ethiq/testdata"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// Regression tests for grant creation in the ethiq precompile: the granter must be
// the immediate EVM caller, never tx.origin. A self-granted MsgMintHaqq
// authorization let a malicious contract burn the victim's aISLM and mint the
// resulting aHAQQ to itself, with no prior approval from the victim.

// mintHaqqGrant returns the MintHaqq authorization for the pair, or nil.
func mintHaqqGrant(ctx sdk.Context, ak authzkeeper.Keeper, grantee, granter sdk.AccAddress) authz.Authorization {
	auth, _ := ak.GetAuthorization(ctx, grantee, granter, ethiq.MintHaqqMsgURL)
	return auth
}

// TestApproveFromContractGrantsFromCallerNotOrigin asserts that a contract calling
// approve creates the grant under its own address, not under the signer's.
func (s *PrecompileTestSuite) TestApproveFromContractGrantsFromCallerNotOrigin() {
	victim := s.keyring.GetKey(0)
	grantee := s.keyring.GetKey(1)

	forwarder, err := contracts.LoadUcdaoForwarderContract()
	s.Require().NoError(err)
	forwarderAddr, err := s.factory.DeployContract(
		victim.Priv,
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: forwarder},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	calldata, err := s.precompile.ABI.Pack(
		authorization.ApproveMethod,
		grantee.Addr,
		math.NewInt(1e18).BigInt(),
		[]string{ethiq.MintHaqqMsgURL},
	)
	s.Require().NoError(err)

	precompileAddr := common.HexToAddress(evmtypes.EthiqPrecompileAddress)
	_, err = s.factory.ExecuteContractCall(
		victim.Priv,
		evmtypes.EvmTxArgs{To: &forwarderAddr, GasLimit: 2_000_000},
		factory.CallArgs{
			ContractABI: forwarder.ABI,
			MethodName:  "forward",
			Args:        []interface{}{precompileAddr, calldata},
		},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	ak := s.network.App.AuthzKeeper

	s.Require().NotNil(
		mintHaqqGrant(ctx, ak, grantee.AccAddr, sdk.AccAddress(forwarderAddr.Bytes())),
		"the grant must be owned by the calling contract",
	)
	s.Require().Nil(
		mintHaqqGrant(ctx, ak, grantee.AccAddr, victim.AccAddr),
		"a nested contract must not be able to create a MintHaqq grant on behalf of tx.origin",
	)
}

// TestDirectApproveFromEOAIsUnchanged pins the backward-compatible path.
func (s *PrecompileTestSuite) TestDirectApproveFromEOAIsUnchanged() {
	granter := s.keyring.GetKey(0)
	grantee := s.keyring.GetKey(1)

	precompileAddr := common.HexToAddress(evmtypes.EthiqPrecompileAddress)
	_, err := s.factory.ExecuteContractCall(
		granter.Priv,
		evmtypes.EvmTxArgs{To: &precompileAddr, GasLimit: 2_000_000},
		factory.CallArgs{
			ContractABI: s.precompile.ABI,
			MethodName:  authorization.ApproveMethod,
			Args: []interface{}{
				grantee.Addr,
				math.NewInt(1e18).BigInt(),
				[]string{ethiq.MintHaqqMsgURL},
			},
		},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	s.Require().NotNil(
		mintHaqqGrant(s.network.GetContext(), s.network.App.AuthzKeeper, grantee.AccAddr, granter.AccAddr),
		"a direct EOA approve must keep working exactly as before",
	)
}

// TestSelfGrantAndConsumeInSameTxFails is the full exploit for ethiq, the precompile
// with the heaviest impact of the four: a contract grants itself MsgMintHaqq on
// behalf of the signer and consumes that grant in the same transaction, burning the
// victim's aISLM and minting the resulting aHAQQ to an attacker address. Both halves
// must now fail, atomically.
//
// testApproveAndThenMintHaqq calls approve(_addr, ...) and then
// mintHaqq(tx.origin, _to_addr, ...) in one contract call. Pointing _addr at the
// contract itself is the exploit exactly as written up: before the fix the approve
// wrote a grant with the victim as granter, and the mintHaqq immediately spent it.
func (s *PrecompileTestSuite) TestSelfGrantAndConsumeInSameTxFails() {
	victim := s.keyring.GetKey(0)
	attackerPayout := s.keyring.GetKey(1)

	caller, err := testdata.LoadEthiqCallerContract()
	s.Require().NoError(err)
	callerAddr, err := s.factory.DeployContract(
		victim.Priv,
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: caller},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	burn := math.NewInt(1e18)
	islmBefore := s.network.App.BankKeeper.GetBalance(
		s.network.GetContext(), victim.AccAddr, utils.BaseDenom,
	).Amount
	haqqBefore := s.network.App.BankKeeper.GetBalance(
		s.network.GetContext(), attackerPayout.AccAddr, ethiqtypes.BaseDenom,
	).Amount

	_, err = s.factory.ExecuteContractCall(
		victim.Priv,
		evmtypes.EvmTxArgs{To: &callerAddr, GasLimit: 2_000_000},
		factory.CallArgs{
			ContractABI: caller.ABI,
			MethodName:  "testApproveAndThenMintHaqq",
			Args: []interface{}{
				callerAddr, // grantee: the contract approves itself
				burn.BigInt(),
				burn.BigInt(),
				attackerPayout.Addr,
			},
		},
	)
	s.Require().Error(err, "a contract must not be able to self-grant from tx.origin and burn the victim's aISLM")
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()

	// The reverted transaction must leave no grant behind, under either granter.
	s.Require().Nil(
		mintHaqqGrant(ctx, s.network.App.AuthzKeeper, sdk.AccAddress(callerAddr.Bytes()), victim.AccAddr),
		"the reverted transaction must leave no grant from the victim",
	)
	s.Require().Nil(
		mintHaqqGrant(ctx, s.network.App.AuthzKeeper, sdk.AccAddress(callerAddr.Bytes()), sdk.AccAddress(callerAddr.Bytes())),
		"the contract's own grant must be rolled back with the failed mint",
	)

	// And no value moved: only the gas of the reverted transaction may have been spent.
	islmAfter := s.network.App.BankKeeper.GetBalance(ctx, victim.AccAddr, utils.BaseDenom).Amount
	s.Require().True(
		islmBefore.Sub(islmAfter).LT(burn),
		"the victim's aISLM must not have been burned",
	)
	haqqAfter := s.network.App.BankKeeper.GetBalance(ctx, attackerPayout.AccAddr, ethiqtypes.BaseDenom).Amount
	s.Require().Equal(
		haqqBefore.String(), haqqAfter.String(),
		"the attacker must not have received any aHAQQ",
	)
}
