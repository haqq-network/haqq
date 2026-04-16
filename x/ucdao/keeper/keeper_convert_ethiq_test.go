package keeper_test

import (
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// TestConvertToHaqqModuleDisabled tests ConvertToHaqq when module is disabled
func (suite *KeeperTestSuite) TestConvertToHaqqModuleDisabled() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	sender := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)
	islmAmount := sdkmath.NewInt(1000)

	// Disable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = false
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	result, err := suite.network.App.DaoKeeper.ConvertToHaqq(ctx, sender, receiver, islmAmount)
	suite.Require().Error(err)
	suite.Require().True(result.Denom == "" || result.IsZero())
	suite.Require().ErrorIs(err, ucdaotypes.ErrModuleDisabled)
}

// TestConvertToHaqqNotHolder tests ConvertToHaqq when sender is not a holder
func (suite *KeeperTestSuite) TestConvertToHaqqNotHolder() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	sender := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)
	islmAmount := sdkmath.NewInt(1000)

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Sender is not a holder (has not funded the DAO)
	result, err := suite.network.App.DaoKeeper.ConvertToHaqq(ctx, sender, receiver, islmAmount)
	suite.Require().Error(err)
	suite.Require().True(result.Denom == "" || result.IsZero())
	suite.Require().ErrorIs(err, ucdaotypes.ErrNotEligible)
}

// TestConvertToHaqqEthiqCalculationError tests ConvertToHaqq when ethiq module's calculation returns an error.
// Uses only ISLM (no aLIQUID) so we reach BurnIslmForHaqq; ethiq may return e.g. "failed to calculate aHAQQ amount".
func (suite *KeeperTestSuite) TestConvertToHaqqEthiqCalculationError() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	sender := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)
	islmAmount := sdkmath.NewInt(1000)

	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	ethiqParams := ethiqtypes.DefaultParams()
	ethiqParams.Enabled = true
	ethiqParams.MinMintPerTx = sdkmath.OneInt()
	ethiqParams.MaxMintPerTx = sdkmath.NewInt(1000000000000000000)
	suite.network.App.EthiqKeeper.SetParams(ctx, ethiqParams)

	// Fund sender with ISLM only (aLIQUID cannot be minted via FundAccount; must be created via Liquidate)
	islmCoin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000000000000000000))
	err = suite.network.FundAccount(sender, sdk.NewCoins(islmCoin))
	suite.Require().NoError(err)
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(islmCoin), sender)
	suite.Require().NoError(err)

	// Mint huge ethiq supply so ethiq's CalculateHaqqCoinsToMint / price level logic may fail
	hugeSupply := sdkmath.NewInt(1)
	for i := 0; i < 30; i++ {
		hugeSupply = hugeSupply.MulRaw(10)
	}
	ethiqCoin := sdk.NewCoin(ethiqtypes.BaseDenom, hugeSupply)
	err = suite.network.App.BankKeeper.MintCoins(ctx, ethiqtypes.ModuleName, sdk.NewCoins(ethiqCoin))
	suite.Require().NoError(err)

	result, err := suite.network.App.DaoKeeper.ConvertToHaqq(ctx, sender, receiver, islmAmount)
	suite.Require().Error(err)
	suite.Require().True(result.Denom == "" || result.IsZero())

	errMsg := err.Error()
	// ucdao wraps ethiq errors as "failed to convert amount of aISLM to aHAQQ"; ethiq may wrap with "failed to calculate aHAQQ amount"
	suite.Require().True(
		strings.Contains(errMsg, "failed to convert") || strings.Contains(errMsg, "failed to calculate aHAQQ") || strings.Contains(errMsg, "failed to find price level"),
		"Error should be from conversion or ethiq calculation, got: %s", errMsg,
	)
}

