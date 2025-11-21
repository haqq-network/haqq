package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/haqq-network/haqq/utils"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// TestGetTotalBalance tests the GetTotalBalance function
func (suite *KeeperTestSuite) TestGetTotalBalance() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	coin1 := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	coin2 := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

	addr1 := suite.keyring.GetAccAddr(0)
	addr2 := suite.keyring.GetAccAddr(1)

	// Enable module (default already has EnableDao: true)
	params := ucdaotypes.DefaultParams()
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// First, fund the accounts with the coins so they have them to send
	err = suite.network.FundAccount(addr1, sdk.NewCoins(coin1))
	suite.Require().NoError(err)
	err = suite.network.FundAccount(addr2, sdk.NewCoins(coin2))
	suite.Require().NoError(err)

	// Fund first account
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin1), addr1)
	suite.Require().NoError(err)

	// Fund second account
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin2), addr2)
	suite.Require().NoError(err)

	// Get total balance
	totalBalance := suite.network.App.DaoKeeper.GetTotalBalance(ctx)
	suite.Require().True(totalBalance.AmountOf(coin1.Denom).Equal(coin1.Amount))
	suite.Require().True(totalBalance.AmountOf(coin2.Denom).Equal(coin2.Amount))
}

// TestGetTotalBalanceOf tests the GetTotalBalanceOf function
func (suite *KeeperTestSuite) TestGetTotalBalanceOf() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	denom := utils.BaseDenom
	expectedAmount := sdkmath.NewInt(1000)

	// Initially should return zero
	balance := suite.network.App.DaoKeeper.GetTotalBalanceOf(ctx, denom)
	suite.Require().True(balance.Amount.IsZero())
	suite.Require().Equal(denom, balance.Denom)

	// Enable module (default already has EnableDao: true)
	params := ucdaotypes.DefaultParams()
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Set up a balance
	coin := sdk.NewCoin(denom, expectedAmount)
	addr := suite.keyring.GetAccAddr(0)

	// First, fund the account with the coin so they have it to send
	err = suite.network.FundAccount(addr, sdk.NewCoins(coin))
	suite.Require().NoError(err)

	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), addr)
	suite.Require().NoError(err)

	// Now should return the amount
	balance = suite.network.App.DaoKeeper.GetTotalBalanceOf(ctx, denom)
	suite.Require().True(balance.Amount.Equal(expectedAmount))
	suite.Require().Equal(denom, balance.Denom)
}

// TestHasTotalBalanceOf tests the HasTotalBalanceOf function
func (suite *KeeperTestSuite) TestHasTotalBalanceOf() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	denom := utils.BaseDenom

	// Initially should return false
	hasBalance := suite.network.App.DaoKeeper.HasTotalBalanceOf(ctx, denom)
	suite.Require().False(hasBalance)

	// Enable module (default already has EnableDao: true)
	params := ucdaotypes.DefaultParams()
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Set up a balance
	coin := sdk.NewCoin(denom, sdkmath.NewInt(1000))
	addr := suite.keyring.GetAccAddr(0)

	// First, fund the account with the coin so they have it to send
	err = suite.network.FundAccount(addr, sdk.NewCoins(coin))
	suite.Require().NoError(err)

	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), addr)
	suite.Require().NoError(err)

	// Now should return true
	hasBalance = suite.network.App.DaoKeeper.HasTotalBalanceOf(ctx, denom)
	suite.Require().True(hasBalance)
}

// TestIterateTotalBalance tests the IterateTotalBalance function
func (suite *KeeperTestSuite) TestIterateTotalBalance() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	coin1 := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	coin2 := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

	addr1 := suite.keyring.GetAccAddr(0)
	addr2 := suite.keyring.GetAccAddr(1)

	// Enable module (default already has EnableDao: true)
	params := ucdaotypes.DefaultParams()
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// First, fund the accounts with the coins so they have them to send
	err = suite.network.FundAccount(addr1, sdk.NewCoins(coin1))
	suite.Require().NoError(err)
	err = suite.network.FundAccount(addr2, sdk.NewCoins(coin2))
	suite.Require().NoError(err)

	// Fund accounts
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin1), addr1)
	suite.Require().NoError(err)

	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin2), addr2)
	suite.Require().NoError(err)

	// Iterate and collect coins
	collectedCoins := make(map[string]sdk.Coin)
	suite.network.App.DaoKeeper.IterateTotalBalance(ctx, func(c sdk.Coin) bool {
		collectedCoins[c.Denom] = c
		return false // continue iteration
	})

	suite.Require().Len(collectedCoins, 2)
	suite.Require().True(collectedCoins[coin1.Denom].Amount.Equal(coin1.Amount))
	suite.Require().True(collectedCoins[coin2.Denom].Amount.Equal(coin2.Amount))
}

