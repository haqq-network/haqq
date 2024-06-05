package app_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/ibc-go/v8/testing/mock"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/encoding"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	"github.com/haqq-network/haqq/utils"
)

func TestExport(t *testing.T) {
	nw := network.NewUnitTestNetwork()
	exported, err := nw.App.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")

	require.NotEmpty(t, exported.AppState)
	require.NotEmpty(t, exported.Validators)
	require.Equal(t, int64(2), exported.Height)
	require.Equal(t, *app.DefaultConsensusParams, exported.ConsensusParams)
}

func TestPoA(t *testing.T) {
	// create public key
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err, "public key should be created without error")

	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, math.NewInt(100000000000000))),
	}

	chainID := utils.MainNetChainID + "-1"
	db := dbm.NewMemDB()
	logger := log.NewTestLogger(t)
	appHaqq := app.NewHaqq(
		logger.With("instance", "poa"),
		db, nil, true,
		map[int64]bool{}, app.DefaultNodeHome, 0,
		encoding.MakeConfig(app.ModuleBasics),
		simtestutil.NewAppOptionsWithFlagHome(app.DefaultNodeHome),
		baseapp.SetChainID(chainID),
	)

	genesisState := app.NewDefaultGenesisState()
	genesisState = app.GenesisStateWithValSet(appHaqq, genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)
	stateBytes, err := json.MarshalIndent(genesisState, "", "  ")
	require.NoError(t, err)

	// Initialize the chain
	_, err = appHaqq.InitChain(
		&abci.RequestInitChain{
			ChainId:       chainID,
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)
	require.NoError(t, err)

	_, err = appHaqq.Commit()
	require.NoError(t, err)

	ctx := appHaqq.NewUncachedContext(false, tmproto.Header{})
	validatorUpdates, err := appHaqq.StakingKeeper.BlockValidatorUpdates(ctx)
	require.NoError(t, err)
	require.Equal(t, len(validatorUpdates), 0)
}
