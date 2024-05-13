package liquidvesting_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"
	"github.com/cometbft/cometbft/version"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/suite"

	"github.com/haqq-network/haqq/app"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	feemarkettypes "github.com/haqq-network/haqq/x/feemarket/types"
	"github.com/haqq-network/haqq/x/liquidvesting"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
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

func (suite *GenesisTestSuite) TestLiquidVestingInitGenesis() {
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
			types.NewGenesisState(
				types.DefaultParams(),
				1,
				[]types.Denom{
					{
						BaseDenom:     "aLIQUID0",
						DisplayDenom:  "LIQUID0",
						OriginalDenom: "aISLM",
						StartTime:     time.Unix(171326008, 0).UTC(),
						EndTime:       time.Unix(1713260189, 0).UTC(),
						LockupPeriods: sdkvesting.Periods{{
							Length: 100,
							Amount: sdk.NewCoins(sdk.NewCoin("aISLM", sdkmath.NewIntWithDecimal(1, 18))),
						}},
					},
				}),
		},
	}

	for _, tc := range testCases {

		suite.Require().NotPanics(func() {
			liquidvesting.InitGenesis(suite.ctx, suite.app.LiquidVestingKeeper, tc.genesisState)
		})
		params := suite.app.LiquidVestingKeeper.GetParams(suite.ctx)

		denoms := suite.app.LiquidVestingKeeper.GetAllDenoms(suite.ctx)
		suite.Require().Equal(tc.genesisState.Params, params)
		if len(denoms) > 0 {
			suite.Require().Equal(tc.genesisState.Denoms, denoms)
		} else {
			suite.Require().Len(tc.genesisState.Denoms, 0)
		}
	}
}

func (suite *GenesisTestSuite) TestLiquidVestingExportGenesis() {
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
			types.NewGenesisState(
				types.DefaultParams(),
				1,
				[]types.Denom{
					{
						BaseDenom:     "aLIQUID0",
						DisplayDenom:  "LIQUID0",
						OriginalDenom: "aISLM",
						StartTime:     time.Unix(1713260089, 0).UTC(),
						EndTime:       time.Unix(1713260189, 0).UTC(),
						LockupPeriods: sdkvesting.Periods{{
							Length: 100,
							Amount: sdk.NewCoins(sdk.NewCoin("aISLM", sdkmath.NewIntWithDecimal(1, 18))),
						}},
					},
				}),
		},
	}

	for _, tc := range testGenCases {
		liquidvesting.InitGenesis(suite.ctx, suite.app.LiquidVestingKeeper, tc.genesisState)
		suite.Require().NotPanics(func() {
			genesisExported := liquidvesting.ExportGenesis(suite.ctx, suite.app.LiquidVestingKeeper)
			params := suite.app.LiquidVestingKeeper.GetParams(suite.ctx)
			suite.Require().Equal(genesisExported.Params, params)

			denoms := suite.app.LiquidVestingKeeper.GetAllDenoms(suite.ctx)
			if len(denoms) > 0 {
				suite.Require().Equal(genesisExported.Denoms, denoms)
			} else {
				suite.Require().Len(genesisExported.Denoms, 0)
			}
		})
	}
}
