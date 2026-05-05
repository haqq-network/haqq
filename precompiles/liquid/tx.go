package liquid

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	liquidkeeper "github.com/haqq-network/haqq/x/liquidvesting/keeper"
)

const (
	// LiquidateMethod defines the ABI method name for the liquidvesting Liquidate transaction.
	LiquidateMethod = "liquidate"
	// RedeemMethod defines the ABI method name for the liquidvesting Redeem transaction.
	RedeemMethod = "redeem"
)

// mirrorBankBaseDeltaIntoStateDB syncs the bank delta of the EVM gas denom (aISLM)
// produced by a keeper call into the EVM StateDB journal when the precompile is
// invoked from another contract (caller != origin).
//
// Why this is needed:
//   - Liquidate/Redeem move aISLM through bankKeeper.SendCoinsFromAccount/ModuleTo* directly.
//     Those calls update the SDK bank state but do not touch the EVM journal.
//   - On EVM tx commit, x/evm reconciles native EVM account balances against the
//     SDK bank for accounts touched by the EVM journal. If the contract account that
//     was actually debited/credited in bank is not in the journal, its EVM-side balance
//     is left unchanged, while bank already reflects the move - leading to a state
//     mismatch and effectively "phantom" coins on the EVM side.
//   - When caller == origin (EOA), x/evm intentionally skips reconciling the origin's
//     balance for these direct keeper movements, so no journal entry is required.
//
// baseBefore must be sampled immediately before the keeper/msg work; the resulting
// delta matches the actual bank movement (which may differ from the nominal msg amount,
// e.g. for Redeem the credit on `to` equals the unlocked principal that round-trips
// through the module account).
func (p *Precompile) mirrorBankBaseDeltaIntoStateDB(
	ctx sdk.Context,
	isCallerOrigin bool,
	addr sdk.AccAddress,
	baseBefore sdkmath.Int,
) {
	if isCallerOrigin {
		return
	}
	netBaseDelta := baseBefore.Sub(p.keeper.BaseDenomBankBalance(ctx, addr))
	if netBaseDelta.IsZero() {
		return
	}
	hexAddr := common.BytesToAddress(addr.Bytes())
	if netBaseDelta.IsNegative() {
		// Bank balance grew: account was credited - mirror as Add.
		p.AddBalanceChangeEntries(cmn.NewBalanceChangeEntry(hexAddr, netBaseDelta.Neg().BigInt(), cmn.Add))
		return
	}
	// Bank balance shrank: account was debited - mirror as Sub.
	p.AddBalanceChangeEntries(cmn.NewBalanceChangeEntry(hexAddr, netBaseDelta.BigInt(), cmn.Sub))
}

// Liquidate executes the liquidvesting Liquidate message.
// It supports authorization when the caller is not the origin account.
func (p *Precompile) Liquidate(
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
		"tx called",
		"sender", sender,
		"method", method.Name,
		"from_address", msg.LiquidateFrom,
		"to_address", msg.LiquidateTo,
		"amount", msg.Amount.String(),
	)

	originAddr := sdk.AccAddress(origin.Bytes())
	callerAddr := sdk.AccAddress(contract.CallerAddress.Bytes())

	// isCallerSender is true when the contract caller is the same as the sender
	isCallerSender := contract.CallerAddress == sender
	// isCallerOrigin is true when the contract caller is the same as the origin
	isCallerOrigin := contract.CallerAddress == origin

	// Ensure the logical sender matches the origin when going through a contract.
	if !isCallerSender && origin != sender {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, origin.String(), sender.String())
	}

	// Check and accept authorization if needed
	if !isCallerOrigin {
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
	}

	// Snapshot the bank balance of the debited account (liquidateFrom) BEFORE the
	// keeper call so the EVM-side mirror can replay the exact aISLM movement that
	// SendCoinsFromAccountToModule performs inside Liquidate.
	liquidateFromAccAddr := sdk.MustAccAddressFromBech32(msg.LiquidateFrom)
	baseBefore := p.keeper.BaseDenomBankBalance(ctx, liquidateFromAccAddr)

	// Execute the message using the message server.
	msgSrv := liquidkeeper.NewMsgServerImpl(p.keeper)
	res, err := msgSrv.Liquidate(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Mirror the bank delta on liquidateFrom into the EVM StateDB journal so that
	// the EVM commit sees the contract caller's aISLM balance as actually debited.
	p.mirrorBankBaseDeltaIntoStateDB(ctx, isCallerOrigin, liquidateFromAccAddr, baseBefore)

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
func (p *Precompile) Redeem(
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
		"tx called",
		"sender", sender,
		"method", method.Name,
		"from_address", msg.RedeemFrom,
		"to_address", msg.RedeemTo,
		"amount", msg.Amount.String(),
	)

	originAddr := sdk.AccAddress(origin.Bytes())
	callerAddr := sdk.AccAddress(contract.CallerAddress.Bytes())

	// isCallerSender is true when the contract caller is the same as the sender
	isCallerSender := contract.CallerAddress == sender
	// isCallerOrigin is true when the contract caller is the same as the origin
	isCallerOrigin := contract.CallerAddress == origin

	// If the contract caller is not the same as the sender, the sender must be the origin
	if !isCallerSender && origin != sender {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, origin.String(), sender.String())
	}

	// Check and accept authorization if needed
	if !isCallerOrigin {
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
	}

	// Snapshot the bank balances of both bank-touching counterparties BEFORE the
	// keeper call so the EVM-side mirror can replay the exact aISLM movement that
	// Redeem performs internally:
	//   - redeemFrom is debited the liquid (aLIQUID*) denom, which is NOT the EVM
	//     gas denom, so its base-denom balance is unchanged - sampling it lets the
	//     mirror no-op for that side without special-casing.
	//   - redeemTo is credited the unlocked principal in aISLM coming from the
	//     liquidvesting module account; this is the credit we must propagate to
	//     the EVM journal when the call originates from a contract.
	redeemFromAccAddr := sdk.MustAccAddressFromBech32(msg.RedeemFrom)
	redeemToAccAddr := sdk.MustAccAddressFromBech32(msg.RedeemTo)
	fromBaseBefore := p.keeper.BaseDenomBankBalance(ctx, redeemFromAccAddr)
	toBaseBefore := p.keeper.BaseDenomBankBalance(ctx, redeemToAccAddr)

	// Execute the message using the message server.
	msgSrv := liquidkeeper.NewMsgServerImpl(p.keeper)
	if _, err := msgSrv.Redeem(ctx, msg); err != nil {
		return nil, err
	}

	// Mirror the bank deltas into the EVM StateDB journal so that the EVM commit
	// sees the contract caller's aISLM balance as actually credited/debited.
	// AddBalanceChangeEntries (rather than SetBalanceChangeEntries) is used so the
	// two entries coexist when redeemFrom == redeemTo or when both sides happen
	// to move base denom (e.g. through a future hook on the from side).
	p.mirrorBankBaseDeltaIntoStateDB(ctx, isCallerOrigin, redeemFromAccAddr, fromBaseBefore)
	p.mirrorBankBaseDeltaIntoStateDB(ctx, isCallerOrigin, redeemToAccAddr, toBaseBefore)

	if err := p.EmitRedeemEvent(ctx, stateDB, sender, common.HexToAddress(msg.RedeemTo), msg.Amount.Denom, msg.Amount.Amount.BigInt()); err != nil {
		return nil, err
	}

	// Redeem has an empty response, so we simply return an empty output.
	return method.Outputs.Pack()
}
