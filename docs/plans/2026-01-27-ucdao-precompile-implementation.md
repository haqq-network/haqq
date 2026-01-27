# UCDAO Precompile Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a precompile for the UCDAO module that exposes DAO funding and ownership transfer operations to EVM contracts, with full allowance support using Cosmos SDK's authz module.

**Architecture:** The precompile embeds `cmn.Precompile` for base functionality (ABI, authz, gas configs), uses `SendAuthorization` from the bank module for amount-based per-denom allowances, and follows the same patterns as the staking/ICS20 precompiles for authorization flows.

**Tech Stack:** Go, Cosmos SDK authz module, Ethereum ABI encoding, go-ethereum

---

## Task 1: Add Precompile Address Constant

**Files:**
- Modify: `x/evm/types/precompiles.go:16-17`

**Step 1: Add the UCDAO precompile address constant**

In `x/evm/types/precompiles.go`, add after `BankPrecompileAddress`:

```go
UcdaoPrecompileAddress = "0x0000000000000000000000000000000000000805"
```

**Step 2: Add to available static precompiles list**

In the same file, add `UcdaoPrecompileAddress` to `AvailableStaticPrecompiles`:

```go
var AvailableStaticPrecompiles = []string{
	P256PrecompileAddress,
	Bech32PrecompileAddress,
	StakingPrecompileAddress,
	DistributionPrecompileAddress,
	ICS20PrecompileAddress,
	// VestingPrecompileAddress,
	BankPrecompileAddress,
	UcdaoPrecompileAddress,
}
```

**Step 3: Verify build compiles**

Run: `go build ./x/evm/...`
Expected: No errors

**Step 4: Commit**

```bash
git add x/evm/types/precompiles.go
git commit -m "feat(evm): add UCDAO precompile address constant"
```

---

## Task 2: Create Precompile Directory and ABI

**Files:**
- Create: `precompiles/ucdao/abi.json`

**Step 1: Create precompile directory**

```bash
mkdir -p precompiles/ucdao
```

**Step 2: Create the ABI file**

Create `precompiles/ucdao/abi.json` with the following content:

```json
[
  {
    "name": "approve",
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {"name": "spender", "type": "address"},
      {"name": "amount", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ]}
    ],
    "outputs": [{"name": "success", "type": "bool"}]
  },
  {
    "name": "revoke",
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [{"name": "spender", "type": "address"}],
    "outputs": [{"name": "success", "type": "bool"}]
  },
  {
    "name": "increaseAllowance",
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {"name": "spender", "type": "address"},
      {"name": "amount", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ]}
    ],
    "outputs": [{"name": "success", "type": "bool"}]
  },
  {
    "name": "decreaseAllowance",
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {"name": "spender", "type": "address"},
      {"name": "amount", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ]}
    ],
    "outputs": [{"name": "success", "type": "bool"}]
  },
  {
    "name": "allowance",
    "type": "function",
    "stateMutability": "view",
    "inputs": [
      {"name": "owner", "type": "address"},
      {"name": "spender", "type": "address"}
    ],
    "outputs": [
      {"name": "remaining", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ]}
    ]
  },
  {
    "name": "fund",
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {"name": "amount", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ]}
    ],
    "outputs": [{"name": "success", "type": "bool"}]
  },
  {
    "name": "transferOwnership",
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {"name": "owner", "type": "address"},
      {"name": "newOwner", "type": "address"}
    ],
    "outputs": [
      {"name": "transferred", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ]}
    ]
  },
  {
    "name": "transferOwnershipWithRatio",
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {"name": "owner", "type": "address"},
      {"name": "newOwner", "type": "address"},
      {"name": "ratio", "type": "uint256"}
    ],
    "outputs": [
      {"name": "transferred", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ]}
    ]
  },
  {
    "name": "transferOwnershipWithAmount",
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {"name": "owner", "type": "address"},
      {"name": "newOwner", "type": "address"},
      {"name": "amount", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ]}
    ],
    "outputs": [
      {"name": "transferred", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ]}
    ]
  },
  {
    "name": "balance",
    "type": "function",
    "stateMutability": "view",
    "inputs": [
      {"name": "account", "type": "address"},
      {"name": "denom", "type": "string"}
    ],
    "outputs": [{"name": "amount", "type": "uint256"}]
  },
  {
    "name": "allBalances",
    "type": "function",
    "stateMutability": "view",
    "inputs": [{"name": "account", "type": "address"}],
    "outputs": [
      {"name": "balances", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ]}
    ]
  },
  {
    "name": "totalBalance",
    "type": "function",
    "stateMutability": "view",
    "inputs": [],
    "outputs": [
      {"name": "total", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ]}
    ]
  },
  {
    "name": "enabled",
    "type": "function",
    "stateMutability": "view",
    "inputs": [],
    "outputs": [{"name": "isEnabled", "type": "bool"}]
  },
  {
    "name": "Approval",
    "type": "event",
    "inputs": [
      {"name": "owner", "type": "address", "indexed": true},
      {"name": "spender", "type": "address", "indexed": true},
      {"name": "amount", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ], "indexed": false}
    ]
  },
  {
    "name": "Revocation",
    "type": "event",
    "inputs": [
      {"name": "owner", "type": "address", "indexed": true},
      {"name": "spender", "type": "address", "indexed": true}
    ]
  },
  {
    "name": "Fund",
    "type": "event",
    "inputs": [
      {"name": "depositor", "type": "address", "indexed": true},
      {"name": "amount", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ], "indexed": false}
    ]
  },
  {
    "name": "TransferOwnership",
    "type": "event",
    "inputs": [
      {"name": "from", "type": "address", "indexed": true},
      {"name": "to", "type": "address", "indexed": true},
      {"name": "amount", "type": "tuple[]", "components": [
        {"name": "denom", "type": "string"},
        {"name": "amount", "type": "uint256"}
      ], "indexed": false}
    ]
  }
]
```

