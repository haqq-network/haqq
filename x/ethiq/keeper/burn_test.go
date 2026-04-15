package keeper_test

import (
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
)

func (suite *KeeperTestSuite) TestBurnIslmForHaqq() {
	var from, to sdk.AccAddress

	testCases := []struct {
		name        string
		malleate    func(ctx sdk.Context)
		amt         sdkmath.Int
		expRes      sdkmath.Int
		expErr      bool
		errContains string
	}{
		{
			name: "fail - module is disabled",
			malleate: func(ctx sdk.Context) {
				p := s.network.App.EthiqKeeper.GetParams(ctx)
				p.Enabled = false
				s.network.App.EthiqKeeper.SetParams(ctx, p)

				from = s.keyring.GetAccAddr(0)
				to = s.keyring.GetAccAddr(1)
			},
			amt:         sdkmath.OneInt(),
			expErr:      true,
			errContains: "module is disabled",
		},
		{
			name: "fail - empty from address",
			malleate: func(_ sdk.Context) {
				from = sdk.AccAddress{}
				to = s.keyring.GetAccAddr(1)
			},
			amt:         sdkmath.OneInt(),
			expErr:      true,
			errContains: "from_address cannot be empty",
		},
		{
			name: "fail - empty to address",
			malleate: func(_ sdk.Context) {
				from = s.keyring.GetAccAddr(0)
				to = sdk.AccAddress{}
			},
			amt:         sdkmath.OneInt(),
			expErr:      true,
			errContains: "to_address cannot be empty",
		},
		{
			name: "fail - invalid amount",
			malleate: func(_ sdk.Context) {
				from = s.keyring.GetAccAddr(0)
				to = s.keyring.GetAccAddr(1)
			},
			amt:         sdkmath.ZeroInt(),
			expErr:      true,
			errContains: "invalid amount",
		},
		{
			name: "fail - too small mint amount",
			malleate: func(_ sdk.Context) {
				from = s.keyring.GetAccAddr(0)
				to = s.keyring.GetAccAddr(1)
			},
			amt:         sdkmath.OneInt(),
			expErr:      true,
			errContains: "haqq_amount is less than min_mint_per_tx",
		},
		{
			name: "fail - too big mint amount",
			malleate: func(ctx sdk.Context) {
				p := s.network.App.EthiqKeeper.GetParams(ctx)
				p.MaxMintPerTx = sdkmath.OneInt().MulRaw(5)
				s.network.App.EthiqKeeper.SetParams(ctx, p)

				from = s.keyring.GetAccAddr(0)
				to = s.keyring.GetAccAddr(1)
				err := s.network.FundAccount(from, sdk.NewCoins(
					sdk.NewCoin(utils.BaseDenom, sdkmath.OneInt().MulRaw(1e18).MulRaw(1e9).MulRaw(30)),
				))
				suite.Require().NoError(err)
			},
			amt:         sdkmath.OneInt().MulRaw(1e18).MulRaw(1e9).MulRaw(30),
			expErr:      true,
			errContains: "burn amount exceeds pricing curve capacity",
		},
		{
			name: "fail - max supply exceeded",
			malleate: func(_ sdk.Context) {
				from = s.keyring.GetAccAddr(0)
				to = s.keyring.GetAccAddr(1)
				err := s.network.FundAccount(to, sdk.NewCoins(
					sdk.NewCoin(ethiqtypes.BaseDenom, sdkmath.OneInt().MulRaw(1e18).MulRaw(1e8).Sub(sdkmath.OneInt())),
				))
				suite.Require().NoError(err)
			},
			amt:         sdkmath.OneInt().MulRaw(1e18).MulRaw(4),
			expErr:      true,
			errContains: "total aHAQQ supply exceeds allowed maximum",
		},
		{
			name: "fail - redeem liquid vesting coins",
			malleate: func(_ sdk.Context) {
				from = s.keyring.GetAccAddr(0)
				to = s.keyring.GetAccAddr(1)
				err := s.network.FundAccount(from, sdk.NewCoins(
					sdk.NewCoin("aLIQUID99", sdkmath.OneInt().MulRaw(1e18)),
				))
				suite.Require().NoError(err)
			},
			amt:         sdkmath.OneInt().MulRaw(1e18).MulRaw(15),
			expErr:      true,
			errContains: "failed to redeem aLIQUID coins",
		},
		{
			name: "fail - insufficient funds",
			malleate: func(_ sdk.Context) {
				from = s.keyring.GetAccAddr(0)
				to = s.keyring.GetAccAddr(1)
			},
			amt:         sdkmath.OneInt().MulRaw(1e18).MulRaw(1e9),
			expErr:      true,
			errContains: "insufficient funds",
		},
		{
			name: "fail - account does not exist",
			malleate: func(_ sdk.Context) {
				from = sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq")
				to = s.keyring.GetAccAddr(1)
			},
			amt:         sdkmath.OneInt().MulRaw(1e18).MulRaw(1e9),
			expErr:      true,
			errContains: "account haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq does not exist",
		},
		{
			name: "fail - blocked receiver",
			malleate: func(_ sdk.Context) {
				from = s.keyring.GetAccAddr(0)
				to = s.network.App.AccountKeeper.GetModuleAddress(ethiqtypes.ModuleName)
			},
			amt:         sdkmath.OneInt().MulRaw(1e18),
			expErr:      true,
			errContains: "not allowed to receive funds",
		},
		{
			name: "success - burn/mint coins, eth account, no liquid vesting coins",
			malleate: func(_ sdk.Context) {
				from = s.keyring.GetAccAddr(0)
				to = s.keyring.GetAccAddr(1)
			},
			amt:    sdkmath.NewIntFromUint64(975), // at this moment, price per token is 9.75
			expRes: sdkmath.NewIntFromUint64(100),
			expErr: false,
		},
		{
			name: "success - burn/mint coins, vesting account, no liquid vesting coins",
			malleate: func(ctx sdk.Context) {
				from = s.keyring.GetAccAddr(0)
				to = s.keyring.GetAccAddr(1)
				funder := s.keyring.GetAccAddr(2)

				suite.Require().NoError(s.setupClawbackVestingAccount(ctx, from, funder, testutil.TestVestingSchedule.TotalVestingCoins, false))
			},
			amt:    sdkmath.NewIntFromUint64(975000000000), // at this moment, price per token is 9.75
			expRes: sdkmath.NewIntFromUint64(100000000000),
			expErr: false,
		},
		{
			name: "success - burn/mint coins, eth account, with liquid vesting coins",
			malleate: func(ctx sdk.Context) {
				from = s.keyring.GetAccAddr(0)
				to = s.keyring.GetAccAddr(1)
				funder := s.keyring.GetAccAddr(2)
				vesting := s.keyring.GetAccAddr(3)

				// custom liquid vesting params
				lvParams := s.network.App.LiquidVestingKeeper.GetParams(ctx)
				lvParams.MinimumLiquidationAmount = sdkmath.OneInt()
				suite.Require().NoError(s.network.App.LiquidVestingKeeper.SetParams(ctx, lvParams))

				suite.Require().NoError(s.setupClawbackVestingAccount(ctx, vesting, funder, testutil.TestVestingSchedule.TotalVestingCoins, false))
				_, _, err := s.network.App.LiquidVestingKeeper.Liquidate(ctx, vesting, from, testutil.TestVestingSchedule.TotalVestingCoins.QuoInt(sdkmath.NewIntFromUint64(2))[0])
				suite.Require().NoError(err)
			},
			amt:    testutil.TestVestingSchedule.TotalVestingCoins.QuoInt(sdkmath.NewIntFromUint64(2))[0].Amount, // at this moment, price per token is 9.75
			expRes: sdkmath.NewIntFromUint64(246153846153846153),
			expErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := s.network.GetContext()

			tc.malleate(ctx)

			res, err := s.network.App.EthiqKeeper.BurnIslmForHaqq(ctx, tc.amt, from, to)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(tc.expRes, res)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestBurnIslmForHaqqByApplicationID() {
	testCases := []struct {
		name          string
		malleate      func(ctx sdk.Context)
		from          sdk.AccAddress
		appID         uint64
		expMintAmt    string
		expMintToAddr sdk.AccAddress
		expErr        bool
		errContains   string
	}{
		{
			name: "fail - module is disabled",
			malleate: func(ctx sdk.Context) {
				p := s.network.App.EthiqKeeper.GetParams(ctx)
				p.Enabled = false
				s.network.App.EthiqKeeper.SetParams(ctx, p)
			},
			expErr:      true,
			errContains: "module is disabled",
		},
		{
			name:        "fail - application not found",
			malleate:    func(_ sdk.Context) {},
			appID:       999,
			expErr:      true,
			errContains: "application 999 not found",
		},
		{
			name: "fail - application already executed",
			malleate: func(ctx sdk.Context) {
				s.network.App.EthiqKeeper.SetApplicationAsExecuted(ctx, 33)
			},
			appID:       33,
			expErr:      true,
			errContains: "application ID 33 is already executed",
		},
		{
			name:        "fail - application is canceled",
			malleate:    func(_ sdk.Context) {},
			appID:       7,
			expErr:      true,
			errContains: "application ID 7 is canceled",
		},
		{
			name:        "fail - wrong sender address",
			malleate:    func(_ sdk.Context) {},
			from:        s.keyring.GetAccAddr(0),
			appID:       5,
			expErr:      true,
			errContains: "application ID 5 can be executed by",
		},
		{
			name: "fail - max supply exceeded",
			malleate: func(_ sdk.Context) {
				err := s.network.FundAccount(s.keyring.GetAccAddr(0), sdk.NewCoins(
					sdk.NewCoin(ethiqtypes.BaseDenom, sdkmath.OneInt().MulRaw(1e18).MulRaw(1e8).Sub(sdkmath.OneInt())),
				))
				suite.Require().NoError(err)
			},
			from:        sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq"),
			appID:       8,
			expErr:      true,
			errContains: "total aHAQQ supply exceeds allowed maximum",
		},
		{
			name:        "fail - account does not exist",
			malleate:    func(_ sdk.Context) {},
			from:        sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq"),
			appID:       8,
			expErr:      true,
			errContains: "account haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq does not exist",
		},
		{
			name: "fail - redeem liquid vesting coins",
			malleate: func(_ sdk.Context) {
				err := s.network.FundAccount(
					sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq"),
					sdk.NewCoins(sdk.NewCoin("aLIQUID99", sdkmath.OneInt().MulRaw(1e18))),
				)
				suite.Require().NoError(err)
			},
			from:        sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq"),
			appID:       8,
			expErr:      true,
			errContains: "failed to redeem aLIQUID coins",
		},
		{
			name: "fail - insufficient funds",
			malleate: func(_ sdk.Context) {
				// Create and fund account, but not enough coins
				fundAmt, ok := sdkmath.NewIntFromString("100000000000000000000")
				suite.Require().True(ok)
				err := s.network.FundAccount(
					sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq"),
					sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmt)),
				)
				suite.Require().NoError(err)
			},
			from:        sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq"),
			appID:       8,
			expErr:      true,
			errContains: "insufficient funds",
		},
		{
			name: "success - bank as source of funds, no liquid vesting coins",
			malleate: func(_ sdk.Context) {
				// Create and fund account
				fundAmt, ok := sdkmath.NewIntFromString("1000000000000000000")
				suite.Require().True(ok)
				err := s.network.FundAccount(
					sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq"),
					sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmt)),
				)
				suite.Require().NoError(err)
			},
			from:          sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq"),
			appID:         29,
			expMintAmt:    "117647058823529411",
			expMintToAddr: sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq"),
			expErr:        false,
			errContains:   "",
		},
		{
			name: "success - bank as source of funds, with liquid vesting coins",
			malleate: func(ctx sdk.Context) {
				// Create and fund account
				fromAddr := sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq")
				fundAmt, ok := sdkmath.NewIntFromString("999999999999999999") // at least 1 coin have to be liquid
				suite.Require().True(ok)
				suite.Require().NoError(s.network.FundAccount(fromAddr, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmt))))

				funder := s.keyring.GetAccAddr(2)
				vesting := s.keyring.GetAccAddr(3)

				// custom liquid vesting params
				lvParams := s.network.App.LiquidVestingKeeper.GetParams(ctx)
				lvParams.MinimumLiquidationAmount = sdkmath.OneInt()
				suite.Require().NoError(s.network.App.LiquidVestingKeeper.SetParams(ctx, lvParams))

				suite.Require().NoError(s.setupClawbackVestingAccount(ctx, vesting, funder, testutil.TestVestingSchedule.TotalVestingCoins, false))
				_, _, err := s.network.App.LiquidVestingKeeper.Liquidate(ctx, vesting, fromAddr, testutil.TestVestingSchedule.TotalVestingCoins.QuoInt(sdkmath.NewIntFromUint64(2))[0])
				suite.Require().NoError(err)
				suite.Require().NoError(s.network.NextBlockAfter(5 * time.Second))
			},
			from:          sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq"),
			appID:         29,
			expMintAmt:    "117647058823529411",
			expMintToAddr: sdk.MustAccAddressFromBech32("haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq"),
			expErr:        false,
			errContains:   "",
		},
		{
			name: "success - UCDAO as source of funds, no liquid vesting coins",
			malleate: func(ctx sdk.Context) {
				fromAddr := sdk.MustAccAddressFromBech32("haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0")
				fundAmt, ok := sdkmath.NewIntFromString("1000000000000000000000000")
				suite.Require().True(ok)
				suite.Require().NoError(s.network.FundAccount(fromAddr, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmt))))
				suite.Require().NoError(s.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmt)), fromAddr))
			},
			from:          sdk.MustAccAddressFromBech32("haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0"),
			appID:         12,
			expMintAmt:    "333333333333333333333333",
			expMintToAddr: sdk.MustAccAddressFromBech32("haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0"),
			expErr:        false,
			errContains:   "",
		},
		{
			name: "success - UCDAO as source of funds, with liquid vesting coins",
			malleate: func(ctx sdk.Context) {
				// fund funder account
				fromAddr := sdk.MustAccAddressFromBech32("haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0")
				fundAmt, ok := sdkmath.NewIntFromString("999999999999999999999999") // at least one coin have to be liquid
				suite.Require().True(ok)
				suite.Require().NoError(s.network.FundAccount(fromAddr, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmt))))
				// custom liquid vesting params
				lvParams := s.network.App.LiquidVestingKeeper.GetParams(ctx)
				lvParams.MinimumLiquidationAmount = sdkmath.OneInt()
				suite.Require().NoError(s.network.App.LiquidVestingKeeper.SetParams(ctx, lvParams))

				funder := s.keyring.GetAccAddr(2)
				vesting := s.keyring.GetAccAddr(3)
				suite.Require().NoError(s.setupClawbackVestingAccount(ctx, vesting, funder, testutil.TestVestingSchedule.TotalVestingCoins, false))
				liquidCoin, _, err := s.network.App.LiquidVestingKeeper.Liquidate(ctx, vesting, fromAddr, testutil.TestVestingSchedule.TotalVestingCoins.QuoInt(sdkmath.NewIntFromUint64(2))[0])
				suite.Require().NoError(err)
				suite.Require().NoError(s.network.NextBlockAfter(5 * time.Second))

				suite.Require().NoError(s.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmt), liquidCoin), fromAddr))
			},
			from:          sdk.MustAccAddressFromBech32("haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0"),
			appID:         12,
			expMintAmt:    "333333333333333333333333",
			expMintToAddr: sdk.MustAccAddressFromBech32("haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0"),
			expErr:        false,
			errContains:   "",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := s.network.GetContext()

			tc.malleate(ctx)
			// refresh context
			ctx = s.network.GetContext()

			resAmt, toAddr, err := s.network.App.EthiqKeeper.BurnIslmForHaqqByApplicationID(ctx, tc.from, tc.appID)
			if tc.expErr {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(resAmt)
				suite.Require().NotNil(toAddr)
				expMintAmt, ok := sdkmath.NewIntFromString(tc.expMintAmt)
				suite.Require().True(ok)
				suite.Require().Equal(expMintAmt.String(), resAmt.String())
				suite.Require().Equal(tc.expMintToAddr, toAddr)
			}
		})
	}
}
