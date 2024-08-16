package keeper_test

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/tests"
	"github.com/haqq-network/haqq/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	ethtypes "github.com/haqq-network/haqq/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	"github.com/haqq-network/haqq/x/vesting/types"
)

var (
	vestAmount     = int64(1000)
	balances       = sdk.NewCoins(sdk.NewInt64Coin("aISLM", vestAmount))
	quarter        = sdk.NewCoins(sdk.NewInt64Coin("aISLM", 250))
	funder         = sdk.AccAddress(tests.GenerateAddress().Bytes())
	vestingAddr    = sdk.AccAddress(tests.GenerateAddress().Bytes())
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
		funder             sdk.AccAddress
		vestingAddr        sdk.AccAddress
		lockup             sdkvesting.Periods
		vesting            sdkvesting.Periods
		merge              bool
		expectExtraBalance int64
		expPass            bool
		errContains        string
	}{
		{
			name:               "ok - new account",
			malleate:           func() {},
			funder:             funder,
			vestingAddr:        vestingAddr,
			lockup:             lockupPeriods,
			vesting:            vestingPeriods,
			merge:              false,
			expectExtraBalance: 0,
			expPass:            true,
		},
		{
			name:               "ok - new account - default lockup",
			malleate:           func() {},
			funder:             funder,
			vestingAddr:        vestingAddr,
			lockup:             nil,
			vesting:            vestingPeriods,
			merge:              false,
			expectExtraBalance: 0,
			expPass:            true,
		},
		{
			name:               "ok - new account - default vesting",
			malleate:           func() {},
			funder:             funder,
			vestingAddr:        vestingAddr,
			lockup:             lockupPeriods,
			vesting:            nil,
			merge:              false,
			expectExtraBalance: 0,
			expPass:            true,
		},
		{
			name:        "fail - different locking and vesting amounts",
			malleate:    func() {},
			funder:      funder,
			vestingAddr: vestingAddr,
			lockup: sdkvesting.Periods{
				{Length: 5000, Amount: quarter},
			},
			vesting:            vestingPeriods,
			merge:              false,
			expectExtraBalance: 0,
			expPass:            false,
			errContains:        "lockup and vesting amounts must be equal",
		},
		{
			name: "fail - account exists - clawback but no merge",
			malleate: func() {
				// Existing clawback account
				vestingStart := s.ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, vestingAddr, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			funder:             funder,
			vestingAddr:        vestingAddr,
			lockup:             lockupPeriods,
			vesting:            vestingPeriods,
			merge:              false,
			expectExtraBalance: 0,
			expPass:            false,
			errContains:        "already exists; consider using --merge",
		},
		{
			name:               "fail - account exists - no clawback",
			malleate:           func() {},
			funder:             funder,
			vestingAddr:        funder,
			lockup:             lockupPeriods,
			vesting:            vestingPeriods,
			merge:              false,
			expectExtraBalance: 0,
			expPass:            false,
			errContains:        "already exists: invalid request",
		},
		{
			name:               "fail - account exists - merge but not clawback",
			malleate:           func() {},
			funder:             funder,
			vestingAddr:        funder,
			lockup:             lockupPeriods,
			vesting:            vestingPeriods,
			merge:              true,
			expectExtraBalance: 0,
			expPass:            false,
			errContains:        "must be a clawback vesting account",
		},
		{
			name: "fail - account exists - wrong funder",
			malleate: func() {
				// Existing clawback account
				vestingStart := s.ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, vestingAddr, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			funder:             addr3,
			vestingAddr:        vestingAddr,
			lockup:             lockupPeriods,
			vesting:            vestingPeriods,
			merge:              true,
			expectExtraBalance: 0,
			expPass:            false,
			errContains:        "can only accept grants from account",
		},
		{
			name: "ok - account exists - addGrant",
			malleate: func() {
				// Existing clawback account
				vestingStart := s.ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, vestingAddr, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			funder:             funder,
			vestingAddr:        vestingAddr,
			lockup:             lockupPeriods,
			vesting:            vestingPeriods,
			merge:              true,
			expectExtraBalance: 1000,
			expPass:            true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // Reset
			ctx := sdk.WrapSDKContext(suite.ctx)

			tc.malleate()

			err := testutil.FundAccount(s.ctx, s.app.BankKeeper, funder, balances)
			suite.Require().NoError(err)

			msg := types.NewMsgCreateClawbackVestingAccount(
				tc.funder,
				tc.vestingAddr,
				suite.ctx.BlockTime().Add(-1*time.Second),
				tc.lockup,
				tc.vesting,
				tc.merge,
			)
			res, err := suite.app.VestingKeeper.CreateClawbackVestingAccount(ctx, msg)

			expRes := &types.MsgCreateClawbackVestingAccountResponse{}
			balanceFunder := suite.app.BankKeeper.GetBalance(suite.ctx, tc.funder, "aISLM")
			balanceVestingAddr := suite.app.BankKeeper.GetBalance(suite.ctx, tc.vestingAddr, "aISLM")

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(expRes, res)

				accI := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.vestingAddr)
				suite.Require().NotNil(accI)
				suite.Require().IsType(&types.ClawbackVestingAccount{}, accI)
				suite.Require().Equal(sdk.NewInt64Coin("aISLM", 0), balanceFunder)
				suite.Require().Equal(sdk.NewInt64Coin("aISLM", vestAmount+tc.expectExtraBalance), balanceVestingAddr)
			} else {
				suite.Require().Error(err, tc.name)
				suite.Require().ErrorContains(err, tc.errContains)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgClawback() {
	testCases := []struct {
		name        string
		malleate    func()
		funder      sdk.AccAddress
		vestingAddr sdk.AccAddress
		// clawbackDest is the address to send the coins that were clawed back to
		clawbackDest sdk.AccAddress
		startTime    time.Time
		expPass      bool
		errContains  string
	}{
		{
			name:        "fail - account does not exist",
			malleate:    func() {},
			funder:      funder,
			vestingAddr: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			startTime:   suite.ctx.BlockTime(),
			expPass:     false,
			errContains: "does not exist",
		},
		{
			name: "fail - no clawback account",
			malleate: func() {
				err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, addr4, balances)
				suite.Require().NoError(err)
			},
			funder:       funder,
			vestingAddr:  addr4,
			clawbackDest: addr3,
			startTime:    suite.ctx.BlockTime(),
			expPass:      false,
			errContains:  types.ErrNotSubjectToClawback.Error(),
		},
		{
			name: "fail - wrong account type",
			malleate: func() {
				// create a base vesting account instead of a clawback vesting account at the vesting address
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				acc := sdkvesting.NewBaseVestingAccount(baseAccount, balances, 500000)
				s.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			funder:       funder,
			vestingAddr:  vestingAddr,
			clawbackDest: addr3,
			startTime:    suite.ctx.BlockTime(),
			expPass:      false,
			errContains:  types.ErrNotSubjectToClawback.Error(),
		},
		{
			name:         "fail - wrong funder",
			malleate:     func() {},
			funder:       addr3,
			vestingAddr:  vestingAddr,
			clawbackDest: addr3,
			startTime:    suite.ctx.BlockTime(),
			expPass:      false,
			errContains:  "clawback can only be requested by original funder",
		},
		{
			name:         "fail - clawback destination is blocked",
			malleate:     func() {},
			funder:       funder,
			vestingAddr:  vestingAddr,
			clawbackDest: authtypes.NewModuleAddress("transfer"),
			startTime:    suite.ctx.BlockTime(),
			expPass:      false,
			errContains:  "is not allowed to receive funds",
		},
		{
			name:        "pass - before start time",
			malleate:    func() {},
			funder:      funder,
			vestingAddr: vestingAddr,
			startTime:   suite.ctx.BlockTime().Add(time.Hour),
			expPass:     true,
		},
		{
			name:         "pass - with clawback destination",
			malleate:     func() {},
			funder:       funder,
			vestingAddr:  vestingAddr,
			clawbackDest: addr3,
			startTime:    suite.ctx.BlockTime(),
			expPass:      true,
		},
		{
			name:        "pass - without clawback destination",
			malleate:    func() {},
			funder:      funder,
			vestingAddr: vestingAddr,
			startTime:   suite.ctx.BlockTime(),
			expPass:     true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset
			ctx := sdk.WrapSDKContext(suite.ctx)

			// Set funder
			txFfunder := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, tc.funder)
			suite.app.AccountKeeper.SetAccount(suite.ctx, txFfunder)
			err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, funder, balances)
			suite.Require().NoError(err)

			// Create Clawback Vesting Account
			createMsg := types.NewMsgCreateClawbackVestingAccount(funder, vestingAddr, tc.startTime, lockupPeriods, vestingPeriods, false)
			createRes, err := suite.app.VestingKeeper.CreateClawbackVestingAccount(ctx, createMsg)
			suite.Require().NoError(err)
			suite.Require().NotNil(createRes)

			balanceDest := suite.app.BankKeeper.GetBalance(suite.ctx, vestingAddr, "aISLM")
			suite.Require().Equal(balanceDest, sdk.NewInt64Coin("aISLM", 1000))

			tc.malleate()

			// Perform clawback
			msg := types.NewMsgClawback(tc.funder, tc.vestingAddr, tc.clawbackDest)
			res, err := suite.app.VestingKeeper.Clawback(ctx, msg)

			balanceDest = suite.app.BankKeeper.GetBalance(suite.ctx, vestingAddr, "aISLM")
			balanceClaw := suite.app.BankKeeper.GetBalance(suite.ctx, tc.clawbackDest, "aISLM")
			if len(tc.clawbackDest) == 0 {
				balanceClaw = suite.app.BankKeeper.GetBalance(suite.ctx, tc.funder, "aISLM")
			}

			if tc.expPass {
				suite.Require().NoError(err)

				expRes := &types.MsgClawbackResponse{}
				suite.Require().Equal(expRes, res, "expected full balances to be clawed back")
				suite.Require().Equal(sdk.NewInt64Coin("aISLM", 0), balanceDest)
				suite.Require().Equal(balances[0], balanceClaw)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
				suite.Require().Nil(res)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgUpdateVestingFunder() {
	newFunder := sdk.AccAddress(utiltx.GenerateAddress().Bytes())

	testCases := []struct {
		name        string
		malleate    func()
		funder      sdk.AccAddress
		vestingAcc  sdk.AccAddress
		newFunder   sdk.AccAddress
		expPass     bool
		errContains string
	}{
		{
			name:        "fail - non-existent account",
			malleate:    func() {},
			funder:      funder,
			vestingAcc:  sdk.AccAddress(tests.GenerateAddress().Bytes()),
			newFunder:   newFunder,
			expPass:     false,
			errContains: "does not exist",
		},
		{
			name: "fail - wrong account type",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(addr4)
				acc := sdkvesting.NewBaseVestingAccount(baseAccount, balances, 500000)
				s.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			funder:      funder,
			vestingAcc:  addr4,
			newFunder:   newFunder,
			expPass:     false,
			errContains: types.ErrNotSubjectToClawback.Error(),
		},
		{
			name:        "fail - wrong funder",
			malleate:    func() {},
			funder:      newFunder,
			vestingAcc:  vestingAddr,
			newFunder:   newFunder,
			expPass:     false,
			errContains: "is not the current funder and cannot update the funder address",
		},
		{
			name:        "fail - new funder is blocked",
			malleate:    func() {},
			funder:      funder,
			vestingAcc:  vestingAddr,
			newFunder:   authtypes.NewModuleAddress("transfer"),
			expPass:     false,
			errContains: "is a blocked address and not allowed to fund vesting accounts",
		},
		{
			name: "pass - update funder successfully",
			malleate: func() {
			},
			funder:     funder,
			vestingAcc: vestingAddr,
			newFunder:  newFunder,
			expPass:    true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset
			ctx := sdk.WrapSDKContext(suite.ctx)
			startTime := suite.ctx.BlockTime()

			// Set funder
			tcFunder := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, tc.funder)
			suite.app.AccountKeeper.SetAccount(suite.ctx, tcFunder)
			err := testutil.FundAccount(suite.ctx, suite.app.BankKeeper, funder, balances)
			suite.Require().NoError(err)

			// Create Clawback Vesting Account
			createMsg := types.NewMsgCreateClawbackVestingAccount(funder, vestingAddr, startTime, lockupPeriods, vestingPeriods, false)
			createRes, err := suite.app.VestingKeeper.CreateClawbackVestingAccount(ctx, createMsg)
			suite.Require().NoError(err)
			suite.Require().NotNil(createRes)

			balanceDest := suite.app.BankKeeper.GetBalance(suite.ctx, vestingAddr, "aISLM")
			suite.Require().Equal(balanceDest, sdk.NewInt64Coin("aISLM", 1000))

			tc.malleate()

			// Perform Vesting account update
			msg := types.NewMsgUpdateVestingFunder(tc.funder, tc.newFunder, tc.vestingAcc)
			res, err := suite.app.VestingKeeper.UpdateVestingFunder(ctx, msg)

			expRes := &types.MsgUpdateVestingFunderResponse{}

			if tc.expPass {
				// get the updated vesting account
				vestingAcc := suite.app.AccountKeeper.GetAccount(suite.ctx, tc.vestingAcc)
				va, ok := vestingAcc.(*types.ClawbackVestingAccount)
				suite.Require().True(ok, "vesting account could not be casted to ClawbackVestingAccount")

				suite.Require().NoError(err)
				suite.Require().Equal(expRes, res)
				suite.Require().Equal(va.FunderAddress, tc.newFunder.String())
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.errContains)
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
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			acc := tc.malleate()

			msg := types.NewMsgConvertVestingAccount(acc.GetAddress())
			res, err := suite.app.VestingKeeper.ConvertVestingAccount(sdk.WrapSDKContext(suite.ctx), msg)

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
		})
	}
}

func (suite *KeeperTestSuite) TestConvertIntoVestingAccount() {
	testCases := []struct {
		name               string
		malleate           func()
		from               sdk.AccAddress
		to                 sdk.AccAddress
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
			funder,
			vestingAddr,
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
			funder,
			vestingAddr,
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
			funder,
			vestingAddr,
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
			funder,
			vestingAddr,
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
			funder,
			vestingAddr,
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
				testutil.FundAccount(s.ctx, s.app.BankKeeper, funder, sdk.NewCoins(sdk.NewInt64Coin("NBND", 500))) //nolint:errcheck
			},
			funder,
			vestingAddr,
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
				ethAccount.CodeHash = common.BytesToHash(evmtypes.EmptyCodeHash).Hex()
				s.app.AccountKeeper.SetAccount(s.ctx, ethAccount)
			},
			funder,
			addr3,
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
				ethAccount.CodeHash = common.BytesToHash(evmtypes.EmptyCodeHash).Hex()
				s.app.AccountKeeper.SetAccount(s.ctx, ethAccount)
			},
			funder,
			addr3,
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
			funder,
			vestingAddr,
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
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, vestingAddr, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			funder,
			vestingAddr,
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
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, vestingAddr, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			vestingAddr,
			vestingAddr,
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
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, vestingAddr, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			funder,
			vestingAddr,
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
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, vestingAddr, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			funder,
			vestingAddr,
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
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				testutil.FundAccount(s.ctx, s.app.BankKeeper, vestingAddr, balances) //nolint:errcheck
				s.app.AccountKeeper.SetAccount(s.ctx, clawbackAccount)
			},
			funder,
			vestingAddr,
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
			funder,
			addr5,
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

			err = testutil.FundAccount(s.ctx, s.app.BankKeeper, funder, balances)
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
				suite.ctx.BlockTime().Add(-1*time.Second),
				tc.lockup,
				tc.vesting,
				tc.merge,
				tc.stake,
				valAddr,
			)
			res, err := suite.app.VestingKeeper.ConvertIntoVestingAccount(ctx, msg)

			expResponse := &types.MsgConvertIntoVestingAccountResponse{}
			balanceSource := suite.app.BankKeeper.GetBalance(suite.ctx, tc.from, "aISLM")
			balanceDest := suite.app.BankKeeper.GetBalance(suite.ctx, tc.to, "aISLM")
			balanceBonded := suite.app.StakingKeeper.GetDelegatorBonded(suite.ctx, tc.to)

			if tc.expectPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(expResponse, res)

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
