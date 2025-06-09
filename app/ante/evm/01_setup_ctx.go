package evm

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	evmante "github.com/haqq-network/haqq/x/evm/ante"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// EthSetupContextDecorator is adapted from SetUpContextDecorator from cosmos-sdk, it ignores gas consumption
// by setting the gas meter to infinite
type EthSetupContextDecorator struct {
	evmKeeper EVMKeeper
}

func NewEthSetUpContextDecorator(evmKeeper EVMKeeper) EthSetupContextDecorator {
	return EthSetupContextDecorator{
		evmKeeper: evmKeeper,
	}
}

func (esc EthSetupContextDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	newCtx, err = SetupContext(ctx, tx, esc.evmKeeper)
	if err != nil {
		return ctx, err
	}
	return next(newCtx, tx, simulate)
}

func SetupContext(ctx sdk.Context, tx sdk.Tx, evmKeeper EVMKeeper) (sdk.Context, error) {
	// all transactions must implement GasTx
	_, ok := tx.(authante.GasTx)
	if !ok {
		return ctx, errorsmod.Wrapf(errortypes.ErrInvalidType, "invalid transaction type %T, expected GasTx", tx)
	}

	// We need to set up an empty gas config so that the gas is consistent with Ethereum.
	newCtx := evmante.BuildEvmExecutionCtx(ctx).
		WithGasMeter(sdk.NewInfiniteGasMeter())
	// Reset transient gas used to prepare the execution of current cosmos tx.
	// Transient gas-used is necessary to sum the gas-used of cosmos tx, when it contains multiple eth msgs.
	evmKeeper.ResetTransientGasUsed(ctx)

	return newCtx, nil
}

// EthEmitEventDecorator emit events in ante handler in case of tx execution failed (out of block gas limit).
type EthEmitEventDecorator struct {
	evmKeeper EVMKeeper
}

// NewEthEmitEventDecorator creates a new EthEmitEventDecorator
func NewEthEmitEventDecorator(evmKeeper EVMKeeper) EthEmitEventDecorator {
	return EthEmitEventDecorator{evmKeeper}
}

// AnteHandle emits some basic events for the eth messages
func (eeed EthEmitEventDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// After eth tx passed ante handler, the fee is deducted and nonce increased, it shouldn't be ignored by json-rpc,
	// we need to emit some basic events at the very end of ante handler to be indexed by tendermint.
	txIndex := eeed.evmKeeper.GetTxIndexTransient(ctx)

	for i, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		// emit ethereum tx hash as an event so that it can be indexed by Tendermint for query purposes
		// it's emitted in ante handler, so we can query failed transaction (out of block gas limit).
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			evmtypes.EventTypeEthereumTx,
			sdk.NewAttribute(evmtypes.AttributeKeyEthereumTxHash, msgEthTx.Hash),
			sdk.NewAttribute(evmtypes.AttributeKeyTxIndex, strconv.FormatUint(txIndex+uint64(i), 10)), // #nosec G701
		))
	}

	return next(ctx, tx, simulate)
}
