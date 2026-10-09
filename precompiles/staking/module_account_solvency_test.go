package staking_test

import (
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/staking"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	testutils "github.com/haqq-network/haqq/testutil/integration/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// Module accounts and the EVM StateDB.
//
// x/evm SetBalance reconciles every dirty EVM account against the bank on commit and
// mints or burns the difference. Module accounts are not exempt, and their balances
// move through bank calls that leave no EVM journal entry - which is exactly what a
// stateful precompile does when it delegates, undelegates or funds the community pool.
//
// What makes that reconciliation wrong rather than merely redundant is where the two
// sides read from. StateDB.getStateObject loads an account's balance from s.ctx, while
// a precompile writes through s.cacheCtx, so an account first loaded into the StateDB
// *after* a precompile moved it carries the balance it had *before* the call. Commit
// then treats that stale number as the truth.
//
// The precompiles paper over this for the accounts they know about by mirroring the
// bank movement into the journal (delegator, distribution withdrawer, ucDAO escrow,
// IBC escrow). The staking pools are moved by every delegation and are mirrored by
// nothing, which is what the first test below exercises.

// bondedObligations is the bonded side of the staking module account invariant: what
// the bonded pool has to cover.
func (s *PrecompileTestSuite) bondedObligations(ctx sdk.Context) math.Int {
	total := math.ZeroInt()
	s.Require().NoError(s.network.App.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) bool {
		if val.IsBonded() {
			total = total.Add(val.GetTokens())
		}
		return false
	}))

	return total
}

// deployToucher deploys the helper that dirties a module account and calls a
// precompile in the same transaction.
func (s *PrecompileTestSuite) deployToucher() (common.Address, evmtypes.CompiledContract) {
	toucher, err := contracts.LoadModuleAccountToucherContract()
	s.Require().NoError(err)

	addr, err := s.factory.DeployContract(
		s.keyring.GetPrivKey(0),
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: toucher},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	return addr, toucher
}

// TestDelegateThenTouchKeepsBondedPoolSolvent is the reproduction.
//
// In one transaction the contract:
//  1. delegates X of its own funds - the staking keeper moves X into the bonded pool
//     through the bank, inside the StateDB's cache context, with no journal entry;
//  2. sends 1 wei to the bonded pool - the StateDB loads the pool from s.ctx, which
//     still shows the pre-delegation balance B, and journals it dirty at B+1.
//
// On commit SetBalance compares the EVM view B+1 against the bank's B+X and burns
// X-1 out of the pool. The delegation shares survive, so the bonded pool is left short
// of what the bonded validators are owed and the shortfall lands on whoever unbonds
// last.
//
// Doing it in the other order does not work, and that is why the order is spelled out:
// touching first makes the EVM view exceed the bank, and the precompile's own
// pre-commit then tries to *credit* the pool, which bank refuses. Only the stale-read
// direction burns.
//
// The assertion is the invariant, not the mechanism: the bonded pool must always cover
// the bonded validator tokens, and the delegation must not destroy supply. That holds
// whether the transaction is made to work or made to fail - silently burning the pool
// is what must not happen.
func (s *PrecompileTestSuite) TestDelegateThenTouchKeepsBondedPoolSolvent() {
	signer := s.keyring.GetKey(0)
	validator := s.network.GetValidators()[0]

	toucherAddr, toucher := s.deployToucher()

	delAmt := math.NewInt(1e18)
	s.Require().NoError(testutils.FundAccountWithBaseDenom(
		s.factory, s.network, signer, toucherAddr.Bytes(), delAmt.MulRaw(3),
	))
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	bondedPool := s.network.App.AccountKeeper.GetModuleAddress(stakingtypes.BondedPoolName)
	bondedPoolEVM := common.BytesToAddress(bondedPool.Bytes())

	poolBefore := s.network.App.BankKeeper.GetBalance(ctx, bondedPool, s.bondDenom).Amount
	owedBefore := s.bondedObligations(ctx)
	supplyBefore := s.network.App.BankKeeper.GetSupply(ctx, s.bondDenom).Amount
	s.Require().True(
		poolBefore.GTE(owedBefore),
		"precondition: the bonded pool must start solvent (pool %s, owed %s)", poolBefore, owedBefore,
	)

	delegateData, err := s.precompile.ABI.Pack(
		staking.DelegateMethod, toucherAddr, validator.OperatorAddress, delAmt.BigInt(),
	)
	s.Require().NoError(err)

	// The contract delegates its own funds, so caller == delegator and no grant is
	// needed; forwardThenTouch then sends the pool 1 wei in the same transaction.
	precompileAddr := common.HexToAddress(evmtypes.StakingPrecompileAddress)
	_, callErr := s.factory.ExecuteContractCall(
		signer.Priv,
		evmtypes.EvmTxArgs{To: &toucherAddr, GasLimit: 2_000_000, Amount: big.NewInt(1)},
		factory.CallArgs{
			ContractABI: toucher.ABI,
			MethodName:  "forwardThenTouch",
			Args:        []interface{}{precompileAddr, delegateData, bondedPoolEVM},
		},
	)
	s.Require().NoError(s.network.NextBlock())

	ctx = s.network.GetContext()
	poolAfter := s.network.App.BankKeeper.GetBalance(ctx, bondedPool, s.bondDenom).Amount
	owedAfter := s.bondedObligations(ctx)
	supplyAfter := s.network.App.BankKeeper.GetSupply(ctx, s.bondDenom).Amount
	s.T().Logf(
		"tx error: %v | pool %s -> %s | owed %s -> %s | supply delta %s",
		callErr, poolBefore, poolAfter, owedBefore, owedAfter, supplyAfter.Sub(supplyBefore),
	)

	s.Require().True(
		poolAfter.GTE(owedAfter),
		"bonded pool is short by %s: it holds %s but bonded validators are owed %s",
		owedAfter.Sub(poolAfter), poolAfter, owedAfter,
	)
	s.Require().True(
		supplyAfter.GTE(supplyBefore),
		"delegating destroyed %s of supply", supplyBefore.Sub(supplyAfter),
	)
}

