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

	isCallerOwner := contract.CallerAddress == owner
	isCallerOrigin := contract.CallerAddress == origin
	needsAuthorization := origin != owner && !isCallerOrigin

	// If the contract caller address equals the owner address provided,
	// update the owner address to be the origin address.
	if isCallerOwner {
		ownerAddr = sdk.AccAddress(origin.Bytes())
	} else if needsAuthorization {
		if err := p.checkAndUpdateAuthorization(ctx, contract.CallerAddress, owner, amount); err != nil {
			return nil, err
		}
	}

	return p.ucdaoKeeper.TransferOwnership(ctx, ownerAddr, newOwnerAddr, amount)
}

// checkAndUpdateAuthorization verifies authorization and updates the spend limit.
func (p Precompile) checkAndUpdateAuthorization(
	ctx sdk.Context,
	caller, owner common.Address,
	amount sdk.Coins,
) error {
	auth, expiration := p.AuthzKeeper.GetAuthorization(ctx, caller.Bytes(), owner.Bytes(), TransferOwnershipMsgURL)
	if auth == nil {
		return fmt.Errorf(ErrAuthorizationNotFound, caller)
	}

	sendAuth, ok := auth.(*banktypes.SendAuthorization)
	if !ok {
		return fmt.Errorf("unexpected authorization type: %T", auth)
	}

	if err := validateSpendLimit(sendAuth.SpendLimit, amount); err != nil {
		return err
	}

	newSpendLimit := sendAuth.SpendLimit.Sub(amount...)
	if newSpendLimit.IsZero() {
		return p.AuthzKeeper.DeleteGrant(ctx, caller.Bytes(), owner.Bytes(), TransferOwnershipMsgURL)
	}

	sendAuth.SpendLimit = newSpendLimit
	return p.AuthzKeeper.SaveGrant(ctx, caller.Bytes(), owner.Bytes(), sendAuth, expiration)
}

// validateSpendLimit checks if the requested amount is within the spend limit.
func validateSpendLimit(spendLimit, amount sdk.Coins) error {
	for _, coin := range amount {
		found, limitCoin := spendLimit.Find(coin.Denom)
		if !found && len(spendLimit) > 0 {
			return fmt.Errorf(ErrInsufficientAllowance, coin.Amount, "0")
		}
		if found && coin.Amount.GT(limitCoin.Amount) {
			return fmt.Errorf(ErrInsufficientAllowance, coin.Amount, limitCoin.Amount)
		}
	}
	return nil
}
