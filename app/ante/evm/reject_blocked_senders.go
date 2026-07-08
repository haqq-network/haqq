package evm

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// EthRejectBlockedSendersDecorator rejects any Ethereum transaction sent from a
// blocked account. It freezes accounts that illegitimately acquired funds
// through the precompile abuse exploit, preventing them from moving or hiding
// stolen funds via EVM transactions.
//
// The blacklist is keyed by the bech32 (haqq1...) representation of the account
// address. A haqq1... account and its 0x... EVM address share the same 20-byte
// value, so the recovered sender's bytes are converted to bech32 before lookup.
// A nil or empty blacklist makes the decorator a no-op.
//
// CONTRACT: must run after EthSigVerificationDecorator, which recovers the
// sender and sets MsgEthereumTx.From.
type EthRejectBlockedSendersDecorator struct {
	blacklist map[string]bool
}

// NewEthRejectBlockedSendersDecorator creates a new EthRejectBlockedSendersDecorator.
func NewEthRejectBlockedSendersDecorator(blacklist map[string]bool) EthRejectBlockedSendersDecorator {
	return EthRejectBlockedSendersDecorator{blacklist: blacklist}
}

// AnteHandle rejects the tx if its sender is blocked. It runs on both CheckTx
// and DeliverTx so blocked accounts are kept out of the mempool and out of
// blocks.
func (d EthRejectBlockedSendersDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if len(d.blacklist) == 0 {
		return next(ctx, tx, simulate)
	}

	for _, msg := range tx.GetMsgs() {
		ethMsg, _, _, err := evmtypes.UnpackEthMsg(msg)
		if err != nil {
			return ctx, err
		}

		// From is set by EthSigVerificationDecorator. Convert the sender's address
		// bytes to their bech32 form, which is how the blacklist is keyed.
		sender := sdk.AccAddress(common.HexToAddress(ethMsg.From).Bytes()).String()
		if d.blacklist[sender] {
			return ctx, errorsmod.Wrapf(
				errortypes.ErrUnauthorized,
				"account %s is blocked from sending transactions", sender,
			)
		}
	}

	return next(ctx, tx, simulate)
}
