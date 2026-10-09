package ucdao_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/testutil"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	"github.com/haqq-network/haqq/precompiles/ucdao"
	haqqtestutil "github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
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

// TestContractTransfersItsOwnOwnership covers the consumption side of the identity
// model for the one ucDAO method that cannot be delegated.
//
// transferOwnership moves the whole escrow and carries no amount, so no spend limit
// can be expressed for it and ucDAO registers no authorization type under
// MsgTransferOwnership -- nobody can ever be authorized to call it for someone else.
// That is a statement about the owner, not about tx.origin: when the caller *is* the
// owner nothing is delegated, the account moves its own escrow, and a contract
// wallet's own threshold is the authorization. Gating on `caller == origin` instead
// turned away a Safe acting on itself, while the very same escrow could be moved
// through transferOwnershipWithAmount.
func (s *PrecompileTestSuite) TestContractTransfersItsOwnOwnership() {
	s.SetupTest()
	ctx := s.network.GetContext()

	// The "contract" owns the ucDAO position it moves. An unrelated EOA signs the
	// transaction, so origin is neither the caller nor the owner.
	callerAddr := utiltx.GenerateAddress()
	callerAccAddr := sdk.AccAddress(callerAddr.Bytes())
	originAddr := utiltx.GenerateAddress()
	newOwner := s.keyring.GetKey(1)

	amount := math.NewInt(1e18)
	coins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, amount))
	s.Require().NoError(haqqtestutil.FundAccount(ctx, s.network.App.BankKeeper, callerAccAddr, coins))
	s.Require().NoError(s.network.App.DaoKeeper.Fund(ctx, coins, callerAccAddr))
	s.Require().Equal(
		amount, s.network.App.DaoKeeper.GetAccountBalances(ctx, callerAccAddr).AmountOf(utils.BaseDenom),
		"fixture invariant: the caller must hold the position it is about to move",
	)

	method := s.precompile.Methods[ucdao.TransferOwnershipMethod]
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, callerAddr, s.precompile, 1e6)
	_, err := s.precompile.TransferOwnership(ctx, originAddr, contract, s.network.GetStateDB(), &method,
		[]interface{}{callerAddr, newOwner.Addr})
	s.Require().NoError(err, "a contract must be able to move its own ucDAO position without being tx.origin")

	s.Require().True(
		s.network.App.DaoKeeper.GetAccountBalances(ctx, callerAccAddr).IsZero(),
		"the caller's escrow must be emptied",
	)
	s.Require().Equal(
		amount, s.network.App.DaoKeeper.GetAccountBalances(ctx, newOwner.AccAddr).AmountOf(utils.BaseDenom),
		"the position must have landed on the new owner",
	)
}

// TestTransferOwnershipForAnotherAccountRejected is the other half: the guard is
// about the owner, so acting for an account that is not the caller stays rejected --
// including for a direct EOA call, where it used to be a separate origin check.
func (s *PrecompileTestSuite) TestTransferOwnershipForAnotherAccountRejected() {
	s.SetupTest()
	ctx := s.network.GetContext()

	signer := s.keyring.GetKey(0)
	victim := utiltx.GenerateAddress()
	newOwner := s.keyring.GetKey(1)

	method := s.precompile.Methods[ucdao.TransferOwnershipMethod]
	// Direct EOA call: caller == origin == signer, but the owner argument is someone else.
	contract, ctx := testutil.NewPrecompileContract(s.T(), ctx, signer.Addr, s.precompile, 1e6)
	_, err := s.precompile.TransferOwnership(ctx, signer.Addr, contract, s.network.GetStateDB(), &method,
		[]interface{}{victim, newOwner.Addr})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "cannot be called on behalf of another account")
	s.Require().ErrorContains(err, ucdao.TransferOwnershipWithAmountMsgURL)
}
