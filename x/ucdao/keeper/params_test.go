package keeper_test

import (
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// TestGetParams tests the GetParams function
func (suite *KeeperTestSuite) TestGetParams() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Initially should return default params (EnableDao: true)
	params := suite.getBaseKeeper().GetParams(ctx)
	expectedDefaultParams := ucdaotypes.DefaultParams()
	suite.Require().Equal(expectedDefaultParams, params)

	// Set custom params
	customParams := ucdaotypes.DefaultParams()
	customParams.EnableDao = false
	err := suite.getBaseKeeper().SetParams(ctx, customParams)
	suite.Require().NoError(err)

	// Get params
	params = suite.getBaseKeeper().GetParams(ctx)
	suite.Require().Equal(customParams, params)
}

// TestSetParams tests the SetParams function
func (suite *KeeperTestSuite) TestSetParams() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	params := ucdaotypes.DefaultParams()
	params.EnableDao = false

	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Verify params were set
	retrievedParams := suite.getBaseKeeper().GetParams(ctx)
	suite.Require().Equal(params, retrievedParams)
}

// TestIsModuleEnabled tests the IsModuleEnabled function
func (suite *KeeperTestSuite) TestIsModuleEnabled() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Initially should return true (default params have EnableDao: true)
	enabled := suite.getBaseKeeper().IsModuleEnabled(ctx)
	suite.Require().True(enabled)

	// Disable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = false
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	enabled = suite.getBaseKeeper().IsModuleEnabled(ctx)
	suite.Require().False(enabled)

	// Enable module again
	params.EnableDao = true
	err = suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	enabled = suite.getBaseKeeper().IsModuleEnabled(ctx)
	suite.Require().True(enabled)
}
