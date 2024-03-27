package app

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	tmed25519 "github.com/cometbft/cometbft/crypto/ed25519"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/haqq-network/haqq/encoding"
)

func TestHaqqAnteHandlerDecorator(t *testing.T) {
	valPkey := ed25519.GenPrivKey()
	valAddr := sdk.ValAddress(valPkey.PubKey().Address())

	// tmPubKey implements tmcrypto.PubKey interface
	tmPubKey := tmed25519.PubKey(valPkey.PubKey().Bytes())

	// create validator set with single validator
	validator := tmtypes.NewValidator(tmPubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin("aISLM", math.NewInt(100000000000000))),
	}

	chainID := MainnetChainID + "-1"
	db := dbm.NewMemDB()
	app := NewHaqq(
		log.NewTestLogger(t),
		db, nil, true,
		map[int64]bool{}, DefaultNodeHome, 0,
		encoding.MakeConfig(ModuleBasics),
		simtestutil.NewAppOptionsWithFlagHome(DefaultNodeHome),
		baseapp.SetChainID(chainID),
	)

	genesisState := NewDefaultGenesisState()
	genesisState = GenesisStateWithValSet(app, genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)
	stateBytes, err := json.MarshalIndent(genesisState, "", "  ")
	require.NoError(t, err)

	reqInit := abci.RequestInitChain{
		ChainId:       chainID,
		Validators:    []abci.ValidatorUpdate{},
		AppStateBytes: stateBytes,
	}
	_, err = app.InitChain(&reqInit)
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)

	handler := NewHaqqAnteHandlerDecorator(app.StakingKeeper, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})

	t.Run("set validator", func(t *testing.T) {
		ctx := app.NewContextLegacy(true, tmproto.Header{})
		validator, err := stakingtypes.NewValidator(
			valAddr.String(),
			valPkey.PubKey(),
			stakingtypes.NewDescription("validator", "", "", "", ""),
		)
		require.NoError(t, err)
		err = app.StakingKeeper.SetValidator(ctx, validator)
		require.NoError(t, err)
		err = app.StakingKeeper.SetNewValidatorByPowerIndex(ctx, validator)
		require.NoError(t, err)
	})

	// TODO Need to fix. Distribution module doesn't have a handler for gov proposal now
	// t.Run("try add gov spend proposal", func(t *testing.T) {
	//	coins := sdk.NewCoins(sdk.NewCoin("aISLM", sdk.NewInt(100)))
	//	newAccPkey := ed25519.GenPrivKey()
	//	recipient := sdk.AccAddress(newAccPkey.PubKey().Address())
	//
	//	// generate submit proposal
	//	cpsp := distrtypes.NewCommunityPoolSpendProposal("Test", "description", recipient, coins)
	//	sp, _ := govtypes.NewMsgSubmitProposal(cpsp, coins, recipient)
	//
	//	/// build tx
	//	builder := app.GetTxConfig().NewTxBuilder()
	//	require.NoError(t, builder.SetMsgs(sp))
	//	require.NoError(t, builder.SetSignatures(signing.SignatureV2{
	//		PubKey: newAccPkey.PubKey(),
	//		Data:   nil,
	//	}))
	//
	//	// run tx
	//	ctx := app.NewContext(true, tmproto.Header{Height: 1})
	//	_, err = handler(ctx, builder.GetTx(), true)
	//
	//	require.Error(t, ErrCommunitySpendingComingLater, err)
	// })

	t.Run("create validator", func(t *testing.T) {
		t.Run("from unknown address", func(t *testing.T) {
			newValPkey := ed25519.GenPrivKey()
			newValAddr := sdk.ValAddress(newValPkey.PubKey().Address())
			msg, err := stakingtypes.NewMsgCreateValidator(
				newValAddr.String(),
				newValPkey.PubKey(),
				sdk.NewInt64Coin(sdk.DefaultBondDenom, 50),
				stakingtypes.NewDescription("testname", "", "", "", ""),
				stakingtypes.CommissionRates{},
				math.OneInt(),
			)
			require.NoError(t, err)

			builder := app.GetTxConfig().NewTxBuilder()
			require.NoError(t, builder.SetMsgs(msg))
			require.NoError(t, builder.SetSignatures(signing.SignatureV2{
				PubKey: newValPkey.PubKey(),
				Data:   nil,
			}))

			ctx := app.NewContextLegacy(true, tmproto.Header{Height: 1})
			_, err = handler(ctx, builder.GetTx(), true)

			t.Logf("### from unknown address %v ###", err)

			require.NoError(t, err)
		})

		t.Run("from validator address", func(t *testing.T) {
			msg, err := stakingtypes.NewMsgCreateValidator(
				valAddr.String(),
				valPkey.PubKey(),
				sdk.NewInt64Coin(sdk.DefaultBondDenom, 50),
				stakingtypes.NewDescription("testname", "", "", "", ""),
				stakingtypes.CommissionRates{},
				math.OneInt(),
			)
			require.NoError(t, err)

			builder := app.GetTxConfig().NewTxBuilder()
			require.NoError(t, builder.SetMsgs(msg))
			require.NoError(t, builder.SetSignatures(signing.SignatureV2{
				PubKey: valPkey.PubKey(),
				Data:   nil,
			}))

			ctx := app.NewContextLegacy(true, tmproto.Header{Height: 1})
			_, err = handler(ctx, builder.GetTx(), true)
			require.NoError(t, err)
		})
	})

	t.Run("create delegation", func(t *testing.T) {
		t.Run("from unknown address", func(t *testing.T) {
			delPkey := ed25519.GenPrivKey()
			delAddr := sdk.AccAddress(delPkey.PubKey().Address())

			msg := stakingtypes.NewMsgDelegate(delAddr.String(), valAddr.String(), sdk.NewCoin("aISLM", math.NewInt(10)))
			builder := app.GetTxConfig().NewTxBuilder()
			require.NoError(t, builder.SetMsgs(msg))
			require.NoError(t, builder.SetSignatures(signing.SignatureV2{
				PubKey: delPkey.PubKey(),
				Data:   nil,
			}))

			ctx := app.NewContextLegacy(true, tmproto.Header{Height: 1})
			_, err := handler(ctx, builder.GetTx(), true)

			require.NoError(t, err)
		})

		t.Run("from validator address", func(t *testing.T) {
			delAddr := sdk.AccAddress(valPkey.PubKey().Address())

			msg := stakingtypes.NewMsgDelegate(delAddr.String(), valAddr.String(), sdk.NewCoin("aISLM", math.NewInt(10)))
			builder := app.GetTxConfig().NewTxBuilder()
			require.NoError(t, builder.SetMsgs(msg))
			require.NoError(t, builder.SetSignatures(signing.SignatureV2{
				PubKey: valPkey.PubKey(),
				Data:   nil,
			}))

			ctx := app.NewContextLegacy(true, tmproto.Header{Height: 1})
			_, err := handler(ctx, builder.GetTx(), true)

			require.NoError(t, err)
		})
	})
}
