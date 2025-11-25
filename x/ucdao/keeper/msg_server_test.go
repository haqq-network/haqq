package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ucdao/keeper"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// TestNewMsgServerImpl tests the NewMsgServerImpl function
func (suite *KeeperTestSuite) TestNewMsgServerImpl() {
	suite.SetupTest()
	msgServer := keeper.NewMsgServerImpl(suite.network.App.DaoKeeper)
	suite.Require().NotNil(msgServer)
}

// TestMsgServerFund tests the msgServer Fund function
func (suite *KeeperTestSuite) TestMsgServerFund() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	msg := ucdaotypes.NewMsgFund(sdk.NewCoins(coin), addr)
	msgServer := keeper.NewMsgServerImpl(suite.network.App.DaoKeeper)

	resp, err := msgServer.Fund(ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
}

// TestMsgServerFundInvalidMessage tests msgServer Fund with invalid message
func (suite *KeeperTestSuite) TestMsgServerFundInvalidMessage() {
	suite.SetupTest()
	ctx := suite.network.GetContext()

	msg := &ucdaotypes.MsgFund{
		Depositor: "invalid-address",
		Amount:    sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))),
	}
	msgServer := keeper.NewMsgServerImpl(suite.network.App.DaoKeeper)

	resp, err := msgServer.Fund(ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(resp)
}

// TestMsgServerFundInvalidDenom tests msgServer Fund with invalid denom
func (suite *KeeperTestSuite) TestMsgServerFundInvalidDenom() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	coin := sdk.NewCoin("invalid-denom", sdkmath.NewInt(1000))

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Fund the account first so they have tokens to send
	err = suite.network.FundAccount(addr, sdk.NewCoins(coin))
	suite.Require().NoError(err)

	msg := ucdaotypes.NewMsgFund(sdk.NewCoins(coin), addr)
	msgServer := keeper.NewMsgServerImpl(suite.network.App.DaoKeeper)

	resp, err := msgServer.Fund(ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(resp)
	suite.Require().ErrorIs(err, ucdaotypes.ErrInvalidDenom)
}

// TestMsgServerTransferOwnership tests the msgServer TransferOwnership function
func (suite *KeeperTestSuite) TestMsgServerTransferOwnership() {
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
	msg := ucdaotypes.NewMsgTransferOwnership(owner, newOwner)
	msgServer := keeper.NewMsgServerImpl(suite.network.App.DaoKeeper)

	resp, err := msgServer.TransferOwnership(ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
}

// TestMsgServerTransferOwnershipNotEligible tests msgServer TransferOwnership when owner has no balance
func (suite *KeeperTestSuite) TestMsgServerTransferOwnershipNotEligible() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	owner := suite.keyring.GetAccAddr(0)
	newOwner := suite.keyring.GetAccAddr(1)

	msg := ucdaotypes.NewMsgTransferOwnership(owner, newOwner)
	msgServer := keeper.NewMsgServerImpl(suite.network.App.DaoKeeper)

	resp, err := msgServer.TransferOwnership(ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(resp)
	suite.Require().ErrorIs(err, ucdaotypes.ErrNotEligible)
}

// TestMsgServerTransferOwnershipWithRatio tests the msgServer TransferOwnershipWithRatio function
func (suite *KeeperTestSuite) TestMsgServerTransferOwnershipWithRatio() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	owner := suite.keyring.GetAccAddr(0)
	newOwner := suite.keyring.GetAccAddr(1)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	halfCoin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(500))

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Fund owner account
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), owner)
	suite.Require().NoError(err)

	// Transfer ownership with ratio (50%)
	ratio := sdkmath.LegacyNewDecWithPrec(5, 1) // 0.5
	msg := ucdaotypes.NewMsgTransferOwnershipWithRatio(owner, newOwner, ratio)
	msgServer := keeper.NewMsgServerImpl(suite.network.App.DaoKeeper)

	resp, err := msgServer.TransferOwnershipWithRatio(ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Equal(sdk.NewCoins(halfCoin), resp.Coins)
}

// TestMsgServerTransferOwnershipWithRatioInvalidRatio tests msgServer TransferOwnershipWithRatio with invalid ratio
func (suite *KeeperTestSuite) TestMsgServerTransferOwnershipWithRatioInvalidRatio() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	owner := suite.keyring.GetAccAddr(0)
	newOwner := suite.keyring.GetAccAddr(1)

	// Invalid ratio > 1
	ratio := sdkmath.LegacyNewDecWithPrec(15, 1) // 1.5
	msg := ucdaotypes.NewMsgTransferOwnershipWithRatio(owner, newOwner, ratio)
	msgServer := keeper.NewMsgServerImpl(suite.network.App.DaoKeeper)

	resp, err := msgServer.TransferOwnershipWithRatio(ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(resp)
}

// TestMsgServerTransferOwnershipWithAmount tests the msgServer TransferOwnershipWithAmount function
func (suite *KeeperTestSuite) TestMsgServerTransferOwnershipWithAmount() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	owner := suite.keyring.GetAccAddr(0)
	newOwner := suite.keyring.GetAccAddr(1)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))
	transferAmount := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(300))

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Fund owner account
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), owner)
	suite.Require().NoError(err)

	// Transfer ownership with specific amount
	msg := ucdaotypes.NewMsgTransferOwnershipWithAmount(owner, newOwner, sdk.NewCoins(transferAmount))
	msgServer := keeper.NewMsgServerImpl(suite.network.App.DaoKeeper)

	resp, err := msgServer.TransferOwnershipWithAmount(ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
}
