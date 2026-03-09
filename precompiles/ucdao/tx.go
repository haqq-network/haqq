package ucdao

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	ucdaokeeper "github.com/haqq-network/haqq/x/ucdao/keeper"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

const (
	// ConvertToHaqqMethod defines the ABI method name for ConvertToHaqq transaction.
	ConvertToHaqqMethod = "convertToHaqq"
	// TransferOwnershipMethod defines the ABI method name for TransferOwnership transaction.
	TransferOwnershipMethod = "transferOwnership"
	// TransferOwnershipWithAmountMethod defines the ABI method name for TransferOwnershipWithAmount transaction.
	TransferOwnershipWithAmountMethod = "transferOwnershipWithAmount"
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

func (p Precompile) ConvertToHaqq(
	ctx sdk.Context,
	origin common.Address,
	_ *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, sender, _, err := NewConvertToHaqqMsg(args)
	if err != nil {
		return nil, err
	}

	// Ensure origin is the sender (no authz support for now)
	if origin != sender {
		return nil, fmt.Errorf("origin (%s) must be the sender (%s)", origin.String(), sender.String())
	}

	msgSrv := ucdaokeeper.NewMsgServerImpl(p.daoKeeper)
	res, err := msgSrv.ConvertToHaqq(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Return minted amount as uint256
	return method.Outputs.Pack(res.MintedCoin.Amount.BigInt())
}

func (p Precompile) TransferOwnership(
	ctx sdk.Context,
	origin common.Address,
	_ *vm.Contract,
	_ vm.StateDB,
	_ *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, owner, _, err := NewTransferOwnershipMsg(args)
	if err != nil {
		return nil, err
	}

	// Ensure origin is the owner
	if origin != owner {
		return nil, fmt.Errorf("origin (%s) must be the owner (%s)", origin.String(), owner.String())
	}

	msgSrv := ucdaokeeper.NewMsgServerImpl(p.daoKeeper)
	_, err = msgSrv.TransferOwnership(ctx, msg)
	if err != nil {
		return nil, err
	}

	return []byte{}, nil
}

func (p Precompile) TransferOwnershipWithAmount(
	ctx sdk.Context,
	origin common.Address,
	_ *vm.Contract,
	_ vm.StateDB,
	_ *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, owner, _, err := NewTransferOwnershipWithAmountMsg(args)
	if err != nil {
		return nil, err
	}

	// Ensure origin is the owner
	if origin != owner {
		return nil, fmt.Errorf("origin (%s) must be the owner (%s)", origin.String(), owner.String())
	}

	msgSrv := ucdaokeeper.NewMsgServerImpl(p.daoKeeper)
	_, err = msgSrv.TransferOwnershipWithAmount(ctx, msg)
	if err != nil {
		return nil, err
	}

	return []byte{}, nil
}
