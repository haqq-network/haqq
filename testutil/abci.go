package testutil

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

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
	newCtx := app.BaseApp.NewContextLegacy(false, header)
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
	appHaqq *app.Haqq,
	priv cryptotypes.PrivKey,
	gasPrice *sdkmath.Int,
	msgs ...sdk.Msg,
) (abci.ExecTxResult, error) {
	txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
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
	)
	if err != nil {
		return abci.ExecTxResult{}, err
	}

	return BroadcastTxBytes(appHaqq, txConfig.TxEncoder(), cosmosTx)
}

// DeliverEthTx generates and broadcasts a Cosmos Tx populated with MsgEthereumTx messages.
// If a private key is provided, it will attempt to sign all messages with the given private key,
// otherwise, it will assume the messages have already been signed.
func DeliverEthTx(
	appHaqq *app.Haqq,
	priv cryptotypes.PrivKey,
	msgs ...sdk.Msg,
) (abci.ExecTxResult, error) {
	// TODO: replace with app.config.GetConfig()
	cfg := encoding.MakeConfig(app.ModuleBasics)

	ethTx, err := tx.PrepareEthTx(cfg.TxConfig, appHaqq, priv, msgs...)
	if err != nil {
		return abci.ExecTxResult{}, err
	}

	res, err := BroadcastTxBytes(appHaqq, cfg.TxConfig.TxEncoder(), ethTx)
	if err != nil {
		return abci.ExecTxResult{}, err
	}

	if _, err := CheckEthTxResponse(res, cfg.Codec); err != nil {
		return abci.ExecTxResult{}, err
	}

	return res, nil
}

// CheckTx checks a cosmos tx for a given set of msgs
func CheckTx(
	ctx sdk.Context,
	appHaqq *app.Haqq,
	priv cryptotypes.PrivKey,
	gasPrice *sdkmath.Int,
	msgs ...sdk.Msg,
) (abci.ResponseCheckTx, error) {
	// TODO: replace with app.config.GetConfig()
	txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig

	cosmosTx, err := tx.PrepareCosmosTx(
		ctx,
		appHaqq,
		tx.CosmosTxArgs{
			TxCfg:    txConfig,
			Priv:     priv,
			ChainID:  ctx.ChainID(),
			GasPrice: gasPrice,
			Gas:      10_000_000,
			Msgs:     msgs,
		},
	)
	if err != nil {
		return abci.ResponseCheckTx{}, err
	}
	return checkTxBytes(appHaqq, txConfig.TxEncoder(), cosmosTx)
}

// CheckEthTx checks a Ethereum tx for a given set of msgs
func CheckEthTx(
	appHaqq *app.Haqq,
	priv cryptotypes.PrivKey,
	msgs ...sdk.Msg,
) (abci.ResponseCheckTx, error) {
	// TODO: replace with app.config.GetConfig()
	txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig

	ethTx, err := tx.PrepareEthTx(txConfig, appHaqq, priv, msgs...)
	if err != nil {
		return abci.ResponseCheckTx{}, err
	}
	return checkTxBytes(appHaqq, txConfig.TxEncoder(), ethTx)
}

// BroadcastTxBytes encodes a transaction and calls DeliverTx on the app.
func BroadcastTxBytes(app *app.Haqq, txEncoder sdk.TxEncoder, tx sdk.Tx) (abci.ExecTxResult, error) {
	// bz are bytes to be broadcasted over the network
	bz, err := txEncoder(tx)
	if err != nil {
		return abci.ExecTxResult{}, err
	}

	req := abci.RequestFinalizeBlock{Txs: [][]byte{bz}}
	res, err := app.BaseApp.FinalizeBlock(&req)
	if err != nil {
		return abci.ExecTxResult{}, err
	}
	if len(res.TxResults) != 1 {
		return abci.ExecTxResult{}, fmt.Errorf("unexpected transaction results; expected 1, got: %d", len(res.TxResults))
	}
	txRes := res.TxResults[0]
	if txRes.Code != 0 {
		return abci.ExecTxResult{}, errorsmod.Wrapf(errortypes.ErrInvalidRequest, txRes.Log)
	}

	return *txRes, nil
}

// commit is a private helper function that runs the EndBlocker logic, commits the changes,
// updates the header, runs the BeginBlocker function and returns the updated header
func commit(ctx sdk.Context, app *app.Haqq, t time.Duration, vs *tmtypes.ValidatorSet) (tmproto.Header, error) {
	header := ctx.BlockHeader()
	req := abci.RequestFinalizeBlock{Height: header.Height}

	if vs != nil {
		res, err := app.FinalizeBlock(&req)
		if err != nil {
			return header, err
		}

		nextVals, err := applyValSetChanges(vs, res.ValidatorUpdates)
		if err != nil {
			return header, err
		}
		header.ValidatorsHash = vs.Hash()
		header.NextValidatorsHash = nextVals.Hash()
	} else {
		if _, err := app.EndBlocker(ctx); err != nil {
			return header, err
		}
	}

	if _, err := app.Commit(); err != nil {
		return header, err
	}

	header.Height++
	header.Time = header.Time.Add(t)
	header.AppHash = app.LastCommitID().Hash

	if _, err := app.BeginBlocker(ctx); err != nil {
		return header, err
	}

	return header, nil
}

// checkTxBytes encodes a transaction and calls checkTx on the app.
func checkTxBytes(app *app.Haqq, txEncoder sdk.TxEncoder, tx sdk.Tx) (abci.ResponseCheckTx, error) {
	bz, err := txEncoder(tx)
	if err != nil {
		return abci.ResponseCheckTx{}, err
	}

	req := abci.RequestCheckTx{Tx: bz}
	res, err := app.BaseApp.CheckTx(&req)
	if err != nil {
		return abci.ResponseCheckTx{}, err
	}
	if res.Code != 0 {
		return abci.ResponseCheckTx{}, errorsmod.Wrapf(errortypes.ErrInvalidRequest, res.Log)
	}

	return *res, nil
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
