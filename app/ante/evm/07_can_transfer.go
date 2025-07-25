package evm

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/haqq-network/haqq/x/evm/statedb"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// CanTransferDecorator checks if the sender is allowed to transfer funds according to the EVM block
// context rules.
type CanTransferDecorator struct {
	evmKeeper EVMKeeper
}

// NewCanTransferDecorator creates a new CanTransferDecorator instance.
func NewCanTransferDecorator(evmKeeper EVMKeeper) CanTransferDecorator {
	return CanTransferDecorator{
		evmKeeper: evmKeeper,
	}
}

// AnteHandle creates an EVM from the message and calls the BlockContext CanTransfer function to
// see if the address can execute the transaction.
func (ctd CanTransferDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	evmParams := ctd.evmKeeper.GetParams(ctx)
	chainCfg := evmParams.GetChainConfig()
	ethCfg := chainCfg.EthereumConfig(ctd.evmKeeper.ChainID())
	baseFee := ctd.evmKeeper.GetBaseFee(ctx, ethCfg)
	blockHeight := big.NewInt(ctx.BlockHeight())
	rules := ethCfg.Rules(blockHeight, true)
	signer := ethtypes.MakeSigner(ethCfg, blockHeight)

	if rules.IsLondon && baseFee == nil {
		return ctx, errorsmod.Wrap(
			evmtypes.ErrInvalidBaseFee,
			"base fee is supported but evm block context value is nil",
		)
	}

	for _, msg := range tx.GetMsgs() {
		ethMsg, _, _, err := evmtypes.UnpackEthMsg(msg)
		if err != nil {
			return ctx, err
		}

		coreMsg, err := ethMsg.AsMessage(signer, baseFee)
		if err != nil {
			return ctx, errorsmod.Wrapf(
				err,
				"failed to create an ethereum core.Message from signer %T", signer,
			)
		}

		if err := CanTransfer(
			ctx,
			ctd.evmKeeper,
			coreMsg,
			baseFee,
			ethCfg,
			evmParams,
			rules.IsLondon,
		); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

// CanTransfer checks if the sender is allowed to transfer funds according to the EVM block
func CanTransfer(
	ctx sdk.Context,
	evmKeeper EVMKeeper,
	msg core.Message,
	baseFee *big.Int,
	ethCfg *params.ChainConfig,
	params evmtypes.Params,
	isLondon bool,
) error {
	if isLondon && msg.GasFeeCap().Cmp(baseFee) < 0 {
		return errorsmod.Wrapf(
			errortypes.ErrInsufficientFee,
			"max fee per gas less than block base fee (%s < %s)",
			msg.GasFeeCap(), baseFee,
		)
	}

	// NOTE: pass in an empty coinbase address and nil tracer as we don't need them for the check below
	cfg := &statedb.EVMConfig{
		ChainConfig: ethCfg,
		Params:      params,
		CoinBase:    common.Address{},
		BaseFee:     baseFee,
	}

	stateDB := statedb.New(ctx, evmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash().Bytes())))
	evm := evmKeeper.NewEVM(ctx, msg, cfg, evmtypes.NewNoOpTracer(), stateDB)

	// check that caller has enough balance to cover asset transfer for **topmost** call
	// NOTE: here the gas consumed is from the context with the infinite gas meter
	if msg.Value().Sign() > 0 && !evm.Context.CanTransfer(stateDB, msg.From(), msg.Value()) {
		return errorsmod.Wrapf(
			errortypes.ErrInsufficientFunds,
			"failed to transfer %s from address %s using the EVM block context transfer function",
			msg.Value(),
			msg.From(),
		)
	}

	return nil
}
