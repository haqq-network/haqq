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
	"github.com/ethereum/go-ethereum/crypto"

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
	addr1 = sdk.AccAddress(tests.GenerateAddress().Bytes())
	addr2 = sdk.AccAddress(tests.GenerateAddress().Bytes())
)

func (suite *KeeperTestSuite) TestLiquidate() {
	var ctx sdk.Context

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
				err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin(utils.BaseDenom, third.AmountOf(utils.BaseDenom)),
			expectPass: true,
		},
		{
			name: "ok - standard liquidation two thirds",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				startTime := suite.ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin(utils.BaseDenom, third.AmountOf(utils.BaseDenom)).Add(sdk.NewCoin(utils.BaseDenom, third.AmountOf(utils.BaseDenom))),
			expectPass: true,
		},
		{
			name: "ok - block time matches end of current period",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				startTime := suite.ctx.BlockTime().Add(-100 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin(utils.BaseDenom, third.AmountOf(utils.BaseDenom)),
			expectPass: true,
		},
		{
			name: "ok - standard liquidation full liquidation",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				startTime := suite.ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin(utils.BaseDenom, amount.AmountOf(utils.BaseDenom)),
			expectPass: true,
		},
		{
			name: "ok - liquidate partially staked tokens",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				startTime := suite.ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
				_, err = s.app.StakingKeeper.Delegate(s.ctx, addr1, third.AmountOf(utils.BaseDenom), stakingtypes.Unbonded, s.validator, true)
				suite.Require().NoError(err)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin(utils.BaseDenom, third.AmountOf(utils.BaseDenom).Add(third.AmountOf(utils.BaseDenom))),
			expectPass: true,
		},
		{
			name: "fail - liquidate amount bigger than locked but less than total",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				startTime := suite.ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin(utils.BaseDenom, amount.AmountOf(utils.BaseDenom).Add(math.NewInt(1_500_000))),
			expectPass: false,
		},
		{
			name: "fail - liquidate staked tokens",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				startTime := suite.ctx.BlockTime().Add(-10 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
				_, err = s.app.StakingKeeper.Delegate(s.ctx, addr1, amount.AmountOf(utils.BaseDenom), stakingtypes.Unbonded, s.validator, true)
				suite.Require().NoError(err)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin(utils.BaseDenom, amount.AmountOf(utils.BaseDenom)),
			expectPass: false,
		},
		{
			name: "fail - liquidate tokens partially unlocked",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				startTime := suite.ctx.BlockTime().Add(-200001 * time.Second)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewCoin(utils.BaseDenom, math.NewInt(1_500_000)),
			expectPass: false,
		},
		{
			name: "fail - amount exceeded",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, suite.ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewInt64Coin(utils.BaseDenom, 4_000_000),
			expectPass: false,
		},
		{
			name: "fail - denom is not aISLM",
			malleate: func() {
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
				clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, suite.ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
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
				err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
				suite.Require().NoError(err)
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			from:       addr1,
			to:         addr2,
			amount:     sdk.NewInt64Coin(utils.BaseDenom, 2_000_000),
			expectPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // Reset
			ctx := sdk.WrapSDKContext(suite.ctx)

			tc.malleate()

			accFrom := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.from)
			vaFrom, ok := accFrom.(*vestingtypes.ClawbackVestingAccount)
			if !ok {
				suite.T().Fatal("account is not clawback vesting account")
			}
			suite.T().Logf("locked only coins: %s", vaFrom.GetLockedUpCoins(suite.ctx.BlockTime()).String())
			suite.T().Logf("UN-locked only coins: %s", vaFrom.GetUnlockedCoins(suite.ctx.BlockTime()).String())
			spendable := suite.app.BankKeeper.SpendableCoin(suite.ctx, tc.from, utils.BaseDenom)
			suite.T().Logf("spendable coins: %s", spendable.String())
			suite.T().Logf("liquidation amount: %s", tc.amount.String())

			msg := types.NewMsgLiquidate(tc.from, tc.to, tc.amount)
			resp, err := suite.app.LiquidVestingKeeper.Liquidate(ctx, msg)
			expResponse := &types.MsgLiquidateResponse{
				Minted: sdk.NewCoin(types.DenomBaseNameFromID(0), tc.amount.Amount),
			}

			if tc.expectPass {
				// check returns
				suite.Require().NoError(err)
				suite.Require().Equal(expResponse.Minted, resp.Minted)
				suite.Require().NotEmpty(resp.ContractAddr)

				// check target account exists and has liquid token
				accIto := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.to)
				suite.Require().NotNil(accIto)
				balanceTarget := suite.app.BankKeeper.GetBalance(suite.ctx, tc.to, types.DenomBaseNameFromID(0))
				suite.Require().Equal(expResponse.Minted.String(), balanceTarget.String())

				// check liquidated vesting locked coins are decreased on initial account
				accIFrom := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.from)
				suite.Require().NotNil(accIFrom)
				cva, isClawback := accIFrom.(*vestingtypes.ClawbackVestingAccount)
				suite.Require().True(isClawback)
				suite.Require().Equal(cva.GetLockedUpCoins(suite.ctx.BlockTime()).Add(cva.GetUnlockedCoins(suite.ctx.BlockTime())...), lockupPeriods.TotalAmount().Sub(tc.amount))

				// check newly created liquid denom
				liquidDenom, found := suite.app.LiquidVestingKeeper.GetDenom(suite.ctx, types.DenomBaseNameFromID(0))
				suite.Require().True(found)
				suite.Require().Equal(cva.EndTime, liquidDenom.EndTime.Unix())

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

func (suite *KeeperTestSuite) TestMultipleLiquidationsFromOneAccount() {
	var (
		ctx               sdk.Context
		from              = addr1
		to                = addr2
		liquidationAmount = sdk.NewCoin(utils.BaseDenom, third.AmountOf(utils.BaseDenom))
		funder            = sdk.AccAddress(types.ModuleName)
	)
	suite.SetupTest() // Reset

	baseAccount := authtypes.NewBaseAccountWithAddress(addr1)
	startTime := suite.ctx.BlockTime().Add(-10 * time.Second)
	clawbackAccount := vestingtypes.NewClawbackVestingAccount(baseAccount, funder, amount, startTime, lockupPeriods, vestingPeriods, nil)
	err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, amount)
	suite.Require().NoError(err)
	s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)

	// FIRST LIQUIDATION
	msg := types.NewMsgLiquidate(from, to, liquidationAmount)
	resp, err := suite.app.LiquidVestingKeeper.Liquidate(ctx, msg)
	expResponse := &types.MsgLiquidateResponse{
		Minted: sdk.NewCoin("aLIQUID0", liquidationAmount.Amount),
	}

	// check returns
	suite.Require().NoError(err)
	suite.Require().Equal(expResponse.Minted, resp.Minted)
	suite.Require().NotEmpty(resp.ContractAddr)

	// check target account exists and has liquid token
	accIto := suite.app.AccountKeeper.GetAccount(suite.ctx, to)
	suite.Require().NotNil(accIto)
	balanceTarget := suite.app.BankKeeper.GetBalance(suite.ctx, to, types.DenomBaseNameFromID(0))
	suite.Require().Equal(sdk.NewCoin(types.DenomBaseNameFromID(0), liquidationAmount.Amount).String(), balanceTarget.String())

	// check liquidated vesting locked coins are decreased on initial account
	accIFrom := suite.app.AccountKeeper.GetAccount(suite.ctx, from)
	suite.Require().NotNil(accIFrom)
	cva, isClawback := accIFrom.(*vestingtypes.ClawbackVestingAccount)
	suite.Require().True(isClawback)
	suite.Require().Equal(cva.GetLockedUpCoins(suite.ctx.BlockTime()), lockupPeriods.TotalAmount().Sub(liquidationAmount))

	// check erc20 token contract
	pair0Resp, err := s.app.Erc20Keeper.TokenPair(s.ctx, &erc20types.QueryTokenPairRequest{Token: types.DenomBaseNameFromID(0)})
	s.Require().NoError(err)
	s.Require().True(pair0Resp.TokenPair.Enabled)
	ethAccTo, isEthAccount := accIto.(*haqqtypes.EthAccount)
	s.Require().True(isEthAccount)
	balanceOfLiquidTokeErc20Pair0 := s.app.Erc20Keeper.BalanceOf(
		s.ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		pair0Resp.TokenPair.GetERC20Contract(),
		common.BytesToAddress(ethAccTo.GetAddress().Bytes()),
	)
	s.Require().Equal(liquidationAmount.Amount.String(), balanceOfLiquidTokeErc20Pair0.String())

	// SECOND LIQUIDATION
	msg = types.NewMsgLiquidate(from, to, liquidationAmount)
	resp, err = suite.app.LiquidVestingKeeper.Liquidate(ctx, msg)

	expResponse = &types.MsgLiquidateResponse{
		Minted: sdk.NewCoin("aLIQUID1", liquidationAmount.Amount),
	}

	// check returns
	suite.Require().NoError(err)
	suite.Require().Equal(expResponse.Minted, resp.Minted)
	suite.Require().NotEmpty(resp.ContractAddr)

	// check target account exists and has liquid token
	balanceTarget = suite.app.BankKeeper.GetBalance(suite.ctx, to, types.DenomBaseNameFromID(1))
	suite.Require().Equal(sdk.NewCoin(types.DenomBaseNameFromID(1), liquidationAmount.Amount).String(), balanceTarget.String())

	// check liquidated vesting locked coins are decreased on initial account
	accIFrom = suite.app.AccountKeeper.GetAccount(suite.ctx, from)
	suite.Require().NotNil(accIFrom)
	cva, isClawback = accIFrom.(*vestingtypes.ClawbackVestingAccount)
	suite.Require().True(isClawback)
	suite.Require().Equal(cva.GetLockedUpCoins(suite.ctx.BlockTime()), sdk.NewCoins(liquidationAmount))

	// check erc20 token contract
	pair1Resp, err := s.app.Erc20Keeper.TokenPair(s.ctx, &erc20types.QueryTokenPairRequest{Token: types.DenomBaseNameFromID(1)})
	s.Require().NoError(err)
	s.Require().True(pair1Resp.TokenPair.Enabled)
	balanceOfLiquidTokeErc20Pair1 := s.app.Erc20Keeper.BalanceOf(
		s.ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		pair1Resp.TokenPair.GetERC20Contract(),
		common.BytesToAddress(ethAccTo.GetAddress().Bytes()),
	)
	s.Require().Equal(liquidationAmount.Amount.String(), balanceOfLiquidTokeErc20Pair1.String())
}

