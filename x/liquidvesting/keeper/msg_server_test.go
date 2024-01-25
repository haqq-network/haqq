package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/haqq-network/haqq/tests"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

var (
	amount = sdk.NewCoins(sdk.NewInt64Coin("test", 300))
	third  = sdk.NewCoins(sdk.NewInt64Coin("test", 100))

	lockupPeriods = sdkvesting.Periods{
		{Length: 100, Amount: third},
		{Length: 100, Amount: third},
		{Length: 100, Amount: third},
	}
	vestingPeriods = sdkvesting.Periods{
		{Length: 0, Amount: amount},
	}
	addr1 = sdk.AccAddress(tests.GenerateAddress().Bytes())
	addr2 = sdk.AccAddress(tests.GenerateAddress().Bytes())
)

func (suite *KeeperTestSuite) TestLiquidate() {
	testCases := []struct {
		name       string
		malleate   func()
		from       sdk.AccAddress
		to         sdk.AccAddress
		amount     sdk.Coin
		expectPass bool
	}{
		{
			name: "ok - standard liquidation one third",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, suite.ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin("test", third.AmountOf("test")),
			expectPass: true,
		},
		{
			name: "ok - standard liquidation full liquidation",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, suite.ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin("test", math.NewInt(300)),
			expectPass: true,
		},
		{
			name: "fail - amount exceeded",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, suite.ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin("test", math.NewInt(400)),
			expectPass: false,
		},
		{
			name: "fail - denom is not locked in staking",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, suite.ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin("nonpresent", math.NewInt(200)),
			expectPass: false,
		},
		{
			name: "fail - vesting periods have length",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				vestingPeriods := sdkvesting.Periods{{Length: 100, Amount: amount}}
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, suite.ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin("test", math.NewInt(200)),
			expectPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // Reset
			ctx := sdk.WrapSDKContext(suite.ctx)

			tc.malleate()

			msg := types.NewMsgLiquidate(tc.from, tc.to, tc.amount)
			resp, err := suite.app.LiquidVestingKeeper.Liquidate(ctx, msg)
			expRes := &types.MsgLiquidateResponse{}

			if tc.expectPass {
				// check returns
				suite.Require().NoError(err)
				suite.Require().Equal(expRes, resp)

				// check target account exists and has liquid token
				accIto := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.to)
				suite.Require().NotNil(accIto)
				balanceTarget := suite.app.BankKeeper.GetBalance(suite.ctx, tc.to, types.DenomNameFromID(0))
				suite.Require().Equal(sdk.NewCoin(types.DenomNameFromID(0), tc.amount.Amount), balanceTarget)

				// check liquidated vesting locked coins are decreased on initial account
				accIFrom := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.from)
				suite.Require().NotNil(accIto)
				cva, isClawback := accIFrom.(*vestingtypes.ClawbackVestingAccount)
				suite.Require().True(isClawback)
				suite.Require().Equal(cva.GetLockedOnly(suite.ctx.BlockTime()), lockupPeriods.TotalAmount().Sub(tc.amount))
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
