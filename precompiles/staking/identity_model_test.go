package staking_test

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/staking"
	"github.com/haqq-network/haqq/precompiles/staking/testdata"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	testutils "github.com/haqq-network/haqq/testutil/integration/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// Regression tests for the precompile identity model.
//
// Grant creation binds the granter to the immediate EVM caller; grant consumption
// binds the granter to the logical sender carried by the message. tx.origin is no
// longer an authorization anywhere in this precompile: before, a contract the user
// had called could mint itself a grant on that user's behalf and consume it in the
// same transaction, and a contract moving its own stake was gated on -- and charged
// against -- an unrelated origin's allowance.

// deployForwarder deploys the generic forwarder used to make
// contract.CallerAddress differ from tx.origin, and returns its address.
func (s *PrecompileTestSuite) deployForwarder() (common.Address, evmtypes.CompiledContract) {
	forwarder, err := contracts.LoadUcdaoForwarderContract()
	s.Require().NoError(err)

	addr, err := s.factory.DeployContract(
		s.keyring.GetPrivKey(0),
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: forwarder},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	return addr, forwarder
}

// forward makes the deployed forwarder call the staking precompile with the given
// calldata, so that caller == forwarder and origin == key 0.
func (s *PrecompileTestSuite) forward(
	forwarderAddr common.Address,
	forwarder evmtypes.CompiledContract,
	calldata []byte,
) error {
	precompileAddr := common.HexToAddress(evmtypes.StakingPrecompileAddress)

	_, err := s.factory.ExecuteContractCall(
		s.keyring.GetPrivKey(0),
		evmtypes.EvmTxArgs{To: &forwarderAddr, GasLimit: 2_000_000},
		factory.CallArgs{
			ContractABI: forwarder.ABI,
			MethodName:  "forward",
			Args:        []interface{}{precompileAddr, calldata},
		},
	)
	return err
}

// TestApproveFromContractGrantsFromCallerNotOrigin asserts that a contract calling
// approve creates the grant under its own address, never under the address that
// signed the transaction.
func (s *PrecompileTestSuite) TestApproveFromContractGrantsFromCallerNotOrigin() {
	victim := s.keyring.GetKey(0)
	grantee := s.keyring.GetKey(1)

	forwarderAddr, forwarder := s.deployForwarder()

	amount := math.NewInt(1e18)
	calldata, err := s.precompile.ABI.Pack(
		authorization.ApproveMethod,
		grantee.Addr,
		amount.BigInt(),
		[]string{staking.DelegateMsg},
	)
	s.Require().NoError(err)

	s.Require().NoError(s.forward(forwarderAddr, forwarder, calldata))
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	ak := s.network.App.AuthzKeeper

	fromCaller, _ := CheckAuthorizationWithContext(
		ctx, ak, stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
		grantee.Addr, forwarderAddr,
	)
	s.Require().NotNil(
		fromCaller,
		"the grant must be owned by the calling contract, which is what a contract wallet needs",
	)

	fromOrigin, _ := CheckAuthorizationWithContext(
		ctx, ak, stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
		grantee.Addr, victim.Addr,
	)
	s.Require().Nil(
		fromOrigin,
		"a nested contract must not be able to create a grant on behalf of tx.origin",
	)
}

// TestSelfGrantAndConsumeInSameTxFails is the full exploit: a contract mints
// itself a grant from tx.origin and consumes it in the same transaction to move
// the victim's stake. Both halves must now fail.
func (s *PrecompileTestSuite) TestSelfGrantAndConsumeInSameTxFails() {
	victim := s.keyring.GetKey(0)
	validator := s.network.GetValidators()[0]

	delAmt := math.NewInt(2e18)
	s.Require().NoError(
		s.factory.Delegate(victim.Priv, validator.OperatorAddress, sdk.NewCoin(s.bondDenom, delAmt)),
	)
	s.Require().NoError(s.network.NextBlock())

	stakingCaller, err := testdata.LoadStakingCallerContract()
	s.Require().NoError(err)
	callerAddr, err := s.factory.DeployContract(
		victim.Priv,
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: stakingCaller},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	undelegateAmt := math.NewInt(1e18)

	// testApproveAndThenUndelegate approves _addr and then undelegates from
	// tx.origin in the same call. Pointing _addr at the contract itself is the
	// exploit exactly as written up.
	_, err = s.factory.ExecuteContractCall(
		victim.Priv,
		evmtypes.EvmTxArgs{To: &callerAddr, GasLimit: 2_000_000},
		factory.CallArgs{
			ContractABI: stakingCaller.ABI,
			MethodName:  "testApproveAndThenUndelegate",
			Args: []interface{}{
				callerAddr,
				undelegateAmt.BigInt(),
				undelegateAmt.BigInt(),
				validator.OperatorAddress,
			},
		},
	)
	s.Require().Error(err, "a contract must not be able to self-grant from tx.origin and undelegate the victim's stake")
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	fromVictim, _ := CheckAuthorizationWithContext(
		ctx, s.network.App.AuthzKeeper,
		stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_UNDELEGATE,
		callerAddr, victim.Addr,
	)
	s.Require().Nil(fromVictim, "the reverted transaction must leave no grant from the victim")

	unbonding, err := s.network.App.StakingKeeper.GetAllUnbondingDelegations(ctx, victim.AccAddr)
	s.Require().NoError(err)
	s.Require().Empty(unbonding, "the victim's stake must not have been touched")
}

