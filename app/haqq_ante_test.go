package app

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/ibc-go/v7/testing/mock"

	"github.com/evmos/evmos/v14/encoding"
	"github.com/haqq-network/haqq/utils"
)

func TestHaqqAnteHandlerDecorator(t *testing.T) {
	privVal := mock.NewPV()
	pubKey, _ := privVal.GetPubKey()

	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})

	//
	valAddr := sdk.ValAddress(pubKey.Address())
	valPkey := &ed25519.PrivKey{Key: privVal.PrivKey.Bytes()}

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdk.NewInt(100000000000000))),
	}

	chainID := utils.MainNetChainID + "-1"
	db := dbm.NewMemDB()
	app := NewHaqq(
		log.NewNopLogger(),
		db, nil, true, map[int64]bool{},
		DefaultNodeHome, 5,
		encoding.MakeConfig(ModuleBasics),
		simtestutil.NewAppOptionsWithFlagHome(DefaultNodeHome),
		baseapp.SetChainID(chainID),
	)

	genesisState := NewDefaultGenesisState()
	genesisState = GenesisStateWithValSet(app, genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)
	stateBytes, err := json.MarshalIndent(genesisState, "", "  ")
	require.NoError(t, err)

	// Initialize the chain
	app.InitChain(
		abci.RequestInitChain{
			ChainId:       chainID,
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

	// TODO Add new gov spend test
	t.Run("try add gov spend proposal - legacy", func(t *testing.T) {
		coins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdk.NewInt(100)))
		newAccPkey := ed25519.GenPrivKey()
		recipient := sdk.AccAddress(newAccPkey.PubKey().Address())

		// generate submit proposal
		cpsp := distrtypes.CommunityPoolSpendProposal{
			"Test", "description",
			recipient.String(), coins,
		}
		sp, _ := govtypes.NewMsgSubmitProposal(&cpsp, coins, recipient)

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

			require.NoError(t, err)
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

			msg := stakingtypes.NewMsgDelegate(delAddr, valAddr, sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10)))
			builder := app.GetTxConfig().NewTxBuilder()
			require.NoError(t, builder.SetMsgs(msg))
			require.NoError(t, builder.SetSignatures(signing.SignatureV2{
				PubKey: delPkey.PubKey(),
				Data:   nil,
			}))

			ctx := app.NewContext(true, tmproto.Header{Height: 1})
			_, err := handler(ctx, builder.GetTx(), true)

			require.NoError(t, err)
		})

		t.Run("from validator address", func(t *testing.T) {
			delAddr := sdk.AccAddress(valPkey.PubKey().Address())

			msg := stakingtypes.NewMsgDelegate(delAddr, valAddr, sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10)))
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
