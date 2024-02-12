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

func (suite *DenomTestSuite) TestDenomBaseNameFromID() {
	testCases := []struct {
		name         string
		ID           uint64
		expectedName string
	}{
		{
			name:         "Simple id",
			ID:           1,
			expectedName: "aLIQUID1",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := DenomBaseNameFromID(tc.ID)
			suite.Require().Equal(tc.expectedName, result)
		})
	}
}

func (suite *DenomTestSuite) TestDenomDisplayNameFromID() {
	testCases := []struct {
		name         string
		ID           uint64
		expectedName string
	}{
		{
			name:         "Simple id",
			ID:           1,
			expectedName: "LIQUID1",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := DenomDisplayNameFromID(tc.ID)
			suite.Require().Equal(tc.expectedName, result)
		})
	}
}
