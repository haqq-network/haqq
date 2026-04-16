package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
)

type GenesisTypesTestSuite struct {
	suite.Suite
}

func TestGenesisTypesSuite(t *testing.T) {
	suite.Run(t, new(GenesisTypesTestSuite))
}

// makeValidDenom creates a valid Denom with the given ID and a counter that accepts it.
func makeValidDenom(id uint64) Denom {
	now := time.Now().UTC()
	return Denom{
		BaseDenom:     DenomBaseNameFromID(id),
		DisplayDenom:  DenomDisplayNameFromID(id),
		OriginalDenom: "aISLM",
		StartTime:     now,
		EndTime:       now.Add(time.Hour),
		LockupPeriods: sdkvesting.Periods{
			{Length: 3600, Amount: sdk.NewCoins(sdk.NewInt64Coin("aISLM", 1000))},
		},
	}
}

func (suite *GenesisTypesTestSuite) TestNewGenesisState() {
	params := DefaultParams()
	denoms := []Denom{makeValidDenom(0)}
	var counter uint64 = 1

	gs := NewGenesisState(params, counter, denoms)

	suite.Require().Equal(params, gs.Params)
	suite.Require().Equal(counter, gs.DenomCounter)
	suite.Require().Equal(denoms, gs.Denoms)
}

func (suite *GenesisTypesTestSuite) TestDefaultGenesisState() {
	gs := DefaultGenesisState()

	suite.Require().NotNil(gs)
	suite.Require().Equal(DefaultParams(), gs.Params)
	suite.Require().Equal(uint64(0), gs.DenomCounter)
	suite.Require().Empty(gs.Denoms)
}

func (suite *GenesisTypesTestSuite) TestGenesisStateValidate() {
	now := time.Now().UTC()

	testCases := []struct {
		name        string
		gs          GenesisState
		expectError bool
		errContains string
	}{
		{
			name:        "valid default state",
			gs:          *DefaultGenesisState(),
			expectError: false,
		},
		{
			name: "valid state with denoms",
			gs: NewGenesisState(
				DefaultParams(),
				1,
				[]Denom{makeValidDenom(0)},
			),
			expectError: false,
		},
		{
			name: "invalid params: negative MinimumLiquidationAmount",
			gs: NewGenesisState(
				NewParams(math.NewInt(-1), true),
				0,
				[]Denom{},
			),
			expectError: true,
			errContains: "positive",
		},
		{
			name: "invalid params: zero MinimumLiquidationAmount",
			gs: NewGenesisState(
				NewParams(math.NewInt(0), true),
				0,
				[]Denom{},
			),
			expectError: true,
		},
		{
			name: "denom endTime before startTime",
			gs: NewGenesisState(
				DefaultParams(),
				1,
				[]Denom{
					{
						BaseDenom:     DenomBaseNameFromID(0),
						DisplayDenom:  DenomDisplayNameFromID(0),
						OriginalDenom: "aISLM",
						StartTime:     now.Add(time.Hour),
						EndTime:       now, // EndTime before StartTime
						LockupPeriods: sdkvesting.Periods{
							{Length: 3600, Amount: sdk.NewCoins(sdk.NewInt64Coin("aISLM", 1000))},
						},
					},
				},
			),
			expectError: true,
			errContains: "start time",
		},
		{
			name: "denom period with negative amount",
			gs: NewGenesisState(
				DefaultParams(),
				1,
				[]Denom{
					{
						BaseDenom:     DenomBaseNameFromID(0),
						DisplayDenom:  DenomDisplayNameFromID(0),
						OriginalDenom: "aISLM",
						StartTime:     now,
						EndTime:       now.Add(time.Hour),
						LockupPeriods: sdkvesting.Periods{
							{Length: 3600, Amount: sdk.Coins{sdk.Coin{Denom: "aISLM", Amount: math.NewInt(-100)}}},
						},
					},
				},
			),
			expectError: true,
			errContains: "negative",
		},
		{
			name: "duplicate denom base names",
			gs: NewGenesisState(
				DefaultParams(),
				2,
				[]Denom{
					makeValidDenom(0),
					{
						BaseDenom:     DenomBaseNameFromID(0), // duplicate
						DisplayDenom:  DenomDisplayNameFromID(1),
						OriginalDenom: "aISLM",
						StartTime:     now,
						EndTime:       now.Add(time.Hour),
						LockupPeriods: sdkvesting.Periods{
							{Length: 3600, Amount: sdk.NewCoins(sdk.NewInt64Coin("aISLM", 1000))},
						},
					},
				},
			),
			expectError: true,
			errContains: "duplicated",
		},
		{
			name: "denom with invalid base name",
			gs: NewGenesisState(
				DefaultParams(),
				1,
				[]Denom{
					{
						BaseDenom:     "invalidbasename",
						DisplayDenom:  DenomDisplayNameFromID(0),
						OriginalDenom: "aISLM",
						StartTime:     now,
						EndTime:       now.Add(time.Hour),
						LockupPeriods: sdkvesting.Periods{
							{Length: 3600, Amount: sdk.NewCoins(sdk.NewInt64Coin("aISLM", 1000))},
						},
					},
				},
			),
			expectError: true,
			errContains: "invalid denom base name",
		},
		{
			name: "denom ID >= denomCounter",
			gs: NewGenesisState(
				DefaultParams(),
				1,
				[]Denom{makeValidDenom(1)},
			),
			expectError: true,
			errContains: "lower or equal",
		},
		{
			name: "denom ID equal to denomCounter (boundary)",
			gs: NewGenesisState(
				DefaultParams(),
				0,
				[]Denom{makeValidDenom(0)},
			),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.gs.Validate()
			if tc.expectError {
				suite.Require().Error(err)
				if tc.errContains != "" {
					suite.Require().Contains(err.Error(), tc.errContains)
				}
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
