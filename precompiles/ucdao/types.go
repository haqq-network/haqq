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
func ParseFundArgs(args []interface{}) (sdk.Coins, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	return ParseCoinsArg(args[0])
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
