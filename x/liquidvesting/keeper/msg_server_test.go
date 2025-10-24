package keeper_test

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/testutil"
	haqqtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/utils"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

var (
	amount = sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 3_000_000))
	third  = sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1_000_000))

	liquidDenomAmount = sdk.NewCoins(sdk.NewInt64Coin("aLIQUID0", 3_000_000))

	lockupPeriods = sdkvesting.Periods{
		{Length: 100000, Amount: third},
		{Length: 100000, Amount: third},
		{Length: 100000, Amount: third},
	}
	vestingPeriods = sdkvesting.Periods{
		{Length: 0, Amount: amount},
	}
	funder   = sdk.AccAddress(types.ModuleName)
	fromAddr sdk.AccAddress
	toAddr   sdk.AccAddress
)

func (suite *KeeperTestSuite) TestLiquidate() {
	var ctx sdk.Context

	testCases := []struct {
		name       string
		malleate   func()
		amount     sdk.Coin
		expectPass bool
	}{
		{
			name: "ok - standard liquidation one third",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
				baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				startTime := ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			amount:     sdk.NewCoin(utils.BaseDenom, third.AmountOf(utils.BaseDenom)),
			expectPass: true,
		},
		{
			name: "ok - standard liquidation two thirds",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
				baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				startTime := ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			amount:     sdk.NewCoin(utils.BaseDenom, third.AmountOf(utils.BaseDenom)).Add(sdk.NewCoin(utils.BaseDenom, third.AmountOf(utils.BaseDenom))),
			expectPass: true,
		},
		{
			name: "ok - block time matches end of current period",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
				baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				startTime := ctx.BlockTime().Add(-100 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			amount:     sdk.NewCoin(utils.BaseDenom, third.AmountOf(utils.BaseDenom)),
			expectPass: true,
		},
		{
			name: "ok - standard liquidation full liquidation",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
				baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				startTime := ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			amount:     sdk.NewCoin(utils.BaseDenom, amount.AmountOf(utils.BaseDenom)),
			expectPass: true,
		},
		{
			name: "ok - liquidate partially staked tokens",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
				baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				startTime := ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
				_, err = suite.network.App.StakingKeeper.Delegate(ctx, fromAddr, third.AmountOf(utils.BaseDenom), stakingtypes.Unbonded, suite.network.GetValidators()[0], true)
				suite.Require().NoError(err)
			},
			amount:     sdk.NewCoin(utils.BaseDenom, third.AmountOf(utils.BaseDenom).Add(third.AmountOf(utils.BaseDenom))),
			expectPass: true,
		},
		{
			name: "fail - liquidate amount bigger than locked but less than total",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
				baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				startTime := ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				err = testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			amount:     sdk.NewCoin(utils.BaseDenom, amount.AmountOf(utils.BaseDenom).Add(math.NewInt(1_500_000))),
			expectPass: false,
		},
		{
			name: "fail - liquidate staked tokens",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
				baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				startTime := ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
				_, err = suite.network.App.StakingKeeper.Delegate(ctx, fromAddr, amount.AmountOf(utils.BaseDenom), stakingtypes.Unbonded, suite.network.GetValidators()[0], true)
				suite.Require().NoError(err)
			},
			amount:     sdk.NewCoin(utils.BaseDenom, amount.AmountOf(utils.BaseDenom)),
			expectPass: false,
		},
		{
			name: "fail - liquidate tokens partially unlocked",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
				baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				startTime := ctx.BlockTime().Add(-200001 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			amount:     sdk.NewCoin(utils.BaseDenom, math.NewInt(1_500_000)),
			expectPass: false,
		},
		{
			name: "fail - amount exceeded",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
				baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			amount:     sdk.NewInt64Coin(utils.BaseDenom, 4_000_000),
			expectPass: false,
		},
		{
			name: "fail - denom is not aISLM",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
				baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			amount:     sdk.NewCoin("nonaISLM", math.NewInt(2_000_000)),
			expectPass: false,
		},
		{
			name: "fail - vesting periods have length",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
				baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				vestingPeriods := sdkvesting.Periods{{Length: 100, Amount: amount}}
				startTime := ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
				suite.Require().NoError(err)
				suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			amount:     sdk.NewInt64Coin(utils.BaseDenom, 2_000_000),
			expectPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()

			toAddr = suite.keyring.GetAccAddr(1)
			// Add empty account
			fromAccNumber := suite.keyring.AddKey()
			fromAddr = suite.keyring.GetAccAddr(fromAccNumber)

			tc.malleate()

			fromAcc := suite.network.App.AccountKeeper.GetAccount(ctx, fromAddr)
			fromVa, ok := fromAcc.(*vestingtypes.ClawbackVestingAccount)
			if !ok {
				suite.T().Fatal("account is not clawback vesting account")
			}
			suite.T().Logf("locked only coins: %s", fromVa.GetLockedUpCoins(ctx.BlockTime()).String())
			suite.T().Logf("UN-locked only coins: %s", fromVa.GetUnlockedCoins(ctx.BlockTime()).String())
			spendable := suite.network.App.BankKeeper.SpendableCoin(ctx, fromAddr, utils.BaseDenom)
			suite.T().Logf("spendable coins: %s", spendable.String())
			suite.T().Logf("liquidation amount: %s", tc.amount.String())

			msg := types.NewMsgLiquidate(fromAddr, toAddr, tc.amount)
			resp, err := suite.network.App.LiquidVestingKeeper.Liquidate(ctx, msg)
			expResponse := &types.MsgLiquidateResponse{
				Minted: sdk.NewCoin(types.DenomBaseNameFromID(0), tc.amount.Amount),
			}

			if tc.expectPass {
				// check returns
				suite.Require().NoError(err)
				suite.Require().Equal(expResponse.Minted, resp.Minted)
				suite.Require().NotEmpty(resp.ContractAddr)

				// check target account exists and has liquid token
				toAcc := suite.network.App.AccountKeeper.GetAccount(ctx, toAddr)
				suite.Require().NotNil(toAcc)
				balanceTarget := suite.network.App.BankKeeper.GetBalance(ctx, toAddr, types.DenomBaseNameFromID(0))
				suite.Require().Equal(expResponse.Minted.String(), balanceTarget.String())

				// check liquidated vesting locked coins are decreased on initial account
				fromAccAfter := suite.network.App.AccountKeeper.GetAccount(ctx, fromAddr)
				suite.Require().NotNil(fromAccAfter)
				fromVaAfter, isClawback := fromAccAfter.(*vestingtypes.ClawbackVestingAccount)
				suite.Require().True(isClawback)
				suite.Require().Equal(fromVaAfter.GetLockedUpCoins(ctx.BlockTime()).Add(fromVaAfter.GetUnlockedCoins(ctx.BlockTime())...), lockupPeriods.TotalAmount().Sub(tc.amount))

				// check newly created liquid denom
				liquidDenom, found := suite.network.App.LiquidVestingKeeper.GetDenom(ctx, types.DenomBaseNameFromID(0))
				suite.Require().True(found)
				suite.Require().Equal(fromVaAfter.EndTime, liquidDenom.EndTime.Unix())

				// check erc20 token contract
				pairResp, err := suite.network.App.Erc20Keeper.TokenPair(ctx, &erc20types.QueryTokenPairRequest{Token: types.DenomBaseNameFromID(0)})
				suite.Require().NoError(err)
				suite.Require().True(pairResp.TokenPair.Enabled)
				_, isEthAccount := toAcc.(*haqqtypes.EthAccount)
				suite.Require().True(isEthAccount)
				balanceOfLiquidTokeErc20Pair := suite.network.App.Erc20Keeper.BalanceOf(
					ctx,
					contracts.ERC20MinterBurnerDecimalsContract.ABI,
					pairResp.TokenPair.GetERC20Contract(),
					common.BytesToAddress(toAddr.Bytes()),
				)
				suite.Require().Equal(tc.amount.Amount.String(), balanceOfLiquidTokeErc20Pair.String())
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMultipleLiquidationsFromOneAccount() {
	var ctx sdk.Context

	suite.SetupTest()
	ctx = suite.network.GetContext()
	// Add empty accounts
	fromAccNumber := suite.keyring.AddKey()
	fromAddr = suite.keyring.GetAccAddr(fromAccNumber)
	toAccNumber := suite.keyring.AddKey()
	toAddr = suite.keyring.GetAccAddr(toAccNumber)

	baseAccount := authtypes.NewBaseAccountWithAddress(fromAddr)
	baseAccount.AccountNumber = suite.network.App.AccountKeeper.NextAccountNumber(ctx)
	startTime := ctx.BlockTime().Add(-10 * time.Second)
	clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
	err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, amount)
	suite.Require().NoError(err)
	suite.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)

	// FIRST LIQUIDATION
	msg := types.NewMsgLiquidate(fromAddr, toAddr, third[0])
	resp, err := suite.network.App.LiquidVestingKeeper.Liquidate(ctx, msg)
	expResponse := &types.MsgLiquidateResponse{
		Minted: sdk.NewCoin("aLIQUID0", third[0].Amount),
	}

	// check returns
	suite.Require().NoError(err)
	suite.Require().Equal(expResponse.Minted, resp.Minted)
	suite.Require().NotEmpty(resp.ContractAddr)

	// check target account exists and has liquid token
	toAcc := suite.network.App.AccountKeeper.GetAccount(ctx, toAddr)
	suite.Require().NotNil(toAcc)
	balanceTarget := suite.network.App.BankKeeper.GetBalance(ctx, toAddr, types.DenomBaseNameFromID(0))
	suite.Require().Equal(sdk.NewCoin(types.DenomBaseNameFromID(0), third[0].Amount).String(), balanceTarget.String())

	// check liquidated vesting locked coins are decreased on initial account
	fromAccAfter := suite.network.App.AccountKeeper.GetAccount(ctx, fromAddr)
	suite.Require().NotNil(fromAccAfter)
	fromVaAfter, isClawback := fromAccAfter.(*vestingtypes.ClawbackVestingAccount)
	suite.Require().True(isClawback)
	suite.Require().Equal(fromVaAfter.GetLockedUpCoins(ctx.BlockTime()), lockupPeriods.TotalAmount().Sub(third[0]))

	// check erc20 token contract
	pair0Resp, err := suite.network.App.Erc20Keeper.TokenPair(ctx, &erc20types.QueryTokenPairRequest{Token: types.DenomBaseNameFromID(0)})
	suite.Require().NoError(err)
	suite.Require().True(pair0Resp.TokenPair.Enabled)
	_, isEthAccount := toAcc.(*haqqtypes.EthAccount)
	suite.Require().True(isEthAccount)
	balanceOfLiquidTokeErc20Pair0 := suite.network.App.Erc20Keeper.BalanceOf(
		ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		pair0Resp.TokenPair.GetERC20Contract(),
		common.BytesToAddress(toAddr.Bytes()),
	)
	suite.Require().Equal(third[0].Amount.String(), balanceOfLiquidTokeErc20Pair0.String())

	// SECOND LIQUIDATION
	msg = types.NewMsgLiquidate(fromAddr, toAddr, third[0])
	resp, err = suite.network.App.LiquidVestingKeeper.Liquidate(ctx, msg)
	expResponse = &types.MsgLiquidateResponse{
		Minted: sdk.NewCoin("aLIQUID1", third[0].Amount),
	}

	// check returns
	suite.Require().NoError(err)
	suite.Require().Equal(expResponse.Minted, resp.Minted)
	suite.Require().NotEmpty(resp.ContractAddr)

	// check target account exists and has liquid token
	balanceTarget = suite.network.App.BankKeeper.GetBalance(ctx, toAddr, types.DenomBaseNameFromID(1))
	suite.Require().Equal(sdk.NewCoin(types.DenomBaseNameFromID(1), third[0].Amount).String(), balanceTarget.String())

	// check liquidated vesting locked coins are decreased on initial account
	fromAccFinal := suite.network.App.AccountKeeper.GetAccount(ctx, fromAddr)
	suite.Require().NotNil(fromAccFinal)
	fromVaFinal, isClawback := fromAccFinal.(*vestingtypes.ClawbackVestingAccount)
	suite.Require().True(isClawback)
	suite.Require().Equal(fromVaFinal.GetLockedUpCoins(ctx.BlockTime()), sdk.NewCoins(third[0]))

	// check erc20 token contract
	pair1Resp, err := suite.network.App.Erc20Keeper.TokenPair(ctx, &erc20types.QueryTokenPairRequest{Token: types.DenomBaseNameFromID(1)})
	suite.Require().NoError(err)
	suite.Require().True(pair1Resp.TokenPair.Enabled)
	balanceOfLiquidTokeErc20Pair1 := suite.network.App.Erc20Keeper.BalanceOf(
		ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		pair1Resp.TokenPair.GetERC20Contract(),
		common.BytesToAddress(toAddr.Bytes()),
	)
	suite.Require().Equal(third[0].Amount.String(), balanceOfLiquidTokeErc20Pair1.String())
}

func (suite *KeeperTestSuite) TestRedeem() {
	var ctx sdk.Context

	testCases := []struct {
		name                 string
		malleate             func()
		redeemAmount         int64
		expectedLockedAmount int64
		expectPass           bool
	}{
		{
			name: "ok - standard redeem, fully unlocked schedule",
			malleate: func() {
				// fund liquid vesting module
				err := testutil.FundModuleAccount(ctx, suite.network.App.BankKeeper, types.ModuleName, amount)
				suite.Require().NoError(err)
				// create liquid vesting denom
				suite.network.App.LiquidVestingKeeper.SetDenom(ctx, types.Denom{
					BaseDenom:     "aLIQUID0",
					DisplayDenom:  "LIQUID0",
					OriginalDenom: utils.BaseDenom,
					LockupPeriods: lockupPeriods,
				})
				// create accounts
				suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, fromAddr))
				suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, toAddr))
				// fund account with liquid denom token
				err = testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, liquidDenomAmount)
				suite.Require().NoError(err)
				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.network.App.BankKeeper.SetDenomMetaData(ctx, liquidTokenMetadata)

				// bind newly created denom to erc20 token
				// Create dummy IBC denom, just to bind ERC20 Precompile with newly created aLiquid denom
				fakeIBCDenom := utils.ComputeIBCDenom(types.ModuleName, liquidTokenMetadata.Base, utils.BaseDenom)
				tokenPair, err := erc20types.NewTokenPairSTRv2(fakeIBCDenom)
				suite.Require().NoError(err)
				// Set real denom to token pair, so precompile could handle transfers properly
				tokenPair.Denom = liquidTokenMetadata.Base
				// k.erc20Keeper.SetToken(ctx, tokenPair) unwrap it below due to pointer receiver in original method.
				suite.network.App.Erc20Keeper.SetTokenPair(ctx, tokenPair)
				suite.network.App.Erc20Keeper.SetDenomMap(ctx, tokenPair.Denom, tokenPair.GetID())
				suite.network.App.Erc20Keeper.SetERC20Map(ctx, tokenPair.GetERC20Contract(), tokenPair.GetID())

				err = suite.network.App.Erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract())
				suite.Require().NoError(err)
			},
			redeemAmount: 3_000_000,
			expectPass:   true,
		},
		{
			name: "ok - standard redeem, partially locked",
			malleate: func() {
				// fund liquid vesting module
				err := testutil.FundModuleAccount(ctx, suite.network.App.BankKeeper, types.ModuleName, amount)
				suite.Require().NoError(err)
				// create liquid vesting denom
				// subs 150 second, it is the half of the second period now
				startTime := ctx.BlockTime().Add(-150000 * time.Second)
				suite.network.App.LiquidVestingKeeper.SetDenom(ctx, types.Denom{
					BaseDenom:     "aLIQUID0",
					DisplayDenom:  "LIQUID0",
					OriginalDenom: utils.BaseDenom,
					StartTime:     startTime,
					EndTime:       startTime.Add(lockupPeriods.TotalDuration()),
					LockupPeriods: lockupPeriods,
				})
				// create accounts
				suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, fromAddr))
				suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, toAddr))
				// fund account with liquid denom token
				err = testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, liquidDenomAmount)
				suite.Require().NoError(err)

				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.network.App.BankKeeper.SetDenomMetaData(ctx, liquidTokenMetadata)

				// bind newly created denom to erc20 token
				// Create dummy IBC denom, just to bind ERC20 Precompile with newly created aLiquid denom
				fakeIBCDenom := utils.ComputeIBCDenom(types.ModuleName, liquidTokenMetadata.Base, utils.BaseDenom)
				tokenPair, err := erc20types.NewTokenPairSTRv2(fakeIBCDenom)
				suite.Require().NoError(err)
				// Set real denom to token pair, so precompile could handle transfers properly
				tokenPair.Denom = liquidTokenMetadata.Base
				// k.erc20Keeper.SetToken(ctx, tokenPair) unwrap it below due to pointer receiver in original method.
				suite.network.App.Erc20Keeper.SetTokenPair(ctx, tokenPair)
				suite.network.App.Erc20Keeper.SetDenomMap(ctx, tokenPair.Denom, tokenPair.GetID())
				suite.network.App.Erc20Keeper.SetERC20Map(ctx, tokenPair.GetERC20Contract(), tokenPair.GetID())

				err = suite.network.App.Erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract())
				suite.Require().NoError(err)
			},
			redeemAmount:         600_000,
			expectedLockedAmount: 400_000,
			expectPass:           true,
		},
		{
			name: "fail - insufficient liquid token balance",
			malleate: func() {
				// fund liquid vesting module
				err := testutil.FundModuleAccount(ctx, suite.network.App.BankKeeper, types.ModuleName, amount)
				suite.Require().NoError(err)
				// create liquid vesting denom
				suite.network.App.LiquidVestingKeeper.SetDenom(ctx, types.Denom{
					BaseDenom:     "aLIQUID0",
					DisplayDenom:  "LIQUID0",
					OriginalDenom: utils.BaseDenom,
					LockupPeriods: lockupPeriods,
				})
				// create accounts
				suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, fromAddr))
				suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, toAddr))
				// fund account with liquid denom token
				err = testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, liquidDenomAmount)
				suite.Require().NoError(err)
				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.network.App.BankKeeper.SetDenomMetaData(ctx, liquidTokenMetadata)

				// bind newly created denom to erc20 token
				// Create dummy IBC denom, just to bind ERC20 Precompile with newly created aLiquid denom
				fakeIBCDenom := utils.ComputeIBCDenom(types.ModuleName, liquidTokenMetadata.Base, utils.BaseDenom)
				tokenPair, err := erc20types.NewTokenPairSTRv2(fakeIBCDenom)
				suite.Require().NoError(err)
				// Set real denom to token pair, so precompile could handle transfers properly
				tokenPair.Denom = liquidTokenMetadata.Base
				// k.erc20Keeper.SetToken(ctx, tokenPair) unwrap it below due to pointer receiver in original method.
				suite.network.App.Erc20Keeper.SetTokenPair(ctx, tokenPair)
				suite.network.App.Erc20Keeper.SetDenomMap(ctx, tokenPair.Denom, tokenPair.GetID())
				suite.network.App.Erc20Keeper.SetERC20Map(ctx, tokenPair.GetERC20Contract(), tokenPair.GetID())

				err = suite.network.App.Erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract())
				suite.Require().NoError(err)
			},
			redeemAmount: 4_000_000,
			expectPass:   false,
		},
		{
			name: "fail - liquid denom does not exist",
			malleate: func() {
				// fund liquid vesting module
				err := testutil.FundModuleAccount(ctx, suite.network.App.BankKeeper, types.ModuleName, amount)
				suite.Require().NoError(err)
				// create liquid vesting denom
				suite.network.App.LiquidVestingKeeper.SetDenom(ctx, types.Denom{
					BaseDenom:     "solid",
					DisplayDenom:  "solid18",
					OriginalDenom: utils.BaseDenom,
					LockupPeriods: lockupPeriods,
				})
				// create accounts
				suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, fromAddr))
				suite.network.App.AccountKeeper.SetAccount(ctx, suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, toAddr))
				// fund account with liquid denom token
				err = testutil.FundAccount(ctx, suite.network.App.BankKeeper, fromAddr, liquidDenomAmount)
				suite.Require().NoError(err)
				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.network.App.BankKeeper.SetDenomMetaData(ctx, liquidTokenMetadata)

				// bind newly created denom to erc20 token
				// Create dummy IBC denom, just to bind ERC20 Precompile with newly created aLiquid denom
				fakeIBCDenom := utils.ComputeIBCDenom(types.ModuleName, liquidTokenMetadata.Base, utils.BaseDenom)
				tokenPair, err := erc20types.NewTokenPairSTRv2(fakeIBCDenom)
				suite.Require().NoError(err)
				// Set real denom to token pair, so precompile could handle transfers properly
				tokenPair.Denom = liquidTokenMetadata.Base
				// k.erc20Keeper.SetToken(ctx, tokenPair) unwrap it below due to pointer receiver in original method.
				suite.network.App.Erc20Keeper.SetTokenPair(ctx, tokenPair)
				suite.network.App.Erc20Keeper.SetDenomMap(ctx, tokenPair.Denom, tokenPair.GetID())
				suite.network.App.Erc20Keeper.SetERC20Map(ctx, tokenPair.GetERC20Contract(), tokenPair.GetID())

				err = suite.network.App.Erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract())
				suite.Require().NoError(err)
			},
			redeemAmount: 4_000_000,
			expectPass:   false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()
			// Add empty accounts
			fromAccNumber := suite.keyring.AddKey()
			fromAddr = suite.keyring.GetAccAddr(fromAccNumber)
			toAccNumber := suite.keyring.AddKey()
			toAddr = suite.keyring.GetAccAddr(toAccNumber)

			tc.malleate()

			redeemCoin := sdk.NewInt64Coin("aLIQUID0", tc.redeemAmount)
			msg := types.NewMsgRedeem(fromAddr, toAddr, redeemCoin)
			resp, err := suite.network.App.LiquidVestingKeeper.Redeem(ctx, msg)
			expResponse := &types.MsgRedeemResponse{}
			if tc.expectPass {
				// check returns
				suite.Require().NoError(err)
				suite.Require().Equal(expResponse, resp)

				// check target account has original tokens
				toAcc := suite.network.App.AccountKeeper.GetAccount(ctx, toAddr)
				suite.Require().NotNil(toAcc)
				balanceTarget := suite.network.App.BankKeeper.SpendableCoin(ctx, toAddr, utils.BaseDenom)
				suite.Require().Equal(sdk.NewInt64Coin(utils.BaseDenom, tc.redeemAmount-tc.expectedLockedAmount).String(), balanceTarget.String())
				if tc.expectedLockedAmount > 0 {
					toVa, isClawback := toAcc.(*vestingtypes.ClawbackVestingAccount)
					suite.Require().True(isClawback)
					expectedLockedCoins := sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, tc.expectedLockedAmount))
					actualLockedCoins := toVa.GetLockedUpCoins(ctx.BlockTime())
					suite.Require().Equal(expectedLockedCoins.String(), actualLockedCoins.String())
				}

				// check liquid tokens are burnt
				_, liquidDenomCoin := liquidDenomAmount.Find("aLIQUID0")
				expectedLiquidTokenSupply := liquidDenomCoin.Sub(redeemCoin)
				actualLiquidTokenSupply := suite.network.App.BankKeeper.GetSupply(ctx, "aLIQUID0")
				suite.Require().Equal(expectedLiquidTokenSupply.String(), actualLiquidTokenSupply.String())
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
