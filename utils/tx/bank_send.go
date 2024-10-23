package tx

import (
	"errors"
	"math/big"
	"strconv"

	sdkmath "cosmossdk.io/math"
	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	tmtypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func BankSendTxAsEthereumTx(tx sdk.Tx) (*evmtypes.MsgEthereumTx, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}

	if len(tx.GetMsgs()) == 0 {
		return nil, errors.New("tx has no messages")
	}

	if len(tx.GetMsgs()) > 1 {
		return nil, errors.New("tx has more than one message")
	}

	msg := tx.GetMsgs()[0]
	sendMsg, ok := msg.(*banktypes.MsgSend)
	if !ok {
		return nil, errors.New("tx message is not a bank.MsgSend")
	}

	// Get gas if set
	dTx, ok := tx.(sdk.FeeTx)
	gasLimit := uint64(0)
	gasPrice := sdkmath.ZeroInt()
	if ok {
		gasLimit = dTx.GetGas()
		if gasLimit > 0 {
			gasPrice = sdkmath.NewIntFromBigInt(dTx.GetFee().AmountOf(utils.BaseDenom).BigInt()).Quo(sdkmath.NewIntFromUint64(gasLimit))
		}
	}

	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return nil, errors.New("transaction without signatures")
	}

	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return nil, errors.New("failed to get transaction signatures")
	}

	switch {
	case len(sigTx.GetSigners()) == 1:
		if len(sigs) != 1 {
			return nil, errors.New("transaction with one signer has more than one signature")
		}
	case len(sigTx.GetSigners()) > 1:
		// TODO implement
		return nil, errors.New("miltisign transaction convert not implemented yet")
	default:
		return nil, errors.New("transaction without signers")
	}

	sig, ok := sigs[0].Data.(*signing.SingleSignatureData)
	if !ok {
		return nil, errors.New("transaction signature data is not a signing.SingleSignatureData")
	}

	sigLen := len(sig.Signature)
	if sigLen == 0 {
		return nil, errors.New("empty signature")
	}

	fromAddr := sdk.MustAccAddressFromBech32(sendMsg.FromAddress)
	toAddr := sdk.MustAccAddressFromBech32(sendMsg.ToAddress)
	islmAmount := sendMsg.Amount.AmountOf(utils.BaseDenom)
	amtInt := sdkmath.NewIntFromBigInt(islmAmount.BigInt())

	// Dummy V,R,S values from real Cosmos signature
	chunkSize := sigLen / 2
	if chunkSize > 32 {
		chunkSize = 32
	}

	r := new(big.Int).SetBytes(sig.Signature[:chunkSize])
	s := new(big.Int).SetBytes(sig.Signature[chunkSize : chunkSize*2])
	v := new(big.Int).SetBytes([]byte{sig.Signature[chunkSize*2] + 27})

	txData := &evmtypes.LegacyTx{
		Nonce:    sigs[0].Sequence,
		GasPrice: &gasPrice,
		GasLimit: gasLimit,
		To:       common.BytesToAddress(toAddr.Bytes()).Hex(),
		Amount:   &amtInt,
		Data:     nil,
		V:        v.Bytes(),
		R:        r.Bytes(),
		S:        s.Bytes(),
	}

	dataAny, err := evmtypes.PackTxData(txData)
	if err != nil {
		return nil, err
	}

	ethTx := &evmtypes.MsgEthereumTx{
		From: common.BytesToAddress(fromAddr.Bytes()).Hex(),
		Data: dataAny,
	}
	ethTx.Hash = ethTx.AsTransaction().Hash().Hex()

	return ethTx, nil
}

func InjectEthereumTxEvents(ctx sdk.Context, decoder sdk.TxDecoder) error {
	if len(ctx.TxBytes()) == 0 {
		return errors.New("tx bytes are empty")
	}

	tx, err := decoder(ctx.TxBytes())
	if err != nil {
		return err
	}

	ethMsg, err := BankSendTxAsEthereumTx(tx)
	if err != nil {
		return err
	}

	ethTx := ethMsg.AsTransaction()

	attrs := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyAmount, ethTx.Value().String()),
		// add event for ethereum transaction hash format
		sdk.NewAttribute(evmtypes.AttributeKeyEthereumTxHash, ethMsg.Hash),
		// add event for index of valid ethereum tx
		sdk.NewAttribute(evmtypes.AttributeKeyTxIndex, strconv.FormatUint(0, 10)),
		// add event for eth tx gas used, we can't get it from cosmos tx result when it contains multiple eth tx msgs.
		sdk.NewAttribute(evmtypes.AttributeKeyTxGasUsed, strconv.FormatUint(ethMsg.GetGas(), 10)),
	}

	if len(ctx.TxBytes()) > 0 {
		// add event for tendermint transaction hash format
		hash := tmbytes.HexBytes(tmtypes.Tx(ctx.TxBytes()).Hash())
		attrs = append(attrs, sdk.NewAttribute(evmtypes.AttributeKeyTxHash, hash.String()))
	}

	if to := ethTx.To(); to != nil {
		attrs = append(attrs, sdk.NewAttribute(evmtypes.AttributeKeyRecipient, to.Hex()))
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			evmtypes.EventTypeEthereumTx,
			attrs...,
		),
	})

	return nil
}
