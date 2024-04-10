package v174_test

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	v174 "github.com/haqq-network/haqq/app/upgrades/v1.7.4"
	"github.com/haqq-network/haqq/tests"
	"github.com/haqq-network/haqq/testutil"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

var (
	amount1ISLM            = sdk.NewCoins(sdk.NewCoin("aISLM", math.NewIntWithDecimal(1, 18)))
	amount1ISLMQuarter     = sdk.NewCoins(sdk.NewCoin("aISLM", math.NewIntWithDecimal(1, 18).Quo(math.NewInt(4))))
	amount30kISLM          = sdk.NewCoins(sdk.NewCoin("aISLM", math.NewIntWithDecimal(30_000, 18)))
	amount30kISLMQuarter   = sdk.NewCoins(sdk.NewCoin("aISLM", math.NewIntWithDecimal(30_000, 18).Quo(math.NewInt(4))))
	amount10kkkISLM        = sdk.NewCoins(sdk.NewCoin("aISLM", math.NewIntWithDecimal(12_000_000_000, 18)))
	amount10kkkISLMQuarter = sdk.NewCoins(sdk.NewCoin("aISLM", math.NewIntWithDecimal(12_000_000_000, 18).Quo(math.NewInt(4))))

	addr = sdk.AccAddress(tests.GenerateAddress().Bytes())
)

func (suite *UpgradeTestSuite) TestStretchLockupScheduleForAccounts() {
	testCases := []struct {
		name                  string
		malleate              func()
		addr                  sdk.AccAddress
		expectedLockupPeriods sdkvesting.Periods
		stretchDays           int64
		threshold             time.Time
		expectedEndTime       time.Time
	}{
		{
			name: "stretch account with 1 ISLM by 3 days",
			malleate: func() {
				// create and fund vesting account
				baseAccount := authtypes.NewBaseAccountWithAddress(addr)
				startTime := suite.ctx.BlockTime().Add(-86400 * time.Second)

				clawbackAccount := vestingtypes.NewClawbackVestingAccount(
					baseAccount,
					sdk.AccAddress(liquidvestingtypes.ModuleName),
					amount1ISLM,
					startTime,
					sdkvesting.Periods{
						{Length: 86400, Amount: amount1ISLMQuarter},
						{Length: 86400, Amount: amount1ISLMQuarter},
						{Length: 86400, Amount: amount1ISLMQuarter},
						{Length: 86400, Amount: amount1ISLMQuarter},
					},
					sdkvesting.Periods{{Length: 0, Amount: amount1ISLM}},
					nil,
				)
				suite.app.AccountKeeper.SetAccount(suite.ctx, clawbackAccount)
				testutil.FundAccount(suite.ctx, suite.app.BankKeeper, addr, amount1ISLM) //nolint:errcheck
			},
			addr: addr,
			expectedLockupPeriods: sdkvesting.Periods{
				{Length: 86400, Amount: amount1ISLMQuarter},
				{Length: 86400, Amount: amount1ISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount1ISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount1ISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount1ISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount1ISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount1ISLMQuarter.QuoInt(math.NewInt(2))},
			},
			stretchDays:     3,
			threshold:       suite.ctx.BlockTime().Add(86_400 * 2 * time.Second),
			expectedEndTime: suite.ctx.BlockTime().Add(86_400 * 6 * time.Second),
		},
		{
			name: "stretch account with 30k ISLM by 3 days",
			malleate: func() {
				// create and fund vesting account
				baseAccount := authtypes.NewBaseAccountWithAddress(addr)
				startTime := suite.ctx.BlockTime().Add(-86400 * time.Second)

				clawbackAccount := vestingtypes.NewClawbackVestingAccount(
					baseAccount,
					sdk.AccAddress(liquidvestingtypes.ModuleName),
					amount30kISLM,
					startTime,
					sdkvesting.Periods{
						{Length: 86400, Amount: amount30kISLMQuarter},
						{Length: 86400, Amount: amount30kISLMQuarter},
						{Length: 86400, Amount: amount30kISLMQuarter},
						{Length: 86400, Amount: amount30kISLMQuarter},
					},
					sdkvesting.Periods{{Length: 0, Amount: amount30kISLM}},
					nil,
				)
				suite.app.AccountKeeper.SetAccount(suite.ctx, clawbackAccount)
				testutil.FundAccount(suite.ctx, suite.app.BankKeeper, addr, amount30kISLM) //nolint:errcheck
			},
			addr: addr,
			expectedLockupPeriods: sdkvesting.Periods{
				{Length: 86400, Amount: amount30kISLMQuarter},
				{Length: 86400, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
			},
			stretchDays:     3,
			threshold:       suite.ctx.BlockTime().Add(86_400 * 2 * time.Second),
			expectedEndTime: suite.ctx.BlockTime().Add(10 + 86_400*6*time.Second),
		},
		{
			name: "stretch account with 10kkk ISLM by 3 days",
			malleate: func() {
				// create and fund vesting account
				baseAccount := authtypes.NewBaseAccountWithAddress(addr)
				startTime := suite.ctx.BlockTime().Add(-86400 * time.Second)

				clawbackAccount := vestingtypes.NewClawbackVestingAccount(
					baseAccount,
					sdk.AccAddress(liquidvestingtypes.ModuleName),
					amount10kkkISLM,
					startTime,
					sdkvesting.Periods{
						{Length: 86400, Amount: amount10kkkISLMQuarter},
						{Length: 86400, Amount: amount10kkkISLMQuarter},
						{Length: 86400, Amount: amount10kkkISLMQuarter},
						{Length: 86400, Amount: amount10kkkISLMQuarter},
					},
					sdkvesting.Periods{{Length: 0, Amount: amount10kkkISLM}},
					nil,
				)
				suite.app.AccountKeeper.SetAccount(suite.ctx, clawbackAccount)
				testutil.FundAccount(suite.ctx, suite.app.BankKeeper, addr, amount10kkkISLM) //nolint:errcheck
			},
			addr: addr,
			expectedLockupPeriods: sdkvesting.Periods{
				{Length: 86400, Amount: amount10kkkISLMQuarter},
				{Length: 86400, Amount: amount10kkkISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount10kkkISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount10kkkISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount10kkkISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount10kkkISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount10kkkISLMQuarter.QuoInt(math.NewInt(2))},
			},
			stretchDays:     3,
			threshold:       suite.ctx.BlockTime().Add(86_400 * 2 * time.Second),
			expectedEndTime: suite.ctx.BlockTime().Add(10 + 86_400*6*time.Second),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // Reset

			tc.malleate()

			err := v174.StretchLockupScheduleForAccounts(suite.ctx, suite.app.AccountKeeper, tc.stretchDays, tc.threshold)

			// check returns
			suite.Require().NoError(err)

			// check target account exists
			acc := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.addr)
			suite.Require().NotNil(acc)

			// check acc schedule match expected one
			cva, isClawback := acc.(*vestingtypes.ClawbackVestingAccount)
			suite.Require().True(isClawback)
			suite.Require().Equal(tc.expectedLockupPeriods.String(), cva.LockupPeriods.String())
			suite.Require().Equal(tc.expectedEndTime.Unix(), cva.EndTime)
		})
	}
}

