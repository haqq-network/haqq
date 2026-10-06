package staking_test

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/precompiles/staking"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	"github.com/haqq-network/haqq/x/evm/statedb"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// newPrecompileEVM builds an EVM over a fresh StateDB, with the staking
// precompile loaded, the way a transaction from keyring account 0 sees it.
func (s *PrecompileTestSuite) newPrecompileEVM() *vm.EVM {
	ctx := s.network.GetContext().WithBlockTime(time.Now())
	caller := s.keyring.GetKey(0)
	precompileAddr := s.precompile.Address()

	msg, err := s.factory.GenerateGethCoreMsg(caller.Priv, evmtypes.EvmTxArgs{
		ChainID:   s.network.App.EvmKeeper.ChainID(),
		To:        &precompileAddr,
		GasLimit:  10_000_000,
		GasPrice:  app.MinGasPrices.BigInt(),
		GasFeeCap: s.network.App.FeeMarketKeeper.GetBaseFee(ctx),
		Accesses:  &ethtypes.AccessList{},
	})
	s.Require().NoError(err)

	cfg, err := s.network.App.EvmKeeper.EVMConfig(ctx, ctx.BlockHeader().ProposerAddress, s.network.App.EvmKeeper.ChainID())
	s.Require().NoError(err)
	stDB := statedb.New(ctx, s.network.App.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash())))
	evm := s.network.App.EvmKeeper.NewEVM(ctx, msg, cfg, nil, stDB)

	precompiles, found, err := s.network.App.EvmKeeper.GetPrecompileInstance(ctx, precompileAddr)
	s.Require().NoError(err)
	s.Require().True(found)
	evm.WithPrecompiles(precompiles.Map, precompiles.Addresses)
	return evm
}

// newBudgetRunner builds one EVM over one StateDB and returns a function that
// runs the staking precompile against it, so successive calls share the
// transaction's MaxPrecompileCalls budget the way calls from one contract do.
func (s *PrecompileTestSuite) newBudgetRunner() func(input []byte, readOnly bool) error {
	evm := s.newPrecompileEVM()
	caller := s.keyring.GetAddr(0)

	return func(input []byte, readOnly bool) error {
		contract := vm.NewPrecompile(vm.AccountRef(caller), s.precompile, common.Big0, 1_000_000)
		contract.Input = input
		_, err := s.precompile.Run(evm, contract, readOnly)
		return err
	}
}

// TestPrecompileCallBudget pins down what MaxPrecompileCalls counts. Every call
// that passes calldata validation snapshots the cache context and flushes the
// dirty set before it can fail, so each one is counted whether it succeeds or
// not; calls rejected on their calldata alone touch no state and are free.
func (s *PrecompileTestSuite) TestPrecompileCallBudget() {
	maxCalls := int(evmtypes.MaxPrecompileCalls)
	maxCallsErr := fmt.Sprintf("max calls to precompiles (%d) reached", evmtypes.MaxPrecompileCalls)

	pack := func(method string, args ...interface{}) []byte {
		input, err := s.precompile.Pack(method, args...)
		s.Require().NoError(err)
		return input
	}
	validQuery := func() []byte {
		return pack(staking.DelegationMethod, s.keyring.GetAddr(0), s.network.GetValidators()[0].GetOperator())
	}

	s.Run("calls rejected on their calldata do not use the budget", func() {
		s.SetupTest()
		run := s.newBudgetRunner()

		for i := 0; i < 3*maxCalls; i++ {
			s.Require().ErrorContains(run([]byte{0xde, 0xad, 0xbe, 0xef}, false), "no method with id")
			s.Require().ErrorContains(run(nil, false), "execution reverted")
			s.Require().ErrorIs(run(pack(staking.DelegateMethod, s.keyring.GetAddr(0), s.network.GetValidators()[0].GetOperator(), common.Big1), true), vm.ErrWriteProtection)
		}

		for i := 0; i < maxCalls; i++ {
			s.Require().NoError(run(validQuery(), true), "call %d is within the budget", i+1)
		}
		s.Require().ErrorContains(run(validQuery(), true), maxCallsErr)
	})

	s.Run("calls that fail after the flush use the budget", func() {
		s.SetupTest()
		run := s.newBudgetRunner()

		// a zero delegator passes ABI decoding and fails inside the query
		failing := pack(staking.DelegationMethod, common.Address{}, s.network.GetValidators()[0].GetOperator())
		for i := 0; i < maxCalls; i++ {
			err := run(failing, true)
			s.Require().Error(err)
			s.Require().NotContains(err.Error(), "max calls to precompiles", "call %d is within the budget", i+1)
		}
		s.Require().ErrorContains(run(validQuery(), true), maxCallsErr)
	})

	s.Run("the budget does not wrap around", func() {
		s.SetupTest()
		run := s.newBudgetRunner()

		for i := 0; i < maxCalls; i++ {
			s.Require().NoError(run(validQuery(), true))
		}
		// the counter used to be a uint8 bumped on every attempt, so 256 refused
		// calls brought it back to zero and reopened the budget
		for i := 0; i < 300; i++ {
			s.Require().ErrorContains(run(validQuery(), true), maxCallsErr, "attempt %d past the budget", i+1)
		}
	})
}

// TestDelegateCallWithEmptyCalldata checks that a DELEGATECALL into a stateful
// precompile with empty calldata reverts the call instead of panicking. The EVM
// ran delegatecalled precompiles with a nil value, which the empty-calldata
// routing dereferenced, so the panic aborted the whole transaction rather than
// failing the one call the calling contract could have handled.
func (s *PrecompileTestSuite) TestDelegateCallWithEmptyCalldata() {
	s.SetupTest()
	evm := s.newPrecompileEVM()
	caller := vm.AccountRef(s.keyring.GetAddr(0))

	s.Require().NotPanics(func() {
		_, _, err := evm.DelegateCall(caller, s.precompile.Address(), nil, 1_000_000)
		s.Require().ErrorIs(err, vm.ErrExecutionReverted)
	})
}
