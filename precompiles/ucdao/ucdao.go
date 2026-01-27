// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ucdao

import (
	"embed"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	ucdaokeeper "github.com/haqq-network/haqq/x/ucdao/keeper"
)

var _ vm.PrecompiledContract = &Precompile{}

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

// Precompile defines the precompiled contract for ucdao.
type Precompile struct {
	cmn.Precompile
	ucdaoKeeper ucdaokeeper.Keeper
	bankKeeper  bankkeeper.Keeper
}

// LoadABI loads the ucdao ABI from the embedded abi.json file
// for the ucdao precompile.
func LoadABI() (abi.ABI, error) {
	return cmn.LoadABI(f, "abi.json")
}

// NewPrecompile creates a new ucdao Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile(
	ucdaoKeeper ucdaokeeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	authzKeeper authzkeeper.Keeper,
) (*Precompile, error) {
	abi, err := LoadABI()
	if err != nil {
		return nil, err
	}

	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  abi,
			AuthzKeeper:          authzKeeper,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
			ApprovalExpiration:   cmn.DefaultExpirationDuration,
		},
		ucdaoKeeper: ucdaoKeeper,
		bankKeeper:  bankKeeper,
	}
	// SetAddress defines the address of the ucdao precompiled contract.
	p.SetAddress(common.HexToAddress(evmtypes.UcdaoPrecompileAddress))

	return p, nil
}

// RequiredGas returns the required bare minimum gas to execute the precompile.
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

// Run executes the precompiled contract ucdao methods defined in the ABI.
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
			// Authorization transactions
			case ApproveMethod:
				bz, err = p.Approve(ctx, evm.Origin, stateDB, method, args)
			case RevokeMethod:
				bz, err = p.Revoke(ctx, evm.Origin, stateDB, method, args)
			case IncreaseAllowanceMethod:
				bz, err = p.IncreaseAllowance(ctx, evm.Origin, stateDB, method, args)
			case DecreaseAllowanceMethod:
				bz, err = p.DecreaseAllowance(ctx, evm.Origin, stateDB, method, args)
			// Authorization queries
			case AllowanceMethod:
				bz, err = p.Allowance(ctx, method, args)
			// UCDAO transactions
			case FundMethod:
				bz, err = p.Fund(ctx, evm.Origin, contract, stateDB, method, args)
			case TransferOwnershipMethod:
				bz, err = p.TransferOwnership(ctx, evm.Origin, contract, stateDB, method, args)
			case TransferOwnershipWithRatioMethod:
				bz, err = p.TransferOwnershipWithRatio(ctx, evm.Origin, contract, stateDB, method, args)
			case TransferOwnershipWithAmountMethod:
				bz, err = p.TransferOwnershipWithAmount(ctx, evm.Origin, contract, stateDB, method, args)
			// UCDAO queries
			case BalanceMethod:
				bz, err = p.Balance(ctx, method, args)
			case AllBalancesMethod:
				bz, err = p.AllBalances(ctx, method, args)
			case TotalBalanceMethod:
				bz, err = p.TotalBalance(ctx, method, args)
			case EnabledMethod:
				bz, err = p.Enabled(ctx, method, args)
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
func (Precompile) IsTransaction(method string) bool {
	switch method {
	case ApproveMethod,
		RevokeMethod,
		IncreaseAllowanceMethod,
		DecreaseAllowanceMethod,
		FundMethod,
		TransferOwnershipMethod,
		TransferOwnershipWithRatioMethod,
		TransferOwnershipWithAmountMethod:
		return true
	default:
		return false
	}
}

// Logger returns a precompile-specific logger.
func (p Precompile) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("evm extension", "ucdao")
}
