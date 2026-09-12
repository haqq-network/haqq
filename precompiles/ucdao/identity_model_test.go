package ucdao_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	"github.com/haqq-network/haqq/precompiles/ucdao"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// Regression tests for grant creation in the ucDAO precompile: the granter must be
// the immediate EVM caller, never tx.origin. Otherwise any contract the user had
// called could authorize itself on that user's behalf and consume the grant in the
// same transaction.

func (s *PrecompileTestSuite) forwardToUcdao(calldata []byte) (common.Address, error) {
	forwarder, err := contracts.LoadUcdaoForwarderContract()
	s.Require().NoError(err)

	forwarderAddr, err := s.factory.DeployContract(
		s.keyring.GetPrivKey(0),
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: forwarder},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	precompileAddr := common.HexToAddress(evmtypes.UcdaoPrecompileAddress)
	_, err = s.factory.ExecuteContractCall(
		s.keyring.GetPrivKey(0),
		evmtypes.EvmTxArgs{To: &forwarderAddr, GasLimit: 2_000_000},
		factory.CallArgs{
			ContractABI: forwarder.ABI,
			MethodName:  "forward",
			Args:        []interface{}{precompileAddr, calldata},
		},
	)
	return forwarderAddr, err
}

// TestApproveFromContractGrantsFromCallerNotOrigin asserts that the grant is owned
// by the calling contract, not by the transaction signer.
func (s *PrecompileTestSuite) TestApproveFromContractGrantsFromCallerNotOrigin() {
	victim := s.keyring.GetKey(0)
	grantee := s.keyring.GetKey(1)

	calldata, err := s.precompile.ABI.Pack(
		authorization.ApproveMethod,
		grantee.Addr,
		math.NewInt(1e18).BigInt(),
		[]string{ucdao.ConvertToHaqqMsgURL},
	)
	s.Require().NoError(err)

	forwarderAddr, err := s.forwardToUcdao(calldata)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	ak := s.network.App.AuthzKeeper

	s.Require().NotNil(
		ucdaoGrant(ctx, ak, grantee.AccAddr, sdk.AccAddress(forwarderAddr.Bytes())),
		"the grant must be owned by the calling contract",
	)
	s.Require().Nil(
		ucdaoGrant(ctx, ak, grantee.AccAddr, victim.AccAddr),
		"a nested contract must not be able to create a ucDAO grant on behalf of tx.origin",
	)
}

// TestDirectApproveFromEOAIsUnchanged pins the backward-compatible path.
func (s *PrecompileTestSuite) TestDirectApproveFromEOAIsUnchanged() {
	granter := s.keyring.GetKey(0)
	grantee := s.keyring.GetKey(1)

	precompileAddr := common.HexToAddress(evmtypes.UcdaoPrecompileAddress)
	_, err := s.factory.ExecuteContractCall(
		granter.Priv,
		evmtypes.EvmTxArgs{To: &precompileAddr, GasLimit: 2_000_000},
		factory.CallArgs{
			ContractABI: s.precompile.ABI,
			MethodName:  authorization.ApproveMethod,
			Args: []interface{}{
				grantee.Addr,
				math.NewInt(1e18).BigInt(),
				[]string{ucdao.ConvertToHaqqMsgURL},
			},
		},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	s.Require().NotNil(
		ucdaoGrant(s.network.GetContext(), s.network.App.AuthzKeeper, grantee.AccAddr, granter.AccAddr),
		"a direct EOA approve must keep working exactly as before",
	)
}

// ucdaoGrant returns the ConvertToHaqq authorization for the pair, or nil.
func ucdaoGrant(ctx sdk.Context, ak authzkeeper.Keeper, grantee, granter sdk.AccAddress) authz.Authorization {
	auth, _ := ak.GetAuthorization(ctx, grantee, granter, ucdao.ConvertToHaqqMsgURL)
	return auth
}