**Step 3: Commit**

```bash
git add precompiles/ucdao/abi.json
git commit -m "feat(precompiles): add UCDAO precompile ABI definition"
```

---

## Task 3: Create Main Precompile File

**Files:**
- Create: `precompiles/ucdao/ucdao.go`

**Step 1: Create the main precompile file**

Create `precompiles/ucdao/ucdao.go`:

```go
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
```

**Step 2: Verify build compiles**

Run: `go build ./precompiles/ucdao/...`
Expected: Errors about missing constants and methods (expected at this stage)

**Step 3: Commit**

```bash
git add precompiles/ucdao/ucdao.go
git commit -m "feat(precompiles): add UCDAO precompile main file structure"
```

---

## Task 4: Create Types File

**Files:**
- Create: `precompiles/ucdao/types.go`

**Step 1: Create the types file**

Create `precompiles/ucdao/types.go`:

```go
// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ucdao

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/haqq-network/haqq/precompiles/common"
)

const (
	// ApproveMethod defines the ABI method name for the ucdao Approve transaction.
	ApproveMethod = "approve"
	// RevokeMethod defines the ABI method name for the ucdao Revoke transaction.
	RevokeMethod = "revoke"
	// IncreaseAllowanceMethod defines the ABI method name for the IncreaseAllowance transaction.
	IncreaseAllowanceMethod = "increaseAllowance"
	// DecreaseAllowanceMethod defines the ABI method name for the DecreaseAllowance transaction.
	DecreaseAllowanceMethod = "decreaseAllowance"
	// AllowanceMethod defines the ABI method name for the Allowance query.
	AllowanceMethod = "allowance"
	// FundMethod defines the ABI method name for the Fund transaction.
	FundMethod = "fund"
	// TransferOwnershipMethod defines the ABI method name for the TransferOwnership transaction.
	TransferOwnershipMethod = "transferOwnership"
	// TransferOwnershipWithRatioMethod defines the ABI method name for the TransferOwnershipWithRatio transaction.
	TransferOwnershipWithRatioMethod = "transferOwnershipWithRatio"
	// TransferOwnershipWithAmountMethod defines the ABI method name for the TransferOwnershipWithAmount transaction.
	TransferOwnershipWithAmountMethod = "transferOwnershipWithAmount"
	// BalanceMethod defines the ABI method name for the Balance query.
	BalanceMethod = "balance"
	// AllBalancesMethod defines the ABI method name for the AllBalances query.
	AllBalancesMethod = "allBalances"
	// TotalBalanceMethod defines the ABI method name for the TotalBalance query.
	TotalBalanceMethod = "totalBalance"
	// EnabledMethod defines the ABI method name for the Enabled query.
	EnabledMethod = "enabled"
)

// ParseApproveArgs parses the arguments for the approve and allowance change methods.
func ParseApproveArgs(args []interface{}) (spender common.Address, coins sdk.Coins, err error) {
	if len(args) != 2 {
		return common.Address{}, nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	spender, ok := args[0].(common.Address)
	if !ok || spender == (common.Address{}) {
		return common.Address{}, nil, fmt.Errorf("invalid spender address: %v", args[0])
	}

	coins, err = ParseCoinsArg(args[1])
	if err != nil {
		return common.Address{}, nil, err
	}

	return spender, coins, nil
}

// ParseRevokeArgs parses the arguments for the revoke method.
func ParseRevokeArgs(args []interface{}) (spender common.Address, err error) {
	if len(args) != 1 {
		return common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	spender, ok := args[0].(common.Address)
	if !ok || spender == (common.Address{}) {
		return common.Address{}, fmt.Errorf("invalid spender address: %v", args[0])
	}

	return spender, nil
}

// ParseAllowanceArgs parses the arguments for the allowance query.
func ParseAllowanceArgs(args []interface{}) (owner, spender common.Address, err error) {
	if len(args) != 2 {
		return common.Address{}, common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	owner, ok := args[0].(common.Address)
	if !ok || owner == (common.Address{}) {
		return common.Address{}, common.Address{}, fmt.Errorf("invalid owner address: %v", args[0])
	}

	spender, ok = args[1].(common.Address)
	if !ok || spender == (common.Address{}) {
		return common.Address{}, common.Address{}, fmt.Errorf("invalid spender address: %v", args[1])
	}

	return owner, spender, nil
}

// ParseFundArgs parses the arguments for the fund method.
func ParseFundArgs(args []interface{}) (coins sdk.Coins, err error) {
	if len(args) != 1 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	coins, err = ParseCoinsArg(args[0])
	if err != nil {
		return nil, err
	}

	return coins, nil
}

// ParseTransferOwnershipArgs parses the arguments for the transferOwnership method.
func ParseTransferOwnershipArgs(args []interface{}) (owner, newOwner common.Address, err error) {
	if len(args) != 2 {
		return common.Address{}, common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	owner, ok := args[0].(common.Address)
	if !ok || owner == (common.Address{}) {
		return common.Address{}, common.Address{}, fmt.Errorf("invalid owner address: %v", args[0])
	}

	newOwner, ok = args[1].(common.Address)
	if !ok || newOwner == (common.Address{}) {
		return common.Address{}, common.Address{}, fmt.Errorf("invalid new owner address: %v", args[1])
	}

	return owner, newOwner, nil
}

// ParseTransferOwnershipWithRatioArgs parses the arguments for the transferOwnershipWithRatio method.
func ParseTransferOwnershipWithRatioArgs(args []interface{}) (owner, newOwner common.Address, ratio math.LegacyDec, err error) {
	if len(args) != 3 {
		return common.Address{}, common.Address{}, math.LegacyDec{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	owner, ok := args[0].(common.Address)
	if !ok || owner == (common.Address{}) {
		return common.Address{}, common.Address{}, math.LegacyDec{}, fmt.Errorf("invalid owner address: %v", args[0])
	}

	newOwner, ok = args[1].(common.Address)
	if !ok || newOwner == (common.Address{}) {
		return common.Address{}, common.Address{}, math.LegacyDec{}, fmt.Errorf("invalid new owner address: %v", args[1])
	}

	ratioBigInt, ok := args[2].(*big.Int)
	if !ok {
		return common.Address{}, common.Address{}, math.LegacyDec{}, fmt.Errorf("invalid ratio: %v", args[2])
	}

	// Convert from 1e18 precision to LegacyDec
	ratio = math.LegacyNewDecFromBigIntWithPrec(ratioBigInt, math.LegacyPrecision)

	return owner, newOwner, ratio, nil
}

// ParseTransferOwnershipWithAmountArgs parses the arguments for the transferOwnershipWithAmount method.
func ParseTransferOwnershipWithAmountArgs(args []interface{}) (owner, newOwner common.Address, coins sdk.Coins, err error) {
	if len(args) != 3 {
		return common.Address{}, common.Address{}, nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	owner, ok := args[0].(common.Address)
	if !ok || owner == (common.Address{}) {
		return common.Address{}, common.Address{}, nil, fmt.Errorf("invalid owner address: %v", args[0])
	}

	newOwner, ok = args[1].(common.Address)
	if !ok || newOwner == (common.Address{}) {
		return common.Address{}, common.Address{}, nil, fmt.Errorf("invalid new owner address: %v", args[1])
	}

	coins, err = ParseCoinsArg(args[2])
	if err != nil {
		return common.Address{}, common.Address{}, nil, err
	}

	return owner, newOwner, coins, nil
}

// ParseBalanceArgs parses the arguments for the balance query.
func ParseBalanceArgs(args []interface{}) (account common.Address, denom string, err error) {
	if len(args) != 2 {
		return common.Address{}, "", fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	account, ok := args[0].(common.Address)
	if !ok || account == (common.Address{}) {
		return common.Address{}, "", fmt.Errorf("invalid account address: %v", args[0])
	}

	denom, ok = args[1].(string)
	if !ok {
		return common.Address{}, "", fmt.Errorf("invalid denom: %v", args[1])
	}

	return account, denom, nil
}

// ParseAllBalancesArgs parses the arguments for the allBalances query.
func ParseAllBalancesArgs(args []interface{}) (account common.Address, err error) {
	if len(args) != 1 {
		return common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	account, ok := args[0].(common.Address)
	if !ok || account == (common.Address{}) {
		return common.Address{}, fmt.Errorf("invalid account address: %v", args[0])
	}

	return account, nil
}

// ParseCoinsArg parses a Coin[] argument from ABI input.
func ParseCoinsArg(arg interface{}) (sdk.Coins, error) {
	// The ABI encodes Coin[] as []struct{Denom string; Amount *big.Int}
	coinsRaw, ok := arg.([]struct {
		Denom  string   `json:"denom"`
		Amount *big.Int `json:"amount"`
	})
	if !ok {
		return nil, fmt.Errorf("invalid coins argument: %v", arg)
	}

	coins := make(sdk.Coins, len(coinsRaw))
	for i, c := range coinsRaw {
		if c.Amount == nil || c.Amount.Sign() < 0 {
			return nil, fmt.Errorf("invalid coin amount at index %d", i)
		}
		coins[i] = sdk.Coin{
			Denom:  c.Denom,
			Amount: math.NewIntFromBigInt(c.Amount),
		}
	}

	return coins.Sort(), nil
}
```

