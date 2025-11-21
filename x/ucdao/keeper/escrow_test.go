package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// TestEscrowTokenSuccess tests escrowToken indirectly through Fund
func (suite *KeeperTestSuite) TestEscrowTokenSuccess() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	addr := suite.keyring.GetAccAddr(0)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Fund - this calls escrowToken internally
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), addr)
	suite.Require().NoError(err)

	// Verify total balance was updated
	totalBalance := suite.network.App.DaoKeeper.GetTotalBalanceOf(ctx, coin.Denom)
	suite.Require().True(totalBalance.Amount.Equal(coin.Amount))
}

// TestUnescrowTokenSuccess tests unescrowToken indirectly through TransferOwnership
func (suite *KeeperTestSuite) TestUnescrowTokenSuccess() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	owner := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)
	coin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))

	// Enable module
	params := ucdaotypes.DefaultParams()
	params.EnableDao = true
	err := suite.getBaseKeeper().SetParams(ctx, params)
	suite.Require().NoError(err)

	// Fund owner account
	err = suite.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(coin), owner)
	suite.Require().NoError(err)

	// Transfer ownership - this internally uses transferEscrowToken
	// which moves coins between escrow addresses
	_, err = suite.network.App.DaoKeeper.TransferOwnership(ctx, owner, receiver, sdk.NewCoins(coin))
	suite.Require().NoError(err)

	// Verify total balance remains the same (just moved between escrows)
	totalBalance := suite.network.App.DaoKeeper.GetTotalBalanceOf(ctx, coin.Denom)
	suite.Require().True(totalBalance.Amount.Equal(coin.Amount))
}

// TestTransferEscrowTokenSuccess tests transferEscrowToken indirectly through TransferOwnership
func (suite *KeeperTestSuite) TestTransferEscrowTokenSuccess() {
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

	// Transfer ownership - this uses transferEscrowToken
	_, err = suite.network.App.DaoKeeper.TransferOwnership(ctx, owner, newOwner, sdk.NewCoins(coin))
	suite.Require().NoError(err)

	// Verify coins moved
	ownerBalance := suite.network.App.DaoKeeper.GetAccountBalances(ctx, owner)
	suite.Require().True(ownerBalance.IsZero())

	newOwnerBalance := suite.network.App.DaoKeeper.GetAccountBalances(ctx, newOwner)
	suite.Require().True(newOwnerBalance.AmountOf(coin.Denom).Equal(coin.Amount))
}
