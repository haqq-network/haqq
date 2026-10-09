package distribution_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/distribution"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	testutils "github.com/haqq-network/haqq/testutil/integration/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// Regression tests for tx.origin authorization in the distribution precompile.
//
// x/distribution has no authorization type -- the precompile exposes no approve or
// revoke and never reads its AuthzKeeper -- so admitting `origin == delegator` made
// tx.origin the authorization itself. A single transaction in which
// attacker-controlled code ran was enough to call setWithdrawAddress for the signer
// and redirect their reward stream permanently, with nothing to revoke afterwards.

// deployDistributionCaller deploys the DistributionCaller helper contract, which
// forwards calls to the distribution precompile so that
// contract.CallerAddress != tx.origin.
func (s *PrecompileTestSuite) deployDistributionCaller() (common.Address, evmtypes.CompiledContract) {
	caller, err := contracts.LoadDistributionCallerContract()
	s.Require().NoError(err)

	addr, err := s.factory.DeployContract(
		s.keyring.GetPrivKey(0),
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: caller},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	return addr, caller
}

func (s *PrecompileTestSuite) callDistributionCaller(
	callerAddr common.Address,
	caller evmtypes.CompiledContract,
	method string,
	args ...interface{},
) error {
	_, err := s.factory.ExecuteContractCall(
		s.keyring.GetPrivKey(0),
		evmtypes.EvmTxArgs{To: &callerAddr, GasLimit: 2_000_000},
		factory.CallArgs{ContractABI: caller.ABI, MethodName: method, Args: args},
	)
	return err
}

// TestSetWithdrawAddressForOriginFromContractFails is the exploit itself: a contract
// pointing the signer's reward stream at an attacker address. This was the whole
// attack -- no grant, no prior approval, permanent.
func (s *PrecompileTestSuite) TestSetWithdrawAddressForOriginFromContractFails() {
	victim := s.keyring.GetKey(0)
	attacker := s.keyring.GetKey(1)

	callerAddr, caller := s.deployDistributionCaller()

	err := s.callDistributionCaller(
		callerAddr, caller,
		"testSetWithdrawAddress", victim.Addr, attacker.AccAddr.String(),
	)
	s.Require().Error(err, "a contract must not be able to redirect tx.origin's reward stream")
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	withdrawer, err := s.network.App.DistrKeeper.GetDelegatorWithdrawAddr(ctx, victim.AccAddr)
	s.Require().NoError(err)
	s.Require().Equal(
		victim.AccAddr.String(),
		withdrawer.String(),
		"the victim's withdraw address must be untouched",
	)
}

// TestSetWithdrawAddressForSelfFromContractWorks pins the legitimate case that
// must keep working: a contract (or contract wallet) setting its own withdraw
// address, where the caller is the delegator.
func (s *PrecompileTestSuite) TestSetWithdrawAddressForSelfFromContractWorks() {
	receiver := s.keyring.GetKey(1)

	callerAddr, caller := s.deployDistributionCaller()

	s.Require().NoError(s.callDistributionCaller(
		callerAddr, caller,
		"testSetWithdrawAddressFromContract", receiver.AccAddr.String(),
	), "a contract must still be able to set its own withdraw address")
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	withdrawer, err := s.network.App.DistrKeeper.GetDelegatorWithdrawAddr(ctx, callerAddr.Bytes())
	s.Require().NoError(err)
	s.Require().Equal(receiver.AccAddr.String(), withdrawer.String())
}

