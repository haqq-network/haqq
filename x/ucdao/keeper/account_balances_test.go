package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/haqq-network/haqq/utils"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// TestGetBalance tests the GetBalance function
func (suite *KeeperTestSuite) TestGetBalance() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	denom := utils.BaseDenom
	coin := sdk.NewCoin(denom, sdkmath.NewInt(1000))

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

	// Get balance
	balance := suite.network.App.DaoKeeper.GetBalance(ctx, addr, denom)
	suite.Require().Equal(coin, balance)
}

// TestGetAccountBalances tests the GetAccountBalances function
func (suite *KeeperTestSuite) TestGetAccountBalances() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	expectedBalances := sdk.NewCoins(
		sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000)),
		sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500)),
	)

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// First, fund the account with the coins so they have them to send
	err = suite.network.FundAccount(addr, expectedBalances)
	suite.Require().NoError(err)

	// Now fund the DAO
	err = suite.network.App.DaoKeeper.Fund(ctx, expectedBalances, addr)
	suite.Require().NoError(err)

	// Get balances
	balances := suite.network.App.DaoKeeper.GetAccountBalances(ctx, addr)
	suite.Require().Equal(expectedBalances, balances)
}

// TestHasBalance tests the HasBalance function
func (suite *KeeperTestSuite) TestHasBalance() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	amt := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))

	// Enable module (default already has EnableDao: true)
	params := ucdaotypes.DefaultParams()
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// First, fund the account with the coin so they have it to send
	err = suite.network.FundAccount(addr, sdk.NewCoins(amt))
	suite.Require().NoError(err)

	// Fund account
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(amt), addr)
	suite.Require().NoError(err)

	// Check has balance
	hasBalance := suite.network.App.DaoKeeper.HasBalance(ctx, addr, amt)
	suite.Require().True(hasBalance)
}

// TestGetHolders tests the GetHolders function
func (suite *KeeperTestSuite) TestGetHolders() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr1 := suite.keyring.GetAccAddr(0)
	addr2 := suite.keyring.GetAccAddr(1)

	coin1 := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	coin2 := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

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

	// Get holders
	holders := suite.network.App.DaoKeeper.GetHolders(ctx)
	suite.Require().Len(holders, 2)
	suite.Require().Contains(holders, addr1)
	suite.Require().Contains(holders, addr2)
}

// TestGetAccountsBalances tests the GetAccountsBalances function
func (suite *KeeperTestSuite) TestGetAccountsBalances() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr1 := suite.keyring.GetAccAddr(0)
	addr2 := suite.keyring.GetAccAddr(1)

	coin1 := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	coin2 := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

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

	// Get accounts balances
	balances := suite.network.App.DaoKeeper.GetAccountsBalances(ctx)
	suite.Require().Len(balances, 2)

	// Verify balances
	balanceMap := make(map[string]ucdaotypes.Balance)
	for _, balance := range balances {
		balanceMap[balance.Address] = balance
	}

	suite.Require().Contains(balanceMap, addr1.String())
	suite.Require().Contains(balanceMap, addr2.String())
	suite.Require().True(coin1.Equal(balanceMap[addr1.String()].Coins[0]))
	suite.Require().True(coin2.Equal(balanceMap[addr2.String()].Coins[0]))
}

// TestGetPaginatedAccountsBalances tests the GetPaginatedAccountsBalances function
func (suite *KeeperTestSuite) TestGetPaginatedAccountsBalances() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr1 := suite.keyring.GetAccAddr(0)
	addr2 := suite.keyring.GetAccAddr(1)

	coin1 := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	coin2 := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

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

	// Get paginated accounts balances
	pagination := &query.PageRequest{
		Limit: 1,
	}

	balances, pageRes, err := suite.getBaseKeeper().GetPaginatedAccountsBalances(ctx, pagination)
	suite.Require().NoError(err)
	suite.Require().Len(balances, 1)
	suite.Require().NotNil(pageRes)
}