// TestZeroValueCallCannotDirtyModuleAccount pins the property that keeps the same
// stale-read trick out of reach without a value transfer.
//
// Geth's stateObject.AddBalance touches an empty account when the amount is zero,
// which journals it dirty at whatever balance was read. This fork's AddBalance returns
// early on a zero amount and journals nothing, so a zero-value CALL cannot stale-load
// an account. Restoring the upstream touch semantics would reopen the hole for every
// module account that is empty at rest.
func (s *PrecompileTestSuite) TestZeroValueCallCannotDirtyModuleAccount() {
	signer := s.keyring.GetKey(0)
	validator := s.network.GetValidators()[0]

	toucherAddr, toucher := s.deployToucher()

	delAmt := math.NewInt(1e18)
	s.Require().NoError(testutils.FundAccountWithBaseDenom(
		s.factory, s.network, signer, toucherAddr.Bytes(), delAmt.MulRaw(3),
	))
	s.Require().NoError(s.network.NextBlock())

	precompileAddr := common.HexToAddress(evmtypes.StakingPrecompileAddress)
	delegateData, err := s.precompile.ABI.Pack(
		staking.DelegateMethod, toucherAddr, validator.OperatorAddress, delAmt.BigInt(),
	)
	s.Require().NoError(err)

	ctx := s.network.GetContext()
	bondedPool := s.network.App.AccountKeeper.GetModuleAddress(stakingtypes.BondedPoolName)
	poolBefore := s.network.App.BankKeeper.GetBalance(ctx, bondedPool, s.bondDenom).Amount

	_, callErr := s.factory.ExecuteContractCall(
		signer.Priv,
		evmtypes.EvmTxArgs{To: &toucherAddr, GasLimit: 2_000_000},
		factory.CallArgs{
			ContractABI: toucher.ABI,
			MethodName:  "touchZeroAndForward",
			Args:        []interface{}{common.BytesToAddress(bondedPool.Bytes()), precompileAddr, delegateData},
		},
	)
	s.Require().NoError(callErr, "a zero-value call plus a delegation must simply work")
	s.Require().NoError(s.network.NextBlock())

	ctx = s.network.GetContext()
	poolAfter := s.network.App.BankKeeper.GetBalance(ctx, bondedPool, s.bondDenom).Amount
	s.Require().Equal(
		poolBefore.Add(delAmt).String(), poolAfter.String(),
		"the delegated amount must land in the bonded pool and stay there",
	)
	s.Require().True(poolAfter.GTE(s.bondedObligations(ctx)), "bonded pool must stay solvent")
}

// TestValueTransferToModuleAccountIsRefused pins the direction that is already safe.
// A value-bearing CALL raises the EVM's view of a module account above the bank, so
// commit tries to credit it and bank refuses - module accounts are blocked recipients.
// The transaction must fail rather than move the module account's balance.
func (s *PrecompileTestSuite) TestValueTransferToModuleAccountIsRefused() {
	signer := s.keyring.GetKey(0)

	toucherAddr, toucher := s.deployToucher()
	s.Require().NoError(testutils.FundAccountWithBaseDenom(
		s.factory, s.network, signer, toucherAddr.Bytes(), math.NewInt(1e18),
	))
	s.Require().NoError(s.network.NextBlock())

	ctx := s.network.GetContext()
	bondedPool := s.network.App.AccountKeeper.GetModuleAddress(stakingtypes.BondedPoolName)
	poolBefore := s.network.App.BankKeeper.GetBalance(ctx, bondedPool, s.bondDenom).Amount

	_, callErr := s.factory.ExecuteContractCall(
		signer.Priv,
		evmtypes.EvmTxArgs{To: &toucherAddr, GasLimit: 2_000_000, Amount: big.NewInt(1)},
		factory.CallArgs{
			ContractABI: toucher.ABI,
			MethodName:  "forward",
			Args:        []interface{}{common.BytesToAddress(bondedPool.Bytes()), []byte{}},
		},
	)
	s.Require().Error(callErr, "the EVM must not be able to credit a module account")
	s.Require().NoError(s.network.NextBlock())

	ctx = s.network.GetContext()
	poolAfter := s.network.App.BankKeeper.GetBalance(ctx, bondedPool, s.bondDenom).Amount
	s.Require().Equal(
		poolBefore.String(), poolAfter.String(),
		"a module account's balance must not move because of an EVM transfer",
	)
}
