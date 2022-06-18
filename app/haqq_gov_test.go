package app

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"
	"github.com/tharsis/ethermint/encoding"
)

func TestGovPoolSpendProposal(t *testing.T) {
	pkey := ed25519.GenPrivKey()
	addr := sdk.AccAddress(pkey.PubKey().Address())
	prop := types.NewCommunityPoolSpendProposal("Test", "description", addr, sdk.NewCoins(
		sdk.NewCoin("ISLM", sdk.NewInt(1)),
	))

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

	ctx := app.NewContext(true, tmproto.Header{})
	_, err = app.GovKeeper.SubmitProposal(ctx, prop)
	require.Contains(t, err.Error(), ErrFundGovComingLater.Error())
}