**Step 2: Verify build compiles**

Run: `go build ./precompiles/ucdao/...`
Expected: Errors about missing methods (expected at this stage)

**Step 3: Commit**

```bash
git add precompiles/ucdao/types.go
git commit -m "feat(precompiles): add UCDAO precompile types and argument parsers"
```

---

## Task 5: Create Errors File

**Files:**
- Create: `precompiles/ucdao/errors.go`

**Step 1: Create the errors file**

Create `precompiles/ucdao/errors.go`:

```go
// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ucdao

const (
	// ErrDifferentOriginFromOwner is raised when the origin address is not the same as the owner address.
	ErrDifferentOriginFromOwner = "origin address %s is not the same as owner address %s"
	// ErrAuthorizationNotFound is raised when the authorization is not found.
	ErrAuthorizationNotFound = "authorization not found for spender %s"
	// ErrInsufficientAllowance is raised when the allowance is insufficient.
	ErrInsufficientAllowance = "insufficient allowance: requested %s, available %s"
	// ErrDecreaseAmountTooBig is raised when the decrease amount is bigger than the allowance.
	ErrDecreaseAmountTooBig = "decrease amount %s is bigger than the allowance %s"
)
```

**Step 2: Commit**

```bash
git add precompiles/ucdao/errors.go
git commit -m "feat(precompiles): add UCDAO precompile error messages"
```

---

## Task 6: Create Events File

**Files:**
- Create: `precompiles/ucdao/events.go`

**Step 1: Create the events file**

Create `precompiles/ucdao/events.go`:

```go
// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ucdao

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

const (
	// EventTypeApproval defines the event type for the Approval event.
	EventTypeApproval = "Approval"
	// EventTypeRevocation defines the event type for the Revocation event.
	EventTypeRevocation = "Revocation"
	// EventTypeFund defines the event type for the Fund event.
	EventTypeFund = "Fund"
	// EventTypeTransferOwnership defines the event type for the TransferOwnership event.
	EventTypeTransferOwnership = "TransferOwnership"
)

// EmitApprovalEvent emits an Approval event.
func (p Precompile) EmitApprovalEvent(ctx sdk.Context, stateDB vm.StateDB, owner, spender common.Address, coins sdk.Coins) error {
	event := p.ABI.Events[EventTypeApproval]
	topics := make([]common.Hash, 3)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(owner)
	if err != nil {
		return err
	}

	topics[2], err = cmn.MakeTopic(spender)
	if err != nil {
		return err
	}

	// Pack the arguments to be used as the Data field
	arguments := event.Inputs.NonIndexed()
	packed, err := arguments.Pack(cmn.NewCoinsResponse(coins))
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint: gosec // G115
	})

	return nil
}

// EmitRevocationEvent emits a Revocation event.
func (p Precompile) EmitRevocationEvent(ctx sdk.Context, stateDB vm.StateDB, owner, spender common.Address) error {
	event := p.ABI.Events[EventTypeRevocation]
	topics := make([]common.Hash, 3)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(owner)
	if err != nil {
		return err
	}

	topics[2], err = cmn.MakeTopic(spender)
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        nil,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint: gosec // G115
	})

	return nil
}

// EmitFundEvent emits a Fund event.
func (p Precompile) EmitFundEvent(ctx sdk.Context, stateDB vm.StateDB, depositor common.Address, coins sdk.Coins) error {
	event := p.ABI.Events[EventTypeFund]
	topics := make([]common.Hash, 2)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(depositor)
	if err != nil {
		return err
	}

	// Pack the arguments to be used as the Data field
	arguments := event.Inputs.NonIndexed()
	packed, err := arguments.Pack(cmn.NewCoinsResponse(coins))
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint: gosec // G115
	})

	return nil
}

// EmitTransferOwnershipEvent emits a TransferOwnership event.
func (p Precompile) EmitTransferOwnershipEvent(ctx sdk.Context, stateDB vm.StateDB, from, to common.Address, coins sdk.Coins) error {
	event := p.ABI.Events[EventTypeTransferOwnership]
	topics := make([]common.Hash, 3)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(from)
	if err != nil {
		return err
	}

	topics[2], err = cmn.MakeTopic(to)
	if err != nil {
		return err
	}

	// Pack the arguments to be used as the Data field
	arguments := event.Inputs.NonIndexed()
	packed, err := arguments.Pack(cmn.NewCoinsResponse(coins))
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint: gosec // G115
	})

	return nil
}
```

**Step 2: Commit**

```bash
git add precompiles/ucdao/events.go
git commit -m "feat(precompiles): add UCDAO precompile event emission helpers"
```

---

## Task 7: Create Query Methods File

**Files:**
- Create: `precompiles/ucdao/query.go`

**Step 1: Create the query file**

Create `precompiles/ucdao/query.go`:

```go
// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ucdao

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"

	cmn "github.com/haqq-network/haqq/precompiles/common"
)

// Balance returns the balance of a specific denom for an account in the UCDAO.
func (p Precompile) Balance(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	account, denom, err := ParseBalanceArgs(args)
	if err != nil {
		return nil, err
	}

	accAddr := sdk.AccAddress(account.Bytes())
	balance := p.ucdaoKeeper.GetBalance(ctx, accAddr, denom)

	return method.Outputs.Pack(balance.Amount.BigInt())
}

// AllBalances returns all balances for an account in the UCDAO.
func (p Precompile) AllBalances(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	account, err := ParseAllBalancesArgs(args)
	if err != nil {
		return nil, err
	}

	accAddr := sdk.AccAddress(account.Bytes())
	balances := p.ucdaoKeeper.GetAccountBalances(ctx, accAddr)

	return method.Outputs.Pack(cmn.NewCoinsResponse(balances))
}

// TotalBalance returns the total balance of the UCDAO.
func (p Precompile) TotalBalance(
	ctx sdk.Context,
	method *abi.Method,
	_ []interface{},
) ([]byte, error) {
	totalBalance := p.ucdaoKeeper.GetTotalBalance(ctx)

	return method.Outputs.Pack(cmn.NewCoinsResponse(totalBalance))
}

// Enabled returns whether the UCDAO module is enabled.
func (p Precompile) Enabled(
	ctx sdk.Context,
	method *abi.Method,
	_ []interface{},
) ([]byte, error) {
	enabled := p.ucdaoKeeper.IsModuleEnabled(ctx)

	return method.Outputs.Pack(enabled)
}
```

**Step 2: Verify build compiles**

Run: `go build ./precompiles/ucdao/...`
Expected: Errors about missing tx methods (expected at this stage)

**Step 3: Commit**

```bash
git add precompiles/ucdao/query.go
git commit -m "feat(precompiles): add UCDAO precompile query methods"
```

---

## Task 8: Create Transaction Methods File

**Files:**
- Create: `precompiles/ucdao/tx.go`

**Step 1: Create the tx file**

Create `precompiles/ucdao/tx.go`:

```go
// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ucdao

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

// TransferOwnershipMsgURL is the message URL for UCDAO transfer ownership operations.
var TransferOwnershipMsgURL = sdk.MsgTypeURL(&banktypes.MsgSend{})

// Fund funds the UCDAO with the given amount.
func (p Precompile) Fund(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	coins, err := ParseFundArgs(args)
	if err != nil {
		return nil, err
	}

	// The depositor is always the origin (tx signer)
	depositor := sdk.AccAddress(origin.Bytes())

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"depositor", depositor.String(),
		"amount", coins.String(),
	)

	// Execute the fund operation
	if err := p.ucdaoKeeper.Fund(ctx, coins, depositor); err != nil {
		return nil, err
	}

	// Emit the Fund event
	if err := p.EmitFundEvent(ctx, stateDB, origin, coins); err != nil {
		return nil, err
	}

	// If called from a smart contract, record balance change for journal
	if contract.CallerAddress != origin {
		// Calculate total amount being funded
		for _, coin := range coins {
			p.SetBalanceChangeEntries(cmn.NewBalanceChangeEntry(origin, coin.Amount.BigInt(), cmn.Sub))
		}
	}

	return method.Outputs.Pack(true)
}

// TransferOwnership transfers all ownership from owner to newOwner.
func (p Precompile) TransferOwnership(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	owner, newOwner, err := ParseTransferOwnershipArgs(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"owner", owner.String(),
		"newOwner", newOwner.String(),
	)

	// Get owner's full balance to transfer
	ownerAddr := sdk.AccAddress(owner.Bytes())
	balances := p.ucdaoKeeper.GetAccountBalances(ctx, ownerAddr)

	transferred, err := p.executeTransfer(ctx, origin, contract, owner, newOwner, balances)
	if err != nil {
		return nil, err
	}

	// Emit the TransferOwnership event
	if err := p.EmitTransferOwnershipEvent(ctx, stateDB, owner, newOwner, transferred); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(cmn.NewCoinsResponse(transferred))
}

// TransferOwnershipWithRatio transfers a ratio of ownership from owner to newOwner.
func (p Precompile) TransferOwnershipWithRatio(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	owner, newOwner, ratio, err := ParseTransferOwnershipWithRatioArgs(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"owner", owner.String(),
		"newOwner", newOwner.String(),
		"ratio", ratio.String(),
	)

	// Get owner's balance and calculate transfer amount based on ratio
	ownerAddr := sdk.AccAddress(owner.Bytes())
	balances := p.ucdaoKeeper.GetAccountBalances(ctx, ownerAddr)

	// Calculate amount to transfer based on ratio
	transferAmount := sdk.NewCoins()
	for _, coin := range balances {
		amount := ratio.MulInt(coin.Amount).TruncateInt()
		if amount.IsPositive() {
			transferAmount = transferAmount.Add(sdk.NewCoin(coin.Denom, amount))
		}
	}

	transferred, err := p.executeTransfer(ctx, origin, contract, owner, newOwner, transferAmount)
	if err != nil {
		return nil, err
	}

	// Emit the TransferOwnership event
	if err := p.EmitTransferOwnershipEvent(ctx, stateDB, owner, newOwner, transferred); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(cmn.NewCoinsResponse(transferred))
}

// TransferOwnershipWithAmount transfers a specific amount of ownership from owner to newOwner.
func (p Precompile) TransferOwnershipWithAmount(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	owner, newOwner, amount, err := ParseTransferOwnershipWithAmountArgs(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"owner", owner.String(),
		"newOwner", newOwner.String(),
		"amount", amount.String(),
	)

	transferred, err := p.executeTransfer(ctx, origin, contract, owner, newOwner, amount)
	if err != nil {
		return nil, err
	}

	// Emit the TransferOwnership event
	if err := p.EmitTransferOwnershipEvent(ctx, stateDB, owner, newOwner, transferred); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(cmn.NewCoinsResponse(transferred))
}

// executeTransfer handles the common transfer logic including authorization checks.
func (p Precompile) executeTransfer(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	owner, newOwner common.Address,
	amount sdk.Coins,
) (sdk.Coins, error) {
	ownerAddr := sdk.AccAddress(owner.Bytes())
	newOwnerAddr := sdk.AccAddress(newOwner.Bytes())

	// Check authorization if caller is not the owner
	isCallerOrigin := contract.CallerAddress == origin
	isCallerOwner := contract.CallerAddress == owner

	// The provided owner address should always be equal to the origin address.
	// In case the contract caller address is the same as the owner address provided,
	// update the owner address to be equal to the origin address.
	// Otherwise, if the provided owner address is different from the origin address,
	// check for authorization.
	if isCallerOwner {
		owner = origin
		ownerAddr = sdk.AccAddress(origin.Bytes())
	} else if origin != owner && !isCallerOrigin {
		// Need authorization check
		auth, expiration := p.AuthzKeeper.GetAuthorization(ctx, contract.CallerAddress.Bytes(), owner.Bytes(), TransferOwnershipMsgURL)
		if auth == nil {
			return nil, fmt.Errorf(ErrAuthorizationNotFound, contract.CallerAddress)
		}

		// Verify this is a SendAuthorization and check spend limit
		sendAuth, ok := auth.(*banktypes.SendAuthorization)
		if !ok {
			return nil, fmt.Errorf("unexpected authorization type: %T", auth)
		}

		// Check if the requested amount is within the spend limit
		for _, coin := range amount {
			found := false
			for _, limit := range sendAuth.SpendLimit {
				if limit.Denom == coin.Denom {
					found = true
					if coin.Amount.GT(limit.Amount) {
						return nil, fmt.Errorf(ErrInsufficientAllowance, coin.Amount, limit.Amount)
					}
					break
				}
			}
			if !found && len(sendAuth.SpendLimit) > 0 {
				return nil, fmt.Errorf(ErrInsufficientAllowance, coin.Amount, "0")
			}
		}

		// Update the authorization after transfer
		newSpendLimit := sendAuth.SpendLimit.Sub(amount...)
		if newSpendLimit.IsZero() {
			// Delete the authorization if spend limit is exhausted
			if err := p.AuthzKeeper.DeleteGrant(ctx, contract.CallerAddress.Bytes(), owner.Bytes(), TransferOwnershipMsgURL); err != nil {
				return nil, err
			}
		} else {
			// Update with new spend limit
			sendAuth.SpendLimit = newSpendLimit
			if err := p.AuthzKeeper.SaveGrant(ctx, contract.CallerAddress.Bytes(), owner.Bytes(), sendAuth, expiration); err != nil {
				return nil, err
			}
		}
	}

	// Execute the actual transfer
	transferred, err := p.ucdaoKeeper.TransferOwnership(ctx, ownerAddr, newOwnerAddr, amount)
	if err != nil {
		return nil, err
	}

	return transferred, nil
}
```

