package keeper_test

import (
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

func (suite *KeeperTestSuite) TestBurnIslmForHaqq() {
	var from, to sdk.AccAddress

	testCases := []struct {
		name        string
		malleate    func(ctx sdk.Context)
		amt         sdkmath.Int
		expRes      sdkmath.Int
		calcExpRes  bool
		expErr      bool
		errContains string
	}{
		{
			name: "fail - module is disabled",
			malleate: func(ctx sdk.Context) {
				p := s.network.App.EthiqKeeper.GetParams(ctx)
				p.Enabled = false
				suite.Require().NoError(s.network.App.EthiqKeeper.SetParams(ctx, p))

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
				suite.Require().NoError(s.network.App.EthiqKeeper.SetParams(ctx, p))

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
			amt:        sdkmath.NewIntFromUint64(975), // at this moment, price per token is 9.75
			expRes:     sdkmath.NewIntFromUint64(100),
			calcExpRes: true,
			expErr:     false,
		},
		{
			name: "success - burn/mint coins, vesting account, no liquid vesting coins",
			malleate: func(ctx sdk.Context) {
				from = s.keyring.GetAccAddr(0)
				to = s.keyring.GetAccAddr(1)
				funder := s.keyring.GetAccAddr(2)

				suite.Require().NoError(s.setupClawbackVestingAccount(ctx, from, funder, testutil.TestVestingSchedule.TotalVestingCoins, false))
			},
			amt:        sdkmath.NewIntFromUint64(975000000000), // at this moment, price per token is 9.75
			expRes:     sdkmath.NewIntFromUint64(100000000000),
			calcExpRes: true,
			expErr:     false,
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
			amt:        testutil.TestVestingSchedule.TotalVestingCoins.QuoInt(sdkmath.NewIntFromUint64(2))[0].Amount, // at this moment, price per token is 9.75
			expRes:     sdkmath.NewIntFromUint64(246153846153846153),
			calcExpRes: true,
			expErr:     false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := s.network.GetContext()

			tc.malleate(ctx)
			if tc.calcExpRes {
				expRes, expErr := s.network.App.EthiqKeeper.CalculateHaqqCoinsToMint(ctx, tc.amt)
				suite.Require().NoError(expErr)
				tc.expRes = expRes
			}

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
	var canceledAppID uint64
	var canceledAppFrom string
	var ucdaoAppID uint64
	var ucdaoAppFrom string

	totalApps := ethiqtypes.TotalNumberOfApplications()
	for i := uint64(0); i < totalApps; i++ {
		app, err := ethiqtypes.GetApplicationByID(i)
		suite.Require().NoError(err)

		if app.IsCanceled && canceledAppFrom == "" {
			canceledAppID = i
			canceledAppFrom = app.FromAddress
		}
		if app.Source == ethiqtypes.SourceOfFunds_SOURCE_OF_FUNDS_UCDAO && !app.IsCanceled && ucdaoAppFrom == "" {
			ucdaoAppID = i
			ucdaoAppFrom = app.FromAddress
		}
	}

	suite.Require().NotEmpty(canceledAppFrom, "canceled application not found in waitlist")
	suite.Require().NotEmpty(ucdaoAppFrom, "UCDAO application not found in waitlist")

	testCases := []struct {
		name          string
		malleate      func(ctx sdk.Context)
		from          sdk.AccAddress
		appID         uint64
		expMintAmt    string
		expMintToAddr sdk.AccAddress
		calcExpected  bool
		expErr        bool
		errContains   string
	}{
		{
			name: "fail - module is disabled",
			malleate: func(ctx sdk.Context) {
				p := s.network.App.EthiqKeeper.GetParams(ctx)
				p.Enabled = false
				suite.Require().NoError(s.network.App.EthiqKeeper.SetParams(ctx, p))
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
			from:        sdk.MustAccAddressFromBech32(canceledAppFrom),
			appID:       canceledAppID,
			expErr:      true,
			errContains: "is canceled",
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
			from:        sdk.MustAccAddressFromBech32("haqq13tznakuqvyr4q8kflalrkmf7uqnsg0psqhawlv"),
			appID:       8,
			expErr:      true,
			errContains: "total aHAQQ supply exceeds allowed maximum",
		},
		{
			name:        "fail - account does not exist",
			malleate:    func(_ sdk.Context) {},
			from:        sdk.MustAccAddressFromBech32("haqq13tznakuqvyr4q8kflalrkmf7uqnsg0psqhawlv"),
			appID:       8,
			expErr:      true,
			errContains: "account haqq13tznakuqvyr4q8kflalrkmf7uqnsg0psqhawlv does not exist",
		},
		{
			name: "fail - redeem liquid vesting coins",
			malleate: func(_ sdk.Context) {
				err := s.network.FundAccount(
					sdk.MustAccAddressFromBech32("haqq13tznakuqvyr4q8kflalrkmf7uqnsg0psqhawlv"),
					sdk.NewCoins(sdk.NewCoin("aLIQUID99", sdkmath.OneInt().MulRaw(1e18))),
				)
				suite.Require().NoError(err)
			},
			from:        sdk.MustAccAddressFromBech32("haqq13tznakuqvyr4q8kflalrkmf7uqnsg0psqhawlv"),
			appID:       8,
			expErr:      true,
			errContains: "failed to redeem aLIQUID coins",
		},
		{
			name: "fail - insufficient funds",
			malleate: func(_ sdk.Context) {
				// Create and fund account, but not enough coins
				err := s.network.FundAccount(
					sdk.MustAccAddressFromBech32("haqq13tznakuqvyr4q8kflalrkmf7uqnsg0psqhawlv"),
					sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.OneInt())),
				)
				suite.Require().NoError(err)
			},
			from:        sdk.MustAccAddressFromBech32("haqq13tznakuqvyr4q8kflalrkmf7uqnsg0psqhawlv"),
			appID:       8,
			expErr:      true,
			errContains: "insufficient funds",
		},
		{
			name: "success - bank as source of funds, no liquid vesting coins",
			malleate: func(_ sdk.Context) {
				application, err := ethiqtypes.GetApplicationByID(29)
				suite.Require().NoError(err)

				// Create and fund account
				err = s.network.FundAccount(
					sdk.MustAccAddressFromBech32("haqq1jt70r5w5q56fers0a4z2x95l92v360pwtey60k"),
					sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, application.BurnAmount.Amount.MulRaw(2))),
				)
				suite.Require().NoError(err)
			},
			from:         sdk.MustAccAddressFromBech32("haqq1jt70r5w5q56fers0a4z2x95l92v360pwtey60k"),
			appID:        29,
			calcExpected: true,
			expErr:       false,
			errContains:  "",
		},
		{
			name: "success - bank as source of funds, with liquid vesting coins",
			malleate: func(ctx sdk.Context) {
				// Create and fund account
				fromAddr := sdk.MustAccAddressFromBech32("haqq1jt70r5w5q56fers0a4z2x95l92v360pwtey60k")
				application, err := ethiqtypes.GetApplicationByID(29)
				suite.Require().NoError(err)
				suite.Require().NoError(s.network.FundAccount(fromAddr, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, application.BurnAmount.Amount.MulRaw(2)))))

				funder := s.keyring.GetAccAddr(2)
				vesting := s.keyring.GetAccAddr(3)

				// custom liquid vesting params
				lvParams := s.network.App.LiquidVestingKeeper.GetParams(ctx)
				lvParams.MinimumLiquidationAmount = sdkmath.OneInt()
				suite.Require().NoError(s.network.App.LiquidVestingKeeper.SetParams(ctx, lvParams))

				suite.Require().NoError(s.setupClawbackVestingAccount(ctx, vesting, funder, testutil.TestVestingSchedule.TotalVestingCoins, false))
				_, _, err = s.network.App.LiquidVestingKeeper.Liquidate(ctx, vesting, fromAddr, testutil.TestVestingSchedule.TotalVestingCoins.QuoInt(sdkmath.NewIntFromUint64(2))[0])
				suite.Require().NoError(err)
				suite.Require().NoError(s.network.NextBlockAfter(5 * time.Second))
			},
			from:         sdk.MustAccAddressFromBech32("haqq1jt70r5w5q56fers0a4z2x95l92v360pwtey60k"),
			appID:        29,
			calcExpected: true,
			expErr:       false,
			errContains:  "",
		},
		{
			name: "success - UCDAO as source of funds, no liquid vesting coins",
			malleate: func(ctx sdk.Context) {
				application, err := ethiqtypes.GetApplicationByID(ucdaoAppID)
				suite.Require().NoError(err)
				fromAddr := sdk.MustAccAddressFromBech32(application.FromAddress)
				fundAmt := application.BurnAmount.Amount.MulRaw(2)
				suite.Require().NoError(s.network.FundAccount(fromAddr, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmt))))
				suite.Require().NoError(s.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmt)), fromAddr))
			},
			from:         sdk.MustAccAddressFromBech32(ucdaoAppFrom),
			appID:        ucdaoAppID,
			calcExpected: true,
			expErr:       false,
			errContains:  "",
		},
		{
			name: "success - UCDAO application burn of entire escrow removes holder",
			malleate: func(ctx sdk.Context) {
				application, err := ethiqtypes.GetApplicationByID(ucdaoAppID)
				suite.Require().NoError(err)
				fromAddr := sdk.MustAccAddressFromBech32(application.FromAddress)
				fundAmt := application.BurnAmount.Amount
				suite.Require().NoError(s.network.FundAccount(fromAddr, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmt))))
				suite.Require().NoError(s.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmt)), fromAddr))
			},
			from:         sdk.MustAccAddressFromBech32(ucdaoAppFrom),
			appID:        ucdaoAppID,
			calcExpected: true,
			expErr:       false,
			errContains:  "",
		},
		{
			name: "success - UCDAO as source of funds, with liquid vesting coins",
			malleate: func(ctx sdk.Context) {
				// fund funder account
				application, err := ethiqtypes.GetApplicationByID(ucdaoAppID)
				suite.Require().NoError(err)
				fromAddr := sdk.MustAccAddressFromBech32(application.FromAddress)
				fundAmt := application.BurnAmount.Amount.MulRaw(2)
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
			from:         sdk.MustAccAddressFromBech32(ucdaoAppFrom),
			appID:        ucdaoAppID,
			calcExpected: true,
			expErr:       false,
			errContains:  "",
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
				if tc.calcExpected {
					app, appErr := ethiqtypes.GetApplicationByID(tc.appID)
					suite.Require().NoError(appErr)
					calcRes, calcErr := s.network.App.EthiqKeeper.CalculateForApplication(ctx, &ethiqtypes.QueryCalculateForApplicationRequest{
						ApplicationId: tc.appID,
					})
					suite.Require().NoError(calcErr)
					suite.Require().Equal(calcRes.EstimatedHaqqAmount.String(), resAmt.String())
					suite.Require().Equal(sdk.MustAccAddressFromBech32(app.ToAddress), toAddr)
					if app.Source == ethiqtypes.SourceOfFunds_SOURCE_OF_FUNDS_UCDAO {
						assertUCDAOAccountingMatchesEscrow(suite, ctx, tc.from)
					}
				} else {
					expMintAmt, ok := sdkmath.NewIntFromString(tc.expMintAmt)
					suite.Require().True(ok)
					suite.Require().Equal(expMintAmt.String(), resAmt.String())
					suite.Require().Equal(tc.expMintToAddr, toAddr)
				}
			}
		})
	}
}

