package evm

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// EthMempoolFeeDecorator will check if the transaction's effective fee is at least as large
// as the local validator's minimum gasFee (defined in validator config).
// If fee is too low, decorator returns error and tx is rejected from mempool.
// Note this only applies when ctx.CheckTx = true
// If fee is high enough or not CheckTx, then call next AnteHandler
// CONTRACT: Tx must implement FeeTx to use MempoolFeeDecorator
type EthMempoolFeeDecorator struct {
	evmKeeper EVMKeeper
}

// NewEthMempoolFeeDecorator creates a new NewEthMempoolFeeDecorator instance used only for
// Ethereum transactions.
func NewEthMempoolFeeDecorator(ek EVMKeeper) EthMempoolFeeDecorator {
	return EthMempoolFeeDecorator{
		evmKeeper: ek,
	}
}

// AnteHandle ensures that the provided fees meet a minimum threshold for the validator.
// This check only for local mempool purposes, and thus it is only run on (Re)CheckTx.
// The logic is also skipped if the London hard fork and EIP-1559 are enabled.
func (mfd EthMempoolFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}

	evmParams := mfd.evmKeeper.GetParams(ctx)
	chainCfg := evmParams.GetChainConfig()
	ethCfg := chainCfg.EthereumConfig(mfd.evmKeeper.ChainID())
	blockHeight := big.NewInt(ctx.BlockHeight())
	rules := ethCfg.Rules(blockHeight, true)
	baseFee := mfd.evmKeeper.GetBaseFee(ctx, ethCfg)
	// skip check as the London hard fork and EIP-1559 are enabled
	if baseFee != nil {
		return next(ctx, tx, simulate)
	}

	minGasPrice := ctx.MinGasPrices().AmountOf(evmParams.EvmDenom)

	for _, msg := range tx.GetMsgs() {
		_, txData, _, err := evmtypes.UnpackEthMsg(msg)
		if err != nil {
			return ctx, err
		}

		feeAmt := txData.Fee()
		gas := txData.GetGas()
		fee := sdkmath.LegacyNewDecFromBigInt(feeAmt)
		gasLimit := sdkmath.LegacyNewDecFromBigInt(new(big.Int).SetUint64(gas))

		if err := CheckMempoolFee(fee, minGasPrice, gasLimit, rules.IsLondon); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

// CheckMempoolFee checks if the provided fee is at least as large as the local validator's
func CheckMempoolFee(fee, mempoolMinGasPrice, gasLimit sdkmath.LegacyDec, isLondon bool) error {
	if isLondon {
		return nil
	}

	requiredFee := mempoolMinGasPrice.Mul(gasLimit)

	if fee.LT(requiredFee) {
		return errorsmod.Wrapf(
			errortypes.ErrInsufficientFee,
			"insufficient fee; got: %s required: %s",
			fee, requiredFee,
		)
	}

	return nil
}
