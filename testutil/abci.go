package testutil

import (
	"time"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/encoding"
	"github.com/haqq-network/haqq/testutil/tx"
)

// Commit commits a block at a given time. Reminder: At the end of each
// Tendermint Consensus round the following methods are run
//  1. BeginBlock
//  2. DeliverTx
//  3. EndBlock
//  4. Commit
func Commit(ctx sdk.Context, app *app.Haqq, t time.Duration, vs *tmtypes.ValidatorSet) (sdk.Context, error) {
	header, err := commit(ctx, app, t, vs)
	if err != nil {
		return ctx, err
	}

	return ctx.WithBlockHeader(header), nil
}

// CommitAndCreateNewCtx commits a block at a given time creating a ctx with the current settings
// This is useful to keep test settings that could be affected by EndBlockers, e.g.
// setting a baseFee == 0 and expecting this condition to continue after commit
func CommitAndCreateNewCtx(ctx sdk.Context, app *app.Haqq, t time.Duration, vs *tmtypes.ValidatorSet) (sdk.Context, error) {
	header, err := commit(ctx, app, t, vs)
	if err != nil {
		return ctx, err
	}

	// NewContext function keeps the multistore
	// but resets other context fields
	// GasMeter is set as InfiniteGasMeter
	newCtx := app.BaseApp.NewContext(false, header)
	// set the reseted fields to keep the current ctx settings
	newCtx = newCtx.WithMinGasPrices(ctx.MinGasPrices())
	newCtx = newCtx.WithEventManager(ctx.EventManager())
	newCtx = newCtx.WithKVGasConfig(ctx.KVGasConfig())
	newCtx = newCtx.WithTransientKVGasConfig(ctx.TransientKVGasConfig())

	return newCtx, nil
}

// DeliverTx delivers a cosmos tx for a given set of msgs
func DeliverTx(
	ctx sdk.Context,
	appEvmos *app.Haqq,
	priv cryptotypes.PrivKey,
	gasPrice *sdkmath.Int,
	signMode signing.SignMode,
	msgs ...sdk.Msg,
) (abci.ResponseDeliverTx, error) {
	txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
	cosmosTx, err := tx.PrepareCosmosTx(
		ctx,
		appEvmos,
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
		return abci.ResponseDeliverTx{}, err
	}

	return BroadcastTxBytes(appEvmos, txConfig.TxEncoder(), cosmosTx)
}

// DeliverEthTx generates and broadcasts a Cosmos Tx populated with MsgEthereumTx messages.
// If a private key is provided, it will attempt to sign all messages with the given private key,
// otherwise, it will assume the messages have already been signed.
func DeliverEthTx(
	appEvmos *app.Haqq,
	priv cryptotypes.PrivKey,
	msgs ...sdk.Msg,
) (abci.ResponseDeliverTx, error) {
	txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig

	ethTx, err := tx.PrepareEthTx(txConfig, appEvmos, priv, msgs...)
	if err != nil {
		return abci.ResponseDeliverTx{}, err
	}
	res, err := BroadcastTxBytes(appEvmos, txConfig.TxEncoder(), ethTx)
	if err != nil {
		return res, err
	}

	codec := encoding.MakeConfig(app.ModuleBasics).Codec
	if _, err := CheckEthTxResponse(res, codec); err != nil {
		return res, err
	}

	return res, nil
}

// DeliverEthTxWithoutCheck generates and broadcasts a Cosmos Tx populated with MsgEthereumTx messages.
// If a private key is provided, it will attempt to sign all messages with the given private key,
// otherwise, it will assume the messages have already been signed. It does not check if the Eth tx is
// successful or not.
func DeliverEthTxWithoutCheck(
	appEvmos *app.Haqq,
	priv cryptotypes.PrivKey,
	msgs ...sdk.Msg,
) (abci.ResponseDeliverTx, error) {
	txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig

	ethTx, err := tx.PrepareEthTx(txConfig, appEvmos, priv, msgs...)
	if err != nil {
		return abci.ResponseDeliverTx{}, err
	}

	res, err := BroadcastTxBytes(appEvmos, txConfig.TxEncoder(), ethTx)
	if err != nil {
		return abci.ResponseDeliverTx{}, err
	}

	return res, nil
}

