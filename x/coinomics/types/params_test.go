package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/suite"
)

type ParamsTestSuite struct {
	suite.Suite
}

func TestParamsTestSuite(t *testing.T) {
	suite.Run(t, new(ParamsTestSuite))
}

func (suite *ParamsTestSuite) TestNewParams() {
	mintDenom := "aISLM"
	rewardCoeff := sdkmath.LegacyNewDecWithPrec(78, 1)
	p := NewParams(mintDenom, rewardCoeff, true)
	suite.Require().Equal(mintDenom, p.MintDenom)
	suite.Require().Equal(rewardCoeff, p.RewardCoefficient)
	suite.Require().True(p.EnableCoinomics)

	p2 := NewParams("uatom", sdkmath.LegacyZeroDec(), false)
	suite.Require().Equal("uatom", p2.MintDenom)
	suite.Require().False(p2.EnableCoinomics)
}

func (suite *ParamsTestSuite) TestDefaultParams() {
	p := DefaultParams()
	suite.Require().Equal(DefaultMintDenom, p.MintDenom)
	suite.Require().True(p.EnableCoinomics)
	suite.Require().Equal(sdkmath.LegacyNewDecWithPrec(78, 1), p.RewardCoefficient)
}

func (suite *ParamsTestSuite) TestParamKeyTable() {
	suite.Require().NotPanics(func() {
		_ = ParamKeyTable()
	})
}

func (suite *ParamsTestSuite) TestParamSetPairs() {
	p := DefaultParams()
	pairs := p.ParamSetPairs()
	suite.Require().Len(pairs, 3)
}

func (suite *ParamsTestSuite) TestValidateMintDenom() {
	testCases := []struct {
		name        string
		value       any
		expectError bool
	}{
		{"valid denom", "aISLM", false},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"wrong type int", 42, true},
		{"wrong type bool", true, true},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := validateMintDenom(tc.value)
			if tc.expectError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *ParamsTestSuite) TestValidateRewardCoefficient() {
	testCases := []struct {
		name        string
		value       any
		expectError bool
	}{
		{"valid dec", sdkmath.LegacyNewDecWithPrec(78, 1), false},
		{"zero dec", sdkmath.LegacyZeroDec(), false},
		{"wrong type string", "0.78", true},
		{"wrong type int", 78, true},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := validateRewardCoefficient(tc.value)
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
		{"true value", true, false},
		{"false value", false, false},
		{"wrong type string", "true", true},
		{"wrong type int", 1, true},
		{"wrong type math.Int", sdkmath.NewInt(1), true},
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

func (suite *ParamsTestSuite) TestParamsValidate() {
	testCases := []struct {
		name        string
		params      Params
		expectError bool
	}{
		{"valid default params", DefaultParams(), false},
		{"valid custom params", NewParams("aISLM", sdkmath.LegacyNewDecWithPrec(5, 1), false), false},
		{"empty mint denom", NewParams("", sdkmath.LegacyNewDecWithPrec(78, 1), true), true},
		{"whitespace mint denom", NewParams("   ", sdkmath.LegacyNewDecWithPrec(78, 1), true), true},
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
