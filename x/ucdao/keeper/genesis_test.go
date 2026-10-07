package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
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

// TestExportGenesisRoundTripsWithUntrackedEscrowCredit is the regression guard for
// an export the module refused to import. The exported Balances are the bank
// balances of the holders' escrows, while TotalBalance used to be the internal
// counter. A plain bank transfer to a holder's escrow - aISLM, or a denom the
// counter never tracks - moved the first and not the second, and InitGenesis
// panicked with "genesis total balance is incorrect". The total is now derived
// from the exported balances, so export -> import -> export is stable.
func (suite *KeeperTestSuite) TestExportGenesisRoundTripsWithUntrackedEscrowCredit() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	dao := suite.network.App.DaoKeeper
	bank := suite.network.App.BankKeeper

	holder := suite.keyring.GetAccAddr(0)
	outsider := suite.keyring.GetAccAddr(1)

	funded := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	islmDonation := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(250))
	haqqDonation := sdk.NewCoin(ethiqtypes.BaseDenom, sdkmath.NewInt(7))

	suite.Require().NoError(suite.network.FundAccount(outsider, sdk.NewCoins(haqqDonation)))
	suite.Require().NoError(dao.Fund(ctx, sdk.NewCoins(funded), holder))

	// Credit the holder's escrow behind the module's back.
	escrow := ucdaotypes.GetEscrowAddress(holder)
	suite.Require().NoError(bank.SendCoins(ctx, outsider, escrow, sdk.NewCoins(islmDonation, haqqDonation)))
	suite.Require().True(dao.GetTotalBalance(ctx).Equal(sdk.NewCoins(funded)), "the counter does not see the transfer")

	exported := dao.ExportGenesis(ctx)
	escrowCoins := sdk.NewCoins(funded.Add(islmDonation), haqqDonation)
	suite.Require().Len(exported.Balances, 1)
	suite.Require().True(exported.Balances[0].Coins.Equal(escrowCoins), exported.Balances[0].Coins.String())
	suite.Require().True(exported.TotalBalance.Equal(escrowCoins), exported.TotalBalance.String())

	// Import into a fresh chain whose bank holds the same escrow balance, as the
	// exported bank genesis would.
	suite.SetupTest()
	ctx = suite.network.GetContext()
	dao = suite.network.App.DaoKeeper
	suite.Require().NoError(suite.network.FundAccount(escrow, escrowCoins))

	suite.Require().NotPanics(func() { dao.InitGenesis(ctx, exported) })
	suite.Require().True(dao.GetTotalBalance(ctx).Equal(escrowCoins), dao.GetTotalBalance(ctx).String())
	suite.Require().Equal([]sdk.AccAddress{holder}, dao.GetHolders(ctx))

	reexported := dao.ExportGenesis(ctx)
	suite.Require().Equal(exported.Balances, reexported.Balances)
	suite.Require().True(reexported.TotalBalance.Equal(exported.TotalBalance), reexported.TotalBalance.String())
}

// initGenesisPanic runs InitGenesis and returns the recovered panic message ("" if none).
func (suite *KeeperTestSuite) initGenesisPanic(ctx sdk.Context, gs *ucdaotypes.GenesisState) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint(r)
		}
	}()
	suite.network.App.DaoKeeper.InitGenesis(ctx, gs)
	return ""
}

