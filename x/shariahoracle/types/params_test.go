package types_test

import (
	"testing"

	"github.com/haqq-network/haqq/x/shariahoracle/types"
	"github.com/stretchr/testify/suite"
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
		params   types.Params
		expError bool
	}{
		{"default", types.DefaultParams(), false},
		{
			"valid",
			types.NewParams("0xdac17f958d2ee523a2206206994597c13d831ec7"),
			false,
		},
		{
			"empty address",
			types.NewParams(""),
			true,
		},
		{
			"empty",
			types.Params{},
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
