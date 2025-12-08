package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ethiq/types"
)

// TestLogger tests the Logger function
func (suite *KeeperTestSuite) TestLogger() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	logger := suite.network.App.EthiqKeeper.Logger(ctx)
	suite.Require().NotNil(logger)
}

// TestGetEthiqSupply tests the GetEthiqSupply function
func (suite *KeeperTestSuite) TestGetEthiqSupply() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Initially should be zero
	supply := suite.network.App.EthiqKeeper.GetEthiqSupply(ctx)
	suite.Require().True(supply.IsZero())

	// Mint some ethiq coins
	ethiqAmount := sdkmath.NewInt(1000)
	ethiqCoin := sdk.NewCoin(types.BaseDenom, ethiqAmount)
	err := suite.network.App.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(ethiqCoin))
	suite.Require().NoError(err)

	// Check supply again
	supply = suite.network.App.EthiqKeeper.GetEthiqSupply(ctx)
	suite.Require().Equal(ethiqAmount, supply)
}

// TestGetParams tests the GetParams function
func (suite *KeeperTestSuite) TestGetParams() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Get default params
	params := suite.network.App.EthiqKeeper.GetParams(ctx)
	suite.Require().NotNil(params)
	suite.Require().False(params.Enabled) // Default is disabled
	suite.Require().True(params.StartRate.IsPositive())
	suite.Require().True(params.CurveCoefficient.IsPositive())
	suite.Require().True(params.PowerCoefficient.IsPositive())
}

// TestSetParams tests the SetParams function
func (suite *KeeperTestSuite) TestSetParams() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Create custom params
	customParams := types.DefaultParams()
	customParams.Enabled = true
	customParams.StartRate = sdkmath.LegacyNewDec(2)
	customParams.MinMintPerTx = sdkmath.NewInt(100)
	customParams.MaxMintPerTx = sdkmath.NewInt(1000000)

	suite.network.App.EthiqKeeper.SetParams(ctx, customParams)

	// Verify params were set
	params := suite.network.App.EthiqKeeper.GetParams(ctx)
	suite.Require().True(params.Enabled)
	suite.Require().Equal(customParams.StartRate, params.StartRate)
	suite.Require().Equal(customParams.MinMintPerTx, params.MinMintPerTx)
	suite.Require().Equal(customParams.MaxMintPerTx, params.MaxMintPerTx)
}

// TestIsModuleEnabled tests the IsModuleEnabled function
func (suite *KeeperTestSuite) TestIsModuleEnabled() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Default should be disabled
	enabled := suite.network.App.EthiqKeeper.IsModuleEnabled(ctx)
	suite.Require().False(enabled)

	// Enable module
	params := types.DefaultParams()
	params.Enabled = true
	suite.network.App.EthiqKeeper.SetParams(ctx, params)

	// Check again
	enabled = suite.network.App.EthiqKeeper.IsModuleEnabled(ctx)
	suite.Require().True(enabled)
}

// TestCalculateRequiredISLM tests the CalculateRequiredISLM function
func (suite *KeeperTestSuite) TestCalculateRequiredISLM() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Enable module
	params := types.DefaultParams()
	params.Enabled = true
	params.StartRate = sdkmath.LegacyNewDec(1)
	params.CurveCoefficient = sdkmath.LegacyNewDecWithPrec(5, 9)  // 0.000000005
	params.PowerCoefficient = sdkmath.LegacyNewDecWithPrec(11, 1) // 1.1
	params.MinMintPerTx = sdkmath.NewInt(1)
	params.MaxMintPerTx = sdkmath.NewInt(1000000000000000000) // 1e18
	suite.network.App.EthiqKeeper.SetParams(ctx, params)

	// Test with valid amount
	ethiqAmount := sdkmath.NewInt(1000)
	requiredIslm, pricePerUnit, err := suite.network.App.EthiqKeeper.CalculateRequiredISLM(ctx, ethiqAmount)
	suite.Require().NoError(err)
	suite.Require().True(requiredIslm.IsPositive())
	suite.Require().True(pricePerUnit.IsPositive())
	suite.Require().True(pricePerUnit.GTE(sdkmath.LegacyNewDec(1))) // Price should be at least 1
}

