package evm

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// EthSigVerificationDecorator validates an ethereum signatures
type EthSigVerificationDecorator struct {
	evmKeeper EVMKeeper
}

// NewEthSigVerificationDecorator creates a new EthSigVerificationDecorator
func NewEthSigVerificationDecorator(ek EVMKeeper) EthSigVerificationDecorator {
	return EthSigVerificationDecorator{
		evmKeeper: ek,
	}
}

// AnteHandle validates checks that the registered chain id is the same as the one on the message, and
// that the signer address matches the one defined on the message.
// It's not skipped for RecheckTx, because it set `From` address which is critical from other ante handler to work.
// Failure in RecheckTx will prevent tx to be included into block, especially when CheckTx succeed, in which case user
// won't see the error message.
func (esvd EthSigVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	evmParams := esvd.evmKeeper.GetParams(ctx)
	chainCfg := evmParams.GetChainConfig()
	ethCfg := chainCfg.EthereumConfig(esvd.evmKeeper.ChainID())
	blockNum := big.NewInt(ctx.BlockHeight())
	signer := ethtypes.MakeSigner(ethCfg, blockNum)
	allowUnprotectedTxs := evmParams.GetAllowUnprotectedTxs()

	for _, msg := range tx.GetMsgs() {
		ethMsg, _, _, err := evmtypes.UnpackEthMsg(msg)
		if err != nil {
			return ctx, err
		}

		if err := SignatureVerification(ethMsg, signer, allowUnprotectedTxs); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

// SignatureVerification checks that the registered chain id is the same as the one on the message, and
// that the signer address matches the one defined on the message.
func SignatureVerification(
	msg *evmtypes.MsgEthereumTx,
	signer ethtypes.Signer,
	allowUnprotectedTxs bool,
) error {
	ethTx := msg.AsTransaction()

	if !allowUnprotectedTxs && !ethTx.Protected() {
		return errorsmod.Wrapf(
			errortypes.ErrNotSupported,
			"rejected unprotected Ethereum transaction. Please EIP155 sign your transaction to protect it against replay-attacks")
	}

	sender, err := signer.Sender(ethTx)
	if err != nil {
		return errorsmod.Wrapf(
			errortypes.ErrorInvalidSigner,
			"couldn't retrieve sender address from the ethereum transaction: %s",
			err.Error(),
		)
	}

	// set up the sender to the transaction field if not already
	msg.From = sender.Hex()
	return nil
}