// CheckTx checks a cosmos tx for a given set of msgs
func CheckTx(
	ctx sdk.Context,
	appEvmos *app.Haqq,
	priv cryptotypes.PrivKey,
	gasPrice *sdkmath.Int,
	signMode signing.SignMode,
	msgs ...sdk.Msg,
) (abci.ResponseCheckTx, error) {
	txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig

	cosmosTx, err := tx.PrepareCosmosTx(
		ctx,
		appEvmos,
		tx.CosmosTxArgs{
			TxCfg:    txConfig,
			Priv:     priv,
			ChainID:  ctx.ChainID(),
			GasPrice: gasPrice,
			Gas:      10_000_000,
			Msgs:     msgs,
		},
		signMode,
	)
	if err != nil {
		return abci.ResponseCheckTx{}, err
	}
	return checkTxBytes(appEvmos, txConfig.TxEncoder(), cosmosTx)
}

// CheckEthTx checks a Ethereum tx for a given set of msgs
func CheckEthTx(
	appEvmos *app.Haqq,
	priv cryptotypes.PrivKey,
	msgs ...sdk.Msg,
) (abci.ResponseCheckTx, error) {
	txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig

	ethTx, err := tx.PrepareEthTx(txConfig, appEvmos, priv, msgs...)
	if err != nil {
		return abci.ResponseCheckTx{}, err
	}

	return checkTxBytes(appEvmos, txConfig.TxEncoder(), ethTx)
}

// BroadcastTxBytes encodes a transaction and calls DeliverTx on the app.
func BroadcastTxBytes(app *app.Haqq, txEncoder sdk.TxEncoder, tx sdk.Tx) (abci.ResponseDeliverTx, error) {
	// bz are bytes to be broadcasted over the network
	bz, err := txEncoder(tx)
	if err != nil {
		return abci.ResponseDeliverTx{}, err
	}

	req := abci.RequestDeliverTx{Tx: bz}
	res := app.BaseApp.DeliverTx(req)
	if res.Code != 0 {
		return abci.ResponseDeliverTx{}, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "log: %s", res.Log)
	}

	return res, nil
}

// commit is a private helper function that runs the EndBlocker logic, commits the changes,
// updates the header, runs the BeginBlocker function and returns the updated header
func commit(ctx sdk.Context, app *app.Haqq, t time.Duration, vs *tmtypes.ValidatorSet) (tmproto.Header, error) {
	header := ctx.BlockHeader()

	if vs != nil {
		res := app.EndBlock(abci.RequestEndBlock{Height: header.Height})

		nextVals, err := applyValSetChanges(vs, res.ValidatorUpdates)
		if err != nil {
			return header, err
		}
		header.ValidatorsHash = vs.Hash()
		header.NextValidatorsHash = nextVals.Hash()
	} else {
		app.EndBlocker(ctx, abci.RequestEndBlock{Height: header.Height})
	}

	_ = app.Commit()

	header.Height++
	header.Time = header.Time.Add(t)
	header.AppHash = app.LastCommitID().Hash

	app.BeginBlock(abci.RequestBeginBlock{
		Header: header,
	})

	return header, nil
}

// checkTxBytes encodes a transaction and calls checkTx on the app.
func checkTxBytes(app *app.Haqq, txEncoder sdk.TxEncoder, tx sdk.Tx) (abci.ResponseCheckTx, error) {
	bz, err := txEncoder(tx)
	if err != nil {
		return abci.ResponseCheckTx{}, err
	}

	req := abci.RequestCheckTx{Tx: bz}
	res := app.BaseApp.CheckTx(req)
	if res.Code != 0 {
		return abci.ResponseCheckTx{}, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "log: %s", res.Log)
	}

	return res, nil
}

// applyValSetChanges takes in tmtypes.ValidatorSet and []abci.ValidatorUpdate and will return a new tmtypes.ValidatorSet which has the
// provided validator updates applied to the provided validator set.
func applyValSetChanges(valSet *tmtypes.ValidatorSet, valUpdates []abci.ValidatorUpdate) (*tmtypes.ValidatorSet, error) {
	updates, err := tmtypes.PB2TM.ValidatorUpdates(valUpdates)
	if err != nil {
		return nil, err
	}

	// must copy since validator set will mutate with UpdateWithChangeSet
	newVals := valSet.Copy()
	err = newVals.UpdateWithChangeSet(updates)
	if err != nil {
		return nil, err
	}

	return newVals, nil
}
