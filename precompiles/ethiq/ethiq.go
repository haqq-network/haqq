package ethiq

import (
	"embed"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	ethiqkeeper "github.com/haqq-network/haqq/x/ethiq/keeper"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

// Precompile defines the precompiled contract for ethiq.
type Precompile struct {
	cmn.Precompile
	ethiqKeeper *ethiqkeeper.Keeper
}

// NewPrecompile creates a new ethiq Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile(
	ethiqKeeper *ethiqkeeper.Keeper,
	authzKeeper authzkeeper.Keeper,
) (*Precompile, error) {
	loadedAbi, err := cmn.LoadABI(f, "abi.json")
	if err != nil {
		return nil, err
	}

	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  loadedAbi,
			AuthzKeeper:          authzKeeper,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
			ApprovalExpiration:   cmn.DefaultExpirationDuration, // should be configurable in the future.
		},
		ethiqKeeper: ethiqKeeper,
	}
	// SetAddress defines the address of the ethiq precompiled contract.
	p.SetAddress(common.HexToAddress(evmtypes.EthiqPrecompileAddress))

	return p, nil
}

// RequiredGas returns the required bare minimum gas to execute precompile.
func (p Precompile) RequiredGas(input []byte) uint64 {
	// NOTE: This check avoid panicking when trying to decode the method ID
	if len(input) < 4 {
		return 0
	}

	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		// This should never happen since this method is going to fail during Run
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
			// Authorization Methods:
			case authorization.ApproveMethod:
				bz, err = p.Approve(ctx, evm.Origin, stateDB, method, args)
			case authorization.RevokeMethod:
				bz, err = p.Revoke(ctx, evm.Origin, stateDB, method, args)
			case ApproveApplicationIDMethod:
				bz, err = p.ApproveApplicationID(ctx, evm.Origin, stateDB, method, args)
			case RevokeApplicationIDMethod:
				bz, err = p.RevokeApplicationID(ctx, evm.Origin, stateDB, method, args)
			case authorization.IncreaseAllowanceMethod:
				bz, err = p.IncreaseAllowance(ctx, evm.Origin, stateDB, method, args)
			case authorization.DecreaseAllowanceMethod:
				bz, err = p.DecreaseAllowance(ctx, evm.Origin, stateDB, method, args)
			// Transactions
			case MintHaqq:
				bz, err = p.MintHaqq(ctx, evm.Origin, contract, stateDB, method, args)
			case MintHaqqByApplication:
				bz, err = p.MintHaqqByApplication(ctx, evm.Origin, contract, stateDB, method, args)
			// Queries
			case Calculate:
				bz, err = p.Calculate(ctx, contract, method, args)
			case authorization.AllowanceMethod:
				bz, err = p.Allowance(ctx, method, contract, args)
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

// IsTransaction checks if the given method name corresponds to a transaction or query.
//
// Available ethiq transactions are:
//   - MintHaqq
//   - MintHaqqByApplication
//
// Available authorization transactions are:
//   - Approve
//   - ApproveApplicationID
//   - Revoke
//   - RevokeApplicationID
//   - IncreaseAllowance
//   - DecreaseAllowance
func (Precompile) IsTransaction(method string) bool {
	switch method {
	case MintHaqq,
		MintHaqqByApplication,
		authorization.ApproveMethod,
		authorization.RevokeMethod,
		ApproveApplicationIDMethod,
		RevokeApplicationIDMethod,
		authorization.IncreaseAllowanceMethod,
		authorization.DecreaseAllowanceMethod:
		return true
	default:
		return false
	}
}

// Logger returns a precompile-specific logger.
func (p Precompile) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("evm extension", "ethiq")
}
