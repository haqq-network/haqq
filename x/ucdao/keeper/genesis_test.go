package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// TestInitGenesis tests the InitGenesis function
func (suite *KeeperTestSuite) TestInitGenesis() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr1 := suite.keyring.GetAccAddr(0)
	addr2 := suite.keyring.GetAccAddr(1)

	coin1 := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	coin2 := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

	genState := &ucdaotypes.GenesisState{
		Params: ucdaotypes.DefaultParams(),
		Balances: []ucdaotypes.Balance{
			{
				Address: addr1.String(),
				Coins:   sdk.NewCoins(coin1),
			},
			{
				Address: addr2.String(),
				Coins:   sdk.NewCoins(coin2),
			},
		},
		TotalBalance: sdk.NewCoins(coin1, coin2),
	}

	suite.network.App.DaoKeeper.InitGenesis(ctx, genState)

	// Verify params
	params := suite.getBaseKeeper().GetParams(ctx)
	suite.Require().Equal(genState.Params, params)

	// Verify total balance
	totalBalance := suite.network.App.DaoKeeper.GetTotalBalance(ctx)
	suite.Require().True(totalBalance.AmountOf(coin1.Denom).Equal(coin1.Amount))
	suite.Require().True(totalBalance.AmountOf(coin2.Denom).Equal(coin2.Amount))
}

// TestExportGenesis tests the ExportGenesis function
func (suite *KeeperTestSuite) TestExportGenesis() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr1 := suite.keyring.GetAccAddr(0)
	addr2 := suite.keyring.GetAccAddr(1)

	coin1 := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	coin2 := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

	// Set params (default already has EnableDao: true)
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

	// Export genesis
	genState := suite.network.App.DaoKeeper.ExportGenesis(ctx)
	suite.Require().NotNil(genState)
	suite.Require().Equal(params, genState.Params)
	suite.Require().Len(genState.Balances, 2)
	suite.Require().True(genState.TotalBalance.AmountOf(coin1.Denom).Equal(coin1.Amount))
	suite.Require().True(genState.TotalBalance.AmountOf(coin2.Denom).Equal(coin2.Amount))
}

// TestInitGenesisWithEmptyBalances tests InitGenesis with empty balances
func (suite *KeeperTestSuite) TestInitGenesisWithEmptyBalances() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	genState := &ucdaotypes.GenesisState{
		Params:       ucdaotypes.DefaultParams(),
		Balances:     []ucdaotypes.Balance{},
		TotalBalance: sdk.Coins{},
	}

	suite.network.App.DaoKeeper.InitGenesis(ctx, genState)

	// Verify params
	params := suite.getBaseKeeper().GetParams(ctx)
	suite.Require().Equal(genState.Params, params)

	// Verify total balance is zero
	totalBalance := suite.network.App.DaoKeeper.GetTotalBalance(ctx)
	suite.Require().True(totalBalance.IsZero())
}

// TestInitGenesisWithMismatchedTotalBalance tests InitGenesis with mismatched total balance (should panic)
func (suite *KeeperTestSuite) TestInitGenesisWithMismatchedTotalBalance() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))

	genState := &ucdaotypes.GenesisState{
		Params: ucdaotypes.DefaultParams(),
		Balances: []ucdaotypes.Balance{
			{
				Address: addr.String(),
				Coins:   sdk.NewCoins(coin),
			},
		},
		TotalBalance: sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(2000))), // Mismatched
	}

	suite.Require().Panics(func() {
		suite.network.App.DaoKeeper.InitGenesis(ctx, genState)
	})
}

// TestInitGenesisWithMultipleAccounts tests InitGenesis with multiple accounts
func (suite *KeeperTestSuite) TestInitGenesisWithMultipleAccounts() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr1 := suite.keyring.GetAccAddr(0)
	addr2 := suite.keyring.GetAccAddr(1)

	coin1 := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	coin2 := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

	genState := &ucdaotypes.GenesisState{
		Params: ucdaotypes.DefaultParams(),
		Balances: []ucdaotypes.Balance{
			{
				Address: addr1.String(),
				Coins:   sdk.NewCoins(coin1),
			},
			{
				Address: addr2.String(),
				Coins:   sdk.NewCoins(coin2),
			},
		},
		TotalBalance: sdk.NewCoins(coin1, coin2),
	}

	// Fund escrow addresses directly (InitGenesis sets internal state but doesn't move coins)
	escrowAddr1 := ucdaotypes.GetEscrowAddress(addr1)
	escrowAddr2 := ucdaotypes.GetEscrowAddress(addr2)
	err := suite.network.FundAccount(escrowAddr1, sdk.NewCoins(coin1))
	suite.Require().NoError(err)
	err = suite.network.FundAccount(escrowAddr2, sdk.NewCoins(coin2))
	suite.Require().NoError(err)

	suite.network.App.DaoKeeper.InitGenesis(ctx, genState)

	// Verify balances
	balances1 := suite.network.App.DaoKeeper.GetAccountBalances(ctx, addr1)
	suite.Require().True(balances1.AmountOf(coin1.Denom).Equal(coin1.Amount))

	balances2 := suite.network.App.DaoKeeper.GetAccountBalances(ctx, addr2)
	suite.Require().True(balances2.AmountOf(coin2.Denom).Equal(coin2.Amount))

	// Verify holders
	holders := suite.network.App.DaoKeeper.GetHolders(ctx)
	suite.Require().Contains(holders, addr1)
	suite.Require().Contains(holders, addr2)
}

// TestInitGenesisWithZeroCoins tests InitGenesis with zero coins (should be filtered)
func (suite *KeeperTestSuite) TestInitGenesisWithZeroCoins() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	zeroCoin := sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt())
	validCoin := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

	genState := &ucdaotypes.GenesisState{
		Params: ucdaotypes.DefaultParams(),
		Balances: []ucdaotypes.Balance{
			{
				Address: addr.String(),
				Coins:   sdk.NewCoins(zeroCoin, validCoin),
			},
		},
		TotalBalance: sdk.NewCoins(validCoin),
	}

	// Fund escrow address directly (InitGenesis sets internal state but doesn't move coins)
	escrowAddr := ucdaotypes.GetEscrowAddress(addr)
	err := suite.network.FundAccount(escrowAddr, sdk.NewCoins(validCoin))
	suite.Require().NoError(err)

	suite.network.App.DaoKeeper.InitGenesis(ctx, genState)

	// Verify only valid coin is stored
	balances := suite.network.App.DaoKeeper.GetAccountBalances(ctx, addr)
	suite.Require().True(balances.AmountOf(validCoin.Denom).Equal(validCoin.Amount))
	suite.Require().True(balances.AmountOf(zeroCoin.Denom).IsZero())
}

// TestExportGenesisWithEmptyState tests ExportGenesis with empty state
func (suite *KeeperTestSuite) TestExportGenesisWithEmptyState() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	genState := suite.network.App.DaoKeeper.ExportGenesis(ctx)
	suite.Require().NotNil(genState)
	suite.Require().Len(genState.Balances, 0)
	suite.Require().True(genState.TotalBalance.IsZero())
}
