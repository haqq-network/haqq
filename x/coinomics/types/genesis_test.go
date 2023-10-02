package types

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
)

type GenesisTestSuite struct {
	suite.Suite
}

func (suite *GenesisTestSuite) SetupTest() {
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) TestValidateGenesis() {
	// Team Address needs to be set manually at Genesis
	validParams := DefaultParams()

	newGen := NewGenesisState(
		validParams,
		sdk.NewDec(100),
		1,
		100,
		sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000)),
		sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000_0)),
		sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000_000)),
	)

	testCases := []struct {
		name     string
		genState *GenesisState
		expPass  bool
	}{
		{
			"empty genesis",
			&GenesisState{},
			false,
		},
		{
			"invalid default genesis",
			DefaultGenesisState(),
			true,
		},
		{
			"valid genesis constructor",
			&newGen,
			true,
		},
		{
			"valid genesis",
			&GenesisState{
				Params:            validParams,
				Inflation:         sdk.NewDec(100),
				Era:               1,
				EraStartedAtBlock: 100,
				EraTargetMint:     sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000)),
				EraClosingSupply:  sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000_00)),
				MaxSupply:         sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000_000)),
			},
			true,
		},
		{
			"invalid genesis",
			&GenesisState{
				Params: validParams,
			},
			false,
		},
		{
			"invalid genesis - EraTargetMint is nil",
			&GenesisState{
				Params:            validParams,
				Inflation:         sdk.NewDec(100),
				Era:               1,
				EraStartedAtBlock: 100,
				EraClosingSupply:  sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000_00)),
				MaxSupply:         sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000_000)),
			},
			false,
		},
		{
			"invalid genesis - EraClosingSupply is nil",
			&GenesisState{
				Params:            validParams,
				Inflation:         sdk.NewDec(100),
				Era:               1,
				EraStartedAtBlock: 100,
				EraTargetMint:     sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000)),
				MaxSupply:         sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000_000)),
			},
			false,
		},
		{
			"invalid genesis - MaxSupply is nil",
			&GenesisState{
				Params:            validParams,
				Inflation:         sdk.NewDec(100),
				Era:               1,
				EraStartedAtBlock: 100,
				EraTargetMint:     sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000)),
				MaxSupply:         sdk.NewCoin(utils.BaseDenom, sdk.NewInt(10_000_000_00)),
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.genState.Validate()
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}
