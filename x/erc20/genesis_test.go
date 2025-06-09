package erc20_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"
	"github.com/cometbft/cometbft/version"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/erc20"
	"github.com/haqq-network/haqq/x/erc20/types"
	feemarkettypes "github.com/haqq-network/haqq/x/feemarket/types"
)

type GenesisTestSuite struct {
	suite.Suite
	ctx     sdk.Context
	app     *app.Haqq
	genesis types.GenesisState
}

const osmoERC20ContractAddr = "0x5dCA2483280D9727c80b5518faC4556617fb19ZZ"

var osmoDenomTrace = transfertypes.DenomTrace{
	BaseDenom: "uosmo",
	Path:      "transfer/channel-0",
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) SetupTest() {
	// consensus key
	consAddress := sdk.ConsAddress(utiltx.GenerateAddress().Bytes())

	chainID := utils.TestEdge2ChainID + "-3"
	suite.app, _ = app.Setup(false, feemarkettypes.DefaultGenesisState(), chainID)
	suite.ctx = suite.app.BaseApp.NewContext(false, tmproto.Header{
		Height:          1,
		ChainID:         chainID,
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

func (suite *GenesisTestSuite) TestERC20InitGenesis() {
	testCases := []struct {
		name         string
		genesisState types.GenesisState
	}{
		{
			name:         "empty genesis",
			genesisState: types.GenesisState{},
		},
		{
			name:         "default genesis",
			genesisState: *types.DefaultGenesisState(),
		},
		{
			name: "custom genesis",
			genesisState: types.NewGenesisState(
				types.DefaultParams(),
				[]types.TokenPair{
					{
						Erc20Address:  osmoERC20ContractAddr,
						Denom:         osmoDenomTrace.IBCDenom(),
						Enabled:       true,
						ContractOwner: types.OWNER_MODULE,
					},
				},
			),
		},
	}

	for _, tc := range testCases {
		gen := network.CustomGenesisState{
			types.ModuleName: &tc.genesisState, // #nosec G601
		}
		nw := network.NewUnitTestNetwork(
			network.WithCustomGenesis(gen),
		)

		params := nw.App.Erc20Keeper.GetParams(nw.GetContext())

		tokenPairs := nw.App.Erc20Keeper.GetTokenPairs(nw.GetContext())
		suite.Require().Equal(tc.genesisState.Params, params)
		if len(tokenPairs) > 0 {
			suite.Require().Equal(tc.genesisState.TokenPairs, tokenPairs, tc.name)
		} else {
			suite.Require().Len(tc.genesisState.TokenPairs, 0, tc.name)
		}
	}
}

func (suite *GenesisTestSuite) TestErc20ExportGenesis() {
	testGenCases := []struct {
		name         string
		genesisState types.GenesisState
	}{
		{
			name:         "empty genesis",
			genesisState: types.GenesisState{},
		},
		{
			name:         "default genesis",
			genesisState: *types.DefaultGenesisState(),
		},
		{
			name: "custom genesis",
			genesisState: types.NewGenesisState(
				types.DefaultParams(),
				[]types.TokenPair{
					{
						Erc20Address:  osmoERC20ContractAddr,
						Denom:         osmoDenomTrace.IBCDenom(),
						Enabled:       true,
						ContractOwner: types.OWNER_MODULE,
					},
				},
			),
		},
	}

	for _, tc := range testGenCases {
		erc20.InitGenesis(suite.ctx, suite.app.Erc20Keeper, suite.app.AccountKeeper, tc.genesisState)
		suite.Require().NotPanics(func() {
			genesisExported := erc20.ExportGenesis(suite.ctx, suite.app.Erc20Keeper)
			params := suite.app.Erc20Keeper.GetParams(suite.ctx)
			suite.Require().Equal(genesisExported.Params, params)

			tokenPairs := suite.app.Erc20Keeper.GetTokenPairs(suite.ctx)
			if len(tokenPairs) > 0 {
				suite.Require().Equal(genesisExported.TokenPairs, tokenPairs)
			} else {
				suite.Require().Len(genesisExported.TokenPairs, 0)
			}
		})
	}
}
