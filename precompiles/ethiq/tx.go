package ethiq

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
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

// mirrorBankBaseDeltaIntoStateDB syncs the debited account's aISLM (EVM gas denom) bank delta into the EVM journal when
// the precompile is invoked from another contract (caller != origin). baseBefore must be read immediately before the
// keeper/msg work. Delta matches bank movements from redeem-liquid-then-burn (not necessarily the nominal msg amount).
func (p *Precompile) mirrorBankBaseDeltaIntoStateDB(ctx sdk.Context, isCallerOrigin bool, debitAccAddr sdk.AccAddress, baseBefore sdkmath.Int) {
	if isCallerOrigin {
		return
	}
	netBaseDelta := baseBefore.Sub(p.ethiqKeeper.BaseDenomBankBalance(ctx, debitAccAddr))
	if netBaseDelta.IsZero() {
		return
	}
	debitHexAddr := common.BytesToAddress(debitAccAddr.Bytes())
	if netBaseDelta.IsNegative() {
		p.SetBalanceChangeEntries(cmn.NewBalanceChangeEntry(debitHexAddr, netBaseDelta.Neg().BigInt(), cmn.Add))
		return
	}
	p.SetBalanceChangeEntries(cmn.NewBalanceChangeEntry(debitHexAddr, netBaseDelta.BigInt(), cmn.Sub))
}

func (p *Precompile) MintHaqq(
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
		"origin", origin.String(),
		"caller", contract.CallerAddress.String(),
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ from_address: %s, to_address: %s, islm_amount: %s }",
			msg.FromAddress,
			msg.ToAddress,
			msg.IslmAmount.String(),
		),
	)

	// isCallerSender is true when the contract caller is the same as the sender
	isCallerSender := contract.CallerAddress == sender
	// isCallerOrigin is true when the contract caller is the same as the origin
	isCallerOrigin := contract.CallerAddress == origin

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

	fromAccAddr := sdk.MustAccAddressFromBech32(msg.FromAddress)
	baseBefore := p.ethiqKeeper.BaseDenomBankBalance(ctx, fromAccAddr)

	// Execute the transaction using the message server
	msgSrv := ethiqkeeper.NewMsgServerImpl(p.ethiqKeeper)
	res, err := msgSrv.MintHaqq(ctx, msg)
	if err != nil {
		return nil, err
	}

	p.mirrorBankBaseDeltaIntoStateDB(ctx, isCallerOrigin, fromAccAddr, baseBefore)

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

func (p *Precompile) MintHaqqByApplication(
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
		"origin", origin.String(),
		"caller", contract.CallerAddress.String(),
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ from_address: %s, application_id: %d }",
			msg.FromAddress,
			msg.ApplicationId,
		),
	)

	// isCallerSender is true when the contract caller is the same as the sender
	isCallerSender := contract.CallerAddress == sender
	// isCallerOrigin is true when the contract caller is the same as the origin
	isCallerOrigin := contract.CallerAddress == origin

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

	application, err := ethiqtypes.GetApplicationByID(msg.ApplicationId)
	if err != nil {
		return nil, err
	}

	// Same debit account as BurnIslmForHaqqByApplicationID (owner bank vs UCDAO escrow).
	debitAccAddr := sdk.MustAccAddressFromBech32(msg.FromAddress)
	if application.Source == ethiqtypes.SourceOfFunds_SOURCE_OF_FUNDS_UCDAO {
		debitAccAddr = ethiqkeeper.GetUCDAOEscrowAddress(debitAccAddr)
	}
	baseBefore := p.ethiqKeeper.BaseDenomBankBalance(ctx, debitAccAddr)

	// Execute the transaction using the message server
	msgSrv := ethiqkeeper.NewMsgServerImpl(p.ethiqKeeper)
	res, err := msgSrv.MintHaqqByApplication(ctx, msg)
	if err != nil {
		return nil, err
	}

	receiver := common.BytesToAddress(sdk.MustAccAddressFromBech32(res.ToAddress).Bytes())

	p.mirrorBankBaseDeltaIntoStateDB(ctx, isCallerOrigin, debitAccAddr, baseBefore)

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