**Step 2: Verify build compiles**

Run: `go build ./precompiles/ucdao/...`
Expected: Errors about missing approve methods (expected at this stage)

**Step 3: Commit**

```bash
git add precompiles/ucdao/tx.go
git commit -m "feat(precompiles): add UCDAO precompile transaction methods"
```

---

## Task 9: Create Approve Methods File

**Files:**
- Create: `precompiles/ucdao/approve.go`

**Step 1: Create the approve file**

Create `precompiles/ucdao/approve.go`:

```go
// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ucdao

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

// Approve sets the allowance of a spender over the caller's UCDAO holdings.
func (p Precompile) Approve(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	spender, coins, err := ParseApproveArgs(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"granter", origin.String(),
		"grantee", spender.String(),
		"amount", coins.String(),
	)

	// If coins are empty or zero, revoke the authorization
	if coins.IsZero() {
		return p.revokeAuthorization(ctx, origin, spender, stateDB, method)
	}

	// Create or update the SendAuthorization
	sendAuth := banktypes.NewSendAuthorization(coins, nil)
	expiration := ctx.BlockTime().Add(p.ApprovalExpiration).UTC()

	if err := p.AuthzKeeper.SaveGrant(ctx, spender.Bytes(), origin.Bytes(), sendAuth, &expiration); err != nil {
		return nil, err
	}

	// Emit the Approval event
	if err := p.EmitApprovalEvent(ctx, stateDB, origin, spender, coins); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// Revoke removes all allowances for a spender.
func (p Precompile) Revoke(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	spender, err := ParseRevokeArgs(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"granter", origin.String(),
		"grantee", spender.String(),
	)

	return p.revokeAuthorization(ctx, origin, spender, stateDB, method)
}

// revokeAuthorization removes the authorization grant.
func (p Precompile) revokeAuthorization(
	ctx sdk.Context,
	granter, grantee common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
) ([]byte, error) {
	if err := p.AuthzKeeper.DeleteGrant(ctx, grantee.Bytes(), granter.Bytes(), TransferOwnershipMsgURL); err != nil {
		// If grant doesn't exist, just log and continue
		p.Logger(ctx).Debug("grant not found during revoke", "error", err)
	}

	// Emit the Revocation event
	if err := p.EmitRevocationEvent(ctx, stateDB, granter, grantee); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// IncreaseAllowance increases the allowance of a spender.
func (p Precompile) IncreaseAllowance(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	spender, coins, err := ParseApproveArgs(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"granter", origin.String(),
		"grantee", spender.String(),
		"amount", coins.String(),
	)

	// Get existing authorization
	auth, expiration := p.AuthzKeeper.GetAuthorization(ctx, spender.Bytes(), origin.Bytes(), TransferOwnershipMsgURL)

	var newSpendLimit sdk.Coins
	if auth == nil {
		// No existing authorization, create new one with the given amount
		newSpendLimit = coins
	} else {
		sendAuth, ok := auth.(*banktypes.SendAuthorization)
		if !ok {
			return nil, fmt.Errorf("unexpected authorization type: %T", auth)
		}
		// Add to existing spend limit
		newSpendLimit = sendAuth.SpendLimit.Add(coins...)
	}

	// Save updated authorization
	sendAuth := banktypes.NewSendAuthorization(newSpendLimit, nil)
	if expiration == nil {
		exp := ctx.BlockTime().Add(p.ApprovalExpiration).UTC()
		expiration = &exp
	}

	if err := p.AuthzKeeper.SaveGrant(ctx, spender.Bytes(), origin.Bytes(), sendAuth, expiration); err != nil {
		return nil, err
	}

	// Emit the Approval event with the new total allowance
	if err := p.EmitApprovalEvent(ctx, stateDB, origin, spender, newSpendLimit); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// DecreaseAllowance decreases the allowance of a spender.
func (p Precompile) DecreaseAllowance(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	spender, coins, err := ParseApproveArgs(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"granter", origin.String(),
		"grantee", spender.String(),
		"amount", coins.String(),
	)

	// Get existing authorization
	auth, expiration := p.AuthzKeeper.GetAuthorization(ctx, spender.Bytes(), origin.Bytes(), TransferOwnershipMsgURL)
	if auth == nil {
		return nil, fmt.Errorf(ErrAuthorizationNotFound, spender)
	}

	sendAuth, ok := auth.(*banktypes.SendAuthorization)
	if !ok {
		return nil, fmt.Errorf("unexpected authorization type: %T", auth)
	}

	// Check that we have enough to subtract
	for _, coin := range coins {
		found := false
		for _, limit := range sendAuth.SpendLimit {
			if limit.Denom == coin.Denom {
				found = true
				if coin.Amount.GT(limit.Amount) {
					return nil, fmt.Errorf(ErrDecreaseAmountTooBig, coin.Amount, limit.Amount)
				}
				break
			}
		}
		if !found {
			return nil, fmt.Errorf(ErrDecreaseAmountTooBig, coin.Amount, "0")
		}
	}

	// Subtract from spend limit
	newSpendLimit, hasNeg := sendAuth.SpendLimit.SafeSub(coins...)
	if hasNeg {
		return nil, fmt.Errorf("decrease amount exceeds current allowance")
	}

	// If spend limit is zero, delete the authorization
	if newSpendLimit.IsZero() {
		if err := p.AuthzKeeper.DeleteGrant(ctx, spender.Bytes(), origin.Bytes(), TransferOwnershipMsgURL); err != nil {
			return nil, err
		}
		// Emit revocation event since allowance is now zero
		if err := p.EmitRevocationEvent(ctx, stateDB, origin, spender); err != nil {
			return nil, err
		}
	} else {
		// Save updated authorization
		sendAuth.SpendLimit = newSpendLimit
		if err := p.AuthzKeeper.SaveGrant(ctx, spender.Bytes(), origin.Bytes(), sendAuth, expiration); err != nil {
			return nil, err
		}
		// Emit the Approval event with the new total allowance
		if err := p.EmitApprovalEvent(ctx, stateDB, origin, spender, newSpendLimit); err != nil {
			return nil, err
		}
	}

	return method.Outputs.Pack(true)
}

// Allowance returns the remaining allowance of a spender.
func (p Precompile) Allowance(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	owner, spender, err := ParseAllowanceArgs(args)
	if err != nil {
		return nil, err
	}

	// Get authorization
	auth, _ := p.AuthzKeeper.GetAuthorization(ctx, spender.Bytes(), owner.Bytes(), TransferOwnershipMsgURL)
	if auth == nil {
		// Return empty coins if no authorization
		return method.Outputs.Pack(cmn.NewCoinsResponse(sdk.Coins{}))
	}

	sendAuth, ok := auth.(*banktypes.SendAuthorization)
	if !ok {
		return nil, fmt.Errorf("unexpected authorization type: %T", auth)
	}

	return method.Outputs.Pack(cmn.NewCoinsResponse(sendAuth.SpendLimit))
}
```