// assertUCDAOAccountingMatchesEscrow checks that ucDAO TotalBalance equals the sum of
// holder escrow bank balances and that empty escrows are dropped from the holders index.
func assertUCDAOAccountingMatchesEscrow(suite *KeeperTestSuite, ctx sdk.Context, owner sdk.AccAddress) {
	suite.T().Helper()

	dao := suite.network.App.DaoKeeper
	sum := sdk.NewCoins()
	for _, holder := range dao.GetHolders(ctx) {
		sum = sum.Add(dao.GetAccountBalances(ctx, holder)...)
	}
	suite.Require().True(
		dao.GetTotalBalance(ctx).Equal(sum),
		"ucDAO TotalBalance %s must equal sum of holder escrow balances %s after application burn",
		dao.GetTotalBalance(ctx),
		sum,
	)

	remaining := dao.GetAccountBalances(ctx, owner)
	if remaining.IsZero() {
		suite.Require().False(dao.IsHolder(ctx, owner), "empty escrow must be removed from holders index")
		return
	}
	suite.Require().True(dao.IsHolder(ctx, owner), "non-empty escrow must stay in holders index")
}

// TestApplicationBurnEventIdentifiesOwner checks who the application-mint event names.
//
// For a ucDAO application the coins leave a derived escrow account. Reporting that address as
// the sender forced indexers to recompute sha256 over the module name to get back to a user,
// so the owner is the sender and the escrow is a separate attribute.
func (suite *KeeperTestSuite) TestApplicationBurnEventIdentifiesOwner() {
	var bankAppID, ucdaoAppID uint64
	var bankAppFrom, ucdaoAppFrom string

	totalApps := ethiqtypes.TotalNumberOfApplications()
	for i := uint64(0); i < totalApps; i++ {
		app, err := ethiqtypes.GetApplicationByID(i)
		suite.Require().NoError(err)
		if app.IsCanceled {
			continue
		}
		if app.Source == ethiqtypes.SourceOfFunds_SOURCE_OF_FUNDS_BANK && bankAppFrom == "" {
			bankAppID, bankAppFrom = i, app.FromAddress
		}
		if app.Source == ethiqtypes.SourceOfFunds_SOURCE_OF_FUNDS_UCDAO && ucdaoAppFrom == "" {
			ucdaoAppID, ucdaoAppFrom = i, app.FromAddress
		}
	}
	suite.Require().NotEmpty(bankAppFrom, "bank application not found in waitlist")
	suite.Require().NotEmpty(ucdaoAppFrom, "UCDAO application not found in waitlist")

	// eventAttributes runs the application burn on a clean event manager and returns the
	// attributes of the emitted mint event.
	eventAttributes := func(ctx sdk.Context, owner sdk.AccAddress, appID uint64) map[string]string {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		_, _, err := s.network.App.EthiqKeeper.BurnIslmForHaqqByApplicationID(ctx, owner, appID)
		suite.Require().NoError(err)

		attrs := make(map[string]string)
		for _, event := range ctx.EventManager().Events() {
			if event.Type != ethiqtypes.EventTypeMintByApplicationIDExecuted {
				continue
			}
			for _, attr := range event.Attributes {
				attrs[attr.Key] = attr.Value
			}
		}
		suite.Require().NotEmpty(attrs, "mint event was not emitted")
		return attrs
	}

	suite.Run("bank source names the owner and reports no escrow", func() {
		suite.SetupTest()
		ctx := s.network.GetContext()

		owner := sdk.MustAccAddressFromBech32(bankAppFrom)
		app, err := ethiqtypes.GetApplicationByID(bankAppID)
		suite.Require().NoError(err)
		suite.Require().NoError(s.network.FundAccount(owner, sdk.NewCoins(app.BurnAmount)))

		attrs := eventAttributes(ctx, owner, bankAppID)
		suite.Require().Equal(bankAppFrom, attrs[ethiqtypes.AttributeKeySender])
		suite.Require().Empty(attrs[ethiqtypes.AttributeKeyApplicationEscrow])
	})

	suite.Run("ucDAO source names the owner and reports the escrow", func() {
		suite.SetupTest()
		ctx := s.network.GetContext()

		owner := sdk.MustAccAddressFromBech32(ucdaoAppFrom)
		app, err := ethiqtypes.GetApplicationByID(ucdaoAppID)
		suite.Require().NoError(err)
		suite.Require().NoError(s.network.FundAccount(owner, sdk.NewCoins(app.BurnAmount)))
		suite.Require().NoError(s.network.App.DaoKeeper.Fund(ctx, sdk.NewCoins(app.BurnAmount), owner))

		attrs := eventAttributes(ctx, owner, ucdaoAppID)
		suite.Require().Equal(ucdaoAppFrom, attrs[ethiqtypes.AttributeKeySender],
			"sender must be the ucDAO holder, not the derived escrow")
		suite.Require().Equal(
			ucdaotypes.GetEscrowAddress(owner).String(),
			attrs[ethiqtypes.AttributeKeyApplicationEscrow],
			"the escrow the coins left must be reported separately",
		)
	})
}