// TestSetWithdrawAddressFromEOAIsUnchanged pins the backward-compatible path: a
// direct EOA call has caller == delegator and is unaffected.
func (s *PrecompileTestSuite) TestSetWithdrawAddressFromEOAIsUnchanged() {
	delegator := s.keyring.GetKey(0)
	receiver := s.keyring.GetKey(1)

	precompileAddr := common.HexToAddress(evmtypes.DistributionPrecompileAddress)

	_, err := s.factory.ExecuteContractCall(
		delegator.Priv,
		evmtypes.EvmTxArgs{To: &precompileAddr, GasLimit: 2_000_000},
		factory.CallArgs{
			ContractABI: s.precompile.ABI,
			MethodName:  distribution.SetWithdrawAddressMethod,
			Args:        []interface{}{delegator.Addr, receiver.AccAddr.String()},
		},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	withdrawer, err := s.network.App.DistrKeeper.GetDelegatorWithdrawAddr(ctx, delegator.AccAddr)
	s.Require().NoError(err)
	s.Require().Equal(receiver.AccAddr.String(), withdrawer.String())
}

// TestFundCommunityPoolForOriginFromContractFails covers the griefing half: a
// contract draining the signer's balance into the community pool.
func (s *PrecompileTestSuite) TestFundCommunityPoolForOriginFromContractFails() {
	victim := s.keyring.GetKey(0)

	callerAddr, caller := s.deployDistributionCaller()

	amount := math.NewInt(1e18)
	before := s.network.App.BankKeeper.GetBalance(
		s.network.GetContext(), victim.AccAddr, s.bondDenom,
	).Amount

	err := s.callDistributionCaller(
		callerAddr, caller,
		"testFundCommunityPool", victim.Addr, amount.BigInt(),
	)
	s.Require().Error(err, "a contract must not be able to donate tx.origin's balance")
	s.Require().NoError(s.network.NextBlock())

	after := s.network.App.BankKeeper.GetBalance(
		s.network.GetContext(), victim.AccAddr, s.bondDenom,
	).Amount
	s.Require().True(
		before.Sub(after).LT(amount),
		"only the gas of the reverted transaction may have been spent, not the donated amount",
	)
}

// TestFundCommunityPoolForSelfFromContractWorks pins that a contract funding the
// pool from its own balance still works.
func (s *PrecompileTestSuite) TestFundCommunityPoolForSelfFromContractWorks() {
	signer := s.keyring.GetKey(0)

	callerAddr, caller := s.deployDistributionCaller()

	amount := math.NewInt(1e18)
	s.Require().NoError(testutils.FundAccountWithBaseDenom(
		s.factory, s.network, signer, callerAddr.Bytes(), amount.MulRaw(2),
	))
	s.Require().NoError(s.network.NextBlock())

	s.Require().NoError(s.callDistributionCaller(
		callerAddr, caller,
		"testFundCommunityPool", callerAddr, amount.BigInt(),
	), "a contract must still be able to fund the pool from its own balance")
}

// TestClaimRewardsForOriginFromContractStillWorks pins the residual that was
// deliberately kept. With setWithdrawAddress locked, a forced claim can only pay
// the address the delegator chose, so it is griefing rather than theft -- and
// keeping it preserves the auto-compounder pattern.
func (s *PrecompileTestSuite) TestClaimRewardsForOriginFromContractStillWorks() {
	delegator := s.keyring.GetKey(0)
	validator := s.network.GetValidators()[0]

	s.Require().NoError(s.factory.Delegate(
		delegator.Priv, validator.OperatorAddress, sdk.NewCoin(s.bondDenom, math.NewInt(1e18)),
	))
	s.Require().NoError(s.network.NextBlock())
	s.Require().NoError(s.network.NextBlock())

	callerAddr, caller := s.deployDistributionCaller()

	s.Require().NoError(s.callDistributionCaller(
		callerAddr, caller,
		"testClaimRewards", delegator.Addr, uint32(1),
	), "claimRewards for the origin is deliberately still permitted from a contract")
}

// TestSetWithdrawAddressRejectsNonEVMLength covers the second half of the
// distribution write-up: a withdraw address that does not decode to 20 bytes has
// no EVM representation, so rewards paid there cannot be recovered from the EVM
// side. Longer addresses stay legitimate on the Cosmos path.
func (s *PrecompileTestSuite) TestSetWithdrawAddressRejectsNonEVMLength() {
	delegator := s.keyring.GetKey(0)

	wideBytes := make([]byte, 32)
	copy(wideBytes, []byte("thirtytwobytewithdrawaddress1234"))
	wideAddr := sdk.AccAddress(wideBytes).String()

	_, _, err := distribution.NewMsgSetWithdrawAddress([]interface{}{delegator.Addr, wideAddr})
	s.Require().Error(err, "a 32-byte withdraw address must be rejected on the EVM path")
	s.Require().Contains(err.Error(), "must be a 20-byte address")

	precompileAddr := common.HexToAddress(evmtypes.DistributionPrecompileAddress)
	_, err = s.factory.ExecuteContractCall(
		delegator.Priv,
		evmtypes.EvmTxArgs{To: &precompileAddr, GasLimit: 2_000_000},
		factory.CallArgs{
			ContractABI: s.precompile.ABI,
			MethodName:  distribution.SetWithdrawAddressMethod,
			Args:        []interface{}{delegator.Addr, wideAddr},
		},
	)
	s.Require().Error(err)
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	withdrawer, err := s.network.App.DistrKeeper.GetDelegatorWithdrawAddr(ctx, delegator.AccAddr)
	s.Require().NoError(err)
	s.Require().Equal(delegator.AccAddr.String(), withdrawer.String())
}
