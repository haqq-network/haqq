package keeper_test

import (
	"fmt"
	"time"

	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	ethtypes "github.com/evmos/ethermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	"github.com/evmos/ethermint/tests"

	"github.com/haqq-network/haqq/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/x/vesting/types"
)

var (
	balances       = sdk.NewCoins(sdk.NewInt64Coin("aISLM", 1000))
	quarter        = sdk.NewCoins(sdk.NewInt64Coin("aISLM", 250))
	addr           = sdk.AccAddress(tests.GenerateAddress().Bytes())
	addr2          = sdk.AccAddress(tests.GenerateAddress().Bytes())
	addr3          = sdk.AccAddress(tests.GenerateAddress().Bytes())
	addr4          = sdk.AccAddress(tests.GenerateAddress().Bytes())
	addr5          = sdk.AccAddress(tests.GenerateAddress().Bytes())
	lockupPeriods  = sdkvesting.Periods{{Length: 5000, Amount: balances}}
	vestingPeriods = sdkvesting.Periods{
		{Length: 2000, Amount: quarter},
		{Length: 2000, Amount: quarter},
		{Length: 2000, Amount: quarter},
		{Length: 2000, Amount: quarter},
	}
)

func (suite *KeeperTestSuite) TestMsgCreateClawbackVestingAccount() {
	testCases := []struct {
		name               string
		malleate           func()
		from               sdk.AccAddress
		to                 sdk.AccAddress
		startTime          time.Time
		lockup             sdkvesting.Periods
		vesting            sdkvesting.Periods
		merge              bool
		expectExtraBalance int64
		expectPass         bool
	}{
		{
			"ok - new account",
			func() {},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			vestingPeriods,
			false,
			0,
			true,
		},
		{
			"ok - new account - default lockup",
			func() {},
			addr,
			addr2,
			time.Now(),
			nil,
			vestingPeriods,
			false,
			0,
			true,
		},
		{
			"ok - new account - default vesting",
			func() {},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			nil,
			false,
			0,
			true,
		},
		{
			"fail - different locking and vesting amounts",
			func() {},
			addr,
			addr2,
			time.Now(),
			sdkvesting.Periods{
				{Length: 5000, Amount: quarter},
			},
			vestingPeriods,
			false,
			0,
			false,
		},
		{
			"fail - account exists - clawback but no merge",
			func() {
				// Existing clawback account
				vestingStart := s.ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr2, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			vestingPeriods,
			false,
			0,
			false,
		},
		{
			"fail - account exists - no clawback",
			func() {},
			addr,
			addr,
			time.Now(),
			lockupPeriods,
			vestingPeriods,
			false,
			0,
			false,
		},
		{
			"fail - account exists - merge but not clawback",
			func() {},
			addr,
			addr,
			time.Now(),
			lockupPeriods,
			vestingPeriods,
			true,
			0,
			false,
		},
		{
			"fail - account exists - wrong funder",
			func() {
				// Existing clawback account
				vestingStart := s.ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr2, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			addr2,
			addr2,
			time.Now(),
			lockupPeriods,
			vestingPeriods,
			true,
			0,
			false,
		},
		{
			"ok - account exists - addGrant",
			func() {
				// Existing clawback account
				vestingStart := s.ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				funder := addr
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr2, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			vestingPeriods,
			true,
			1000,
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // Reset
			ctx := sdk.WrapSDKContext(suite.ctx)

			tc.malleate()

			err := testutil.FundAccount(s.ctx, s.app.BankKeeper, addr, balances)
			suite.Require().NoError(err)

			msg := types.NewMsgCreateClawbackVestingAccount(
				tc.from,
				tc.to,
				tc.startTime,
				tc.lockup,
				tc.vesting,
				tc.merge,
			)
			res, err := suite.app.VestingKeeper.CreateClawbackVestingAccount(ctx, msg)

			expRes := &types.MsgCreateClawbackVestingAccountResponse{}
			balanceSource := suite.app.BankKeeper.GetBalance(suite.ctx, tc.from, "aISLM")
			balanceDest := suite.app.BankKeeper.GetBalance(suite.ctx, tc.to, "aISLM")

			if tc.expectPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(expRes, res)

				accI := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.to)
				suite.Require().NotNil(accI)
				suite.Require().IsType(&types.ClawbackVestingAccount{}, accI)
				suite.Require().Equal(sdk.NewInt64Coin("aISLM", 0), balanceSource)
				suite.Require().Equal(sdk.NewInt64Coin("aISLM", 1000+tc.expectExtraBalance), balanceDest)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgClawback() {
	testCases := []struct {
		name         string
		malleate     func()
		funder       sdk.AccAddress
		addr         sdk.AccAddress
		dest         sdk.AccAddress
		startTime    time.Time
		expectedPass bool
	}{
		{
			"no clawback account",
			func() {},
			addr,
			sdk.AccAddress(tests.GenerateAddress().Bytes()),
			addr3,
			suite.ctx.BlockTime(),
			false,
		},
		{
			"wrong account type",
			func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(addr4)
				acc := sdkvesting.NewBaseVestingAccount(baseAccount, balances, 500000)
				s.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			addr,
			addr4,
			addr3,
			suite.ctx.BlockTime(),
			false,
		},
		{
			"wrong funder",
			func() {},
			addr3,
			addr2,
			addr3,
			suite.ctx.BlockTime(),
			false,
		},
		{
			"before start time",
			func() {
			},
			addr,
			addr2,
			addr3,
			suite.ctx.BlockTime().Add(time.Hour),
			false,
		},
		{
			"pass",
			func() {
			},
			addr,
			addr2,
			addr3,
			suite.ctx.BlockTime(),
			true,
		},
		{
			"pass - without dest",
			func() {
			},
			addr,
			addr2,
			sdk.AccAddress([]byte{}),
			suite.ctx.BlockTime(),
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset
			ctx := sdk.WrapSDKContext(suite.ctx)

			// Set funder
			funder := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, tc.funder)
			suite.app.AccountKeeper.SetAccount(suite.ctx, funder)
			err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, addr, balances)
			suite.Require().NoError(err)

			// Create Clawback Vesting Account
			createMsg := types.NewMsgCreateClawbackVestingAccount(addr, addr2, tc.startTime, lockupPeriods, vestingPeriods, false)
			createRes, err := suite.app.VestingKeeper.CreateClawbackVestingAccount(ctx, createMsg)
			suite.Require().NoError(err)
			suite.Require().NotNil(createRes)

			balanceDest := suite.app.BankKeeper.GetBalance(suite.ctx, addr2, "aISLM")
			suite.Require().Equal(balanceDest, sdk.NewInt64Coin("aISLM", 1000))

			tc.malleate()

			// Perform clawback
			msg := types.NewMsgClawback(tc.funder, tc.addr, tc.dest)
			res, err := suite.app.VestingKeeper.Clawback(ctx, msg)

			expRes := &types.MsgClawbackResponse{}
			balanceDest = suite.app.BankKeeper.GetBalance(suite.ctx, addr2, "aISLM")
			balanceClaw := suite.app.BankKeeper.GetBalance(suite.ctx, tc.dest, "aISLM")
			if len(tc.dest) == 0 {
				balanceClaw = suite.app.BankKeeper.GetBalance(suite.ctx, tc.funder, "aISLM")
			}

			if tc.expectedPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expRes, res)
				suite.Require().Equal(sdk.NewInt64Coin("aISLM", 0), balanceDest)
				suite.Require().Equal(balances[0], balanceClaw)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(res)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgUpdateVestingFunder() {
	testCases := []struct {
		name         string
		malleate     func()
		funder       sdk.AccAddress
		vestingAcc   sdk.AccAddress
		newFunder    sdk.AccAddress
		expectedPass bool
	}{
		{
			"non-existent vesting account",
			func() {},
			addr,
			sdk.AccAddress(tests.GenerateAddress().Bytes()),
			addr3,
			false,
		},
		{
			"wrong account type",
			func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(addr4)
				acc := sdkvesting.NewBaseVestingAccount(baseAccount, balances, 500000)
				s.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			addr,
			addr4,
			addr3,
			false,
		},
		{
			"wrong funder",
			func() {},
			addr3,
			addr2,
			addr3,
			false,
		},
		{
			"new funder is blocked",
			func() {},
			addr,
			addr2,
			authtypes.NewModuleAddress("transfer"),
			false,
		},
		{
			"update funder successfully",
			func() {
			},
			addr,
			addr2,
			addr3,
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset
			ctx := sdk.WrapSDKContext(suite.ctx)
			startTime := suite.ctx.BlockTime()

			// Set funder
			funder := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, tc.funder)
			suite.app.AccountKeeper.SetAccount(suite.ctx, funder)
			err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, addr, balances)
			suite.Require().NoError(err)

			// Create Clawback Vesting Account
			createMsg := types.NewMsgCreateClawbackVestingAccount(addr, addr2, startTime, lockupPeriods, vestingPeriods, false)
			createRes, err := suite.app.VestingKeeper.CreateClawbackVestingAccount(ctx, createMsg)
			suite.Require().NoError(err)
			suite.Require().NotNil(createRes)

			balanceDest := suite.app.BankKeeper.GetBalance(suite.ctx, addr2, "aISLM")
			suite.Require().Equal(balanceDest, sdk.NewInt64Coin("aISLM", 1000))

			tc.malleate()

			// Perform Vesting account update
			msg := types.NewMsgUpdateVestingFunder(tc.funder, tc.newFunder, tc.vestingAcc)
			res, err := suite.app.VestingKeeper.UpdateVestingFunder(ctx, msg)

			expRes := &types.MsgUpdateVestingFunderResponse{}

			if tc.expectedPass {
				// get the updated vesting account
				vestingAcc := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.vestingAcc)
				va, ok := vestingAcc.(*types.ClawbackVestingAccount)
				suite.Require().True(ok, "vesting account could not be casted to ClawbackVestingAccount")

				suite.Require().NoError(err)
				suite.Require().Equal(expRes, res)
				suite.Require().Equal(va.FunderAddress, tc.newFunder.String())
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(res)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestClawbackVestingAccountStore() {
	suite.SetupTest()

	// Create and set clawback vesting account
	vestingStart := s.ctx.BlockTime()
	funder := sdk.AccAddress(types.ModuleName)
	addr := sdk.AccAddress(tests.GenerateAddress().Bytes())
	baseAccount := authtypes.NewBaseAccountWithAddress(addr)
	acc := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	acc2 := suite.app.AccountKeeper.GetAccount(suite.ctx, acc.GetAddress())
	suite.Require().IsType(&types.ClawbackVestingAccount{}, acc2)
	suite.Require().Equal(acc.String(), acc2.String())
}

func (suite *KeeperTestSuite) TestClawbackVestingAccountMarshal() {
	suite.SetupTest()

	// Create and set clawback vesting account
	vestingStart := s.ctx.BlockTime()
	funder := sdk.AccAddress(types.ModuleName)
	addr := sdk.AccAddress(tests.GenerateAddress().Bytes())
	baseAccount := authtypes.NewBaseAccountWithAddress(addr)
	acc := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)

	bz, err := suite.app.AccountKeeper.MarshalAccount(acc)
	suite.Require().NoError(err)

	acc2, err := suite.app.AccountKeeper.UnmarshalAccount(bz)
	suite.Require().NoError(err)
	suite.Require().IsType(&types.ClawbackVestingAccount{}, acc2)
	suite.Require().Equal(acc.String(), acc2.String())

	// error on bad bytes
	_, err = suite.app.AccountKeeper.UnmarshalAccount(bz[:len(bz)/2])
	suite.Require().Error(err)
}

func (suite *KeeperTestSuite) TestConvertVestingAccount() {
	startTime := s.ctx.BlockTime().Add(-5 * time.Second)
	testCases := []struct {
		name     string
		malleate func() authtypes.AccountI
		expPass  bool
	}{
		{
			"fail - no account found",
			func() authtypes.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				return baseAcc
			},
			false,
		},
		{
			"fail - not a vesting account",
			func() authtypes.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				suite.app.AccountKeeper.SetAccount(suite.ctx, baseAcc)
				return baseAcc
			},
			false,
		},
		{
			"fail - unlocked & unvested",
			func() authtypes.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				lockupPeriods := sdkvesting.Periods{{Length: 0, Amount: balances}}
				vestingPeriods := sdkvesting.Periods{
					{Length: 0, Amount: quarter},
					{Length: 2000, Amount: quarter},
					{Length: 2000, Amount: quarter},
					{Length: 2000, Amount: quarter},
				}
				vestingAcc := types.NewClawbackVestingAccount(baseAcc, from, balances, startTime, lockupPeriods, vestingPeriods, nil)
				suite.app.AccountKeeper.SetAccount(suite.ctx, vestingAcc)
				return vestingAcc
			},
			false,
		},
		{
			"fail - locked & vested",
			func() authtypes.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				vestingPeriods := sdkvesting.Periods{{Length: 0, Amount: balances}}
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				vestingAcc := types.NewClawbackVestingAccount(baseAcc, from, balances, startTime, lockupPeriods, vestingPeriods, nil)
				suite.app.AccountKeeper.SetAccount(suite.ctx, vestingAcc)
				return vestingAcc
			},
			false,
		},
		{
			"fail - locked & unvested",
			func() authtypes.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				vestingAcc := types.NewClawbackVestingAccount(baseAcc, from, balances, suite.ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				suite.app.AccountKeeper.SetAccount(suite.ctx, vestingAcc)
				return vestingAcc
			},
			false,
		},
		{
			"success - unlocked & vested convert to base account",
			func() authtypes.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				vestingPeriods := sdkvesting.Periods{{Length: 0, Amount: balances}}
				vestingAcc := types.NewClawbackVestingAccount(baseAcc, from, balances, startTime, nil, vestingPeriods, nil)
				suite.app.AccountKeeper.SetAccount(suite.ctx, vestingAcc)
				return vestingAcc
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.SetupTest() // reset
		ctx := sdk.WrapSDKContext(suite.ctx)

		acc := tc.malleate()

		msg := types.NewMsgConvertVestingAccount(acc.GetAddress())
		res, err := suite.app.VestingKeeper.ConvertVestingAccount(ctx, msg)

		if tc.expPass {
			suite.Require().NoError(err)
			suite.Require().NotNil(res)

			account := suite.app.AccountKeeper.GetAccount(suite.ctx, acc.GetAddress())

			_, ok := account.(vestingexported.VestingAccount)
			suite.Require().False(ok)

			_, ok = account.(ethtypes.EthAccountI)
			suite.Require().True(ok)

		} else {
			suite.Require().Error(err)
			suite.Require().Nil(res)
		}
	}
}

func (suite *KeeperTestSuite) TestConvertIntoVestingAccount() {
	testCases := []struct {
		name               string
		malleate           func()
		from               sdk.AccAddress
		to                 sdk.AccAddress
		startTime          time.Time
		lockup             sdkvesting.Periods
		vesting            sdkvesting.Periods
		merge              bool
		stake              bool
		expectDelegation   uint64
		expectExtraBalance int64
		expectPass         bool
	}{
		{
			"ok - new account, no delegation",
			func() {},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			vestingPeriods,
			false,
			false,
			0,
			0,
			true,
		},
		{
			"ok - new account - default lockup",
			func() {},
			addr,
			addr2,
			time.Now(),
			nil,
			vestingPeriods,
			false,
			false,
			0,
			0,
			true,
		},
		{
			"ok - new account - default vesting without staking",
			func() {},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			nil,
			false,
			false,
			0,
			0,
			true,
		},
		{
			"ok - new account - default vesting with staking",
			func() {},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			nil,
			false,
			true,
			1000,
			-1000,
			true,
		},
		{
			"fail - new account - no vested coins to delegate",
			func() {},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			vestingPeriods,
			false,
			true,
			0,
			0,
			false,
		},
		{
			"fail - new account - vested coins is not bondable",
			func() {
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr, sdk.NewCoins(sdk.NewInt64Coin("NBND", 500))) //nolint:errcheck
			},
			addr,
			addr2,
			time.Now(),
			sdkvesting.Periods{{Length: 5000, Amount: sdk.NewCoins(sdk.NewInt64Coin("NBND", 500))}},
			nil,
			false,
			true,
			0,
			0,
			false,
		},
		{
			"ok - account exists - no clawback - convert without staking",
			func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(addr3)
				ethAccount := ethtypes.ProtoAccount().(*ethtypes.EthAccount)
				ethAccount.BaseAccount = baseAccount
				s.app.AccountKeeper.SetAccount(s.ctx, ethAccount)
			},
			addr,
			addr3,
			time.Now(),
			lockupPeriods,
			nil,
			false,
			false,
			0,
			0,
			true,
		},
		{
			"ok - account exists - no clawback - convert with staking",
			func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(addr3)
				ethAccount := ethtypes.ProtoAccount().(*ethtypes.EthAccount)
				ethAccount.BaseAccount = baseAccount
				s.app.AccountKeeper.SetAccount(s.ctx, ethAccount)
			},
			addr,
			addr3,
			time.Now(),
			lockupPeriods,
			nil,
			false,
			true,
			1000,
			-1000,
			true,
		},
		{
			"fail - different locking and vesting amounts",
			func() {},
			addr,
			addr2,
			time.Now(),
			sdkvesting.Periods{
				{Length: 5000, Amount: quarter},
			},
			vestingPeriods,
			false,
			false,
			0,
			0,
			false,
		},
		{
			"fail - account exists - clawback but no merge",
			func() {
				// Existing clawback account
				vestingStart := s.ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr2, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			vestingPeriods,
			false,
			false,
			0,
			0,
			false,
		},
		{
			"fail - account exists - wrong funder",
			func() {
				// Existing clawback account
				vestingStart := s.ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr2, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			addr2,
			addr2,
			time.Now(),
			lockupPeriods,
			vestingPeriods,
			true,
			false,
			0,
			0,
			false,
		},
		{
			"ok - account exists - addGrant",
			func() {
				// Existing clawback account
				vestingStart := s.ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				funder := addr
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr2, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			vestingPeriods,
			true,
			false,
			0,
			1000,
			true,
		},
		{
			"ok - account exists - addGrant with full vesting and staking",
			func() {
				// Existing clawback account
				vestingStart := s.ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				funder := addr
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr2, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			nil,
			true,
			true,
			1000,
			0,
			true,
		},
		{
			"ok - account exists - addGrant with partial vesting and staking",
			func() {
				// Existing clawback account
				vestingStart := s.ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				funder := addr
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, addr2, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			addr,
			addr2,
			time.Now(),
			lockupPeriods,
			sdkvesting.Periods{
				{Length: 0, Amount: quarter},
				{Length: 2000, Amount: quarter},
				{Length: 2000, Amount: quarter},
				{Length: 2000, Amount: quarter},
			},
			true,
			true,
			250,
			750,
			true,
		},
		{
			"fail - account exists - not eth, not vesting, unsupported",
			func() {
				// Existing module account
				baseAccount := authtypes.NewBaseAccountWithAddress(addr5)
				s.app.AccountKeeper.SetAccount(s.ctx, authtypes.NewModuleAccount(baseAccount, "testmodule"))
			},
			addr,
			addr5,
			time.Now(),
			lockupPeriods,
			nil,
			true,
			true,
			0,
			0,
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			var err error
			suite.SetupTest() // Reset
			ctx := sdk.WrapSDKContext(suite.ctx)

			tc.malleate()

			err = testutil.FundAccount(s.ctx, s.app.BankKeeper, addr, balances)
			suite.Require().NoError(err)

			valAddr := sdk.ValAddress{}
			expBalanceBonded := sdk.ZeroInt()
			if tc.stake {
				vals := suite.app.StakingKeeper.GetAllValidators(suite.ctx)
				suite.Require().Greater(len(vals), 0)
				valopAddress := vals[0].OperatorAddress
				valAddr, err = sdk.ValAddressFromBech32(valopAddress)
				suite.Require().NoError(err)
				expBalanceBonded = sdk.NewIntFromUint64(tc.expectDelegation)
			}

			msg := types.NewMsgConvertIntoVestingAccount(
				tc.from,
				tc.to,
				tc.startTime,
				tc.lockup,
				tc.vesting,
				tc.merge,
				tc.stake,
				valAddr,
			)
			res, err := suite.app.VestingKeeper.ConvertIntoVestingAccount(ctx, msg)

			expRes := &types.MsgConvertIntoVestingAccountResponse{}
			balanceSource := suite.app.BankKeeper.GetBalance(suite.ctx, tc.from, "aISLM")
			balanceDest := suite.app.BankKeeper.GetBalance(suite.ctx, tc.to, "aISLM")
			balanceBonded := suite.app.StakingKeeper.GetDelegatorBonded(suite.ctx, tc.to)

			if tc.expectPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(expRes, res)

				accI := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.to)
				suite.Require().NotNil(accI)
				suite.Require().IsType(&types.ClawbackVestingAccount{}, accI)
				suite.Require().Equal(sdk.NewInt64Coin("aISLM", 0), balanceSource)
				suite.Require().Equal(sdk.NewInt64Coin("aISLM", 1000+tc.expectExtraBalance), balanceDest)
				if tc.stake {
					suite.Require().True(balanceBonded.GT(sdk.ZeroInt()))
				}
				suite.Require().True(expBalanceBonded.Equal(balanceBonded))
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}