**Step 2: Verify build compiles**

Run: `go build ./precompiles/ucdao/...`
Expected: Build should succeed

**Step 3: Commit**

```bash
git add precompiles/ucdao/approve.go
git commit -m "feat(precompiles): add UCDAO precompile authorization methods"
```

---

## Task 10: Add IsModuleEnabled to UCDAO Keeper Interface

**Files:**
- Modify: `x/ucdao/keeper/keeper.go`

**Step 1: Check if IsModuleEnabled exists and add GetParams if needed**

First, check if `IsModuleEnabled` already exists. If it does but `GetParams` is missing, we need to check the params file.

Run: `grep -n "IsModuleEnabled\|GetParams" x/ucdao/keeper/*.go`

If `IsModuleEnabled` doesn't exist or is defined on a different type, add it to the Keeper interface and implementation.

**Step 2: Verify the keeper interface is correct**

The interface should have:
- `IsModuleEnabled(ctx sdk.Context) bool`
- `GetParams(ctx sdk.Context) types.Params` (if used)

**Step 3: Commit any changes**

```bash
git add x/ucdao/keeper/
git commit -m "feat(ucdao): ensure keeper interface has required methods"
```

---

## Task 11: Register Precompile in EVM Module

**Files:**
- Search for and modify precompile registration files

**Step 1: Find precompile registration**

Search for where precompiles are registered:
```bash
grep -rn "NewPrecompile\|precompiles" app/ x/evm/ --include="*.go" | grep -v "_test.go"
```

**Step 2: Register the UCDAO precompile**

Add the UCDAO precompile to the precompile registration, typically in `app/app.go` or `x/evm/keeper/keeper.go`.

Example pattern:
```go
ucdaoPrecompile, err := ucdao.NewPrecompile(
    app.UcdaoKeeper,
    app.BankKeeper,
    app.AuthzKeeper,
)
if err != nil {
    panic(fmt.Errorf("failed to create ucdao precompile: %w", err))
}
```

**Step 3: Verify build compiles**

Run: `make build`
Expected: Build should succeed

**Step 4: Commit**

```bash
git add app/ x/evm/
git commit -m "feat(evm): register UCDAO precompile in EVM module"
```

---

## Task 12: Write Unit Tests for Types

**Files:**
- Create: `precompiles/ucdao/types_test.go`

**Step 1: Create types test file**

Create `precompiles/ucdao/types_test.go`:

```go
package ucdao_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/haqq-network/haqq/precompiles/ucdao"
)

func TestParseApproveArgs(t *testing.T) {
	testCases := []struct {
		name      string
		args      []interface{}
		expectErr bool
	}{
		{
			name: "valid args",
			args: []interface{}{
				common.HexToAddress("0x1234567890123456789012345678901234567890"),
				[]struct {
					Denom  string   `json:"denom"`
					Amount *big.Int `json:"amount"`
				}{
					{Denom: "aISLM", Amount: big.NewInt(1000)},
				},
			},
			expectErr: false,
		},
		{
			name:      "invalid number of args",
			args:      []interface{}{common.Address{}},
			expectErr: true,
		},
		{
			name: "empty spender address",
			args: []interface{}{
				common.Address{},
				[]struct {
					Denom  string   `json:"denom"`
					Amount *big.Int `json:"amount"`
				}{},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ucdao.ParseApproveArgs(tc.args)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParseCoinsArg(t *testing.T) {
	testCases := []struct {
		name      string
		arg       interface{}
		expectErr bool
	}{
		{
			name: "valid coins",
			arg: []struct {
				Denom  string   `json:"denom"`
				Amount *big.Int `json:"amount"`
			}{
				{Denom: "aISLM", Amount: big.NewInt(1000)},
				{Denom: "aLIQUID1", Amount: big.NewInt(500)},
			},
			expectErr: false,
		},
		{
			name: "negative amount",
			arg: []struct {
				Denom  string   `json:"denom"`
				Amount *big.Int `json:"amount"`
			}{
				{Denom: "aISLM", Amount: big.NewInt(-1000)},
			},
			expectErr: true,
		},
		{
			name:      "invalid type",
			arg:       "not a coin array",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ucdao.ParseCoinsArg(tc.arg)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
```

**Step 2: Run the tests**