// TestIterateTotalBalanceEarlyStop tests that iteration stops when callback returns true
func (suite *KeeperTestSuite) TestIterateTotalBalanceEarlyStop() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	coin1 := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	coin2 := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

	addr1 := suite.keyring.GetAccAddr(0)
	addr2 := suite.keyring.GetAccAddr(1)

	// Enable module (default already has EnableDao: true)
	params := ucdaotypes.DefaultParams()
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// First, fund the accounts with the coins so they have them to send
	err = suite.network.FundAccount(addr1, sdk.NewCoins(coin1))
	suite.Require().NoError(err)
	err = suite.network.FundAccount(addr2, sdk.NewCoins(coin2))
	suite.Require().NoError(err)

	// Fund accounts
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin1), addr1)
	suite.Require().NoError(err)

	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin2), addr2)
	suite.Require().NoError(err)

	// Iterate and stop early
	count := 0
	suite.network.App.DaoKeeper.IterateTotalBalance(ctx, func(c sdk.Coin) bool {
		count++
		return true // stop iteration
	})

	suite.Require().Equal(1, count)
}

// TestGetPaginatedTotalBalance tests the GetPaginatedTotalBalance function
func (suite *KeeperTestSuite) TestGetPaginatedTotalBalance() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	coin1 := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	coin2 := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

	addr1 := suite.keyring.GetAccAddr(0)
	addr2 := suite.keyring.GetAccAddr(1)

	// Enable module (default already has EnableDao: true)
	params := ucdaotypes.DefaultParams()
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// First, fund the accounts with the coins so they have them to send
	err = suite.network.FundAccount(addr1, sdk.NewCoins(coin1))
	suite.Require().NoError(err)
	err = suite.network.FundAccount(addr2, sdk.NewCoins(coin2))
	suite.Require().NoError(err)

	// Fund accounts
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin1), addr1)
	suite.Require().NoError(err)

	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin2), addr2)
	suite.Require().NoError(err)

	// Get paginated total balance
	pagination := &query.PageRequest{
		Limit: 1,
	}

	totalBalance, pageRes, err := suite.network.App.DaoKeeper.GetPaginatedTotalBalance(ctx, pagination)
	suite.Require().NoError(err)
	suite.Require().NotNil(pageRes)
	suite.Require().True(totalBalance.AmountOf(coin1.Denom).Equal(coin1.Amount) || totalBalance.AmountOf(coin2.Denom).Equal(coin2.Amount))
}

// TestGetTotalBalanceWithZeroCoins tests GetTotalBalance with zero coins (should be filtered)
func (suite *KeeperTestSuite) TestGetTotalBalanceWithZeroCoins() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Get total balance when empty
	totalBalance := suite.network.App.DaoKeeper.GetTotalBalance(ctx)
	suite.Require().True(totalBalance.IsZero())
}

// TestGetPaginatedTotalBalanceWithNilPagination tests GetPaginatedTotalBalance with nil pagination
func (suite *KeeperTestSuite) TestGetPaginatedTotalBalanceWithNilPagination() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	addr := suite.keyring.GetAccAddr(0)

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

	// Get paginated total balance with nil pagination
	totalBalance, pageRes, err := suite.network.App.DaoKeeper.GetPaginatedTotalBalance(ctx, nil)
	suite.Require().NoError(err)
	suite.Require().NotNil(pageRes)
	suite.Require().True(totalBalance.AmountOf(coin.Denom).Equal(coin.Amount))
}

// TestGetPaginatedTotalBalanceWithEmptyStore tests GetPaginatedTotalBalance with empty store
func (suite *KeeperTestSuite) TestGetPaginatedTotalBalanceWithEmptyStore() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	pagination := &query.PageRequest{
		Limit: 10,
	}

	totalBalance, pageRes, err := suite.network.App.DaoKeeper.GetPaginatedTotalBalance(ctx, pagination)
	suite.Require().NoError(err)
	suite.Require().NotNil(pageRes)
	suite.Require().True(totalBalance.IsZero())
}

// TestIterateTotalBalanceWithEmptyStore tests IterateTotalBalance with empty store
func (suite *KeeperTestSuite) TestIterateTotalBalanceWithEmptyStore() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	count := 0
	suite.network.App.DaoKeeper.IterateTotalBalance(ctx, func(c sdk.Coin) bool {
		count++
		return false
	})
	suite.Require().Equal(0, count)
}

// TestGetTotalBalanceOfNonExistentDenom tests GetTotalBalanceOf with non-existent denom
func (suite *KeeperTestSuite) TestGetTotalBalanceOfNonExistentDenom() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	denom := "nonexistent"
	balance := suite.network.App.DaoKeeper.GetTotalBalanceOf(ctx, denom)
	suite.Require().Equal(denom, balance.Denom)
	suite.Require().True(balance.Amount.IsZero())
}

// TestHasTotalBalanceOfNonExistentDenom tests HasTotalBalanceOf with non-existent denom
func (suite *KeeperTestSuite) TestHasTotalBalanceOfNonExistentDenom() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	denom := "nonexistent"
	hasBalance := suite.network.App.DaoKeeper.HasTotalBalanceOf(ctx, denom)
	suite.Require().False(hasBalance)
}
