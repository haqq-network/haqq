package ucdao

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

func (p Precompile) Approve(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	grantee, coin, typeURLs, err := authorization.CheckApprovalArgs(args, utils.BaseDenom)
	if err != nil {
		return nil, err
	}

	for _, typeURL := range typeURLs {
		switch typeURL {
		case ConvertToHaqqMsgURL, TransferOwnershipMsgURL:
			if err = p.grantOrDeleteAuthz(ctx, grantee, origin, coin, typeURL); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf(cmn.ErrInvalidMsgType, "ucdao", typeURL)
		}
	}

	if err := p.EmitApprovalEvent(ctx, stateDB, grantee, origin, coin, typeURLs); err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) Revoke(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	grantee, typeURLs, err := authorization.CheckRevokeArgs(args)
	if err != nil {
		return nil, err
	}

	for _, typeURL := range typeURLs {
		switch typeURL {
		case ConvertToHaqqMsgURL, TransferOwnershipMsgURL:
			if err = p.AuthzKeeper.DeleteGrant(ctx, grantee.Bytes(), origin.Bytes(), typeURL); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf(cmn.ErrInvalidMsgType, "ucdao", typeURL)
		}
	}

	if err = authorization.EmitRevocationEvent(cmn.EmitEventArgs{
		Ctx:            ctx,
		StateDB:        stateDB,
		ContractAddr:   p.Address(),
		ContractEvents: p.ABI.Events,
		EventData: authorization.EventRevocation{
			Granter:  origin,
			Grantee:  grantee,
			TypeUrls: typeURLs,
		},
	}); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// IncreaseAllowance implements the ethiq increase allowance transactions.
func (p Precompile) IncreaseAllowance(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	grantee, coin, typeURLs, err := authorization.CheckApprovalArgs(args, utils.BaseDenom)
	if err != nil {
		return nil, err
	}

	for _, typeURL := range typeURLs {
		switch typeURL {
		case ConvertToHaqqMsgURL, TransferOwnershipMsgURL:
			if err = p.increaseAllowance(ctx, grantee, origin, coin, typeURL); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf(cmn.ErrInvalidMsgType, "ethiq", typeURL)
		}
	}

	if err := p.EmitAllowanceChangeEvent(ctx, stateDB, grantee, origin, typeURLs); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// DecreaseAllowance implements the ethiq decrease allowance transactions.
func (p Precompile) DecreaseAllowance(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	grantee, coin, typeURLs, err := authorization.CheckApprovalArgs(args, utils.BaseDenom)
	if err != nil {
		return nil, err
	}

	for _, typeURL := range typeURLs {
		switch typeURL {
		case ConvertToHaqqMsgURL, TransferOwnershipMsgURL:
			authzGrant, expiration, err := authorization.CheckAuthzExists(ctx, p.AuthzKeeper, grantee, origin, typeURL)
			if err != nil {
				return nil, err
			}

			if err = p.decreaseAllowance(ctx, grantee, origin, coin, authzGrant, expiration); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf(cmn.ErrInvalidMsgType, "ethiq", typeURL)
		}
	}

	if err := p.EmitAllowanceChangeEvent(ctx, stateDB, grantee, origin, typeURLs); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// grantOrDeleteAuthz grants ucdao method authorization to the precompiled contract for a spender.
// If the amount is zero, it deletes the authorization if it exists.
func (p Precompile) grantOrDeleteAuthz(
	ctx sdk.Context,
	grantee, granter common.Address,
	coin *sdk.Coin,
	msgType string,
) error {
	// Case 1: coin is nil -> set authorization with no limit
	if coin == nil || coin.IsNil() {
		p.Logger(ctx).Debug(
			"setting authorization without limit",
			"grantee", grantee.String(),
			"granter", granter.String(),
		)
		return p.createAuthz(ctx, grantee, granter, coin, msgType)
	}

	// Case 2: coin amount is zero or negative -> delete the authorization
	if !coin.Amount.IsPositive() {
		p.Logger(ctx).Debug(
			"deleting authorization",
			"grantee", grantee.String(),
			"granter", granter.String(),
		)

		switch msgType {
		case ConvertToHaqqMsgURL, TransferOwnershipMsgURL:
			return p.AuthzKeeper.DeleteGrant(ctx, grantee.Bytes(), granter.Bytes(), msgType)
		default:
			return fmt.Errorf(cmn.ErrInvalidMsgType, "ucdao", msgType)
		}
	}

	// Case 3: coin amount is non zero -> and not coin is not nil set with custom amount
	return p.createAuthz(ctx, grantee, granter, coin, msgType)
}

// createAuthz creates a ucdao authorization for a spender.
func (p Precompile) createAuthz(
	ctx sdk.Context,
	grantee, granter common.Address,
	coin *sdk.Coin,
	msgType string,
) error {
	expiration := ctx.BlockTime().Add(p.ApprovalExpiration).UTC()

	switch msgType {
	case ConvertToHaqqMsgURL:
		convAuthz, err := ucdaotypes.NewConvertToHaqqAuthorization(coin)
		if err != nil {
			return err
		}
		if err := convAuthz.ValidateBasic(); err != nil {
			return err
		}
		return p.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), convAuthz, &expiration)
	case TransferOwnershipMsgURL:
		transferAuthz, err := ucdaotypes.NewTransferOwnershipAuthorization(coin)
		if err != nil {
			return err
		}
		if err := transferAuthz.ValidateBasic(); err != nil {
			return err
		}
		return p.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), transferAuthz, &expiration)
	default:
		return fmt.Errorf(cmn.ErrInvalidMsgType, "ucdao", msgType)
	}
}

