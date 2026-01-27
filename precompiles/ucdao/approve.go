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

	auth, expiration := p.AuthzKeeper.GetAuthorization(ctx, spender.Bytes(), origin.Bytes(), TransferOwnershipMsgURL)

	var newSpendLimit sdk.Coins
	if auth == nil {
		newSpendLimit = coins
	} else {
		sendAuth, err := asSendAuthorization(auth)
		if err != nil {
			return nil, err
		}
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

	auth, expiration := p.AuthzKeeper.GetAuthorization(ctx, spender.Bytes(), origin.Bytes(), TransferOwnershipMsgURL)
	if auth == nil {
		return nil, fmt.Errorf(ErrAuthorizationNotFound, spender)
	}

	sendAuth, err := asSendAuthorization(auth)
	if err != nil {
		return nil, err
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

	auth, _ := p.AuthzKeeper.GetAuthorization(ctx, spender.Bytes(), owner.Bytes(), TransferOwnershipMsgURL)
	if auth == nil {
		return method.Outputs.Pack(cmn.NewCoinsResponse(sdk.Coins{}))
	}

	sendAuth, err := asSendAuthorization(auth)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(cmn.NewCoinsResponse(sendAuth.SpendLimit))
}

// asSendAuthorization casts an authorization to SendAuthorization or returns an error.
func asSendAuthorization(auth interface{}) (*banktypes.SendAuthorization, error) {
	sendAuth, ok := auth.(*banktypes.SendAuthorization)
	if !ok {
		return nil, fmt.Errorf("unexpected authorization type: %T", auth)
	}
	return sendAuth, nil
}
