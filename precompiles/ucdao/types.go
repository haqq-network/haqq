package ucdao

import (
	"fmt"
	"math/big"

	errorsmod "cosmossdk.io/errors"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// NewConvertToHaqqMsg builds MsgConvertToHaqq from ABI arguments.
func NewConvertToHaqqMsg(args []interface{}) (*ucdaotypes.MsgConvertToHaqq, common.Address, common.Address, error) {
	if len(args) != 3 {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	sender, ok := args[0].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf("invalid sender address: %v", args[0])
	}

	receiver, ok := args[1].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf("invalid receiver address: %v", args[1])
	}

	amount, ok := args[2].(*big.Int)
	if !ok || amount == nil {
		return nil, common.Address{}, common.Address{}, fmt.Errorf("invalid amount: %v", args[2])
	}

	msg := ucdaotypes.NewMsgConvertToHaqq(
		sdk.AccAddress(sender.Bytes()),
		sdk.AccAddress(receiver.Bytes()),
		sdkmath.NewIntFromBigInt(amount),
	)

	// The precompile calls the msg server directly, so baseapp's
	// validateBasicTxMsgs never runs on this path. Validate here or not at all.
	if err := msg.ValidateBasic(); err != nil {
		return nil, common.Address{}, common.Address{}, err
	}

	return msg, sender, receiver, nil
}

// NewTransferOwnershipMsg builds MsgTransferOwnership from ABI arguments.
func NewTransferOwnershipMsg(args []interface{}) (*ucdaotypes.MsgTransferOwnership, common.Address, common.Address, error) {
	if len(args) != 2 {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	owner, ok := args[0].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf("invalid owner address: %v", args[0])
	}

	newOwner, ok := args[1].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf("invalid new owner address: %v", args[1])
	}

	msg := ucdaotypes.NewMsgTransferOwnership(
		sdk.AccAddress(owner.Bytes()),
		sdk.AccAddress(newOwner.Bytes()),
	)

	// The precompile calls the msg server directly, so baseapp's
	// validateBasicTxMsgs never runs on this path. Validate here or not at all.
	if err := msg.ValidateBasic(); err != nil {
		return nil, common.Address{}, common.Address{}, err
	}

	return msg, owner, newOwner, nil
}

// NewTransferOwnershipWithAmountMsg builds MsgTransferOwnershipWithAmount from ABI arguments.
func NewTransferOwnershipWithAmountMsg(args []interface{}) (*ucdaotypes.MsgTransferOwnershipWithAmount, common.Address, common.Address, error) {
	if len(args) != 4 {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 4, len(args))
	}

	owner, ok := args[0].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf("invalid owner address: %v", args[0])
	}

	newOwner, ok := args[1].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf("invalid new owner address: %v", args[1])
	}

	denomsIface, ok := args[2].([]string)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf("invalid denoms type: %T", args[2])
	}

	amountsIface, ok := args[3].([]*big.Int)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf("invalid amounts type: %T", args[3])
	}

	if len(denomsIface) != len(amountsIface) {
		return nil, common.Address{}, common.Address{}, fmt.Errorf("denoms and amounts length mismatch")
	}

	// Both the denom strings and the amounts come straight from calldata.
	// sdk.NewCoin panics on a malformed denom or a negative amount, and
	// cmn.HandleGasError only recovers ErrorOutOfGas - anything else is re-raised
	// and unwinds out of the precompile instead of reverting the call. Build each
	// coin by hand and validate it so bad input is an error, not a panic.
	coins := sdk.Coins{}
	seen := make(map[string]struct{}, len(denomsIface))
	for i := range denomsIface {
		if amountsIface[i] == nil {
			return nil, common.Address{}, common.Address{}, fmt.Errorf("nil amount at index %d", i)
		}
		if amountsIface[i].BitLen() > sdkmath.MaxBitLen {
			return nil, common.Address{}, common.Address{}, fmt.Errorf(
				"amount at index %d does not fit in %d bits", i, sdkmath.MaxBitLen,
			)
		}
		// Duplicate denoms would be summed by Coins.Add, and that sum can overflow
		// sdkmath.Int and panic. The intent is ambiguous anyway, so reject them.
		if _, dup := seen[denomsIface[i]]; dup {
			return nil, common.Address{}, common.Address{}, fmt.Errorf("duplicate denom %q at index %d", denomsIface[i], i)
		}
		seen[denomsIface[i]] = struct{}{}

		coin := sdk.Coin{Denom: denomsIface[i], Amount: sdkmath.NewIntFromBigInt(amountsIface[i])}
		if err := coin.Validate(); err != nil {
			return nil, common.Address{}, common.Address{}, errorsmod.Wrapf(err, "invalid coin at index %d", i)
		}
		coins = coins.Add(coin)
	}

	msg := ucdaotypes.NewMsgTransferOwnershipWithAmount(
		sdk.AccAddress(owner.Bytes()),
		sdk.AccAddress(newOwner.Bytes()),
		coins,
	)

	// The precompile calls the msg server directly, so baseapp's
	// validateBasicTxMsgs never runs on this path. Validate here or not at all.
	if err := msg.ValidateBasic(); err != nil {
		return nil, common.Address{}, common.Address{}, err
	}

	return msg, owner, newOwner, nil
}

// CheckAndAcceptAuthorizationIfNeeded checks if authorization exists and accepts the grant.
// In case the origin is the caller of the address, no authorization is required.
func CheckAndAcceptAuthorizationIfNeeded(
	ctx sdk.Context,
	contract *vm.Contract,
	origin common.Address,
	authzKeeper authzkeeper.Keeper,
	msg sdk.Msg,
) error {
	if contract.CallerAddress == origin {
		return nil
	}

	auth, expiration, err := authorization.CheckAuthzExists(ctx, authzKeeper, contract.CallerAddress, origin, sdk.MsgTypeURL(msg))
	if err != nil {
		return fmt.Errorf(authorization.ErrAuthzDoesNotExistOrExpired, contract.CallerAddress, origin)
	}

	// Accept the grant with the actual message
	_, isMintAuth := auth.(*ucdaotypes.ConvertToHaqqAuthorization)
	_, isTransferAuth := auth.(*ucdaotypes.TransferOwnershipAuthorization)
	if !isMintAuth && !isTransferAuth {
		return fmt.Errorf("expected ConvertToHaqqAuthorization or TransferOwnershipAuthorization, got %T", auth)
	}

	resp, err := auth.Accept(ctx, msg)
	if err != nil {
		return err
	}

	if !resp.Accept {
		return fmt.Errorf("authorization not accepted")
	}

	// Update grant if needed (application-based authz is always deleted after use)
	if resp.Delete {
		if err = authzKeeper.DeleteGrant(ctx, contract.CallerAddress.Bytes(), origin.Bytes(), sdk.MsgTypeURL(msg)); err != nil {
			return err
		}
	} else if resp.Updated != nil {
		if err = authzKeeper.SaveGrant(ctx, contract.CallerAddress.Bytes(), origin.Bytes(), resp.Updated, expiration); err != nil {
			return err
		}
	}

	return nil
}
