package app

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"
	"github.com/tharsis/ethermint/encoding"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestHaqqAnteHandlerDecorator(t *testing.T) {
	valPkey := ed25519.GenPrivKey()
	valAddr := sdk.ValAddress(valPkey.PubKey().Address())

	db := dbm.NewMemDB()
	app := NewHaqq(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, map[int64]bool{}, DefaultNodeHome, 0, encoding.MakeConfig(ModuleBasics), simapp.EmptyAppOptions{})

	genesisState := NewDefaultGenesisState()
	stateBytes, err := json.MarshalIndent(genesisState, "", "  ")
	require.NoError(t, err)

	app.InitChain(
		abci.RequestInitChain{
			ChainId:       "evmos_9000-1",
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)
	app.Commit()

	handler := NewHaqqAnteHandlerDecorator(app.StakingKeeper, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return ctx, nil
	})

	t.Run("set validator", func(t *testing.T) {
		ctx := app.NewContext(true, tmproto.Header{})
		validator, err := stakingtypes.NewValidator(
			valAddr,
			valPkey.PubKey(),
			stakingtypes.NewDescription("validator", "", "", "", ""),
		)
		require.NoError(t, err)
		app.StakingKeeper.SetValidator(ctx, validator)
		app.StakingKeeper.SetNewValidatorByPowerIndex(ctx, validator)
	})

	t.Run("try add gov spend proposal", func(t *testing.T) {
		coins := sdk.NewCoins(sdk.NewCoin("aISLM", sdk.NewInt(100)))
		newAccPkey := ed25519.GenPrivKey()
		recipient := sdk.AccAddress(newAccPkey.PubKey().Address())

		// generate submit proposal
		cpsp := distrtypes.NewCommunityPoolSpendProposal("Test", "description", recipient, coins)
		sp, _ := govtypes.NewMsgSubmitProposal(cpsp, coins, recipient)

		/// build tx
		builder := app.GetTxConfig().NewTxBuilder()
		require.NoError(t, builder.SetMsgs(sp))
		require.NoError(t, builder.SetSignatures(signing.SignatureV2{
			PubKey: newAccPkey.PubKey(),
			Data:   nil,
		}))

		// run tx
		ctx := app.NewContext(true, tmproto.Header{Height: 1})
		_, err = handler(ctx, builder.GetTx(), true)

		require.Error(t, ErrCommunitySpendingComingLater, err)
	})

	t.Run("create validator", func(t *testing.T) {
		t.Run("from unknown address", func(t *testing.T) {
			newValPkey := ed25519.GenPrivKey()
			newValAddr := sdk.ValAddress(newValPkey.PubKey().Address())
			msg, err := stakingtypes.NewMsgCreateValidator(
				newValAddr,
				newValPkey.PubKey(),
				sdk.NewInt64Coin(sdk.DefaultBondDenom, 50),
				stakingtypes.NewDescription("testname", "", "", "", ""),
				stakingtypes.CommissionRates{},
				sdk.OneInt(),
			)
			require.NoError(t, err)

			builder := app.GetTxConfig().NewTxBuilder()
			require.NoError(t, builder.SetMsgs(msg))
			require.NoError(t, builder.SetSignatures(signing.SignatureV2{
				PubKey: newValPkey.PubKey(),
				Data:   nil,
			}))

			ctx := app.NewContext(true, tmproto.Header{Height: 1})
			_, err = handler(ctx, builder.GetTx(), true)

			t.Logf("### from unknown address %v ###", err)

			require.Error(t, ErrDelegationComingLater, err)
		})

		t.Run("from validator address", func(t *testing.T) {
			msg, err := stakingtypes.NewMsgCreateValidator(
				valAddr,
				valPkey.PubKey(),
				sdk.NewInt64Coin(sdk.DefaultBondDenom, 50),
				stakingtypes.NewDescription("testname", "", "", "", ""),
				stakingtypes.CommissionRates{},
				sdk.OneInt(),
			)
			require.NoError(t, err)

			builder := app.GetTxConfig().NewTxBuilder()
			require.NoError(t, builder.SetMsgs(msg))
			require.NoError(t, builder.SetSignatures(signing.SignatureV2{
				PubKey: valPkey.PubKey(),
				Data:   nil,
			}))

			ctx := app.NewContext(true, tmproto.Header{Height: 1})
			_, err = handler(ctx, builder.GetTx(), true)
			require.NoError(t, err)
		})
	})

	t.Run("create delegation", func(t *testing.T) {
		t.Run("from unknown address", func(t *testing.T) {
			delPkey := ed25519.GenPrivKey()
			delAddr := sdk.AccAddress(delPkey.PubKey().Address())

			msg := stakingtypes.NewMsgDelegate(delAddr, valAddr, sdk.NewCoin("aISLM", sdk.NewInt(10)))
			builder := app.GetTxConfig().NewTxBuilder()
			require.NoError(t, builder.SetMsgs(msg))
			require.NoError(t, builder.SetSignatures(signing.SignatureV2{
				PubKey: delPkey.PubKey(),
				Data:   nil,
			}))

			ctx := app.NewContext(true, tmproto.Header{Height: 1})
			_, err := handler(ctx, builder.GetTx(), true)

			require.Error(t, ErrDelegationComingLater, err)
		})

		t.Run("from validator address", func(t *testing.T) {
			delAddr := sdk.AccAddress(valPkey.PubKey().Address())

			msg := stakingtypes.NewMsgDelegate(delAddr, valAddr, sdk.NewCoin("aISLM", sdk.NewInt(10)))
			builder := app.GetTxConfig().NewTxBuilder()
			require.NoError(t, builder.SetMsgs(msg))
			require.NoError(t, builder.SetSignatures(signing.SignatureV2{
				PubKey: valPkey.PubKey(),
				Data:   nil,
			}))

			ctx := app.NewContext(true, tmproto.Header{Height: 1})
			_, err := handler(ctx, builder.GetTx(), true)

			require.NoError(t, err)
		})
	})
}