// TestApplicationBurnFromUntrackedEscrow covers the escrow that ucDAO never counted.
//
// The applicant topped their escrow up with a plain bank transfer instead of MsgFund, so the
// ucDAO total balance is behind the escrow's real balance. The application burn removes more
// than was ever counted, which used to panic inside sdk.Coin.Sub and made the application
// permanently unexecutable. It must go through and leave the counter at zero.
func (suite *KeeperTestSuite) TestApplicationBurnFromUntrackedEscrow() {
	suite.SetupTest()
	ctx := s.network.GetContext()

	var appID uint64
	var from string
	total := ethiqtypes.TotalNumberOfApplications()
	for i := uint64(0); i < total; i++ {
		app, err := ethiqtypes.GetApplicationByID(i)
		suite.Require().NoError(err)
		if app.Source == ethiqtypes.SourceOfFunds_SOURCE_OF_FUNDS_UCDAO && !app.IsCanceled {
			appID, from = i, app.FromAddress
			break
		}
	}
	suite.Require().NotEmpty(from, "UCDAO application not found in waitlist")

	app, err := ethiqtypes.GetApplicationByID(appID)
	suite.Require().NoError(err)
	owner := sdk.MustAccAddressFromBech32(from)
	escrow := ucdaotypes.GetEscrowAddress(owner)
	dao := s.network.App.DaoKeeper

	// Fund the escrow directly through bank, bypassing DaoKeeper.Fund entirely.
	suite.Require().NoError(s.network.FundAccount(escrow, sdk.NewCoins(app.BurnAmount)))
	suite.Require().True(dao.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount.IsZero(),
		"a direct transfer must not show up in the ucDAO counter")

	var mintedAmt sdkmath.Int
	suite.Require().NotPanics(func() {
		mintedAmt, _, err = s.network.App.EthiqKeeper.BurnIslmForHaqqByApplicationID(ctx, owner, appID)
	})
	suite.Require().NoError(err)
	suite.Require().True(mintedAmt.IsPositive())

	suite.Require().True(dao.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount.IsZero(),
		"the counter must bottom out at zero, not go negative")
	suite.Require().True(dao.GetAccountBalances(ctx, owner).IsZero(), "the escrow must be drained")
	suite.Require().True(s.network.App.EthiqKeeper.IsApplicationExecuted(ctx, appID))
}
