package shariahoracle_test

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"
	"github.com/cometbft/cometbft/version"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/app"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	feemarkettypes "github.com/haqq-network/haqq/x/feemarket/types"
	"github.com/haqq-network/haqq/x/shariahoracle"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
	"github.com/stretchr/testify/suite"
)

type GenesisTestSuite struct {
	suite.Suite
	ctx     sdk.Context
	app     *app.Haqq
	genesis types.GenesisState
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) SetupTest() {
	// consensus key
	consAddress := sdk.ConsAddress(utiltx.GenerateAddress().Bytes())

	suite.app, _ = app.Setup(false, feemarkettypes.DefaultGenesisState())
	suite.ctx = suite.app.BaseApp.NewContext(false, tmproto.Header{
		Height:          1,
		ChainID:         utils.TestEdge2ChainID + "-3",
		Time:            time.Now().UTC(),
		ProposerAddress: consAddress.Bytes(),

		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	})

	suite.genesis = *types.DefaultGenesisState()
}

func (suite *GenesisTestSuite) TestShariahOracleInitGenesis() {
	testCases := []struct {
		name         string
		genesisState types.GenesisState
	}{
		{
			"default genesis",
			*types.DefaultGenesisState(),
		},
		{
			"custom genesis",
			types.NewGenesisState(types.DefaultParams()),
		},
	}

	for _, tc := range testCases {

		suite.Require().NotPanics(func() {
			shariahoracle.InitGenesis(suite.ctx, suite.app.ShariaOracleKeeper, suite.app.AccountKeeper, tc.genesisState)
		})
		params := suite.app.ShariaOracleKeeper.GetParams(suite.ctx)

		suite.Require().Equal(tc.genesisState.Params, params)
	}
}

func (suite *GenesisTestSuite) TestShariahOracleExportGenesis() {
	testGenCases := []struct {
		name         string
		genesisState types.GenesisState
	}{
		{
			"default genesis",
			*types.DefaultGenesisState(),
		},
		{
			"custom genesis",
			types.NewGenesisState(types.DefaultParams()),
		},
	}

	for _, tc := range testGenCases {
		shariahoracle.InitGenesis(suite.ctx, suite.app.ShariaOracleKeeper, suite.app.AccountKeeper, tc.genesisState)
		suite.Require().NotPanics(func() {
			genesisExported := shariahoracle.ExportGenesis(suite.ctx, suite.app.ShariaOracleKeeper)
			params := suite.app.ShariaOracleKeeper.GetParams(suite.ctx)
			suite.Require().Equal(genesisExported.Params, params)
		})
	}
}
