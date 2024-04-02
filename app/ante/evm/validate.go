package evm

import (
	"errors"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// EthValidateBasicDecorator is adapted from ValidateBasicDecorator from cosmos-sdk, it ignores ErrNoSignatures
type EthValidateBasicDecorator struct {
	evmKeeper EVMKeeper
}

// NewEthValidateBasicDecorator creates a new EthValidateBasicDecorator
func NewEthValidateBasicDecorator(ek EVMKeeper) EthValidateBasicDecorator {
	return EthValidateBasicDecorator{
		evmKeeper: ek,
	}
}

// AnteHandle handles basic validation of tx
func (vbd EthValidateBasicDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// no need to validate basic on recheck tx, call next AnteHandler
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	if t, ok := tx.(sdk.HasValidateBasic); ok {
		err := t.ValidateBasic()
		// ErrNoSignatures is fine with eth tx
		if err != nil && !errors.Is(err, errortypes.ErrNoSignatures) {
			return ctx, errorsmod.Wrap(err, "tx basic validation failed")
		}
	}

	// For eth type cosmos tx, some fields should be verified as zero values,
	// since we will only verify the signature against the hash of the MsgEthereumTx.Data
	wrapperTx, ok := tx.(protoTxProvider)
	if !ok {
		return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid tx type %T, didn't implement interface protoTxProvider", tx)
	}

	protoTx := wrapperTx.GetProtoTx()
	body := protoTx.Body
	if body.Memo != "" || body.TimeoutHeight != uint64(0) || len(body.NonCriticalExtensionOptions) > 0 {
		return ctx, errorsmod.Wrap(errortypes.ErrInvalidRequest,
			"for eth tx body Memo TimeoutHeight NonCriticalExtensionOptions should be empty")
	}

	if len(body.ExtensionOptions) != 1 {
		return ctx, errorsmod.Wrap(errortypes.ErrInvalidRequest, "for eth tx length of ExtensionOptions should be 1")
	}

	authInfo := protoTx.AuthInfo
	if len(authInfo.SignerInfos) > 0 {
		return ctx, errorsmod.Wrap(errortypes.ErrInvalidRequest, "for eth tx AuthInfo SignerInfos should be empty")
	}

	if authInfo.Fee.Payer != "" || authInfo.Fee.Granter != "" {
		return ctx, errorsmod.Wrap(errortypes.ErrInvalidRequest, "for eth tx AuthInfo Fee payer and granter should be empty")
	}

	sigs := protoTx.Signatures
	if len(sigs) > 0 {
		return ctx, errorsmod.Wrap(errortypes.ErrInvalidRequest, "for eth tx Signatures should be empty")
	}

	txFee := sdk.Coins{}
	txGasLimit := uint64(0)

	evmParams := vbd.evmKeeper.GetParams(ctx)
	chainCfg := evmParams.GetChainConfig()
	chainID := vbd.evmKeeper.ChainID()
	ethCfg := chainCfg.EthereumConfig(chainID)
	baseFee := vbd.evmKeeper.GetBaseFee(ctx, ethCfg)
	enableCreate := evmParams.GetEnableCreate()
	enableCall := evmParams.GetEnableCall()
	evmDenom := evmParams.GetEvmDenom()

	msgs := protoTx.GetMsgs()
	if msgs == nil {
		return ctx, errorsmod.Wrap(errortypes.ErrUnknownRequest, "invalid transaction. Transaction without messages")
	}

	for _, msg := range msgs {
		_, txData, from, err := evmtypes.UnpackEthMsg(msg)
		if err != nil {
			return ctx, err
		}

		// Validate `From` field
		if from != nil {
			return ctx, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "invalid From %s, expect empty string", from)
		}

		txGasLimit += txData.GetGas()

		// return error if contract creation or call are disabled through governance
		if !enableCreate && txData.GetTo() == nil {
			return ctx, errorsmod.Wrap(evmtypes.ErrCreateDisabled, "failed to create new contract")
		} else if !enableCall && txData.GetTo() != nil {
			return ctx, errorsmod.Wrap(evmtypes.ErrCallDisabled, "failed to call contract")
		}

		if baseFee == nil && txData.TxType() == ethtypes.DynamicFeeTxType {
			return ctx, errorsmod.Wrap(ethtypes.ErrTxTypeNotSupported, "dynamic fee tx not supported")
		}

		txFee = txFee.Add(sdk.Coin{Denom: evmDenom, Amount: sdkmath.NewIntFromBigInt(txData.Fee())})
	}

	if !authInfo.Fee.Amount.Equal(txFee) {
		return ctx, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "invalid AuthInfo Fee Amount (%s != %s)", authInfo.Fee.Amount, txFee)
	}

	if authInfo.Fee.GasLimit != txGasLimit {
		return ctx, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "invalid AuthInfo Fee GasLimit (%d != %d)", authInfo.Fee.GasLimit, txGasLimit)
	}

	return next(ctx, tx, simulate)
}
