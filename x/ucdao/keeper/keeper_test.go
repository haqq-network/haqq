package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// TestLogger tests the Logger function
func (suite *KeeperTestSuite) TestLogger() {
	ctx := suite.network.GetContext()
	logger := suite.getBaseKeeper().Logger(ctx)
	suite.Require().NotNil(logger)
}

// TestFund tests the Fund function
func (suite *KeeperTestSuite) TestFund() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))

	// Enable module (default already has EnableDao: true)
	params := ucdaotypes.DefaultParams()
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// First, fund the account with the coin so they have it to send
	err = suite.network.FundAccount(addr, sdk.NewCoins(coin))
	suite.Require().NoError(err)

	// Fund account
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), addr)
	suite.Require().NoError(err)

	// Verify balance
	balance := suite.network.App.DaoKeeper.GetBalance(ctx, addr, coin.Denom)
	suite.Require().Equal(coin, balance)
}

// TestFundModuleDisabled tests Fund when module is disabled
func (suite *KeeperTestSuite) TestFundModuleDisabled() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))

	// Disable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = false
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), addr)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, ucdaotypes.ErrModuleDisabled)
}

// TestFundInvalidDenom tests Fund with invalid denom
func (suite *KeeperTestSuite) TestFundInvalidDenom() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	coin := sdk.NewCoin("invalid-denom", sdkmath.NewInt(1000))

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), addr)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, ucdaotypes.ErrInvalidDenom)
}

// TestFundZeroCoin tests Fund with zero coin (should be skipped)
func (suite *KeeperTestSuite) TestFundZeroCoin() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt())

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Should not error for zero coins
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), addr)
	suite.Require().NoError(err)
}

// TestFundLiquidToken tests Fund with liquid token
func (suite *KeeperTestSuite) TestFundLiquidToken() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	coin := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(1000))

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// First, fund the account with the liquid token so they have it to send
	err = suite.network.FundAccount(addr, sdk.NewCoins(coin))
	suite.Require().NoError(err)

	// Now fund the DAO with liquid token
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), addr)
	suite.Require().NoError(err)

	// Verify balance
	balance := suite.network.App.DaoKeeper.GetBalance(ctx, addr, coin.Denom)
	suite.Require().Equal(coin, balance)
}

// TestTransferOwnership tests the TransferOwnership function
func (suite *KeeperTestSuite) TestTransferOwnership() {
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

	// Transfer ownership
	transferred, err := suite.network.App.DaoKeeper.TransferOwnership(ctx, owner, newOwner, sdk.NewCoins(coin))
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(coin), transferred)

	// Verify new owner has balance
	newOwnerBalance := suite.network.App.DaoKeeper.GetBalance(ctx, newOwner, coin.Denom)
	suite.Require().Equal(coin, newOwnerBalance)

	// Verify old owner has no balance
	ownerBalance := suite.network.App.DaoKeeper.GetBalance(ctx, owner, coin.Denom)
	suite.Require().True(ownerBalance.IsZero())
}

// TestTransferOwnershipModuleDisabled tests TransferOwnership when module is disabled
func (suite *KeeperTestSuite) TestTransferOwnershipModuleDisabled() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	owner := suite.keyring.GetAccAddr(0)
	newOwner := suite.keyring.GetAccAddr(1)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))

	// Disable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = false
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	transferred, err := suite.network.App.DaoKeeper.TransferOwnership(ctx, owner, newOwner, sdk.NewCoins(coin))
	suite.Require().Error(err)
	suite.Require().Nil(transferred)
	suite.Require().ErrorIs(err, ucdaotypes.ErrModuleDisabled)
}

// TestTransferOwnershipNotEligible tests TransferOwnership when owner has no balance
func (suite *KeeperTestSuite) TestTransferOwnershipNotEligible() {
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

	// Owner has no balance, so transfer should fail
	transferred, err := suite.network.App.DaoKeeper.TransferOwnership(ctx, owner, newOwner, sdk.NewCoins(coin))
	suite.Require().Error(err)
	suite.Require().Nil(transferred)
	suite.Require().ErrorIs(err, ucdaotypes.ErrNotEligible)
}

// TestTransferOwnershipZeroCoin tests TransferOwnership with zero coin (should skip)
func (suite *KeeperTestSuite) TestTransferOwnershipZeroCoin() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	owner := suite.keyring.GetAccAddr(0)
	newOwner := suite.keyring.GetAccAddr(1)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	zeroCoin := sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt())

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Fund owner account
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), owner)
	suite.Require().NoError(err)

	// Transfer ownership with zero coin (should skip)
	transferred, err := suite.network.App.DaoKeeper.TransferOwnership(ctx, owner, newOwner, sdk.NewCoins(zeroCoin))
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(zeroCoin), transferred)
}