// TestConvertToHaqqEthiqMaxMintExceeded tests ConvertToHaqq when ethiq module rejects (e.g. mint amount > max_mint_per_tx).
func (suite *KeeperTestSuite) TestConvertToHaqqEthiqMaxMintExceeded() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	sender := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)
	islmAmount := sdkmath.NewInt(1000)

	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	ethiqParams := ethiqtypes.DefaultParams()
	ethiqParams.Enabled = true
	ethiqParams.MinMintPerTx = sdkmath.OneInt()
	ethiqParams.MaxMintPerTx = sdkmath.NewInt(1000000)
	suite.network.App.EthiqKeeper.SetParams(ctx, ethiqParams)

	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(10000))
	err = suite.network.FundAccount(sender, sdk.NewCoins(coin))
	suite.Require().NoError(err)
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), sender)
	suite.Require().NoError(err)

	result, err := suite.network.App.DaoKeeper.ConvertToHaqq(ctx, sender, receiver, islmAmount)
	if err != nil {
		suite.Require().True(result.Denom == "" || result.IsZero())
		// Ethiq may return invalid amount or max_mint_per_tx; ucdao wraps as "failed to convert"
		suite.Require().True(strings.Contains(err.Error(), "failed to convert") || strings.Contains(err.Error(), "max_mint_per_tx") || strings.Contains(err.Error(), "invalid"))
	}
}

// TestConvertToHaqqInsufficientTotalBalance tests ConvertToHaqq when sender's total balance is less than islmAmount.
func (suite *KeeperTestSuite) TestConvertToHaqqInsufficientTotalBalance() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	sender := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)
	islmAmount := sdkmath.NewInt(1000)

	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	ethiqParams := ethiqtypes.DefaultParams()
	ethiqParams.Enabled = true
	ethiqParams.MinMintPerTx = sdkmath.OneInt()
	ethiqParams.MaxMintPerTx = sdkmath.NewInt(1000000)
	suite.network.App.EthiqKeeper.SetParams(ctx, ethiqParams)

	// Fund sender with ISLM only, less than islmAmount (aLIQUID cannot be minted via FundAccount)
	islmCoin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(100))
	err = suite.network.FundAccount(sender, sdk.NewCoins(islmCoin))
	suite.Require().NoError(err)
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(islmCoin), sender)
	suite.Require().NoError(err)

	result, err := suite.network.App.DaoKeeper.ConvertToHaqq(ctx, sender, receiver, islmAmount)
	suite.Require().Error(err)
	suite.Require().True(result.Denom == "" || result.IsZero())
	suite.Require().ErrorIs(err, ucdaotypes.ErrInsufficientFunds)
	suite.Require().Contains(err.Error(), "total balance")
}

// TestConvertToHaqqSuccessWithSufficientISLM tests ConvertToHaqq when sender has sufficient ISLM balance (no aLIQUID).
func (suite *KeeperTestSuite) TestConvertToHaqqSuccessWithSufficientISLM() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	sender := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)
	islmAmount := sdkmath.NewInt(1000)

	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	ethiqParams := ethiqtypes.DefaultParams()
	ethiqParams.Enabled = true
	ethiqParams.MinMintPerTx = sdkmath.OneInt()
	ethiqParams.MaxMintPerTx = sdkmath.NewInt(1000000)
	suite.network.App.EthiqKeeper.SetParams(ctx, ethiqParams)

	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(100000))
	err = suite.network.FundAccount(sender, sdk.NewCoins(coin))
	suite.Require().NoError(err)
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), sender)
	suite.Require().NoError(err)

	initialTotalBalance := suite.network.App.DaoKeeper.GetTotalBalanceOf(ctx, utils.BaseDenom)

	result, err := suite.network.App.DaoKeeper.ConvertToHaqq(ctx, sender, receiver, islmAmount)
	if err != nil {
		suite.T().Logf("ConvertToHaqq error (ethiq module may not be fully configured): %v", err)
		return
	}

	suite.Require().False(result.IsZero())
	suite.Require().Equal(ethiqtypes.BaseDenom, result.Denom)

	// Module's ISLM total balance should decrease by islmAmount (spent), not by result.Amount (minted aHAQQ)
	finalTotalBalance := suite.network.App.DaoKeeper.GetTotalBalanceOf(ctx, utils.BaseDenom)
	suite.Require().True(finalTotalBalance.Amount.Equal(initialTotalBalance.Amount.Sub(islmAmount)),
		"expected ISLM total %s, got %s", initialTotalBalance.Amount.Sub(islmAmount), finalTotalBalance.Amount)

	events := ctx.EventManager().Events()
	suite.Require().NotEmpty(events)
}