// TestExportGenesisRoundTripsAfterClampedBurn covers the export after the counter has drifted
// the other way. A plain transfer to a holder's escrow is invisible to the counter; burning it
// through ConvertToHaqq then subtracts more than was counted, TrackSubBalance clamps the aISLM
// counter to zero, and the remaining holder's aISLM is no longer in the counter at all. The
// holder's aLIQUID1 keeps the old counter-based total non-empty, so InitGenesis validated it and
// rejected the export. TotalBalance is now taken from the exported escrow balances.
func (suite *KeeperTestSuite) TestExportGenesisRoundTripsAfterClampedBurn() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	app := suite.network.App
	dao := suite.getBaseKeeper() // concrete keeper, captured before freshCtx() swaps suite.network

	holder := suite.keyring.GetAccAddr(0)    // legitimate holder, never touches the attack
	converter := suite.keyring.GetAccAddr(1) // second holder whose escrow receives the untracked credit
	outsider := suite.keyring.GetAccAddr(2)  // plain bank sender
	receiver := suite.keyring.GetAccAddr(3)

	unit := sdkmath.NewInt(1e18)
	daoParams := ucdaotypes.DefaultParams()
	daoParams.EnableDao = true
	suite.Require().NoError(suite.getBaseKeeper().SetParams(ctx, daoParams))
	ethiqParams := ethiqtypes.DefaultParams()
	ethiqParams.Enabled = true
	suite.Require().NoError(app.EthiqKeeper.SetParams(ctx, ethiqParams))

	holderIslm := sdk.NewCoin(utils.BaseDenom, unit)
	holderLiquid := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))
	converterIslm := sdk.NewCoin(utils.BaseDenom, sdkmath.OneInt())
	donation := sdk.NewCoin(utils.BaseDenom, unit.MulRaw(5))

	suite.Require().NoError(suite.network.FundAccount(holder, sdk.NewCoins(holderIslm, holderLiquid)))
	suite.Require().NoError(suite.network.FundAccount(converter, sdk.NewCoins(converterIslm)))
	suite.Require().NoError(suite.network.FundAccount(outsider, sdk.NewCoins(donation)))

	// legitimate, tracked funding
	suite.Require().NoError(dao.Fund(ctx, sdk.NewCoins(holderIslm, holderLiquid), holder))
	suite.Require().NoError(dao.Fund(ctx, sdk.NewCoins(converterIslm), converter))
	suite.Require().True(dao.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount.Equal(holderIslm.Amount.Add(converterIslm.Amount)))

	// baseline: a clean export round-trips
	clean := dao.ExportGenesis(ctx)
	suite.Require().True(clean.TotalBalance.Equal(sdk.NewCoins(holderIslm.Add(converterIslm), holderLiquid)), clean.TotalBalance.String())
	suite.Require().Empty(suite.initGenesisPanic(suite.freshCtx(), clean), "control: untampered export imports cleanly")

	// untracked credit: an ordinary bank send to the converter's escrow address
	converterEscrow := ucdaotypes.GetEscrowAddress(converter)
	suite.Require().False(app.BankKeeper.BlockedAddr(converterEscrow), "escrow addresses accept plain MsgSend")
	suite.Require().NoError(app.BankKeeper.SendCoins(ctx, outsider, converterEscrow, sdk.NewCoins(donation)))
	suite.Require().True(dao.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount.Equal(holderIslm.Amount.Add(converterIslm.Amount)),
		"counter did not see the donation")

	// the counter has already diverged from the escrows, but the export must still import
	diverged := dao.ExportGenesis(ctx)
	suite.Require().Empty(suite.initGenesisPanic(suite.freshCtx(), diverged), "export after an untracked credit imports cleanly")

	// burn path: converter spends everything its escrow holds; the counter only knew 1e18+1,
	// the burn removes 5e18+1, so TrackSubBalance clamps aISLM to zero instead of panicking
	burnAmt := converterIslm.Amount.Add(donation.Amount)
	_, err := dao.ConvertToHaqq(ctx, converter, receiver, burnAmt)
	suite.Require().NoError(err, "the clamp lets the over-counted burn succeed")

	suite.Require().True(dao.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount.IsZero(), "aISLM counter clamped to zero")
	suite.Require().True(app.BankKeeper.GetBalance(ctx, converterEscrow, utils.BaseDenom).Amount.IsZero(), "donation fully burned")
	suite.Require().False(dao.IsHolder(ctx, converter))
	holderBalance := dao.GetBalance(ctx, holder, utils.BaseDenom)
	suite.Require().True(holderBalance.Equal(holderIslm),
		"the untouched holder still owns 1e18 aISLM in escrow that the counter no longer reflects")

	exported := dao.ExportGenesis(ctx)
	suite.Require().True(exported.TotalBalance.Equal(sdk.NewCoins(holderIslm, holderLiquid)),
		"exported total follows the escrows, not the clamped counter: %s", exported.TotalBalance)
	suite.Require().Len(exported.Balances, 1)
	suite.Require().True(exported.Balances[0].Coins.Equal(sdk.NewCoins(holderIslm, holderLiquid)), exported.Balances[0].Coins.String())

	suite.Require().Empty(suite.initGenesisPanic(suite.freshCtx(), exported), "the module imports its own export")
}

// freshCtx rebuilds the network and returns a context of a new chain to import into.
func (suite *KeeperTestSuite) freshCtx() sdk.Context {
	suite.SetupTest()
	return suite.network.GetContext()
}
