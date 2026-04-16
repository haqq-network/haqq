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

func (suite *DenomTestSuite) TestDenomIDFromBaseName() {
	testCases := []struct {
		name        string
		baseName    string
		expectedID  uint64
		expectError bool
	}{
		{
			name:        "ID zero",
			baseName:    "aLIQUID0",
			expectedID:  0,
			expectError: false,
		},
		{
			name:        "ID one",
			baseName:    "aLIQUID1",
			expectedID:  1,
			expectError: false,
		},
		{
			name:        "ID 999",
			baseName:    "aLIQUID999",
			expectedID:  999,
			expectError: false,
		},
		{
			name:        "wrong prefix LIQUID0",
			baseName:    "LIQUID0",
			expectError: true,
		},
		{
			name:        "prefix only with no number",
			baseName:    "aLIQUID",
			expectError: true,
		},
		{
			name:        "completely invalid string",
			baseName:    "invalid",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, err := DenomIDFromBaseName(tc.baseName)
			if tc.expectError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expectedID, result)
			}
		})
	}
}
