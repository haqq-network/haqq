package ethiq_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/ethiq"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
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