// increaseAllowance increases the allowance of spender over the caller’s tokens by the amount.
func (p Precompile) increaseAllowance(
	ctx sdk.Context,
	grantee, granter common.Address,
	coin *sdk.Coin,
	msgURL string,
) error {
	// Check if the authorization exists for the given spender
	existingAuthz, expiration, err := authorization.CheckAuthzExists(ctx, p.AuthzKeeper, grantee, granter, msgURL)
	if err != nil {
		return err
	}

	switch msgURL {
	case ConvertToHaqqMsgURL:
		convAuthz, ok := existingAuthz.(*ucdaotypes.ConvertToHaqqAuthorization)
		if !ok {
			return errorsmod.Wrapf(authz.ErrUnknownAuthorizationType, "expected: *ucdaotypes.ConvertToHaqqAuthorization, received: %T", existingAuthz)
		}

		if convAuthz.SpendLimit == nil {
			p.Logger(ctx).Debug("increaseAllowance called with no limit (convAuthz.SpendLimit == nil): no-op")
			return nil
		}

		convAuthz.SpendLimit.Amount = convAuthz.SpendLimit.Amount.Add(coin.Amount)
		return p.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), convAuthz, expiration)
	case TransferOwnershipMsgURL:
		transferAuthz, ok := existingAuthz.(*ucdaotypes.TransferOwnershipAuthorization)
		if !ok {
			return errorsmod.Wrapf(authz.ErrUnknownAuthorizationType, "expected: *ucdaotypes.TransferOwnershipAuthorization, received: %T", existingAuthz)
		}

		if transferAuthz.SpendLimit == nil {
			p.Logger(ctx).Debug("increaseAllowance called with no limit (transferAuthz.SpendLimit == nil): no-op")
			return nil
		}

		transferAuthz.SpendLimit.Amount = transferAuthz.SpendLimit.Amount.Add(coin.Amount)
		return p.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), transferAuthz, expiration)
	default:
		return errorsmod.Wrapf(authz.ErrUnknownAuthorizationType, "expected: *ucdaotypes.ConvertToHaqqAuthorization or *ucdaotypes.TransferOwnershipAuthorization, received: %T", existingAuthz)
	}
}

// decreaseAllowance decreases the allowance of spender over the caller’s tokens by the amount.
func (p Precompile) decreaseAllowance(
	ctx sdk.Context,
	grantee, granter common.Address,
	coin *sdk.Coin,
	existingAuthz authz.Authorization,
	expiration *time.Time,
) error {
	switch existingAuthz.MsgTypeURL() {
	case ConvertToHaqqMsgURL:
		convAuthz, ok := existingAuthz.(*ucdaotypes.ConvertToHaqqAuthorization)
		if !ok {
			return errorsmod.Wrapf(authz.ErrUnknownAuthorizationType, "expected: *ucdaotypes.ConvertToHaqqAuthorization, received: %T", existingAuthz)
		}

		if convAuthz.SpendLimit == nil {
			p.Logger(ctx).Debug("decreaseAllowance called with no limit (convAuthz.SpendLimit == nil): no-op")
			return nil
		}

		// If the authorization limit is less than the substation amount, return error
		if convAuthz.SpendLimit.Amount.LT(coin.Amount) {
			return fmt.Errorf(ErrDecreaseAmountTooBig, coin.Amount, convAuthz.SpendLimit.Amount)
		}

		// If amount is less than or equal to the Authorization amount, subtract the amount from the limit
		if coin.Amount.LTE(convAuthz.SpendLimit.Amount) {
			convAuthz.SpendLimit.Amount = convAuthz.SpendLimit.Amount.Sub(coin.Amount)
		}

		return p.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), convAuthz, expiration)
	case TransferOwnershipMsgURL:
		transferAuthz, ok := existingAuthz.(*ucdaotypes.TransferOwnershipAuthorization)
		if !ok {
			return errorsmod.Wrapf(authz.ErrUnknownAuthorizationType, "expected: *ucdaotypes.TransferOwnershipAuthorization, received: %T", existingAuthz)
		}

		if transferAuthz.SpendLimit == nil {
			p.Logger(ctx).Debug("decreaseAllowance called with no limit (transferAuthz.SpendLimit == nil): no-op")
			return nil
		}

		// If the authorization limit is less than the substation amount, return error
		if transferAuthz.SpendLimit.Amount.LT(coin.Amount) {
			return fmt.Errorf(ErrDecreaseAmountTooBig, coin.Amount, transferAuthz.SpendLimit.Amount)
		}

		// If amount is less than or equal to the Authorization amount, subtract the amount from the limit
		if coin.Amount.LTE(transferAuthz.SpendLimit.Amount) {
			transferAuthz.SpendLimit.Amount = transferAuthz.SpendLimit.Amount.Sub(coin.Amount)
		}

		return p.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), transferAuthz, expiration)
	default:
		return errorsmod.Wrapf(authz.ErrUnknownAuthorizationType, "expected: *ucdaotypes.ConvertToHaqqAuthorization or *ucdaotypes.TransferOwnershipAuthorization, received: %T", existingAuthz)
	}
}
