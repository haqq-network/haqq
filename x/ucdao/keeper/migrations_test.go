package keeper_test

import (
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ucdao/keeper"
	"github.com/haqq-network/haqq/x/ucdao/types"
)

// TestNewMigrator tests the NewMigrator function
func (suite *KeeperTestSuite) TestNewMigrator() {
	suite.SetupTest()
	migrator := keeper.NewMigrator(suite.getBaseKeeper())
	suite.Require().NotNil(migrator)
}

// TestMigrate1to2 tests the Migrate1to2 migration function
func (suite *KeeperTestSuite) TestMigrate1to2() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	baseKeeper := suite.getBaseKeeper()
	var err error

	// Prepare 5 test wallets
	testWallets := make([]sdk.AccAddress, 5)
	for i := 0; i < 5; i++ {
		testWallets[i] = suite.keyring.GetAccAddr(i)
	}

	// Define aISLM token amounts for each wallet
	amounts := []math.Int{
		math.NewInt(1000000000000000000), // 1 aISLM
		math.NewInt(2000000000000000000), // 2 aISLM
		math.NewInt(3000000000000000000), // 3 aISLM
		math.NewInt(4000000000000000000), // 4 aISLM
		math.NewInt(5000000000000000000), // 5 aISLM
	}

	// Calculate total amount
	totalAmount := math.ZeroInt()
	for _, amt := range amounts {
		totalAmount = totalAmount.Add(amt)
	}

	// Get module account address
	moduleAddr := authtypes.NewModuleAddress(types.ModuleName)

	// Fund test wallets with tokens and send them to ucdao module account
	// (simulating that tokens were previously transferred to module)
	denom := utils.BaseDenom

	// First, fund each test wallet with their respective amounts
	for i, wallet := range testWallets {
		coins := sdk.NewCoins(sdk.NewCoin(denom, amounts[i]))
		err = suite.network.FundAccount(wallet, coins)
		suite.Require().NoError(err, "failed to fund wallet %d", i)
	}

	// Then, send all tokens from test wallets to ucdao module account
	for i, wallet := range testWallets {
		coins := sdk.NewCoins(sdk.NewCoin(denom, amounts[i]))
		err = suite.network.App.BankKeeper.SendCoins(ctx, wallet, moduleAddr, coins)
		suite.Require().NoError(err, "failed to send coins from wallet %d to module", i)
	}

	// Verify module account has the total balance
	moduleBalance := suite.network.App.BankKeeper.GetBalance(ctx, moduleAddr, denom)
	suite.Require().Equal(totalAmount, moduleBalance.Amount, "module account should have total balance")

	// Manually create balancesStore entries (simulating old version storage)
	// Access the store using the module's store key through the context
	storeKey := suite.network.App.GetKey(types.StoreKey)
	store := ctx.KVStore(storeKey)
	balancesStore := prefix.NewStore(store, types.BalancesPrefix)

	for i, wallet := range testWallets {
		// Create key: address length prefix + address + denom
		// Since balancesStore is a prefix store with BalancesPrefix, the key should be:
		// address length prefix + address + denom (without the BalancesPrefix)
		addrKey := address.MustLengthPrefix(wallet)
		key := append(addrKey, []byte(denom)...)

		// Marshal the amount
		amountBytes, err := amounts[i].Marshal()
		suite.Require().NoError(err)

		// Set the balance in the store (old version format)
		balancesStore.Set(key, amountBytes)
	}

	// Verify balances are in the store before migration
	for i, wallet := range testWallets {
		addrKey := address.MustLengthPrefix(wallet)
		key := append(addrKey, []byte(denom)...)
		suite.Require().True(balancesStore.Has(key), "balance should exist in store for wallet %d", i)
	}

	// Run migration
	migrator := keeper.NewMigrator(baseKeeper)
	err = migrator.Migrate1to2(ctx)
	suite.Require().NoError(err, "migration should succeed")

	// 1. Check holders addresses are all correctly migrated
	holders := baseKeeper.GetHolders(ctx)
	suite.Require().Len(holders, 5, "should have 5 holders after migration")

	// Verify all test wallets are in holders
	holderMap := make(map[string]bool)
	for _, holder := range holders {
		holderMap[holder.String()] = true
	}
	for _, wallet := range testWallets {
		suite.Require().True(holderMap[wallet.String()], "wallet %s should be in holders", wallet.String())
	}

	// 2. Check total balance of all escrow wallet addresses equals initial ucdao module balance
	totalEscrowBalance := sdk.NewCoins()
	for _, wallet := range testWallets {
		escrowAddr := types.GetEscrowAddress(wallet)
		escrowBalance := suite.network.App.BankKeeper.GetAllBalances(ctx, escrowAddr)
		totalEscrowBalance = totalEscrowBalance.Add(escrowBalance...)
	}

	// Verify total escrow balance equals initial module balance
	suite.Require().True(totalEscrowBalance.AmountOf(denom).Equal(totalAmount),
		"total escrow balance should equal initial module balance: expected %s, got %s",
		totalAmount.String(), totalEscrowBalance.AmountOf(denom).String())

	// Verify module account balance is now zero (all transferred to escrow accounts)
	moduleBalanceAfter := suite.network.App.BankKeeper.GetBalance(ctx, moduleAddr, denom)
	suite.Require().True(moduleBalanceAfter.IsZero(), "module account should have zero balance after migration")

	// Verify balances are removed from the store after migration
	for i, wallet := range testWallets {
		addrKey := address.MustLengthPrefix(wallet)
		key := append(addrKey, []byte(denom)...)
		suite.Require().False(balancesStore.Has(key),
			"balance should be removed from store for wallet %d after migration", i)
	}

	// Verify individual escrow balances
	for i, wallet := range testWallets {
		escrowAddr := types.GetEscrowAddress(wallet)
		escrowBalance := suite.network.App.BankKeeper.GetBalance(ctx, escrowAddr, denom)
		expectedCoin := sdk.NewCoin(denom, amounts[i])
		suite.Require().Equal(expectedCoin, escrowBalance,
			"escrow balance for wallet %d should match expected amount", i)
	}
}