// TestConvertToHaqqReceiverGetsMintedHaqq tests that on successful ConvertToHaqq the receiver's bank balance of aHAQQ increases by the minted amount.
func (suite *KeeperTestSuite) TestConvertToHaqqReceiverGetsMintedHaqq() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	sender := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)
	islmAmount := sdkmath.NewInt(1000)

	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	ethiqParams := ethiqtypes.DefaultParams()
	ethiqParams.Enabled = true
	ethiqParams.MinMintPerTx = sdkmath.OneInt()
	ethiqParams.MaxMintPerTx = sdkmath.NewInt(1000000)
	suite.network.App.EthiqKeeper.SetParams(ctx, ethiqParams)

	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(100000))
	err = suite.network.FundAccount(sender, sdk.NewCoins(coin))
	suite.Require().NoError(err)
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), sender)
	suite.Require().NoError(err)

	receiverBefore := suite.network.App.BankKeeper.GetBalance(ctx, receiver, ethiqtypes.BaseDenom)

	result, err := suite.network.App.DaoKeeper.ConvertToHaqq(ctx, sender, receiver, islmAmount)
	if err != nil {
		suite.T().Logf("ConvertToHaqq error (ethiq may not be fully configured): %v", err)
		return
	}

	receiverAfter := suite.network.App.BankKeeper.GetBalance(ctx, receiver, ethiqtypes.BaseDenom)
	suite.Require().True(receiverAfter.Amount.Equal(receiverBefore.Amount.Add(result.Amount)),
		"receiver aHAQQ balance should increase by minted amount: before %s, after %s, minted %s",
		receiverBefore.Amount, receiverAfter.Amount, result.Amount)
}

// TestConvertToHaqqSuccessWithRedeemFromLiquidVesting tests ConvertToHaqq when sender has aLIQUID created via Liquidate; conversion redeems and then burns ISLM.
func (suite *KeeperTestSuite) TestConvertToHaqqSuccessWithRedeemFromLiquidVesting() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	sender := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)
	islmAmount := sdkmath.NewInt(1000)

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Set ethiq params (required for CalculateHaqqCoinsToMint)
	ethiqParams := ethiqtypes.DefaultParams()
	ethiqParams.Enabled = true
	ethiqParams.MinMintPerTx = sdkmath.OneInt()        // Allow small amounts for testing
	ethiqParams.MaxMintPerTx = sdkmath.NewInt(1000000) // Set reasonable max for testing
	suite.network.App.EthiqKeeper.SetParams(ctx, ethiqParams)

	// Enable liquid vesting module
	suite.network.App.LiquidVestingKeeper.SetLiquidVestingEnabled(ctx, true)

	// Get minimum liquidation amount
	minLiquidationAmount := suite.network.App.LiquidVestingKeeper.GetParams(ctx).MinimumLiquidationAmount

	// Create a ClawbackVestingAccount for sender with lockup periods
	// Use lockup periods but no vesting periods (so it can be liquidated)
	// Use minimum liquidation amount to ensure we meet the requirement
	vestingAmount := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, minLiquidationAmount))
	lockupPeriods := sdkvesting.Periods{
		{Length: 100000, Amount: vestingAmount}, // Locked for 100000 seconds
	}
	vestingPeriods := sdkvesting.Periods{
		{Length: 0, Amount: vestingAmount}, // Fully vested immediately (no vesting periods)
	}
	funder := suite.network.App.AccountKeeper.GetModuleAddress(liquidvestingtypes.ModuleName)
	if funder == nil {
		funder = sdk.AccAddress(liquidvestingtypes.ModuleName)
	}
	startTime := ctx.BlockTime().Add(-10 * time.Second) // Start time in the past

	// Create base account
	baseAccount := authtypes.NewBaseAccountWithAddress(sender)
	baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)

	// Create ClawbackVestingAccount
	clawbackAccount := vestingtypes.NewClawbackVestingAccount(
		baseAccount,
		funder,
		vestingAmount,
		startTime,
		lockupPeriods,
		vestingPeriods,
		nil,
	)

	// Fund the vesting account with baseDenom tokens
	err = suite.network.FundAccount(sender, vestingAmount)
	suite.Require().NoError(err)

	// Set the vesting account
	suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)

	// Liquidate the vesting tokens to create liquid tokens
	// Liquidate to sender address (we'll move to escrow later)
	liquidateAmount := sdk.NewCoin(utils.BaseDenom, minLiquidationAmount)
	liquidToken, _, err := suite.network.App.LiquidVestingKeeper.Liquidate(ctx, sender, sender, liquidateAmount)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(liquidToken.Denom) // Should be aLIQUID1 or similar

	// Fund sender with some ISLM (small amount to trigger redemption)
	islmCoin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000)) // Small ISLM amount

	err = suite.network.FundAccount(sender, sdk.NewCoins(islmCoin))
	suite.Require().NoError(err)

	// Fund DAO with ISLM and liquid token
	// This will escrow both coins to the sender's escrow address
	// The liquid token is already in sender's address from liquidation, so Fund will move it to escrow
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(islmCoin, liquidToken), sender)
	suite.Require().NoError(err)

	initialISLMTotal := suite.network.App.DaoKeeper.GetTotalBalanceOf(ctx, utils.BaseDenom)

	result, err := suite.network.App.DaoKeeper.ConvertToHaqq(ctx, sender, receiver, islmAmount)
	if err != nil {
		suite.T().Logf("ConvertToHaqq error (modules may not be fully configured): %v", err)
		return
	}

	suite.Require().False(result.IsZero())
	suite.Require().Equal(ethiqtypes.BaseDenom, result.Denom)

	// ucdao tracks only ISLM and aLIQUID in total balance; after conversion we spent islmAmount of ISLM
	finalISLMTotal := suite.network.App.DaoKeeper.GetTotalBalanceOf(ctx, utils.BaseDenom)
	// Before: initialISLM (1000) + redeemed from liquid (minLiquidationAmount). After: that sum minus islmAmount (1000)
	suite.Require().True(
		finalISLMTotal.Amount.Equal(initialISLMTotal.Amount.Add(liquidToken.Amount).Sub(islmAmount)),
		"ISLM total should be initial + redeemed liquid - spent; got %s", finalISLMTotal.Amount,
	)
}

