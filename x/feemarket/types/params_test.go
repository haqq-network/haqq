package types

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
)

type ParamsTestSuite struct {
	suite.Suite
}

func TestParamsTestSuite(t *testing.T) {
	suite.Run(t, new(ParamsTestSuite))
}

func (suite *ParamsTestSuite) TestParamsValidate() {
	testCases := []struct {
		name     string
		params   Params
		expError bool
	}{
		{"default", DefaultParams(), false},
		{
			"valid",
			NewParams(true, 7, 3, 2000000000, int64(544435345345435345), sdkmath.LegacyNewDecWithPrec(20, 4), DefaultMinGasMultiplier),
			false,
		},
		{
			"empty",
			Params{},
			true,
		},
		{
			"base fee change denominator is 0 ",
			NewParams(true, 0, 3, 2000000000, int64(544435345345435345), sdkmath.LegacyNewDecWithPrec(20, 4), DefaultMinGasMultiplier),
			true,
		},
		{
			"invalid: negative base fee",
			Params{
				NoBaseFee:                false,
				BaseFeeChangeDenominator: 8,
				ElasticityMultiplier:     2,
				BaseFee:                  sdkmath.NewInt(-1),
				EnableHeight:             0,
				MinGasPrice:              DefaultMinGasPrice,
				MinGasMultiplier:         DefaultMinGasMultiplier,
			},
			true,
		},
		{
			"invalid: negative enable height",
			NewParams(false, 8, 2, 1000000000, int64(-1), DefaultMinGasPrice, DefaultMinGasMultiplier),
			true,
		},
		{
			"invalid: min gas price negative",
			NewParams(true, 7, 3, 2000000000, int64(544435345345435345), sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(-1)), DefaultMinGasMultiplier),
			true,
		},
		{
			"valid: min gas multiplier zero",
			NewParams(true, 7, 3, 2000000000, int64(544435345345435345), DefaultMinGasPrice, sdkmath.LegacyZeroDec()),
			false,
		},
		{
			"invalid: min gas multiplier is negative",
			NewParams(true, 7, 3, 2000000000, int64(544435345345435345), DefaultMinGasPrice, sdkmath.LegacyNewDecWithPrec(-5, 1)),
			true,
		},
		{
			"invalid: min gas multiplier bigger than 1",
			NewParams(true, 7, 3, 2000000000, int64(544435345345435345), sdkmath.LegacyNewDecWithPrec(20, 4), sdkmath.LegacyNewDec(2)),
			true,
		},
	}

	for _, tc := range testCases {
		err := tc.params.Validate()

		if tc.expError {
			suite.Require().Error(err, tc.name)
		} else {
			suite.Require().NoError(err, tc.name)
		}
	}
}

func (suite *ParamsTestSuite) TestParamsValidatePriv() {
	suite.Require().Error(validateBool(2))
	suite.Require().NoError(validateBool(true))
	suite.Require().Error(validateBaseFeeChangeDenominator(0))
	suite.Require().Error(validateBaseFeeChangeDenominator(uint32(0)))
	suite.Require().NoError(validateBaseFeeChangeDenominator(uint32(7)))
	suite.Require().Error(validateElasticityMultiplier(""))
	suite.Require().NoError(validateElasticityMultiplier(uint32(2)))
	suite.Require().Error(validateBaseFee(""))
	suite.Require().Error(validateBaseFee(int64(2000000000)))
	suite.Require().Error(validateBaseFee(sdkmath.NewInt(-2000000000)))
	suite.Require().NoError(validateBaseFee(sdkmath.NewInt(2000000000)))
	suite.Require().Error(validateEnableHeight(""))
	suite.Require().Error(validateEnableHeight(int64(-544435345345435345)))
	suite.Require().NoError(validateEnableHeight(int64(544435345345435345)))
	suite.Require().Error(validateMinGasPrice(sdkmath.LegacyDec{}))
	suite.Require().Error(validateMinGasMultiplier(sdkmath.LegacyNewDec(-5)))
	suite.Require().Error(validateMinGasMultiplier(sdkmath.LegacyDec{}))
	suite.Require().Error(validateMinGasMultiplier(""))
}

func (suite *ParamsTestSuite) TestParamKeyTable() {
	suite.Require().NotPanics(func() {
		_ = ParamKeyTable()
	})
}

func (suite *ParamsTestSuite) TestParamSetPairs() {
	p := DefaultParams()
	pairs := p.ParamSetPairs()
	suite.Require().Len(pairs, 7)
}

func (suite *ParamsTestSuite) TestIsBaseFeeEnabled() {
	testCases := []struct {
		name       string
		params     Params
		height     int64
		expEnabled bool
	}{
		{
			"enabled: NoBaseFee=false, height >= EnableHeight",
			NewParams(false, 8, 2, 1000000000, 0, DefaultMinGasPrice, DefaultMinGasMultiplier),
			10,
			true,
		},
		{
			"disabled: NoBaseFee=true",
			NewParams(true, 8, 2, 1000000000, 0, DefaultMinGasPrice, DefaultMinGasMultiplier),
			10,
			false,
		},
		{
			"disabled: height < EnableHeight",
			NewParams(false, 8, 2, 1000000000, 100, DefaultMinGasPrice, DefaultMinGasMultiplier),
			5,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := tc.params.IsBaseFeeEnabled(tc.height)
			suite.Require().Equal(tc.expEnabled, result)
		})
	}
}

func (suite *ParamsTestSuite) TestParamsValidateMinGasPrice() {
	testCases := []struct {
		name     string
		value    any
		expError bool
	}{
		{"default", DefaultParams().MinGasPrice, false},
		{"valid", sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(1)), false},
		{"invalid - wrong type - bool", false, true},
		{"invalid - wrong type - string", "", true},
		{"invalid - wrong type - int64", int64(123), true},
		{"invalid - wrong type - sdkmath.Int", sdkmath.NewInt(1), true},
		{"invalid - is nil", nil, true},
		{"invalid - is negative", sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(-1)), true},
	}

	for _, tc := range testCases {
		err := validateMinGasPrice(tc.value)

		if tc.expError {
			suite.Require().Error(err, tc.name)
		} else {
			suite.Require().NoError(err, tc.name)
		}
	}
}
