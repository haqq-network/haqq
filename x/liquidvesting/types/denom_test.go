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

func (suite *DenomTestSuite) TestDenomNameFromID() {
	testCases := []struct {
		name         string
		ID           uint64
		expectedName string
	}{
		{
			name:         "Simple id",
			ID:           1,
			expectedName: "liquidDenom1",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := DenomNameFromID(tc.ID)
			suite.Require().Equal(tc.expectedName, result)
		})
	}
}
