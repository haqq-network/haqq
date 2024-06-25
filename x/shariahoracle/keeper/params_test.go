package keeper_test

import (
	"reflect"

	"github.com/haqq-network/haqq/x/shariahoracle/types"
)

func (suite *KeeperTestSuite) TestParams() {
	params := suite.app.ShariahOracleKeeper.GetParams(suite.ctx)
	suite.app.ShariahOracleKeeper.SetParams(suite.ctx, params) //nolint:errcheck

	testCases := []struct {
		name      string
		paramsFun func() interface{}
		getFun    func() interface{}
		expected  bool
	}{
		{
			"success - Checks if the default params are set correctly",
			func() interface{} {
				return types.DefaultParams()
			},
			func() interface{} {
				return suite.app.ShariahOracleKeeper.GetParams(suite.ctx)
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			outcome := reflect.DeepEqual(tc.paramsFun(), tc.getFun())
			suite.Require().Equal(tc.expected, outcome)
		})
	}
}
