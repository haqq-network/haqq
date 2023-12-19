package types

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
		sdk.NewCoin("aISLM", sdk.NewInt(10_000_000_000)),
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
				Params:    validParams,
				MaxSupply: sdk.NewCoin("aISLM", sdk.NewInt(10_000_000_000)),
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
			"invalid genesis - MaxSupply is nil",
			&GenesisState{
				Params:    validParams,
				MaxSupply: sdk.Coin{},
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
