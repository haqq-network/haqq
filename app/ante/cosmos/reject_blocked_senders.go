package cosmos

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

// RejectBlockedSendersDecorator rejects any transaction whose required signers
// include a blocked account. It freezes accounts that illegitimately acquired
// funds through the precompile abuse exploit, preventing them from moving or
// hiding stolen funds via Cosmos or EIP-712 transactions.
//
// The blacklist is keyed by the bech32 (haqq1...) representation of the account
// address. A nil or empty blacklist makes the decorator a no-op.
type RejectBlockedSendersDecorator struct {
	blacklist map[string]bool
}

// NewRejectBlockedSendersDecorator creates a new RejectBlockedSendersDecorator.
func NewRejectBlockedSendersDecorator(blacklist map[string]bool) RejectBlockedSendersDecorator {
	return RejectBlockedSendersDecorator{blacklist: blacklist}
}

// AnteHandle rejects the tx if any of its required signers is blocked. It runs
// on both CheckTx and DeliverTx so blocked accounts are kept out of the mempool
// and out of blocks.
func (d RejectBlockedSendersDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if len(d.blacklist) == 0 {
		return next(ctx, tx, simulate)
	}

	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return ctx, errorsmod.Wrapf(errortypes.ErrInvalidType, "tx %T doesn't implement authsigning.SigVerifiableTx", tx)
	}

	signers, err := sigTx.GetSigners()
	if err != nil {
		return ctx, err
	}

	for _, signer := range signers {
		addr := sdk.AccAddress(signer).String()
		if d.blacklist[addr] {
			return ctx, errorsmod.Wrapf(
				errortypes.ErrUnauthorized,
				"account %s is blocked from sending transactions", addr,
			)
		}
	}

	return next(ctx, tx, simulate)
}
