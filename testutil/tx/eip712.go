package tx

import (
	"errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/haqq-network/haqq/app"
	cryptocodec "github.com/haqq-network/haqq/crypto/codec"
	"github.com/haqq-network/haqq/ethereum/eip712"
	"github.com/haqq-network/haqq/types"
)

type EIP712TxArgs struct {
	CosmosTxArgs       CosmosTxArgs
	UseLegacyExtension bool
	UseLegacyTypedData bool
}

type typedDataArgs struct {
	chainID        uint64
	data           []byte
	legacyFeePayer sdk.AccAddress
	legacyMsg      sdk.Msg
}

type signatureV2Args struct {
	pubKey    cryptotypes.PubKey
	signature []byte
	nonce     uint64
}

type legacyWeb3ExtensionArgs struct {
	feePayer  string
	chainID   uint64
	signature []byte
}

// CreateEIP712CosmosTx creates a cosmos tx for typed data according to EIP712.
// Also, signs the tx with the provided messages and private key.
// It returns the signed transaction and an error
func CreateEIP712CosmosTx(
	ctx sdk.Context,
	haqqApp *app.Haqq,
	args EIP712TxArgs,
) (sdk.Tx, error) {
	builder, err := PrepareEIP712CosmosTx(
		ctx,
		haqqApp,
		args,
	)
	return builder.GetTx(), err
}

// PrepareEIP712CosmosTx creates a cosmos tx for typed data according to EIP712.
// Also, signs the tx with the provided messages and private key.
// It returns the tx builder with the signed transaction and an error
func PrepareEIP712CosmosTx(
	ctx sdk.Context,
	haqqApp *app.Haqq,
	args EIP712TxArgs,
) (client.TxBuilder, error) {
	txArgs := args.CosmosTxArgs

	pc, err := types.ParseChainID(txArgs.ChainID)
	if err != nil {
		return nil, err
	}
	chainIDNum := pc.Uint64()

	from := sdk.AccAddress(txArgs.Priv.PubKey().Address().Bytes())
	accNumber := haqqApp.AccountKeeper.GetAccount(ctx, from).GetAccountNumber()

	nonce, err := haqqApp.AccountKeeper.GetSequence(ctx, from)
	if err != nil {
		return nil, err
	}

	fee := legacytx.NewStdFee(txArgs.Gas, txArgs.Fees)

	msgs := txArgs.Msgs
	data := legacytx.StdSignBytes(ctx.ChainID(), accNumber, nonce, 0, fee, msgs, "", nil)

	typedDataArgs := typedDataArgs{
		chainID:        chainIDNum,
		data:           data,
		legacyFeePayer: from,
		legacyMsg:      msgs[0],
	}
	typedData, err := createTypedData(typedDataArgs, args.UseLegacyTypedData)
	if err != nil {
		return nil, err
	}

	txBuilder := txArgs.TxCfg.NewTxBuilder()
	builder, ok := txBuilder.(authtx.ExtensionOptionsTxBuilder)
	if !ok {
		return nil, errors.New("txBuilder could not be casted to authtx.ExtensionOptionsTxBuilder type")
	}

	builder.SetFeeAmount(fee.Amount)
	builder.SetGasLimit(txArgs.Gas)

	err = builder.SetMsgs(txArgs.Msgs...)
	if err != nil {
		return nil, err
	}

	return signCosmosEIP712Tx(
		ctx,
		haqqApp,
		args,
		builder,
		chainIDNum,
		typedData,
	)
}

// signCosmosEIP712Tx signs the cosmos transaction on the txBuilder provided using
// the provided private key and the typed data
func signCosmosEIP712Tx(
	ctx sdk.Context,
	haqqApp *app.Haqq,
	args EIP712TxArgs,
	builder authtx.ExtensionOptionsTxBuilder,
	chainID uint64,
	data apitypes.TypedData,
) (client.TxBuilder, error) {
	priv := args.CosmosTxArgs.Priv

	from := sdk.AccAddress(priv.PubKey().Address().Bytes())
	nonce, err := haqqApp.AccountKeeper.GetSequence(ctx, from)
	if err != nil {
		return nil, err
	}

	sigHash, _, err := apitypes.TypedDataAndHash(data)
	if err != nil {
		return nil, err
	}

	keyringSigner := NewSigner(priv)
	signature, pubKey, err := keyringSigner.SignByAddress(from, sigHash)
	if err != nil {
		return nil, err
	}
	signature[crypto.RecoveryIDOffset] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper

	if args.UseLegacyExtension {
		if err := setBuilderLegacyWeb3Extension(
			builder,
			legacyWeb3ExtensionArgs{
				feePayer:  from.String(),
				chainID:   chainID,
				signature: signature,
			}); err != nil {
			return nil, err
		}
	}

	sigsV2 := getTxSignatureV2(
		signatureV2Args{
			pubKey:    pubKey,
			signature: signature,
			nonce:     nonce,
		},
		args.UseLegacyExtension,
	)

	err = builder.SetSignatures(sigsV2)
	if err != nil {
		return nil, err
	}

	return builder, nil
}

// createTypedData creates the TypedData object corresponding to
// the arguments, using the legacy implementation as specified.
func createTypedData(args typedDataArgs, useLegacy bool) (apitypes.TypedData, error) {
	if useLegacy {
		registry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(registry)
		cryptocodec.RegisterInterfaces(registry)
		evmosCodec := codec.NewProtoCodec(registry)

		feeDelegation := &eip712.FeeDelegationOptions{
			FeePayer: args.legacyFeePayer,
		}

		return eip712.LegacyWrapTxToTypedData(
			evmosCodec,
			args.chainID,
			args.legacyMsg,
			args.data,
			feeDelegation,
		)
	}

	return eip712.WrapTxToTypedData(args.chainID, args.data)
}

// setBuilderLegacyWeb3Extension creates a legacy ExtensionOptionsWeb3Tx and
// appends it to the builder options.
func setBuilderLegacyWeb3Extension(builder authtx.ExtensionOptionsTxBuilder, args legacyWeb3ExtensionArgs) error {
	option, err := codectypes.NewAnyWithValue(&types.ExtensionOptionsWeb3Tx{
		FeePayer:         args.feePayer,
		TypedDataChainID: args.chainID,
		FeePayerSig:      args.signature,
	})
	if err != nil {
		return err
	}

	builder.SetExtensionOptions(option)
	return nil
}

// getTxSignatureV2 returns the SignatureV2 object corresponding to
// the arguments, using the legacy implementation as needed.
func getTxSignatureV2(args signatureV2Args, useLegacyExtension bool) signing.SignatureV2 {
	if useLegacyExtension {
		return signing.SignatureV2{
			PubKey: args.pubKey,
			Data: &signing.SingleSignatureData{
				SignMode: signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
			},
			Sequence: args.nonce,
		}
	}

	// Must use SIGN_MODE_DIRECT, since Amino has some trouble parsing certain Any values from a SignDoc
	// with the Legacy EIP-712 TypedData encodings. This is not an issue with the latest encoding.
	return signing.SignatureV2{
		PubKey: args.pubKey,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: args.signature,
		},
		Sequence: args.nonce,
	}
}
