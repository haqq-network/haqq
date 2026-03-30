package types

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"
)

type ParamsTestSuite struct {
	suite.Suite
}

func TestParamsSuite(t *testing.T) {
	suite.Run(t, new(ParamsTestSuite))
}

func (suite *ParamsTestSuite) TestNewParams() {
	amount := math.NewInt(5000)
	p := NewParams(amount, true)
	suite.Require().Equal(amount, p.MinimumLiquidationAmount)
	suite.Require().True(p.EnableLiquidVesting)

	p2 := NewParams(math.NewInt(1), false)
	suite.Require().Equal(math.NewInt(1), p2.MinimumLiquidationAmount)
	suite.Require().False(p2.EnableLiquidVesting)
}

func (suite *ParamsTestSuite) TestDefaultParams() {
	p := DefaultParams()
	suite.Require().Equal(DefaultMinimumLiquidationAmount, p.MinimumLiquidationAmount)
	suite.Require().Equal(DefaultEnableLiquidVesting, p.EnableLiquidVesting)
	suite.Require().True(p.MinimumLiquidationAmount.IsPositive())
}

func (suite *ParamsTestSuite) TestParamsValidate() {
	testCases := []struct {
		name        string
		params      Params
		expectError bool
	}{
		{
			name:        "valid params",
			params:      DefaultParams(),
			expectError: false,
		},
		{
			name:        "valid custom params",
			params:      NewParams(math.NewInt(1), false),
			expectError: false,
		},
		{
			name:        "zero amount",
			params:      NewParams(math.NewInt(0), true),
			expectError: true,
		},
		{
			name:        "negative amount",
			params:      NewParams(math.NewInt(-1), true),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.params.Validate()
			if tc.expectError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *ParamsTestSuite) TestValidateMathIntPositive() {
	testCases := []struct {
		name        string
		value       any
		expectError bool
	}{
		{
			name:        "valid positive int",
			value:       math.NewInt(1000),
			expectError: false,
		},
		{
			name:        "one",
			value:       math.NewInt(1),
			expectError: false,
		},
		{
			name:        "zero int",
			value:       math.NewInt(0),
			expectError: true,
		},
		{
			name:        "negative int",
			value:       math.NewInt(-100),
			expectError: true,
		},
		{
			name:        "wrong type string",
			value:       "not-a-math-int",
			expectError: true,
		},
		{
			name:        "wrong type int",
			value:       42,
			expectError: true,
		},
		{
			name:        "wrong type bool",
			value:       true,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := validateMathIntPositive(tc.value)
			if tc.expectError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *ParamsTestSuite) TestValidateBool() {
	testCases := []struct {
		name        string
		value       any
		expectError bool
	}{
		{
			name:        "true value",
			value:       true,
			expectError: false,
		},
		{
			name:        "false value",
			value:       false,
			expectError: false,
		},
		{
			name:        "wrong type string",
			value:       "true",
			expectError: true,
		},
		{
			name:        "wrong type int",
			value:       1,
			expectError: true,
		},
		{
			name:        "wrong type math.Int",
			value:       math.NewInt(1),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := validateBool(tc.value)
			if tc.expectError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *ParamsTestSuite) TestParamKeyTable() {
	// Verify no panic when calling ParamKeyTable
	suite.Require().NotPanics(func() {
		_ = ParamKeyTable()
	})
}

func (suite *ParamsTestSuite) TestParamSetPairs() {
	p := DefaultParams()
	pairs := p.ParamSetPairs()
	suite.Require().Len(pairs, 2)
}