// TestCalculateRequiredISLMModuleDisabled tests CalculateRequiredISLM when module is disabled
func (suite *KeeperTestSuite) TestCalculateRequiredISLMModuleDisabled() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Module is disabled by default
	ethiqAmount := sdkmath.NewInt(1000)
	requiredIslm, pricePerUnit, err := suite.network.App.EthiqKeeper.CalculateRequiredISLM(ctx, ethiqAmount)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrModuleDisabled)
	// When error occurs, zero values are returned
	suite.Require().NotNil(requiredIslm)
	suite.Require().NotNil(pricePerUnit)
}

// TestCalculateRequiredISLMInvalidAmount tests CalculateRequiredISLM with invalid amounts
func (suite *KeeperTestSuite) TestCalculateRequiredISLMInvalidAmount() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Enable module
	params := types.DefaultParams()
	params.Enabled = true
	suite.network.App.EthiqKeeper.SetParams(ctx, params)

	// Test with zero amount
	requiredIslm, pricePerUnit, err := suite.network.App.EthiqKeeper.CalculateRequiredISLM(ctx, sdkmath.ZeroInt())
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInvalidAmount)
	// When error occurs, zero values are returned
	suite.Require().NotNil(requiredIslm)
	suite.Require().NotNil(pricePerUnit)

	// Test with negative amount
	requiredIslm, pricePerUnit, err = suite.network.App.EthiqKeeper.CalculateRequiredISLM(ctx, sdkmath.NewInt(-100))
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInvalidAmount)
	// When error occurs, zero values are returned
	suite.Require().NotNil(requiredIslm)
	suite.Require().NotNil(pricePerUnit)
}

// TestCalculateRequiredISLMWithExistingSupply tests CalculateRequiredISLM with existing supply
func (suite *KeeperTestSuite) TestCalculateRequiredISLMWithExistingSupply() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Enable module
	params := types.DefaultParams()
	params.Enabled = true
	params.StartRate = sdkmath.LegacyNewDec(1)
	params.CurveCoefficient = sdkmath.LegacyNewDecWithPrec(5, 9)
	params.PowerCoefficient = sdkmath.LegacyNewDecWithPrec(11, 1)
	suite.network.App.EthiqKeeper.SetParams(ctx, params)

	// Mint some initial supply
	initialSupply := sdkmath.NewInt(1000000)
	ethiqCoin := sdk.NewCoin(types.BaseDenom, initialSupply)
	err := suite.network.App.BankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(ethiqCoin))
	suite.Require().NoError(err)

	// Calculate required ISLM for new mint
	ethiqAmount := sdkmath.NewInt(1000)
	requiredIslm1, pricePerUnit1, err := suite.network.App.EthiqKeeper.CalculateRequiredISLM(ctx, ethiqAmount)
	suite.Require().NoError(err)
	suite.Require().True(requiredIslm1.IsPositive())
	suite.Require().True(pricePerUnit1.IsPositive())

	// Price should be higher than start rate due to bonding curve
	suite.Require().True(pricePerUnit1.GTE(params.StartRate))
}

