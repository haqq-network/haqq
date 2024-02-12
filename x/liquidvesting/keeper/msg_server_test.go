package keeper_test

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/tests"
	"github.com/haqq-network/haqq/testutil"
	haqqtypes "github.com/haqq-network/haqq/types"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

var (
	amount = sdk.NewCoins(sdk.NewInt64Coin("aISLM", 3_000_000))
	third  = sdk.NewCoins(sdk.NewInt64Coin("aISLM", 1_000_000))

	liquidDenomAmount = sdk.NewCoins(sdk.NewInt64Coin("aLIQUID0", 3_000_000))

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
				startTime := suite.ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin("aISLM", third.AmountOf("aISLM")),
			expectPass: true,
		},
		{
			name: "ok - standard liquidation full liquidation",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				startTime := suite.ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin("aISLM", amount.AmountOf("aISLM")),
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
			amount:     sdk.NewInt64Coin("aISLM", 4_000_000),
			expectPass: false,
		},
		{
			name: "fail - denom is not aISLM",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, suite.ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin("nonaISLM", math.NewInt(2_000_000)),
			expectPass: false,
		},
		{
			name: "fail - vesting periods have length",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				vestingPeriods := sdkvesting.Periods{{Length: 100, Amount: amount}}
				startTime := suite.ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewInt64Coin("aISLM", 2_000_000),
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
				balanceTarget := suite.app.BankKeeper.GetBalance(suite.ctx, tc.to, types.DenomBaseNameFromID(0))
				suite.Require().Equal(sdk.NewCoin(types.DenomBaseNameFromID(0), math.ZeroInt()).String(), balanceTarget.String())

				// check liquidated vesting locked coins are decreased on initial account
				accIFrom := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.from)
				suite.Require().NotNil(accIFrom)
				cva, isClawback := accIFrom.(*vestingtypes.ClawbackVestingAccount)
				suite.Require().True(isClawback)
				suite.Require().Equal(cva.GetLockedOnly(suite.ctx.BlockTime()), lockupPeriods.TotalAmount().Sub(tc.amount))

				// check erc20 token contract
				pairResp, err := s.app.Erc20Keeper.TokenPair(s.ctx, &erc20types.QueryTokenPairRequest{Token: types.DenomBaseNameFromID(0)})
				s.Require().NoError(err)
				s.Require().True(pairResp.TokenPair.Enabled)
				ethAccTo, isEthAccount := accIto.(*haqqtypes.EthAccount)
				s.Require().True(isEthAccount)
				balanceOfLiquidTokeErc20Pair := s.app.Erc20Keeper.BalanceOf(
					s.ctx,
					contracts.ERC20MinterBurnerDecimalsContract.ABI,
					pairResp.TokenPair.GetERC20Contract(),
					common.BytesToAddress(ethAccTo.GetAddress().Bytes()),
				)
				s.Require().Equal(tc.amount.Amount.String(), balanceOfLiquidTokeErc20Pair.String())
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestRedeem() {
	testCases := []struct {
		name                 string
		malleate             func()
		redeemFrom           sdk.AccAddress
		redeemTo             sdk.AccAddress
		redeemAmount         int64
		expectedLockedAmount int64
		expectPass           bool
	}{
		{
			name: "ok - standard redeem, fully unlocked schedule",
			malleate: func() {
				// fund liquid vesting module
				testutil.FundModuleAccount(s.ctx, s.app.BankKeeper, types.ModuleName, amount) //nolint:errcheck
				// create liquid vesting denom
				s.app.LiquidVestingKeeper.SetDenom(s.ctx, types.Denom{
					BaseDenom:     "aLIQUID0",
					DisplayDenom:  "LIQUID0",
					OriginalDenom: "aISLM",
					LockupPeriods: lockupPeriods,
				})
				// create accounts
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr1))
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr2))
				// fund account with liquid denom token
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, liquidDenomAmount) //nolint:errcheck
				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, liquidTokenMetadata)
				suite.app.Erc20Keeper.RegisterCoin(suite.ctx, liquidTokenMetadata) //nolint:errcheck
			},
			redeemFrom:   addr1,
			redeemTo:     addr2,
			redeemAmount: 3_000_000,
			expectPass:   true,
		},
		{
			name: "ok - standard redeem, partially locked",
			malleate: func() {
				// fund liquid vesting module
				testutil.FundModuleAccount(s.ctx, s.app.BankKeeper, types.ModuleName, amount) //nolint:errcheck
				// create liquid vesting denom
				// subs 150 second, it is the half of the second period now
				startTime := s.ctx.BlockTime().Add(-150 * time.Second)
				s.app.LiquidVestingKeeper.SetDenom(s.ctx, types.Denom{
					BaseDenom:     "aLIQUID0",
					DisplayDenom:  "LIQUID0",
					OriginalDenom: "aISLM",
					StartTime:     startTime,
					EndTime:       startTime.Add(lockupPeriods.TotalDuration()),
					LockupPeriods: lockupPeriods,
				})
				// create accounts
				acc1 := &haqqtypes.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(addr1),
					CodeHash:    common.BytesToHash(crypto.Keccak256(nil)).String(),
				}
				s.app.AccountKeeper.SetAccount(s.ctx, acc1)
				acc2 := &haqqtypes.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(addr2),
					CodeHash:    common.BytesToHash(crypto.Keccak256(nil)).String(),
				}
				s.app.AccountKeeper.SetAccount(s.ctx, acc2)
				// fund account with liquid denom token
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, liquidDenomAmount) //nolint:errcheck

				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, liquidTokenMetadata)
				suite.app.Erc20Keeper.RegisterCoin(suite.ctx, liquidTokenMetadata) //nolint:errcheck
			},
			redeemFrom:           addr1,
			redeemTo:             addr2,
			redeemAmount:         600_000,
			expectedLockedAmount: 400_000,
			expectPass:           true,
		},
		{
			name: "fail - insufficient liquid token balance",
			malleate: func() {
				// fund liquid vesting module
				testutil.FundModuleAccount(s.ctx, s.app.BankKeeper, types.ModuleName, amount) //nolint:errcheck
				// create liquid vesting denom
				s.app.LiquidVestingKeeper.SetDenom(s.ctx, types.Denom{
					BaseDenom:     "aLIQUID0",
					DisplayDenom:  "LIQUID0",
					OriginalDenom: "aISLM",
					LockupPeriods: lockupPeriods,
				})
				// create accounts
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr1))
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr2))
				// fund account with liquid denom token
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, liquidDenomAmount) //nolint:errcheck
				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, liquidTokenMetadata)
				suite.app.Erc20Keeper.RegisterCoin(suite.ctx, liquidTokenMetadata) //nolint:errcheck
			},
			redeemFrom:   addr1,
			redeemTo:     addr2,
			redeemAmount: 4_000_000,
			expectPass:   false,
		},
		{
			name: "fail - liquid denom does not exist",
			malleate: func() {
				// fund liquid vesting module
				testutil.FundModuleAccount(s.ctx, s.app.BankKeeper, types.ModuleName, amount) //nolint:errcheck
				// create liquid vesting denom
				s.app.LiquidVestingKeeper.SetDenom(s.ctx, types.Denom{
					BaseDenom:     "solid",
					DisplayDenom:  "solid18",
					OriginalDenom: "aISLM",
					LockupPeriods: lockupPeriods,
				})
				// create accounts
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr1))
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr2))
				// fund account with liquid denom token
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, liquidDenomAmount) //nolint:errcheck
				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, liquidTokenMetadata)
				suite.app.Erc20Keeper.RegisterCoin(suite.ctx, liquidTokenMetadata) //nolint:errcheck
			},
			redeemFrom:   addr1,
			redeemTo:     addr2,
			redeemAmount: 4_000_000,
			expectPass:   false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // Reset
			ctx := sdk.WrapSDKContext(suite.ctx)

			tc.malleate()
			redeemCoin := sdk.NewInt64Coin("aLIQUID0", tc.redeemAmount)
			msg := types.NewMsgRedeem(tc.redeemFrom, tc.redeemTo, redeemCoin)
			resp, err := suite.app.LiquidVestingKeeper.Redeem(ctx, msg)
			expRes := &types.MsgRedeemResponse{}
			if tc.expectPass {
				// check returns
				suite.Require().NoError(err)
				suite.Require().Equal(expRes, resp)

				// check target account has original tokens
				accIto := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.redeemTo)
				suite.Require().NotNil(accIto)
				balanceTarget := suite.app.BankKeeper.SpendableCoin(suite.ctx, tc.redeemTo, "aISLM")
				suite.Require().Equal(sdk.NewInt64Coin("aISLM", tc.redeemAmount-tc.expectedLockedAmount).String(), balanceTarget.String())
				if tc.expectedLockedAmount > 0 {
					cva, isClawback := accIto.(*vestingtypes.ClawbackVestingAccount)
					suite.Require().True(isClawback)
					expectedLockedCoins := sdk.NewCoins(sdk.NewInt64Coin("aISLM", tc.expectedLockedAmount))
					actualLockedCoins := cva.GetLockedOnly(s.ctx.BlockTime())
					s.Require().Equal(expectedLockedCoins.String(), actualLockedCoins.String())
				}

				// check liquid tokens are burnt
				_, liquidDenomCoin := liquidDenomAmount.Find("aLIQUID0")
				expectedLiquidTokenSupply := liquidDenomCoin.Sub(redeemCoin)
				actualLiquidTokenSupply := s.app.BankKeeper.GetSupply(s.ctx, "aLIQUID0")
				s.Require().Equal(expectedLiquidTokenSupply.String(), actualLiquidTokenSupply.String())
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
