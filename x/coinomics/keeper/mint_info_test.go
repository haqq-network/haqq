package keeper_test

// func (suite *KeeperTestSuite) TestSetGetPeriod() {
// 	expEra := uint64(9)

// 	testCases := []struct {
// 		name     string
// 		malleate func()
// 		ok       bool
// 	}{
// 		{
// 			"default era",
// 			func() {},
// 			false,
// 		},
// 		{
// 			"period set",
// 			func() {
// 				suite.app.CoinomicsKeeper.SetEra(suite.ctx, expEra)
// 			},
// 			true,
// 		},
// 	}
// 	for _, tc := range testCases {
// 		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
// 			suite.SetupTest() // reset

// 			tc.malleate()

// 			period := suite.app.CoinomicsKeeper.GetEra(suite.ctx)
// 			if tc.ok {
// 				suite.Require().Equal(expEra, period, tc.name)
// 			} else {
// 				suite.Require().Zero(period, tc.name)
// 			}
// 		})
// 	}
// }