// TestConvertToEthiq tests the ConvertToEthiq function
func (suite *KeeperTestSuite) TestConvertToEthiq() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Enable module
	params := types.DefaultParams()
	params.Enabled = true
	params.StartRate = sdkmath.LegacyNewDec(1)
	params.CurveCoefficient = sdkmath.LegacyNewDecWithPrec(5, 9)
	params.PowerCoefficient = sdkmath.LegacyNewDecWithPrec(11, 1)
	params.MinMintPerTx = sdkmath.NewInt(1)
	params.MaxMintPerTx = sdkmath.NewInt(1000000000000000000)
	suite.network.App.EthiqKeeper.SetParams(ctx, params)

	fromAddr := suite.keyring.GetAccAddr(0)
	toAddr := suite.keyring.GetAccAddr(1)

	// Fund from address with ISLM
	islmAmount := sdkmath.NewInt(1000000)
	islmCoin := sdk.NewCoin(utils.BaseDenom, islmAmount)
	err := suite.network.FundAccount(fromAddr, sdk.NewCoins(islmCoin))
	suite.Require().NoError(err)

	// Calculate required ISLM
	ethiqAmount := sdkmath.NewInt(1000)
	requiredIslm, _, err := suite.network.App.EthiqKeeper.CalculateRequiredISLM(ctx, ethiqAmount)
	suite.Require().NoError(err)

	// Convert to ethiq
	maxIslmAmount := requiredIslm.MulRaw(2) // Provide more than required
	actualIslmUsed, err := suite.network.App.EthiqKeeper.ConvertToEthiq(ctx, ethiqAmount, maxIslmAmount, fromAddr, toAddr)
	suite.Require().NoError(err)
	suite.Require().Equal(requiredIslm, actualIslmUsed)

	// Verify ethiq was minted to toAddr
	ethiqBalance := suite.network.App.BankKeeper.GetBalance(ctx, toAddr, types.BaseDenom)
	suite.Require().Equal(ethiqAmount, ethiqBalance.Amount)

	// Verify ISLM was burned (check total burned amount)
	totalBurned := suite.network.App.EthiqKeeper.GetTotalBurnedAmount(ctx)
	suite.Require().Equal(requiredIslm, totalBurned.Amount)
}

// TestConvertToEthiqModuleDisabled tests ConvertToEthiq when module is disabled
func (suite *KeeperTestSuite) TestConvertToEthiqModuleDisabled() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	fromAddr := suite.keyring.GetAccAddr(0)
	toAddr := suite.keyring.GetAccAddr(1)
	ethiqAmount := sdkmath.NewInt(1000)
	maxIslmAmount := sdkmath.NewInt(10000)

	// Module is disabled by default
	actualIslmUsed, err := suite.network.App.EthiqKeeper.ConvertToEthiq(ctx, ethiqAmount, maxIslmAmount, fromAddr, toAddr)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrModuleDisabled)
	suite.Require().True(actualIslmUsed.IsZero())
}

// TestConvertToEthiqInvalidAddresses tests ConvertToEthiq with invalid addresses
func (suite *KeeperTestSuite) TestConvertToEthiqInvalidAddresses() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Enable module
	params := types.DefaultParams()
	params.Enabled = true
	params.MinMintPerTx = sdkmath.NewInt(1)
	params.MaxMintPerTx = sdkmath.NewInt(1000000000000000000)
	suite.network.App.EthiqKeeper.SetParams(ctx, params)

	validAddr := suite.keyring.GetAccAddr(0)
	ethiqAmount := sdkmath.NewInt(1000)
	maxIslmAmount := sdkmath.NewInt(10000)

	// Test with empty from address
	actualIslmUsed, err := suite.network.App.EthiqKeeper.ConvertToEthiq(ctx, ethiqAmount, maxIslmAmount, sdk.AccAddress{}, validAddr)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInvalidAddress)
	suite.Require().True(actualIslmUsed.IsZero())

	// Test with empty to address
	actualIslmUsed, err = suite.network.App.EthiqKeeper.ConvertToEthiq(ctx, ethiqAmount, maxIslmAmount, validAddr, sdk.AccAddress{})
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInvalidAddress)
	suite.Require().True(actualIslmUsed.IsZero())
}