func (suite *UpgradeTestSuite) TestStretchLockupScheduleForLiquidVestingTokens() {
	testCases := []struct {
		name                  string
		malleate              func()
		addr                  sdk.AccAddress
		expectedLockupPeriods sdkvesting.Periods
		stretchDays           int64
		threshold             time.Time
	}{
		{
			name: "stretch account with 30k ISLM by 3 days",
			malleate: func() {
				// create and fund vesting account
				baseAccount := authtypes.NewBaseAccountWithAddress(addr)
				startTime := suite.ctx.BlockTime().Add(-86410 * time.Second)

				clawbackAccount := vestingtypes.NewClawbackVestingAccount(
					baseAccount,
					sdk.AccAddress(liquidvestingtypes.ModuleName),
					amount30kISLM,
					startTime,
					sdkvesting.Periods{
						{Length: 86400, Amount: amount30kISLMQuarter},
						{Length: 86500, Amount: amount30kISLMQuarter},
						{Length: 86400, Amount: amount30kISLMQuarter},
						{Length: 86400, Amount: amount30kISLMQuarter},
					},
					sdkvesting.Periods{{Length: 0, Amount: amount30kISLM}},
					nil,
				)
				suite.app.AccountKeeper.SetAccount(suite.ctx, clawbackAccount)
				testutil.FundAccount(suite.ctx, suite.app.BankKeeper, addr, amount30kISLM) //nolint:errcheck
				msg := liquidvestingtypes.NewMsgLiquidate(
					addr,
					addr,
					sdk.NewCoin("aISLM", amount30kISLM.AmountOf("aISLM").Quo(math.NewInt(4)).Mul(math.NewInt(3))),
				)
				_, err := suite.app.LiquidVestingKeeper.Liquidate(suite.ctx, msg)
				suite.Require().NoError(err)
			},
			addr: addr,
			expectedLockupPeriods: sdkvesting.Periods{
				{Length: 86490, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
				{Length: 86400, Amount: amount30kISLMQuarter.QuoInt(math.NewInt(2))},
			},
			stretchDays: 3,
			threshold:   suite.ctx.BlockTime().Add(86_400 * 2 * time.Second),
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // Reset

			tc.malleate()

			err := v174.StretchLockupScheduleForLiquidVestingTokens(suite.ctx, suite.app.LiquidVestingKeeper, tc.stretchDays, tc.threshold)

			// check returns
			suite.Require().NoError(err)

			// check target account exists
			denom, ok := suite.app.LiquidVestingKeeper.GetDenom(suite.ctx, "aLIQUID0")
			suite.Require().True(ok)

			// check acc schedule match expected one
			suite.Require().Equal(tc.expectedLockupPeriods.String(), denom.LockupPeriods.String())
		})
	}
}
