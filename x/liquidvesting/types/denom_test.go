package types

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DenomTestSuite struct {
	suite.Suite
}

func TestDenomSuite(t *testing.T) {
	suite.Run(t, new(DenomTestSuite))
}

func (suite *DenomTestSuite) TestDenomName0FromID() {
	testCases := []struct {
		name         string
		ID           uint64
		expectedName string
	}{
		{
			name:         "Simple id",
			ID:           1,
			expectedName: "aLIQUIDDENOM1",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := DenomBaseNameFromID(tc.ID)
			suite.Require().Equal(tc.expectedName, result)
		})
	}
}

func (suite *DenomTestSuite) TestDenomName18FromID() {
	testCases := []struct {
		name         string
		ID           uint64
		expectedName string
	}{
		{
			name:         "Simple id",
			ID:           1,
			expectedName: "LIQUIDDENOM1",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := DenomDisplayNameFromID(tc.ID)
			suite.Require().Equal(tc.expectedName, result)
		})
	}
}
