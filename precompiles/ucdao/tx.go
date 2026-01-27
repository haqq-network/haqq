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
