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

// TestTrackSubBalanceSaturatesAtZero covers the case the counter cannot represent.
//
// The total balance counts only what entered through Fund, TransferOwnership or genesis, while
// the escrow balances it mirrors live in bank and accept plain transfers from anyone. Removing
// more than was ever counted used to reach sdk.Coin.Sub and panic; it has to clamp instead.
func (suite *KeeperTestSuite) TestTrackSubBalanceSaturatesAtZero() {
	testCases := []struct {
		name      string
		tracked   sdkmath.Int
		remove    sdkmath.Int
		expResult sdkmath.Int
	}{
		{"partial removal", sdkmath.NewInt(1000), sdkmath.NewInt(400), sdkmath.NewInt(600)},
		{"exact removal", sdkmath.NewInt(1000), sdkmath.NewInt(1000), sdkmath.ZeroInt()},
		{"removal above the tracked amount", sdkmath.NewInt(1000), sdkmath.NewInt(1001), sdkmath.ZeroInt()},
		{"removal with nothing tracked", sdkmath.ZeroInt(), sdkmath.NewInt(1), sdkmath.ZeroInt()},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := suite.network.GetContext()
			dao := suite.network.App.DaoKeeper

			if tc.tracked.IsPositive() {
				addr := suite.keyring.GetAccAddr(0)
				funded := sdk.NewCoin(utils.BaseDenom, tc.tracked)
				suite.Require().NoError(dao.Fund(ctx, sdk.NewCoins(funded), addr))
			}

			suite.Require().NotPanics(func() {
				dao.TrackSubBalance(ctx, sdk.NewCoin(utils.BaseDenom, tc.remove))
			})

			suite.Require().Equal(
				tc.expResult.String(),
				dao.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount.String(),
			)
		})
	}
}

// TestConvertToHaqqWithUntrackedEscrow pins the ConvertToHaqq side of the same problem: the
// holder funded a little through the module and received the rest by a plain bank transfer to
// the escrow, so the conversion removes more than the module ever counted.
func (suite *KeeperTestSuite) TestConvertToHaqqWithUntrackedEscrow() {
	suite.SetupTest()
	ctx := suite.network.GetContext()
	dao := suite.network.App.DaoKeeper

	sender := suite.keyring.GetAccAddr(0)
	receiver := suite.keyring.GetAccAddr(1)

	ethiqParams := suite.network.App.EthiqKeeper.GetParams(ctx)
	ethiqParams.Enabled = true
	ethiqParams.MinMintPerTx = sdkmath.OneInt()
	ethiqParams.MaxMintPerTx = sdkmath.NewInt(1000000)
	suite.Require().NoError(suite.network.App.EthiqKeeper.SetParams(ctx, ethiqParams))

	// 100 through the module, 900 straight into the escrow.
	tracked := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(100))
	untracked := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(900))
	suite.Require().NoError(suite.network.FundAccount(sender, sdk.NewCoins(tracked)))
	suite.Require().NoError(dao.Fund(ctx, sdk.NewCoins(tracked), sender))
	suite.Require().NoError(suite.network.FundAccount(ucdaotypes.GetEscrowAddress(sender), sdk.NewCoins(untracked)))

	suite.Require().Equal(tracked.Amount, dao.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount)

	var err error
	suite.Require().NotPanics(func() {
		_, err = dao.ConvertToHaqq(ctx, sender, receiver, sdkmath.NewInt(1000))
	})
	suite.Require().NoError(err)

	suite.Require().True(dao.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount.IsZero(),
		"the counter must bottom out at zero, not go negative")
	suite.Require().True(dao.GetAccountBalances(ctx, sender).IsZero(), "the escrow must be drained")
	suite.Require().False(dao.IsHolder(ctx, sender), "an empty escrow must leave the holders index")
}
