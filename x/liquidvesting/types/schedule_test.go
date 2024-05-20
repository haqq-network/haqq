package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/suite"
)

type ScheduleTestSuite struct {
	suite.Suite
}

func TestScheduleSuite(t *testing.T) {
	suite.Run(t, new(ScheduleTestSuite))
}

func (suite *ScheduleTestSuite) TestSubtractAmountFromPeriods() {
	testCases := []struct {
		name              string
		minuendPeriods    sdkvesting.Periods
		subtrahend        sdk.Coin
		expectedDecreased sdkvesting.Periods
		expectedDiff      sdkvesting.Periods
		expectError       bool
	}{
		{
			name: " OK Standard subtraction without residue",
			minuendPeriods: []sdkvesting.Period{
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(300)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(300)))},
			},
			subtrahend: sdk.NewCoin("test", math.NewInt(200)),
			expectedDecreased: []sdkvesting.Period{
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(200)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(200)))},
			},
			expectedDiff: []sdkvesting.Period{
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(100)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(100)))},
			},
			expectError: false,
		},
		{
			name: "OK Standard subtraction with residue",
			minuendPeriods: []sdkvesting.Period{
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(10)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(20)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(30)))},
			},
			subtrahend: sdk.NewCoin("test", math.NewInt(20)),
			expectedDecreased: []sdkvesting.Period{
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(7)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(14)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(19)))},
			},
			expectedDiff: []sdkvesting.Period{
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(3)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(6)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(11)))},
			},
			expectError: false,
		},
		{
			name: "OK Standard subtraction with residue and little last period to hold whole residue",
			minuendPeriods: []sdkvesting.Period{
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(100)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(200)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(2)))},
			},
			subtrahend: sdk.NewCoin("test", math.NewInt(220)),
			expectedDecreased: []sdkvesting.Period{
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(28)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(54)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(0)))},
			},
			expectedDiff: []sdkvesting.Period{
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(72)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(146)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(2)))},
			},
			expectError: false,
		},
		{
			name: "FAIL Subtrahend is bigger than total periods amount",
			minuendPeriods: []sdkvesting.Period{
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(10)))},
				{Amount: sdk.NewCoins(sdk.NewCoin("test", math.NewInt(20)))},
			},
			subtrahend:        sdk.NewCoin("test", math.NewInt(50)),
			expectedDecreased: []sdkvesting.Period{},
			expectedDiff:      []sdkvesting.Period{},
			expectError:       true,
		},
		{
			name:              "FAIL Subtract zero from empty periods",
			minuendPeriods:    []sdkvesting.Period{},
			subtrahend:        sdk.NewCoin("test", math.NewInt(0)),
			expectedDecreased: []sdkvesting.Period{},
			expectedDiff:      []sdkvesting.Period{},
			expectError:       true,
		},
		{
			name:              "FAIL Subtrahend is bigger than total periods amount, and periods are empty",
			minuendPeriods:    []sdkvesting.Period{},
			subtrahend:        sdk.NewCoin("test", math.NewInt(50)),
			expectedDecreased: []sdkvesting.Period{},
			expectedDiff:      []sdkvesting.Period{},
			expectError:       true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			decreased, diff, err := SubtractAmountFromPeriods(tc.minuendPeriods, tc.subtrahend)

			if tc.expectError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.Require().Equal(tc.expectedDecreased, decreased)
				suite.Require().Equal(tc.expectedDiff, diff)
			}
		})
	}
}

func (suite *ScheduleTestSuite) TestExtractUpcomingPeriods() {
	testCases := []struct {
		name            string
		startDate       int64
		endDate         int64
		periods         sdkvesting.Periods
		readTime        int64
		expectedPeriods sdkvesting.Periods
	}{
		{
			name:      "Standard extraction",
			startDate: 1700000000,
			endDate:   1700000400,
			periods: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
				{Length: 100},
			},
			readTime: 1700000200,
			expectedPeriods: []sdkvesting.Period{
				{Length: 100},
			},
		},
		{
			name:      "No upcoming periods",
			startDate: 1700000000,
			endDate:   1700000400,
			periods: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
				{Length: 100},
			},
			readTime:        1700000500,
			expectedPeriods: []sdkvesting.Period{},
		},
		{
			name:      "Full extraction",
			startDate: 1700000000,
			endDate:   1700000200,
			periods: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
			},
			readTime: 1600000000,
			expectedPeriods: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := ExtractUpcomingPeriods(tc.startDate, tc.endDate, tc.periods, tc.readTime)
			suite.Require().Equal(tc.expectedPeriods, result)
		})
	}
}

func (suite *ScheduleTestSuite) TestReplacePeriodsTail() {
	testCases := []struct {
		name            string
		periods         sdkvesting.Periods
		replacement     sdkvesting.Periods
		expectedPeriods sdkvesting.Periods
	}{
		{
			name: "Simple replacement",
			periods: []sdkvesting.Period{
				{Length: 200},
				{Length: 200},
				{Length: 200},
			},
			replacement: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
			},
			expectedPeriods: []sdkvesting.Period{
				{Length: 200},
				{Length: 100},
				{Length: 100},
			},
		},
		{
			name: "Full replacement",
			periods: []sdkvesting.Period{
				{Length: 200},
				{Length: 200},
				{Length: 200},
			},
			replacement: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
				{Length: 100},
			},
			expectedPeriods: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
				{Length: 100},
			},
		},
		{
			name: "Over replacement",
			periods: []sdkvesting.Period{
				{Length: 200},
				{Length: 200},
				{Length: 200},
			},
			replacement: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
				{Length: 100},
				{Length: 100},
			},
			expectedPeriods: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
				{Length: 100},
				{Length: 100},
			},
		},
		{
			name: "No replacement",
			periods: []sdkvesting.Period{
				{Length: 200},
				{Length: 200},
				{Length: 200},
			},
			replacement: []sdkvesting.Period{},
			expectedPeriods: []sdkvesting.Period{
				{Length: 200},
				{Length: 200},
				{Length: 200},
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := ReplacePeriodsTail(tc.periods, tc.replacement)
			suite.Require().Equal(tc.expectedPeriods, result)
		})
	}
}

func (suite *ScheduleTestSuite) TestCurrentPeriodShift() {
	testCases := []struct {
		name          string
		startTime     int64
		currentTime   int64
		periods       sdkvesting.Periods
		expectedShift int64
	}{
		{
			name:        "Standard shift",
			startTime:   1700000000,
			currentTime: 1700000150,
			periods: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
			},
			expectedShift: 50,
		},
		{
			name:        "Total length is less than start time and current time diff",
			startTime:   1700000000,
			currentTime: 1700000250,
			periods: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
			},
			expectedShift: 0,
		},
		{
			name:        "Current time before start time",
			startTime:   1700000000,
			currentTime: 1600000000,
			periods: []sdkvesting.Period{
				{Length: 100},
				{Length: 100},
			},
			expectedShift: 0,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			shift := CurrentPeriodShift(tc.startTime, tc.currentTime, tc.periods)
			suite.Require().Equal(tc.expectedShift, shift)
		})
	}
}