// TestConvertToEthiqInvalidAmount tests ConvertToEthiq with invalid amounts
func (suite *KeeperTestSuite) TestConvertToEthiqInvalidAmount() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Enable module
	params := types.DefaultParams()
	params.Enabled = true
	params.MinMintPerTx = sdkmath.NewInt(1000)
	params.MaxMintPerTx = sdkmath.NewInt(1000000)
	suite.network.App.EthiqKeeper.SetParams(ctx, params)

	fromAddr := suite.keyring.GetAccAddr(0)
	toAddr := suite.keyring.GetAccAddr(1)
	maxIslmAmount := sdkmath.NewInt(10000)

	// Test with amount less than min
	ethiqAmount := sdkmath.NewInt(100)
	actualIslmUsed, err := suite.network.App.EthiqKeeper.ConvertToEthiq(ctx, ethiqAmount, maxIslmAmount, fromAddr, toAddr)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInvalidAmount)
	suite.Require().True(actualIslmUsed.IsZero())

	// Test with amount greater than max
	ethiqAmount = sdkmath.NewInt(2000000)
	actualIslmUsed, err = suite.network.App.EthiqKeeper.ConvertToEthiq(ctx, ethiqAmount, maxIslmAmount, fromAddr, toAddr)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInvalidAmount)
	suite.Require().True(actualIslmUsed.IsZero())

	// Test with zero amount
	ethiqAmount = sdkmath.ZeroInt()
	actualIslmUsed, err = suite.network.App.EthiqKeeper.ConvertToEthiq(ctx, ethiqAmount, maxIslmAmount, fromAddr, toAddr)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInvalidAmount)
	suite.Require().True(actualIslmUsed.IsZero())
}

// TestConvertToEthiqInsufficientFunds tests ConvertToEthiq with insufficient max ISLM
func (suite *KeeperTestSuite) TestConvertToEthiqInsufficientFunds() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Enable module
	params := types.DefaultParams()
	params.Enabled = true
	params.StartRate = sdkmath.LegacyNewDec(1)
	params.CurveCoefficient = sdkmath.LegacyNewDecWithPrec(5, 9)
	params.PowerCoefficient = sdkmath.LegacyNewDecWithPrec(11, 1)
	params.MinMintPerTx = sdkmath.NewInt(1)
	params.MaxMintPerTx = sdkmath.NewInt(1000000000000000000)
	suite.network.App.EthiqKeeper.SetParams(ctx, params)

	fromAddr := suite.keyring.GetAccAddr(0)
	toAddr := suite.keyring.GetAccAddr(1)

	// Calculate required ISLM
	ethiqAmount := sdkmath.NewInt(1000)
	requiredIslm, _, err := suite.network.App.EthiqKeeper.CalculateRequiredISLM(ctx, ethiqAmount)
	suite.Require().NoError(err)

	// Try with max ISLM less than required
	maxIslmAmount := requiredIslm.Sub(sdkmath.OneInt())
	actualIslmUsed, err := suite.network.App.EthiqKeeper.ConvertToEthiq(ctx, ethiqAmount, maxIslmAmount, fromAddr, toAddr)
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, types.ErrInsufficientFunds)
	suite.Require().True(actualIslmUsed.IsZero())
}

// TestEnsureEthiqMetadata tests the EnsureEthiqMetadata function
func (suite *KeeperTestSuite) TestEnsureEthiqMetadata() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Ensure metadata
	err := suite.network.App.EthiqKeeper.EnsureEthiqMetadata(ctx)
	suite.Require().NoError(err)

	// Verify metadata exists
	metadata, found := suite.network.App.BankKeeper.GetDenomMetaData(ctx, types.BaseDenom)
	suite.Require().True(found)
	suite.Require().Equal(types.BaseDenom, metadata.Base)
	suite.Require().Equal(types.DisplayDenom, metadata.Display)
	suite.Require().Equal("Ethiq", metadata.Name)
	suite.Require().Equal("ETHIQ", metadata.Symbol)
	suite.Require().Len(metadata.DenomUnits, 2)

	// Call again should not error (idempotent)
	err = suite.network.App.EthiqKeeper.EnsureEthiqMetadata(ctx)
	suite.Require().NoError(err)
}

// TestEnsureEthiqERC20Registration tests the EnsureEthiqERC20Registration function
func (suite *KeeperTestSuite) TestEnsureEthiqERC20Registration() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Ensure ERC20 registration
	err := suite.network.App.EthiqKeeper.EnsureEthiqERC20Registration(ctx)
	suite.Require().NoError(err)

	// Verify denom is registered
	isRegistered := suite.network.App.Erc20Keeper.IsDenomRegistered(ctx, types.BaseDenom)
	suite.Require().True(isRegistered)

	// Call again should not error (idempotent)
	err = suite.network.App.EthiqKeeper.EnsureEthiqERC20Registration(ctx)
	suite.Require().NoError(err)
}

