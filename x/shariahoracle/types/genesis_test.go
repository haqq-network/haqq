package types_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/haqq-network/haqq/x/shariahoracle/types"
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
	newGen := types.NewGenesisState(types.DefaultParams())

	testCases := []struct {
		name     string
		genState *types.GenesisState
		expPass  bool
	}{
		{
			name:     "valid genesis constructor",
			genState: &newGen,
			expPass:  true,
		},
		{
			name:     "default",
			genState: types.DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "valid genesis",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
			},
			expPass: true,
		},
		{
			// Voting period can't be zero
			name:     "empty genesis",
			genState: &types.GenesisState{},
			expPass:  false,
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
