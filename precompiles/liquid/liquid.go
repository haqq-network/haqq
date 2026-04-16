package liquid

import (
	"embed"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	liquidkeeper "github.com/haqq-network/haqq/x/liquidvesting/keeper"
)

//go:embed abi.json
var f embed.FS

// Precompile defines the precompiled contract for the liquidvesting module.
type Precompile struct {
	cmn.Precompile
	keeper liquidkeeper.Keeper
}

// NewPrecompile creates a new liquid Precompile instance as a PrecompiledContract.
func NewPrecompile(
	keeper liquidkeeper.Keeper,
	authzKeeper authzkeeper.Keeper,
) (*Precompile, error) {
	loadedABI, err := cmn.LoadABI(f, "abi.json")
	if err != nil {
		return nil, err
	}

	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  loadedABI,
			AuthzKeeper:          authzKeeper,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
			ApprovalExpiration:   cmn.DefaultExpirationDuration,
		},
		keeper: keeper,
	}

	// SetAddress defines the address of the liquid precompiled contract.
	p.SetAddress(common.HexToAddress(evmtypes.LiquidPrecompileAddress))

	return p, nil
}

// RequiredGas returns the required bare minimum gas to execute the precompile.
func (p Precompile) RequiredGas(input []byte) uint64 {
	// NOTE: This check avoids panicking when trying to decode the method ID.
	if len(input) < 4 {
		return 0
	}

	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		// This should never happen since this method is going to fail during Run.
		return 0
	}

	return p.Precompile.RequiredGas(input, p.IsTransaction(method.Name))
}

// Run executes the precompiled contract methods defined in the ABI.
func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, snapshot, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err, stateDB, snapshot)()

	return p.RunAtomic(
		snapshot,
		stateDB,
		func() ([]byte, error) {
			switch method.Name {
			case LiquidateMethod:
				bz, err = p.Liquidate(ctx, evm.Origin, contract, stateDB, method, args)
			case RedeemMethod:
				bz, err = p.Redeem(ctx, evm.Origin, contract, stateDB, method, args)
			default:
				return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
			}

			if err != nil {
				return nil, err
			}

			cost := ctx.GasMeter().GasConsumed() - initialGas

			if !contract.UseGas(cost) {
				return nil, vm.ErrOutOfGas
			}

			if err := p.AddJournalEntries(stateDB, snapshot); err != nil {
				return nil, err
			}

			return bz, nil
		},
	)
}

// IsTransaction checks if the given method name corresponds to a transaction.
func (Precompile) IsTransaction(method string) bool {
	switch method {
	case LiquidateMethod, RedeemMethod:
		return true
	default:
		return false
	}
}

// Logger returns a precompile-specific logger.
func (p Precompile) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("evm extension", "liquid")
}
