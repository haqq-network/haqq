package ucdao

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

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

// ConvertToHaqqMsgURL defines the authorization type for MsgConvertToHaqq
var ConvertToHaqqMsgURL = sdk.MsgTypeURL(&ucdaotypes.MsgConvertToHaqq{})

// TransferOwnershipMsgURL defines the authorization type for MsgTransferOwnership
var TransferOwnershipMsgURL = sdk.MsgTypeURL(&ucdaotypes.MsgTransferOwnership{})

func (p *Precompile) ConvertToHaqq(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, sender, receiver, err := NewConvertToHaqqMsg(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"origin", origin.String(),
		"caller", contract.CallerAddress.String(),
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ sender: %s, receiver: %s, islm_amount: %s }",
			msg.Sender,
			msg.Receiver,
			msg.IslmAmount.String(),
		),
	)

	// isCallerSender is true when the contract caller is the same as the sender
	isCallerSender := contract.CallerAddress == sender

	// If the contract caller is not the same as the sender, the sender must be the origin
	if isCallerSender {
		sender = origin
	} else if origin != sender {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, origin.String(), sender.String())
	}

	// Check and accept authorization if needed
	if err := CheckAndAcceptAuthorizationIfNeeded(ctx, contract, sender, p.AuthzKeeper, msg); err != nil {
		return nil, err
	}

	msgSrv := ucdaokeeper.NewMsgServerImpl(p.daoKeeper)
	res, err := msgSrv.ConvertToHaqq(ctx, msg)
	if err != nil {
		return nil, err
	}

	if err = EmitMintHaqqEventWithAmount(
		ctx,
		stateDB,
		p.ABI.Events[EventTypeMintHaqq],
		p.Address(),
		sender,
		receiver,
		msg.IslmAmount,
		res.MintedCoin.Amount,
	); err != nil {
		return nil, err
	}

	// Return minted amount as uint256
	return method.Outputs.Pack(res.MintedCoin.Amount.BigInt())
}

func (p *Precompile) TransferOwnership(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, owner, _, err := NewTransferOwnershipMsg(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"origin", origin.String(),
		"caller", contract.CallerAddress.String(),
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ owner: %s, newOwner: %s, amount: %s }",
			msg.Owner,
			msg.NewOwner,
			"all balance",
		),
	)

	// isCallerSender is true when the contract caller is the same as the sender
	isCallerSender := contract.CallerAddress == owner

	// If the contract caller is not the same as the sender, the sender must be the origin
	if isCallerSender {
		owner = origin
	} else if origin != owner {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, origin.String(), owner.String())
	}

	// Check and accept authorization if needed
	if err := CheckAndAcceptAuthorizationIfNeeded(ctx, contract, owner, p.AuthzKeeper, msg); err != nil {
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

func (p *Precompile) TransferOwnershipWithAmount(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, owner, _, err := NewTransferOwnershipWithAmountMsg(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"origin", origin.String(),
		"caller", contract.CallerAddress.String(),
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ owner: %s, newOwner: %s, amount: %s }",
			msg.Owner,
			msg.NewOwner,
			msg.Amount.String(),
		),
	)

	// isCallerSender is true when the contract caller is the same as the sender
	isCallerSender := contract.CallerAddress == owner

	// If the contract caller is not the same as the sender, the sender must be the origin
	if isCallerSender {
		owner = origin
	} else if origin != owner {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, origin.String(), owner.String())
	}

	// Check and accept authorization if needed
	if err := CheckAndAcceptAuthorizationIfNeeded(ctx, contract, owner, p.AuthzKeeper, msg); err != nil {
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
