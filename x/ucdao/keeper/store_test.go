package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// TestSetHoldersIndex tests setHoldersIndex indirectly through Fund
func (suite *KeeperTestSuite) TestSetHoldersIndex() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Fund account - this should add to holders index
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), addr)
	suite.Require().NoError(err)

	// Verify holder is in the list
	holders := suite.network.App.DaoKeeper.GetHolders(ctx)
	suite.Require().Contains(holders, addr)
}

// TestSetHoldersIndexRemoval tests setHoldersIndex removal when balance becomes zero
func (suite *KeeperTestSuite) TestSetHoldersIndexRemoval() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	owner := suite.keyring.GetAccAddr(0)
	newOwner := suite.keyring.GetAccAddr(1)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Fund owner account
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), owner)
	suite.Require().NoError(err)

	// Verify owner is in holders
	holders := suite.network.App.DaoKeeper.GetHolders(ctx)
	suite.Require().Contains(holders, owner)

	// Transfer all ownership - this should remove owner from holders
	_, err = suite.network.App.DaoKeeper.TransferOwnership(ctx, owner, newOwner, sdk.NewCoins(coin))
	suite.Require().NoError(err)

	// Verify owner is removed from holders and newOwner is added
	holders = suite.network.App.DaoKeeper.GetHolders(ctx)
	suite.Require().NotContains(holders, owner)
	suite.Require().Contains(holders, newOwner)
}
