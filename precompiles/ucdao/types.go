package ucdao

import (
	"fmt"
	"math/big"

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

	coins := sdk.Coins{}
	for i := range denomsIface {
		if amountsIface[i] == nil {
			return nil, common.Address{}, common.Address{}, fmt.Errorf("nil amount at index %d", i)
		}
		coin := sdk.NewCoin(denomsIface[i], sdkmath.NewIntFromBigInt(amountsIface[i]))
		coins = coins.Add(coin)
	}

	msg := ucdaotypes.NewMsgTransferOwnershipWithAmount(
		sdk.AccAddress(owner.Bytes()),
		sdk.AccAddress(newOwner.Bytes()),
		coins,
	)

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
