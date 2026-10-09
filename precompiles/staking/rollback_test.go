package staking_test

import (
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/staking"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// Regression guards for PR #460: StateDB flushes a precompile performed before it
// could fail were never journaled.
//
// Every stateful precompile enters through cmn.Precompile.RunSetup, which flushes
// the whole dirty EVM set into the StateDB cache context through x/evm SetAccount -
// real bank mints and burns. It used to do that *before* resolving the selector,
// and only a run that got past setup recorded an AddPrecompileFn journal entry, so
// an error raised in setup left the flushed bank writes in cacheCtx with nothing
// that could undo them. The EVM-side journal revert restored the in-memory balances
// and dropped the dirty markers, and the final StateDB.Commit wrote cacheCtx back
// wholesale, so the bank ended up holding a state the EVM had rolled back.
//
// RunSetup now validates the calldata before it snapshots and flushes anything, and
// StateDB.commitWithCtx stages the flush so a failed one leaves cacheCtx untouched.
// Each exploit run is paired with a control run that differs only in not touching
// the precompile, so any delta is attributable to the precompile call (coinomics is
// switched off so block inflation cannot move the supply).

// rollbackEnv deploys the Rollback fixture, funds it and disables coinomics
// minting so the aISLM supply only moves because of the transactions under test.
func (s *PrecompileTestSuite) rollbackEnv(funding math.Int) (common.Address, evmtypes.CompiledContract) {
	return s.deployFunded(funding, contracts.LoadRollbackContract)
}

func (s *PrecompileTestSuite) bankBalance(addr sdk.AccAddress) math.Int {
	return s.network.App.BankKeeper.GetBalance(s.network.GetContext(), addr, s.bondDenom).Amount
}

func (s *PrecompileTestSuite) spendableBalance(addr sdk.AccAddress) math.Int {
	return s.network.App.BankKeeper.SpendableCoin(s.network.GetContext(), addr, s.bondDenom).Amount
}

func (s *PrecompileTestSuite) bankSupply() math.Int {
	return s.network.App.BankKeeper.GetSupply(s.network.GetContext(), s.bondDenom).Amount
}

type rollbackOutcome struct {
	payerDelta     math.Int // bank balance change of the contract
	payerSpendable math.Int // spendable balance of the contract afterwards
	recipient      math.Int // bank balance of the payee of the reverted payment
	sink           math.Int // bank balance of the payee of the kept 1 wei
	supplyDelta    math.Int
}

// runRollbackThenPay has the fixture pay `amount` to a fresh recipient in a child
// frame, optionally poke 0x0800 with an unknown selector, revert the child, catch
// the revert, and then pay 1 wei to a fresh sink from the outer frame.
func (s *PrecompileTestSuite) runRollbackThenPay(
	contractAddr common.Address, fixture evmtypes.CompiledContract, amount *big.Int, poke bool,
) rollbackOutcome {
	recipient := utiltx.GenerateAddress()
	sink := utiltx.GenerateAddress()
	payer := sdk.AccAddress(contractAddr.Bytes())

	payerBefore := s.bankBalance(payer)
	supplyBefore := s.bankSupply()

	_, err := s.factory.ExecuteContractCall(
		s.keyring.GetPrivKey(0),
		evmtypes.EvmTxArgs{To: &contractAddr, GasLimit: 3_000_000},
		factory.CallArgs{
			ContractABI: fixture.ABI,
			MethodName:  "rollbackThenPay",
			Args:        []interface{}{recipient, amount, sink, poke},
		},
	)
	s.Require().NoError(err, "the outer frame catches the child revert, so the tx must succeed")
	s.Require().NoError(s.network.NextBlock())
	s.Require().NoError(s.network.CheckAccountingInvariants())

	return rollbackOutcome{
		payerDelta:     s.bankBalance(payer).Sub(payerBefore),
		payerSpendable: s.spendableBalance(payer),
		recipient:      s.bankBalance(sdk.AccAddress(recipient.Bytes())),
		sink:           s.bankBalance(sdk.AccAddress(sink.Bytes())),
		supplyDelta:    s.bankSupply().Sub(supplyBefore),
	}
}

// TestRevertedPaymentDoesNotSurvivePrecompileSetupError is the regression guard for
// the C-1 native aISLM inflation.
//
// Child frame: pay X to a fresh recipient, CALL 0x0800 with selector 0xdeadbeef,
// revert. RunSetup used to flush {payer: B-X, recipient: X} into cacheCtx and only
// then fail on the selector, with no journal entry to undo the flush, so the child
// revert left the recipient's minted X in cacheCtx and the final commit sealed an
// inflation of X. RunSetup now rejects the selector before it flushes anything.
//
// The exploit path must be indistinguishable from the control: the payer loses only
// the 1 wei it really sent, the recipient gets nothing, supply is flat.
func (s *PrecompileTestSuite) TestRevertedPaymentDoesNotSurvivePrecompileSetupError() {
	funding := math.NewInt(3e18)
	amount := math.NewInt(5e17)
	contractAddr, fixture := s.rollbackEnv(funding)

	// Control: same transaction without the precompile call behaves correctly.
	control := s.runRollbackThenPay(contractAddr, fixture, amount.BigInt(), false)
	s.T().Logf("control: payer %s, recipient %s, sink %s, supply %s",
		control.payerDelta, control.recipient, control.sink, control.supplyDelta)
	s.Require().Equal("-1", control.payerDelta.String(), "control: payer only loses the kept 1 wei")
	s.Require().True(control.recipient.IsZero(), "control: the reverted payment must not land")
	s.Require().Equal("1", control.sink.String(), "control: the sink receives 1 wei")
	s.Require().True(control.supplyDelta.IsZero(), "control: supply must not move")

	exploit := s.runRollbackThenPay(contractAddr, fixture, amount.BigInt(), true)
	s.T().Logf("exploit: payer %s (spendable %s), recipient %s, sink %s, supply %s",
		exploit.payerDelta, exploit.payerSpendable, exploit.recipient, exploit.sink, exploit.supplyDelta)

	// The poke-through-precompile frame reverts cleanly, so the
	// exploit path matches the control exactly.
	s.Require().Equal("-1", exploit.payerDelta.String(),
		"payer loses only the 1 wei it really sent")
	s.Require().Equal(funding.SubRaw(2).String(), exploit.payerSpendable.String(),
		"payer's spendable aISLM matches the control: unchanged by the reverted frame")
	s.Require().True(exploit.recipient.IsZero(),
		"the reverted payment must not land on the recipient")
	s.Require().Equal("1", exploit.sink.String())
	s.Require().True(exploit.supplyDelta.IsZero(),
		"aISLM supply must not move: the reverted frame's mint is rolled back")
}

// TestValueCallToPrecompileLeavesNoOrphanMint is the regression guard for the C-2
// orphan mint in x/evm.
//
// CALL 0x0800 with value V: the EVM transfer dirties {0x0800: V, payer: B-V}. The
// flush in RunSetup handles 0x0800 first (lowest address): SetBalance mints V to the
// x/evm module, then SendCoinsFromModuleToAccount fails because precompile addresses
// are bank blocked recipients. CommitWithCacheCtx used to write straight into
// cacheCtx, so the mint persisted while the send failed. Now commitWithCtx stages the
// whole dirty-set walk and promotes it only on full success.
//
// The call carries a valid selector on purpose: RunSetup rejects bad calldata before
// it snapshots and flushes anything, so a bogus selector would never reach the flush
// this test guards. The PrecompileChaos fixture makes the call and swallows its
// failure, so the transaction itself succeeds.
//
// Nothing may be minted, the x/evm module is untouched, supply is flat.
func (s *PrecompileTestSuite) TestValueCallToPrecompileLeavesNoOrphanMint() {
	funding := math.NewInt(3e18)
	value := big.NewInt(7e17)
	env := s.newChaosEnv(funding)

	payer := sdk.AccAddress(env.chaos.Bytes())
	evmModule := authtypes.NewModuleAddress(evmtypes.ModuleName)
	precompile := sdk.AccAddress(env.staking.Bytes())

	payerBefore := s.bankBalance(payer)
	moduleBefore := s.bankBalance(evmModule)
	precompileBefore := s.bankBalance(precompile)
	supplyBefore := s.bankSupply()

	// a valid query: it passes calldata validation and reaches the flush
	query := env.pack(env.staking, staking.DelegationMethod, env.chaos, env.validator)
	_, err := s.factory.ExecuteContractCall(
		s.keyring.GetPrivKey(0),
		evmtypes.EvmTxArgs{To: &env.chaos, GasLimit: 3_000_000},
		factory.CallArgs{
			ContractABI: env.fixture.ABI,
			MethodName:  "run",
			Args:        []interface{}{[]chaosStep{call(env.staking, value, query, 200_000)}, false},
		},
	)
	s.Require().NoError(err, "the fixture swallows the failed precompile call, so the tx must succeed")
	s.Require().NoError(s.network.NextBlock())
	s.Require().NoError(s.network.CheckAccountingInvariants())

	payerDelta := s.bankBalance(payer).Sub(payerBefore)
	moduleDelta := s.bankBalance(evmModule).Sub(moduleBefore)
	precompileDelta := s.bankBalance(precompile).Sub(precompileBefore)
	supplyDelta := s.bankSupply().Sub(supplyBefore)
	s.T().Logf("payer %s, x/evm module %s, 0x0800 %s, supply %s",
		payerDelta, moduleDelta, precompileDelta, supplyDelta)

	// The value transfer is rolled back with the failed call, so no mint leaks.
	s.Require().True(payerDelta.IsZero(), "the payer's transfer was rolled back")
	s.Require().True(precompileDelta.IsZero(), "the blocked precompile address received nothing")
	s.Require().True(moduleDelta.IsZero(),
		"x/evm module account keeps no orphaned mint: the atomic flush discarded it")
	s.Require().True(supplyDelta.IsZero(),
		"aISLM supply must not move for a failed value call to a precompile")
}
