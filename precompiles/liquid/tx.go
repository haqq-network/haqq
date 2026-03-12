package liquid

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/x/evm/core/vm"
	liquidkeeper "github.com/haqq-network/haqq/x/liquidvesting/keeper"
)

const (
	// LiquidateMethod defines the ABI method name for the liquidvesting Liquidate transaction.
	LiquidateMethod = "liquidate"
	// RedeemMethod defines the ABI method name for the liquidvesting Redeem transaction.
	RedeemMethod = "redeem"
)

// Liquidate executes the liquidvesting Liquidate message.
// It supports authorization when the caller is not the origin account.
func (p Precompile) Liquidate(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, sender, _, err := NewLiquidateMsg(args)
	if err != nil {
		return nil, err
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"liquidvesting.Liquidate called",
		"sender", sender,
		"method", method.Name,
		"from_address", msg.LiquidateFrom,
		"to_address", msg.LiquidateTo,
		"amount", msg.Amount.String(),
	)

	senderAddr := sdk.AccAddress(sender.Bytes())
	originAddr := sdk.AccAddress(origin.Bytes())
	callerAddr := sdk.AccAddress(contract.CallerAddress.Bytes())

	// Ensure the logical sender matches the origin when going through a contract.
	if !callerAddr.Equals(originAddr) && !senderAddr.Equals(originAddr) {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, originAddr.String(), senderAddr.String())
	}

	// If the contract caller is the origin, execute directly without authorization.
	if contract.CallerAddress == origin {
		msgSrv := liquidkeeper.NewMsgServerImpl(p.keeper)
		res, err := msgSrv.Liquidate(ctx, msg)
		if err != nil {
			return nil, err
		}

		minted := res.Minted.Amount.BigInt()
		erc20Addr := common.HexToAddress(res.ContractAddr)

		if err := p.EmitLiquidateEvent(ctx, stateDB, sender, common.HexToAddress(msg.LiquidateTo), erc20Addr, minted); err != nil {
			return nil, err
		}

		return method.Outputs.Pack(minted, erc20Addr)
	}

	// Otherwise, require an authz grant from the origin (granter) to the contract caller (grantee).
	msgURL := sdk.MsgTypeURL(msg)
	authzGrant, expiration := p.AuthzKeeper.GetAuthorization(ctx, callerAddr, originAddr, msgURL)
	if authzGrant == nil {
		return nil, fmt.Errorf(ErrAuthzDoesNotExistOrExpired, msgURL, callerAddr.String())
	}

	resp, err := authzGrant.Accept(ctx, msg)
	if err != nil {
		return nil, err
	}

	if !resp.Accept {
		return nil, fmt.Errorf("authorization not accepted")
	}

	// Update or delete the grant if required.
	if resp.Delete {
		if err := p.AuthzKeeper.DeleteGrant(ctx, callerAddr, originAddr, msgURL); err != nil {
			return nil, err
		}
	} else if resp.Updated != nil {
		if err := p.AuthzKeeper.SaveGrant(ctx, callerAddr, originAddr, resp.Updated, expiration); err != nil {
			return nil, err
		}
	}

	// Execute the message using the message server.
	msgSrv := liquidkeeper.NewMsgServerImpl(p.keeper)
	res, err := msgSrv.Liquidate(ctx, msg)
	if err != nil {
		return nil, err
	}

	minted := res.Minted.Amount.BigInt()
	erc20Addr := common.HexToAddress(res.ContractAddr)

	if err := p.EmitLiquidateEvent(ctx, stateDB, sender, common.HexToAddress(msg.LiquidateTo), erc20Addr, minted); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(minted, erc20Addr)
}

// Redeem executes the liquidvesting Redeem message.
// It MUST always be executed via authorization, regardless of caller/origin,
// since it needs to transfer liquid ERC20 representation back into native coins.
func (p Precompile) Redeem(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, sender, _, err := NewRedeemMsg(args)
	if err != nil {
		return nil, err
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"liquidvesting.Redeem called",
		"sender", sender,
		"method", method.Name,
		"from_address", msg.RedeemFrom,
		"to_address", msg.RedeemTo,
		"amount", msg.Amount.String(),
	)

	senderAddr := sdk.AccAddress(sender.Bytes())
	originAddr := sdk.AccAddress(origin.Bytes())
	callerAddr := sdk.AccAddress(contract.CallerAddress.Bytes())

	// Enforce that redeem is only callable via allowance (authz).
	// This means the contract caller must NOT be the origin directly.
	if contract.CallerAddress == origin {
		return nil, fmt.Errorf(ErrRedeemRequiresAuthorization)
	}

	// Additionally require the origin to match the logical sender.
	if !senderAddr.Equals(originAddr) {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, originAddr.String(), senderAddr.String())
	}

	msgURL := sdk.MsgTypeURL(msg)

	// Require an authz grant from the origin (granter) to the contract caller (grantee).
	authzGrant, expiration := p.AuthzKeeper.GetAuthorization(ctx, callerAddr, originAddr, msgURL)
	if authzGrant == nil {
		return nil, fmt.Errorf(ErrAuthzDoesNotExistOrExpired, msgURL, callerAddr.String())
	}

	resp, err := authzGrant.Accept(ctx, msg)
	if err != nil {
		return nil, err
	}

	if !resp.Accept {
		return nil, fmt.Errorf("authorization not accepted")
	}

	// Update or delete the grant if required.
	if resp.Delete {
		if err := p.AuthzKeeper.DeleteGrant(ctx, callerAddr, originAddr, msgURL); err != nil {
			return nil, err
		}
	} else if resp.Updated != nil {
		if err := p.AuthzKeeper.SaveGrant(ctx, callerAddr, originAddr, resp.Updated, expiration); err != nil {
			return nil, err
		}
	}

	// Execute the message using the message server.
	msgSrv := liquidkeeper.NewMsgServerImpl(p.keeper)
	if _, err := msgSrv.Redeem(ctx, msg); err != nil {
		return nil, err
	}

	if err := p.EmitRedeemEvent(ctx, stateDB, sender, common.HexToAddress(msg.RedeemTo), msg.Amount.Denom, msg.Amount.Amount.BigInt()); err != nil {
		return nil, err
	}

	// Redeem has an empty response, so we simply return an empty output.
	return method.Outputs.Pack()
}