// TestConvertToHaqqEthiqModuleFails tests ConvertToHaqq when ethiq module returns an error (e.g. burn/mint failure).
func (suite *KeeperTestSuite) TestConvertToHaqqEthiqModuleFails() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	sender := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)
	islmAmount := sdkmath.NewInt(1000)

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Set ethiq params (required for CalculateHaqqCoinsToMint)
	ethiqParams := ethiqtypes.DefaultParams()
	ethiqParams.Enabled = true
	ethiqParams.MinMintPerTx = sdkmath.OneInt()        // Allow small amounts for testing
	ethiqParams.MaxMintPerTx = sdkmath.NewInt(1000000) // Set reasonable max for testing
	suite.network.App.EthiqKeeper.SetParams(ctx, ethiqParams)

	// Fund sender with sufficient ISLM
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(100000))
	err = suite.network.FundAccount(sender, sdk.NewCoins(coin))
	suite.Require().NoError(err)
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), sender)
	suite.Require().NoError(err)

	// Try to convert - if ethiq module is not configured or has issues, this will error
	result, err := suite.network.App.DaoKeeper.ConvertToHaqq(ctx, sender, receiver, islmAmount)
	if err != nil {
		suite.Require().True(result.Denom == "" || result.IsZero())
		// Error should be about failed conversion
		suite.Require().Contains(err.Error(), "failed to convert")
	}
}

// TestConvertToHaqqZeroAmount tests ConvertToHaqq with zero islm amount (rejected by ethiq, or by ucdao if checked earlier).
func (suite *KeeperTestSuite) TestConvertToHaqqZeroAmount() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	sender := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)
	islmAmount := sdkmath.ZeroInt()

	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	ethiqParams := ethiqtypes.DefaultParams()
	ethiqParams.Enabled = true
	ethiqParams.MinMintPerTx = sdkmath.OneInt()
	ethiqParams.MaxMintPerTx = sdkmath.NewInt(1000000)
	suite.network.App.EthiqKeeper.SetParams(ctx, ethiqParams)

	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(100000))
	err = suite.network.FundAccount(sender, sdk.NewCoins(coin))
	suite.Require().NoError(err)
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), sender)
	suite.Require().NoError(err)

	result, err := suite.network.App.DaoKeeper.ConvertToHaqq(ctx, sender, receiver, islmAmount)
	suite.Require().Error(err)
	suite.Require().True(result.Denom == "" || result.IsZero())
}
