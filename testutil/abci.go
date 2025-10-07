package testutil

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/testutil/tx"
)

// DeliverTx delivers a cosmos tx for a given set of msgs
func DeliverTx(
	ctx sdk.Context,
	appHaqq *app.Haqq,
	priv cryptotypes.PrivKey,
	gasPrice *sdkmath.Int,
	signMode signing.SignMode,
	msgs ...sdk.Msg,
) (abci.ExecTxResult, error) {
	txConfig := appHaqq.GetTxConfig()
	cosmosTx, err := tx.PrepareCosmosTx(
		ctx,
		appHaqq,
		tx.CosmosTxArgs{
			TxCfg:    txConfig,
			Priv:     priv,
			ChainID:  ctx.ChainID(),
			Gas:      10_000_000,
			GasPrice: gasPrice,
			Msgs:     msgs,
		},
		signMode,
	)
	if err != nil {
		return abci.ExecTxResult{}, err
	}

	return BroadcastTxBytes(ctx, appHaqq, txConfig.TxEncoder(), cosmosTx)
}

// DeliverEthTx generates and broadcasts a Cosmos Tx populated with MsgEthereumTx messages.
// If a private key is provided, it will attempt to sign all messages with the given private key,
// otherwise, it will assume the messages have already been signed.
func DeliverEthTx(
	ctx sdk.Context,
	appHaqq *app.Haqq,
	priv cryptotypes.PrivKey,
	msgs ...sdk.Msg,
) (abci.ExecTxResult, error) {
	txConfig := appHaqq.GetTxConfig()

	ethTx, err := tx.PrepareEthTx(txConfig, appHaqq, priv, msgs...)
	if err != nil {
		return abci.ExecTxResult{}, err
	}
	res, err := BroadcastTxBytes(ctx, appHaqq, txConfig.TxEncoder(), ethTx)
	if err != nil {
		return res, err
	}

	codec := appHaqq.AppCodec()
	if _, err := CheckEthTxResponse(res, codec); err != nil {
		return res, err
	}

	return res, nil
}

// BroadcastTxBytes encodes a transaction and calls DeliverTx on the app.
func BroadcastTxBytes(ctx sdk.Context, app *app.Haqq, txEncoder sdk.TxEncoder, tx sdk.Tx) (abci.ExecTxResult, error) {
	header := ctx.BlockHeader()
	header.AppHash = app.LastCommitID().Hash
	header.Time = header.Time.Add(time.Second)

	// bz are bytes to be broadcasted over the network
	bz, err := txEncoder(tx)
	if err != nil {
		return abci.ExecTxResult{}, err
	}

	req := &abci.RequestFinalizeBlock{
		Height:             header.Height,
		DecidedLastCommit:  abci.CommitInfo{},
		Hash:               header.AppHash,
		NextValidatorsHash: header.ValidatorsHash,
		ProposerAddress:    header.ProposerAddress,
		Time:               header.Time,
		Txs:                [][]byte{bz},
	}

	res, err := app.BaseApp.FinalizeBlock(req)
	if err != nil {
		return abci.ExecTxResult{}, err
	}
	if len(res.TxResults) != 1 {
		return abci.ExecTxResult{}, fmt.Errorf("unexpected transaction results. Expected 1, got: %d", len(res.TxResults))
	}
	txRes := res.TxResults[0]
	if txRes.Code != 0 {
		return abci.ExecTxResult{}, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "log: %s", txRes.Log)
	}

	return *txRes, nil
}