func (suite *KeeperTestSuite) TestRedeem() {
	var ctx sdk.Context

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
				err := testutil.FundModuleAccount(s.ctx, s.app.BankKeeper, types.ModuleName, amount)
				suite.Require().NoError(err)
				// create liquid vesting denom
				s.app.LiquidVestingKeeper.SetDenom(s.ctx, types.Denom{
					BaseDenom:     "aLIQUID0",
					DisplayDenom:  "LIQUID0",
					OriginalDenom: utils.BaseDenom,
					LockupPeriods: lockupPeriods,
				})
				// create accounts
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr1))
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr2))
				// fund account with liquid denom token
				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, liquidDenomAmount)
				suite.Require().NoError(err)
				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, liquidTokenMetadata)

				// bind newly created denom to erc20 token
				// Create dummy IBC denom, just to bind ERC20 Precompile with newly created aLiquid denom
				fakeIBCDenom := utils.ComputeIBCDenom(types.ModuleName, liquidTokenMetadata.Base, utils.BaseDenom)
				tokenPair, err := erc20types.NewTokenPairSTRv2(fakeIBCDenom)
				suite.Require().NoError(err)
				// Set real denom to token pair, so precompile could handle transfers properly
				tokenPair.Denom = liquidTokenMetadata.Base
				// k.erc20Keeper.SetToken(ctx, tokenPair) unwrap it below due to pointer receiver in original method.
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, tokenPair)
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, tokenPair.Denom, tokenPair.GetID())
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, tokenPair.GetERC20Contract(), tokenPair.GetID())

				err = suite.app.Erc20Keeper.EnableDynamicPrecompiles(suite.ctx, tokenPair.GetERC20Contract())
				suite.Require().NoError(err)
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
				err := testutil.FundModuleAccount(s.ctx, s.app.BankKeeper, types.ModuleName, amount)
				suite.Require().NoError(err)
				// create liquid vesting denom
				// subs 150 second, it is the half of the second period now
				startTime := s.ctx.BlockTime().Add(-150000 * time.Second)
				s.app.LiquidVestingKeeper.SetDenom(s.ctx, types.Denom{
					BaseDenom:     "aLIQUID0",
					DisplayDenom:  "LIQUID0",
					OriginalDenom: utils.BaseDenom,
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
				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, liquidDenomAmount)
				suite.Require().NoError(err)

				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, liquidTokenMetadata)

				// bind newly created denom to erc20 token
				// Create dummy IBC denom, just to bind ERC20 Precompile with newly created aLiquid denom
				fakeIBCDenom := utils.ComputeIBCDenom(types.ModuleName, liquidTokenMetadata.Base, utils.BaseDenom)
				tokenPair, err := erc20types.NewTokenPairSTRv2(fakeIBCDenom)
				suite.Require().NoError(err)
				// Set real denom to token pair, so precompile could handle transfers properly
				tokenPair.Denom = liquidTokenMetadata.Base
				// k.erc20Keeper.SetToken(ctx, tokenPair) unwrap it below due to pointer receiver in original method.
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, tokenPair)
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, tokenPair.Denom, tokenPair.GetID())
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, tokenPair.GetERC20Contract(), tokenPair.GetID())

				err = suite.app.Erc20Keeper.EnableDynamicPrecompiles(suite.ctx, tokenPair.GetERC20Contract())
				suite.Require().NoError(err)
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
				err := testutil.FundModuleAccount(s.ctx, s.app.BankKeeper, types.ModuleName, amount)
				suite.Require().NoError(err)
				// create liquid vesting denom
				s.app.LiquidVestingKeeper.SetDenom(s.ctx, types.Denom{
					BaseDenom:     "aLIQUID0",
					DisplayDenom:  "LIQUID0",
					OriginalDenom: utils.BaseDenom,
					LockupPeriods: lockupPeriods,
				})
				// create accounts
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr1))
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr2))
				// fund account with liquid denom token
				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, liquidDenomAmount)
				suite.Require().NoError(err)
				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, liquidTokenMetadata)

				// bind newly created denom to erc20 token
				// Create dummy IBC denom, just to bind ERC20 Precompile with newly created aLiquid denom
				fakeIBCDenom := utils.ComputeIBCDenom(types.ModuleName, liquidTokenMetadata.Base, utils.BaseDenom)
				tokenPair, err := erc20types.NewTokenPairSTRv2(fakeIBCDenom)
				suite.Require().NoError(err)
				// Set real denom to token pair, so precompile could handle transfers properly
				tokenPair.Denom = liquidTokenMetadata.Base
				// k.erc20Keeper.SetToken(ctx, tokenPair) unwrap it below due to pointer receiver in original method.
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, tokenPair)
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, tokenPair.Denom, tokenPair.GetID())
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, tokenPair.GetERC20Contract(), tokenPair.GetID())

				err = suite.app.Erc20Keeper.EnableDynamicPrecompiles(suite.ctx, tokenPair.GetERC20Contract())
				suite.Require().NoError(err)
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
				err := testutil.FundModuleAccount(s.ctx, s.app.BankKeeper, types.ModuleName, amount)
				suite.Require().NoError(err)
				// create liquid vesting denom
				s.app.LiquidVestingKeeper.SetDenom(s.ctx, types.Denom{
					BaseDenom:     "solid",
					DisplayDenom:  "solid18",
					OriginalDenom: utils.BaseDenom,
					LockupPeriods: lockupPeriods,
				})
				// create accounts
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr1))
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewBaseAccountWithAddress(addr2))
				// fund account with liquid denom token
				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, addr1, liquidDenomAmount)
				suite.Require().NoError(err)
				liquidTokenMetadata := banktypes.Metadata{
					Description: "Liquid vesting token",
					DenomUnits:  []*banktypes.DenomUnit{{Denom: "aLIQUID0", Exponent: 0}, {Denom: "LIQUID0", Exponent: 18}},
					Base:        "aLIQUID0",
					Display:     "LIQUID0",
					Name:        "LIQUID0",
					Symbol:      "LIQUID0",
				}

				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, liquidTokenMetadata)

				// bind newly created denom to erc20 token
				// Create dummy IBC denom, just to bind ERC20 Precompile with newly created aLiquid denom
				fakeIBCDenom := utils.ComputeIBCDenom(types.ModuleName, liquidTokenMetadata.Base, utils.BaseDenom)
				tokenPair, err := erc20types.NewTokenPairSTRv2(fakeIBCDenom)
				suite.Require().NoError(err)
				// Set real denom to token pair, so precompile could handle transfers properly
				tokenPair.Denom = liquidTokenMetadata.Base
				// k.erc20Keeper.SetToken(ctx, tokenPair) unwrap it below due to pointer receiver in original method.
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, tokenPair)
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, tokenPair.Denom, tokenPair.GetID())
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, tokenPair.GetERC20Contract(), tokenPair.GetID())

				err = suite.app.Erc20Keeper.EnableDynamicPrecompiles(suite.ctx, tokenPair.GetERC20Contract())
				suite.Require().NoError(err)
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
			expResponse := &types.MsgRedeemResponse{}
			if tc.expectPass {
				// check returns
				suite.Require().NoError(err)
				suite.Require().Equal(expResponse, resp)

				// check target account has original tokens
				accIto := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.redeemTo)
				suite.Require().NotNil(accIto)
				balanceTarget := suite.app.BankKeeper.SpendableCoin(suite.ctx, tc.redeemTo, utils.BaseDenom)
				suite.Require().Equal(sdk.NewInt64Coin(utils.BaseDenom, tc.redeemAmount-tc.expectedLockedAmount).String(), balanceTarget.String())
				if tc.expectedLockedAmount > 0 {
					cva, isClawback := accIto.(*vestingtypes.ClawbackVestingAccount)
					suite.Require().True(isClawback)
					expectedLockedCoins := sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, tc.expectedLockedAmount))
					actualLockedCoins := cva.GetLockedUpCoins(s.ctx.BlockTime())
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