// TestGetTotalBurnedAmount tests the GetTotalBurnedAmount function
func (suite *KeeperTestSuite) TestGetTotalBurnedAmount() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Initially should be zero
	totalBurned := suite.network.App.EthiqKeeper.GetTotalBurnedAmount(ctx)
	suite.Require().True(totalBurned.Amount.IsZero())
	suite.Require().Equal(utils.BaseDenom, totalBurned.Denom)
}

// TestSetTotalBurnedAmount tests the SetTotalBurnedAmount function
func (suite *KeeperTestSuite) TestSetTotalBurnedAmount() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Set burned amount
	burnedAmount := sdkmath.NewInt(5000)
	burnedCoin := sdk.NewCoin(utils.BaseDenom, burnedAmount)
	suite.network.App.EthiqKeeper.SetTotalBurnedAmount(ctx, burnedCoin)

	// Verify it was set
	totalBurned := suite.network.App.EthiqKeeper.GetTotalBurnedAmount(ctx)
	suite.Require().Equal(burnedAmount, totalBurned.Amount)
	suite.Require().Equal(utils.BaseDenom, totalBurned.Denom)
}

// TestAddToTotalBurnedAmount tests the AddToTotalBurnedAmount function
func (suite *KeeperTestSuite) TestAddToTotalBurnedAmount() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Initially zero
	totalBurned := suite.network.App.EthiqKeeper.GetTotalBurnedAmount(ctx)
	suite.Require().True(totalBurned.Amount.IsZero())

	// Add some amount
	amount1 := sdkmath.NewInt(1000)
	suite.network.App.EthiqKeeper.AddToTotalBurnedAmount(ctx, amount1)

	// Verify
	totalBurned = suite.network.App.EthiqKeeper.GetTotalBurnedAmount(ctx)
	suite.Require().Equal(amount1, totalBurned.Amount)

	// Add more
	amount2 := sdkmath.NewInt(2000)
	suite.network.App.EthiqKeeper.AddToTotalBurnedAmount(ctx, amount2)

	// Verify cumulative
	totalBurned = suite.network.App.EthiqKeeper.GetTotalBurnedAmount(ctx)
	expectedTotal := amount1.Add(amount2)
	suite.Require().Equal(expectedTotal, totalBurned.Amount)
}

// TestConvertToEthiqEvents tests that ConvertToEthiq emits events
func (suite *KeeperTestSuite) TestConvertToEthiqEvents() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	// Enable module
	params := types.DefaultParams()
	params.Enabled = true
	params.StartRate = sdkmath.LegacyNewDec(1)
	params.CurveCoefficient = sdkmath.LegacyNewDecWithPrec(5, 9)
	params.PowerCoefficient = sdkmath.LegacyNewDecWithPrec(11, 1)
	params.MinMintPerTx = sdkmath.NewInt(1)
	params.MaxMintPerTx = sdkmath.NewInt(1000000000000000000)
	suite.network.App.EthiqKeeper.SetParams(ctx, params)

	fromAddr := suite.keyring.GetAccAddr(0)
	toAddr := suite.keyring.GetAccAddr(1)

	// Fund from address
	islmAmount := sdkmath.NewInt(1000000)
	islmCoin := sdk.NewCoin(utils.BaseDenom, islmAmount)
	err := suite.network.FundAccount(fromAddr, sdk.NewCoins(islmCoin))
	suite.Require().NoError(err)

	// Convert to ethiq
	ethiqAmount := sdkmath.NewInt(1000)
	requiredIslm, _, err := suite.network.App.EthiqKeeper.CalculateRequiredISLM(ctx, ethiqAmount)
	suite.Require().NoError(err)

	maxIslmAmount := requiredIslm.MulRaw(2)
	_, err = suite.network.App.EthiqKeeper.ConvertToEthiq(ctx, ethiqAmount, maxIslmAmount, fromAddr, toAddr)
	suite.Require().NoError(err)

	// Check events were emitted
	events := ctx.EventManager().Events()
	suite.Require().NotEmpty(events)
}