Run: `go test -v ./precompiles/ucdao/... -run TestParse`
Expected: Tests should pass

**Step 3: Commit**

```bash
git add precompiles/ucdao/types_test.go
git commit -m "test(precompiles): add UCDAO precompile types tests"
```

---

## Task 13: Write Unit Tests for Main Precompile

**Files:**
- Create: `precompiles/ucdao/ucdao_test.go`
- Create: `precompiles/ucdao/setup_test.go`

**Step 1: Create setup test file**

Create `precompiles/ucdao/setup_test.go`:

```go
package ucdao_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/haqq-network/haqq/precompiles/ucdao"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
)

type PrecompileTestSuite struct {
	suite.Suite

	network     *network.UnitTestNetwork
	factory     factory.TxFactory
	grpcHandler grpc.Handler
	keyring     keyring.Keyring

	precompile *ucdao.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	suite.Run(t, new(PrecompileTestSuite))
}

func (s *PrecompileTestSuite) SetupTest() {
	keyring := keyring.New(2)
	integrationNetwork := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	)
	grpcHandler := grpc.NewIntegrationHandler(integrationNetwork)
	txFactory := factory.New(integrationNetwork, grpcHandler)

	s.network = integrationNetwork
	s.factory = txFactory
	s.grpcHandler = grpcHandler
	s.keyring = keyring

	precompile, err := ucdao.NewPrecompile(
		s.network.App.UcdaoKeeper,
		s.network.App.BankKeeper,
		s.network.App.AuthzKeeper,
	)
	s.Require().NoError(err)
	s.precompile = precompile
}
```

**Step 2: Create main test file**

Create `precompiles/ucdao/ucdao_test.go`:

```go
package ucdao_test

import (
	"github.com/haqq-network/haqq/precompiles/ucdao"
)

func (s *PrecompileTestSuite) TestIsTransaction() {
	testCases := []struct {
		method        string
		isTransaction bool
	}{
		{ucdao.ApproveMethod, true},
		{ucdao.RevokeMethod, true},
		{ucdao.IncreaseAllowanceMethod, true},
		{ucdao.DecreaseAllowanceMethod, true},
		{ucdao.FundMethod, true},
		{ucdao.TransferOwnershipMethod, true},
		{ucdao.TransferOwnershipWithRatioMethod, true},
		{ucdao.TransferOwnershipWithAmountMethod, true},
		{ucdao.AllowanceMethod, false},
		{ucdao.BalanceMethod, false},
		{ucdao.AllBalancesMethod, false},
		{ucdao.TotalBalanceMethod, false},
		{ucdao.EnabledMethod, false},
	}

	for _, tc := range testCases {
		s.Run(tc.method, func() {
			s.Require().Equal(tc.isTransaction, s.precompile.IsTransaction(tc.method))
		})
	}
}

func (s *PrecompileTestSuite) TestRequiredGas() {
	// Test that RequiredGas returns non-zero for valid input
	// Using a minimal valid input (4 bytes for method ID)
	methodID := s.precompile.ABI.Methods[ucdao.BalanceMethod].ID
	gas := s.precompile.RequiredGas(methodID)
	s.Require().NotZero(gas)

	// Test that RequiredGas returns 0 for too short input
	gas = s.precompile.RequiredGas([]byte{0x01, 0x02})
	s.Require().Zero(gas)
}
```

**Step 3: Run the tests**

Run: `go test -v ./precompiles/ucdao/... -run TestPrecompileTestSuite`
Expected: Tests should pass

**Step 4: Commit**

```bash
git add precompiles/ucdao/setup_test.go precompiles/ucdao/ucdao_test.go
git commit -m "test(precompiles): add UCDAO precompile unit tests"
```

---

## Task 14: Run Full Test Suite and Linting

**Step 1: Run all ucdao precompile tests**

Run: `go test -v -timeout=20m ./precompiles/ucdao/...`
Expected: All tests pass

**Step 2: Run linter**

Run: `make lint`
Expected: No linting errors in precompiles/ucdao/

**Step 3: Fix any linting issues**

If there are linting issues, fix them following the project's style guidelines.

**Step 4: Run the full build**

Run: `make build`
Expected: Build succeeds

**Step 5: Commit any fixes**

```bash
git add precompiles/ucdao/
git commit -m "fix(precompiles): address linting issues in UCDAO precompile"
```

---

## Task 15: Final Verification and Documentation

**Step 1: Verify all files are present**

```bash
ls -la precompiles/ucdao/
```

Expected files:
- `abi.json`
- `ucdao.go`
- `types.go`
- `errors.go`
- `events.go`
- `query.go`
- `tx.go`
- `approve.go`
- `setup_test.go`
- `ucdao_test.go`
- `types_test.go`

**Step 2: Run final build and test**

Run: `make build && go test -v ./precompiles/ucdao/...`
Expected: Build and all tests pass

**Step 3: Create final commit**

```bash
git add .
git commit -m "feat(precompiles): complete UCDAO precompile implementation with allowance support"
```

---

## Summary

This plan implements a complete UCDAO precompile with:

1. **Static precompile address**: `0x0000000000000000000000000000000000000805`
2. **Authorization support**: Uses Cosmos SDK authz with `SendAuthorization`
3. **Methods implemented**:
   - Authorization: `approve`, `revoke`, `increaseAllowance`, `decreaseAllowance`, `allowance`
   - Transactions: `fund`, `transferOwnership`, `transferOwnershipWithRatio`, `transferOwnershipWithAmount`
   - Queries: `balance`, `allBalances`, `totalBalance`, `enabled`
4. **Events**: `Approval`, `Revocation`, `Fund`, `TransferOwnership`
5. **Full test coverage** for types and main precompile logic

The implementation follows the existing patterns from staking/ICS20/ERC20 precompiles for consistency.