// TestDirectApproveFromEOAIsUnchanged pins the backward-compatible case: a direct
// EOA call has caller == origin, so the grant still lands under the EOA.
func (s *PrecompileTestSuite) TestDirectApproveFromEOAIsUnchanged() {
	granter := s.keyring.GetKey(0)
	grantee := s.keyring.GetKey(1)

	precompileAddr := common.HexToAddress(evmtypes.StakingPrecompileAddress)
	amount := math.NewInt(1e18)

	_, err := s.factory.ExecuteContractCall(
		granter.Priv,
		evmtypes.EvmTxArgs{To: &precompileAddr, GasLimit: 2_000_000},
		factory.CallArgs{
			ContractABI: s.precompile.ABI,
			MethodName:  authorization.ApproveMethod,
			Args:        []interface{}{grantee.Addr, amount.BigInt(), []string{staking.DelegateMsg}},
		},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	grant, _ := CheckAuthorizationWithContext(
		s.network.GetContext(), s.network.App.AuthzKeeper,
		stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
		grantee.Addr, granter.Addr,
	)
	s.Require().NotNil(grant, "a direct EOA approve must keep working exactly as before")
	s.Require().Equal(amount.String(), grant.MaxTokens.Amount.String())
}

// TestContractDelegatesOwnFundsWithoutGrant covers the consumption side. Before
// the fix, a contract moving its own stake was gated on a grant from an unrelated
// tx.origin, because the delegator was rebound to the origin for the check while
// the message still debited the contract.
func (s *PrecompileTestSuite) TestContractDelegatesOwnFundsWithoutGrant() {
	signer := s.keyring.GetKey(0)
	validator := s.network.GetValidators()[0]

	forwarderAddr, forwarder := s.deployForwarder()

	delAmt := math.NewInt(1e18)
	s.Require().NoError(testutils.FundAccountWithBaseDenom(
		s.factory, s.network, signer, forwarderAddr.Bytes(), delAmt.MulRaw(2),
	))
	s.Require().NoError(s.network.NextBlock())

	calldata, err := s.precompile.ABI.Pack(
		staking.DelegateMethod,
		forwarderAddr,
		validator.OperatorAddress,
		delAmt.BigInt(),
	)
	s.Require().NoError(err)

	// No grant of any kind exists here.
	s.Require().NoError(
		s.forward(forwarderAddr, forwarder, calldata),
		"a contract must be able to move its own stake without a third party's grant",
	)
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
	s.Require().NoError(err)
	del, err := s.network.App.StakingKeeper.GetDelegation(ctx, forwarderAddr.Bytes(), valAddr)
	s.Require().NoError(err)
	val, err := s.network.App.StakingKeeper.GetValidator(ctx, valAddr)
	s.Require().NoError(err)
	s.Require().Equal(
		delAmt.String(),
		val.TokensFromShares(del.Shares).TruncateInt().String(),
		"the delegation must belong to the contract, which is the account that was debited",
	)
}

// TestContractOwnDelegationDoesNotConsumeOriginAllowance is the allowance-griefing
// half: a contract spending its own funds used to charge the origin's
// StakeAuthorization, letting a third party drain a user's allowance without
// touching the user's money.
func (s *PrecompileTestSuite) TestContractOwnDelegationDoesNotConsumeOriginAllowance() {
	signer := s.keyring.GetKey(0)
	validator := s.network.GetValidators()[0]

	forwarderAddr, forwarder := s.deployForwarder()

	delAmt := math.NewInt(1e18)
	s.Require().NoError(testutils.FundAccountWithBaseDenom(
		s.factory, s.network, signer, forwarderAddr.Bytes(), delAmt.MulRaw(2),
	))
	s.Require().NoError(s.network.NextBlock())

	// The signer grants the contract a limited delegate allowance. The contract
	// then delegates its OWN funds; the allowance must be untouched.
	allowance := delAmt.MulRaw(5)
	valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
	s.Require().NoError(err)
	spendLimit := sdk.NewCoin(s.bondDenom, allowance)
	grant, err := stakingtypes.NewStakeAuthorization(
		[]sdk.ValAddress{valAddr}, nil,
		stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
		&spendLimit,
	)
	s.Require().NoError(err)
	expiration := s.network.GetContext().BlockTime().Add(time.Hour)
	s.Require().NoError(s.network.App.AuthzKeeper.SaveGrant(
		s.network.GetContext(), forwarderAddr.Bytes(), signer.AccAddr, grant, &expiration,
	))
	s.Require().NoError(s.network.NextBlock())

	calldata, err := s.precompile.ABI.Pack(
		staking.DelegateMethod,
		forwarderAddr,
		validator.OperatorAddress,
		delAmt.BigInt(),
	)
	s.Require().NoError(err)

	s.Require().NoError(s.forward(forwarderAddr, forwarder, calldata))
	s.Require().NoError(s.network.NextBlock())

	after, _ := CheckAuthorizationWithContext(
		s.network.GetContext(), s.network.App.AuthzKeeper,
		stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
		forwarderAddr, signer.Addr,
	)
	s.Require().NotNil(after, "the signer's grant must survive untouched")
	s.Require().Equal(
		allowance.String(),
		after.MaxTokens.Amount.String(),
		"a contract spending its own funds must not consume the origin's allowance",
	)
}

// TestThirdPartyDelegationWithGranterSignedTxStillWorks pins the branch that is
// deliberately left unchanged: caller != delegator with the delegator as the
// transaction signer still works off an explicit grant.
func (s *PrecompileTestSuite) TestThirdPartyDelegationWithGranterSignedTxStillWorks() {
	delegator := s.keyring.GetKey(0)
	validator := s.network.GetValidators()[0]

	forwarderAddr, forwarder := s.deployForwarder()

	delAmt := math.NewInt(1e18)
	valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
	s.Require().NoError(err)
	spendLimit := sdk.NewCoin(s.bondDenom, delAmt.MulRaw(2))
	grant, err := stakingtypes.NewStakeAuthorization(
		[]sdk.ValAddress{valAddr}, nil,
		stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
		&spendLimit,
	)
	s.Require().NoError(err)
	expiration := s.network.GetContext().BlockTime().Add(time.Hour)
	s.Require().NoError(s.network.App.AuthzKeeper.SaveGrant(
		s.network.GetContext(), forwarderAddr.Bytes(), delegator.AccAddr, grant, &expiration,
	))
	s.Require().NoError(s.network.NextBlock())

	calldata, err := s.precompile.ABI.Pack(
		staking.DelegateMethod,
		delegator.Addr,
		validator.OperatorAddress,
		delAmt.BigInt(),
	)
	s.Require().NoError(err)

	s.Require().NoError(
		s.forward(forwarderAddr, forwarder, calldata),
		"a grant from the delegator, who also signed the tx, must still authorize the call",
	)
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	del, err := s.network.App.StakingKeeper.GetDelegation(ctx, delegator.AccAddr, valAddr)
	s.Require().NoError(err)
	s.Require().True(del.Shares.IsPositive())

	after, _ := CheckAuthorizationWithContext(
		ctx, s.network.App.AuthzKeeper,
		stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
		forwarderAddr, delegator.Addr,
	)
	s.Require().NotNil(after)
	s.Require().Equal(
		delAmt.String(),
		after.MaxTokens.Amount.String(),
		"the delegator's own allowance must be the one that is charged",
	)
}

// TestDelegateForForeignAccountStillRejected pins branch C: acting for an account
// that did not sign the transaction stays rejected, so no pre-existing grant
// created through the old self-grant hole becomes usable without the origin
// coincidence.
func (s *PrecompileTestSuite) TestDelegateForForeignAccountStillRejected() {
	victim := s.keyring.GetKey(1)
	validator := s.network.GetValidators()[0]

	forwarderAddr, forwarder := s.deployForwarder()

	// A grant shaped exactly like an artifact of the old hole: victim -> contract.
	delAmt := math.NewInt(1e18)
	valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
	s.Require().NoError(err)
	spendLimit := sdk.NewCoin(s.bondDenom, delAmt.MulRaw(2))
	grant, err := stakingtypes.NewStakeAuthorization(
		[]sdk.ValAddress{valAddr}, nil,
		stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE,
		&spendLimit,
	)
	s.Require().NoError(err)
	expiration := s.network.GetContext().BlockTime().Add(time.Hour)
	s.Require().NoError(s.network.App.AuthzKeeper.SaveGrant(
		s.network.GetContext(), forwarderAddr.Bytes(), victim.AccAddr, grant, &expiration,
	))
	s.Require().NoError(s.network.NextBlock())

	calldata, err := s.precompile.ABI.Pack(
		staking.DelegateMethod,
		victim.Addr,
		validator.OperatorAddress,
		delAmt.BigInt(),
	)
	s.Require().NoError(err)

	// Key 0 signs, so origin != sender.
	s.Require().Error(
		s.forward(forwarderAddr, forwarder, calldata),
		"a caller must not act for an account that did not sign the transaction, grant or no grant",
	)
}
