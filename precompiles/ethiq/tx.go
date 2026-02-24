package ethiq

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	ethiqkeeper "github.com/haqq-network/haqq/x/ethiq/keeper"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

const (
	// MintHaqq defines the ABI method name for the ethiq mint HAQQ transaction
	MintHaqq = "mintHaqq"
	// MintHaqqByApplication defines the ABI method name for the ethiq mint HAQQ by application transaction
	MintHaqqByApplication = "mintHaqqByApplication"

	// ApproveApplicationIDMethod defines the ABI method name for the authorization Approve transaction.
	ApproveApplicationIDMethod = "approveApplicationID"
	// RevokeApplicationIDMethod defines the ABI method name for the authorization Revoke transaction.
	RevokeApplicationIDMethod = "revokeApplicationID"
)

// MintHaqqMsgURL defines the authorization type for MsgMintHaqq
var MintHaqqMsgURL = sdk.MsgTypeURL(&ethiqtypes.MsgMintHaqq{})

// MsgMintHaqqByApplicationMsgURL defines the authorization type for MsgMintHaqqByApplication
var MsgMintHaqqByApplicationMsgURL = sdk.MsgTypeURL(&ethiqtypes.MsgMintHaqqByApplication{})

func (p Precompile) MintHaqq(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, sender, receiver, err := NewMintHaqqMsg(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"sender", sender,
		"method", method.Name,
		"from_address", msg.FromAddress,
		"to_address", msg.ToAddress,
		"islm_amount", msg.IslmAmount.String(),
	)

	// isCallerSender is true when the contract caller is the same as the sender
	isCallerSender := contract.CallerAddress == sender

	// If the contract caller is not the same as the sender, the sender must be the origin
	if !isCallerSender && origin != sender {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, origin.String(), sender.String())
	}

	// Check and accept authorization if needed
	if contract.CallerAddress != origin {
		auth, expiration, err := authorization.CheckAuthzExists(ctx, p.AuthzKeeper, contract.CallerAddress, origin, MintHaqqMsgURL)
		if err != nil {
			return nil, fmt.Errorf(authorization.ErrAuthzDoesNotExistOrExpired, contract.CallerAddress, origin)
		}

		// Accept the grant with the actual message
		mintAuthz, ok := auth.(*ethiqtypes.MintHaqqAuthorization)
		if !ok {
			return nil, fmt.Errorf("expected MintHaqqAuthorization, got %T", auth)
		}

		resp, err := mintAuthz.Accept(ctx, msg)
		if err != nil {
			return nil, err
		}

		if !resp.Accept {
			return nil, fmt.Errorf("authorization not accepted")
		}

		// Update grant if needed
		if resp.Delete {
			if err = p.AuthzKeeper.DeleteGrant(ctx, contract.CallerAddress.Bytes(), origin.Bytes(), MintHaqqMsgURL); err != nil {
				return nil, err
			}
		} else if resp.Updated != nil {
			if err = p.AuthzKeeper.SaveGrant(ctx, contract.CallerAddress.Bytes(), origin.Bytes(), resp.Updated, expiration); err != nil {
				return nil, err
			}
		}
	}

	// Execute the transaction using the message server
	msgSrv := ethiqkeeper.NewMsgServerImpl(p.ethiqKeeper)
	res, err := msgSrv.MintHaqq(ctx, msg)
	if err != nil {
		return nil, err
	}

	if contract.CallerAddress != origin {
		// NOTE: This ensures that the changes in the bank keeper are correctly mirrored to the EVM stateDB
		// when calling the precompile from another smart contract.
		// This prevents the stateDB from overwriting the changed balance in the bank keeper when committing the EVM state.
		p.SetBalanceChangeEntries(
			cmn.NewBalanceChangeEntry(sender, msg.IslmAmount.BigInt(), cmn.Sub),
		)
	}

	if err = EmitMintHaqqEventWithAmount(
		ctx,
		stateDB,
		p.ABI.Events[EventTypeMintHaqq],
		p.Address(),
		sender,
		receiver,
		msg.IslmAmount,
		res.HaqqAmount,
	); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.HaqqAmount.BigInt())
}

func (p Precompile) MintHaqqByApplication(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, sender, err := NewMintHaqqByApplicationMsg(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"sender", sender,
		"method", method.Name,
		"from_address", msg.FromAddress,
		"application_id", msg.ApplicationId,
	)

	// isCallerSender is true when the contract caller is the same as the sender
	isCallerSender := contract.CallerAddress == sender

	// If the contract caller is not the same as the sender, the sender must be the origin
	if !isCallerSender && origin != sender {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, origin.String(), sender.String())
	}

	// Check and accept authorization if needed
	if contract.CallerAddress != origin {
		auth, expiration, err := authorization.CheckAuthzExists(ctx, p.AuthzKeeper, contract.CallerAddress, origin, MsgMintHaqqByApplicationMsgURL)
		if err != nil {
			return nil, fmt.Errorf(authorization.ErrAuthzDoesNotExistOrExpired, contract.CallerAddress, origin)
		}

		// Accept the grant with the actual message
		appAuthz, ok := auth.(*ethiqtypes.MintHaqqByApplicationIDAuthorization)
		if !ok {
			return nil, fmt.Errorf("expected MintHaqqByApplicationIDAuthorization, got %T", auth)
		}

		resp, err := appAuthz.Accept(ctx, msg)
		if err != nil {
			return nil, err
		}

		if !resp.Accept {
			return nil, fmt.Errorf("authorization not accepted")
		}

		// Update grant if needed (application-based authz is always deleted after use)
		if resp.Delete {
			if err = p.AuthzKeeper.DeleteGrant(ctx, contract.CallerAddress.Bytes(), origin.Bytes(), MsgMintHaqqByApplicationMsgURL); err != nil {
				return nil, err
			}
		} else if resp.Updated != nil {
			if err = p.AuthzKeeper.SaveGrant(ctx, contract.CallerAddress.Bytes(), origin.Bytes(), resp.Updated, expiration); err != nil {
				return nil, err
			}
		}
	}

	// Execute the transaction using the message server
	msgSrv := ethiqkeeper.NewMsgServerImpl(p.ethiqKeeper)
	res, err := msgSrv.MintHaqqByApplication(ctx, msg)
	if err != nil {
		return nil, err
	}

	receiver := common.BytesToAddress(sdk.MustAccAddressFromBech32(res.ToAddress).Bytes())

	if err = EmitMintHaqqEventWithApplicationID(
		ctx,
		stateDB,
		p.ABI.Events[EventTypeMintHaqqByApplication],
		p.Address(),
		sender,
		receiver,
		msg.ApplicationId,
		res.HaqqAmount,
	); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.HaqqAmount.BigInt())
}
