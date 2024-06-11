package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	"github.com/haqq-network/haqq/tests"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	ethtypes "github.com/haqq-network/haqq/types"
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

func TestMsgCreateClawbackVestingAccount(t *testing.T) {
	var (
		nw  *network.UnitTestNetwork
		ctx sdk.Context
	)

	testCases := []struct {
		name               string
		malleate           func()
		from               sdk.AccAddress
		to                 sdk.AccAddress
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				baseAccount.AccountNumber = 44
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr2, balances)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			addr,
			addr2,
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				baseAccount.AccountNumber = 55
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr2, balances)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			addr2,
			addr2,
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				baseAccount.AccountNumber = 66
				funder := addr
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr2, balances)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			addr,
			addr2,
			lockupPeriods,
			vestingPeriods,
			true,
			1000,
			true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// reset network and context
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()

			tc.malleate()

			err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr, balances)
			require.NoError(t, err)

			msg := types.NewMsgCreateClawbackVestingAccount(
				tc.from,
				tc.to,
				ctx.BlockTime().Add(-1*time.Second),
				tc.lockup,
				tc.vesting,
				tc.merge,
			)
			res, err := nw.App.VestingKeeper.CreateClawbackVestingAccount(ctx, msg)

			expRes := &types.MsgCreateClawbackVestingAccountResponse{}
			balanceSource := nw.App.BankKeeper.GetBalance(ctx, tc.from, "aISLM")
			balanceDest := nw.App.BankKeeper.GetBalance(ctx, tc.to, "aISLM")

			if tc.expectPass {
				require.NoError(t, err)
				require.Equal(t, expRes, res)

				accI := nw.App.AccountKeeper.GetAccount(ctx, tc.to)
				require.NotNil(t, accI)
				require.IsType(t, &types.ClawbackVestingAccount{}, accI)
				require.Equal(t, sdk.NewInt64Coin("aISLM", 0), balanceSource)
				require.Equal(t, sdk.NewInt64Coin("aISLM", 1000+tc.expectExtraBalance), balanceDest)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestMsgClawback(t *testing.T) {
	var (
		nw  *network.UnitTestNetwork
		ctx sdk.Context
	)

	testCases := []struct {
		name         string
		malleate     func()
		funder       sdk.AccAddress
		addr         sdk.AccAddress
		dest         sdk.AccAddress
		startTime    func(time.Time) time.Time
		expectedPass bool
	}{
		{
			"no clawback account",
			func() {},
			addr,
			sdk.AccAddress(tests.GenerateAddress().Bytes()),
			addr3,
			func(t time.Time) time.Time { return t },
			false,
		},
		{
			"wrong account type",
			func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(addr4)
				baseAccount.AccountNumber = 33
				acc, err := sdkvesting.NewBaseVestingAccount(baseAccount, balances, 500000)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, acc)
			},
			addr,
			addr4,
			addr3,
			func(t time.Time) time.Time { return t },
			false,
		},
		{
			"wrong funder",
			func() {},
			addr3,
			addr2,
			addr3,
			func(t time.Time) time.Time { return t },
			false,
		},
		{
			"before start time",
			func() {
			},
			addr,
			addr2,
			addr3,
			func(t time.Time) time.Time { return t.Add(time.Hour) },
			false,
		},
		{
			"pass",
			func() {
			},
			addr,
			addr2,
			addr3,
			func(t time.Time) time.Time { return t },
			true,
		},
		{
			"pass - without dest",
			func() {
			},
			addr,
			addr2,
			sdk.AccAddress([]byte{}),
			func(t time.Time) time.Time { return t },
			true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// reset network and context
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()

			// Set funder
			funder := nw.App.AccountKeeper.NewAccountWithAddress(ctx, tc.funder)
			nw.App.AccountKeeper.SetAccount(ctx, funder)
			err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr, balances)
			require.NoError(t, err)

			// Create Clawback Vesting Account
			createMsg := types.NewMsgCreateClawbackVestingAccount(addr, addr2, tc.startTime(ctx.BlockTime()), lockupPeriods, vestingPeriods, false)
			createRes, err := nw.App.VestingKeeper.CreateClawbackVestingAccount(ctx, createMsg)
			require.NoError(t, err)
			require.NotNil(t, createRes)

			balanceDest := nw.App.BankKeeper.GetBalance(ctx, addr2, "aISLM")
			require.Equal(t, balanceDest, sdk.NewInt64Coin("aISLM", 1000))

			tc.malleate()

			// Perform clawback
			msg := types.NewMsgClawback(tc.funder, tc.addr, tc.dest)
			res, err := nw.App.VestingKeeper.Clawback(ctx, msg)

			expRes := &types.MsgClawbackResponse{}
			balanceDest = nw.App.BankKeeper.GetBalance(ctx, addr2, "aISLM")
			balanceClaw := nw.App.BankKeeper.GetBalance(ctx, tc.dest, "aISLM")
			if len(tc.dest) == 0 {
				balanceClaw = nw.App.BankKeeper.GetBalance(ctx, tc.funder, "aISLM")
			}

			if tc.expectedPass {
				require.NoError(t, err)
				require.Equal(t, expRes, res)
				require.Equal(t, sdk.NewInt64Coin("aISLM", 0), balanceDest)
				require.Equal(t, balances[0], balanceClaw)
			} else {
				require.Error(t, err)
				require.Nil(t, res)
			}
		})
	}
}

func TestMsgUpdateVestingFunder(t *testing.T) {
	var (
		nw  *network.UnitTestNetwork
		ctx sdk.Context
	)

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
				baseAccount.AccountNumber = 33
				acc, err := sdkvesting.NewBaseVestingAccount(baseAccount, balances, 500000)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, acc)
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
		t.Run(tc.name, func(t *testing.T) {
			// reset network and context
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()

			startTime := ctx.BlockTime()

			// Set funder
			funder := nw.App.AccountKeeper.NewAccountWithAddress(ctx, tc.funder)
			nw.App.AccountKeeper.SetAccount(ctx, funder)
			err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr, balances)
			require.NoError(t, err)

			// Create Clawback Vesting Account
			createMsg := types.NewMsgCreateClawbackVestingAccount(addr, addr2, startTime, lockupPeriods, vestingPeriods, false)
			createRes, err := nw.App.VestingKeeper.CreateClawbackVestingAccount(ctx, createMsg)
			require.NoError(t, err)
			require.NotNil(t, createRes)

			balanceDest := nw.App.BankKeeper.GetBalance(ctx, addr2, "aISLM")
			require.Equal(t, balanceDest, sdk.NewInt64Coin("aISLM", 1000))

			tc.malleate()

			// Perform Vesting account update
			msg := types.NewMsgUpdateVestingFunder(tc.funder, tc.newFunder, tc.vestingAcc)
			res, err := nw.App.VestingKeeper.UpdateVestingFunder(ctx, msg)

			expRes := &types.MsgUpdateVestingFunderResponse{}

			if tc.expectedPass {
				require.NoError(t, err)
				// get the updated vesting account
				vestingAcc := nw.App.AccountKeeper.GetAccount(ctx, tc.vestingAcc)
				va, ok := vestingAcc.(*types.ClawbackVestingAccount)
				require.True(t, ok, "vesting account could not be casted to ClawbackVestingAccount")
				require.Equal(t, expRes, res)
				require.Equal(t, va.FunderAddress, tc.newFunder.String())
			} else {
				require.Error(t, err)
				require.Nil(t, res)
			}
		})
	}
}

func TestClawbackVestingAccountStore(t *testing.T) {
	var (
		nw  *network.UnitTestNetwork
		ctx sdk.Context
	)

	nw = network.NewUnitTestNetwork()
	ctx = nw.GetContext()

	// Create and set clawback vesting account
	vestingStart := ctx.BlockTime()
	funder := sdk.AccAddress(types.ModuleName)
	addr := sdk.AccAddress(tests.GenerateAddress().Bytes())
	baseAccount := authtypes.NewBaseAccountWithAddress(addr)
	baseAccount.AccountNumber = 33
	acc := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
	nw.App.AccountKeeper.SetAccount(ctx, acc)

	acc2 := nw.App.AccountKeeper.GetAccount(ctx, acc.GetAddress())
	require.IsType(t, &types.ClawbackVestingAccount{}, acc2)
	require.Equal(t, acc.String(), acc2.String())
}

func TestClawbackVestingAccountMarshal(t *testing.T) {
	var (
		nw  *network.UnitTestNetwork
		ctx sdk.Context
	)

	nw = network.NewUnitTestNetwork()
	ctx = nw.GetContext()

	// Create and set clawback vesting account
	vestingStart := ctx.BlockTime()
	funder := sdk.AccAddress(types.ModuleName)
	addr := sdk.AccAddress(tests.GenerateAddress().Bytes())
	baseAccount := authtypes.NewBaseAccountWithAddress(addr)
	baseAccount.AccountNumber = 33
	acc := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)

	bz, err := acc.Marshal()
	require.NoError(t, err)

	var (
		acc2 *types.ClawbackVestingAccount
		acc3 *types.ClawbackVestingAccount
	)
	acc2 = new(types.ClawbackVestingAccount)
	err = acc2.Unmarshal(bz)
	require.NoError(t, err)
	require.IsType(t, &types.ClawbackVestingAccount{}, acc2)
	require.Equal(t, acc.String(), acc2.String())

	// error on bad bytes
	acc3 = new(types.ClawbackVestingAccount)
	err = acc3.Unmarshal(bz[:len(bz)/2])
	require.Error(t, err)
}

func TestConvertVestingAccount(t *testing.T) {
	var (
		nw  *network.UnitTestNetwork
		ctx sdk.Context
	)

	testCases := []struct {
		name     string
		malleate func(startTime time.Time) sdk.AccountI
		expPass  bool
	}{
		{
			"fail - no account found",
			func(_ time.Time) sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 33, 5)
				return baseAcc
			},
			false,
		},
		{
			"fail - not a vesting account",
			func(_ time.Time) sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 33, 5)
				nw.App.AccountKeeper.SetAccount(ctx, baseAcc)
				return baseAcc
			},
			false,
		},
		{
			"fail - unlocked & unvested",
			func(startTime time.Time) sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 33, 5)
				lockupPeriods := sdkvesting.Periods{{Length: 0, Amount: balances}}
				vestingPeriods := sdkvesting.Periods{
					{Length: 0, Amount: quarter},
					{Length: 2000, Amount: quarter},
					{Length: 2000, Amount: quarter},
					{Length: 2000, Amount: quarter},
				}
				vestingAcc := types.NewClawbackVestingAccount(baseAcc, from, balances, startTime, lockupPeriods, vestingPeriods, nil)
				nw.App.AccountKeeper.SetAccount(ctx, vestingAcc)
				return vestingAcc
			},
			false,
		},
		{
			"fail - locked & vested",
			func(startTime time.Time) sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				vestingPeriods := sdkvesting.Periods{{Length: 0, Amount: balances}}
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 33, 5)
				vestingAcc := types.NewClawbackVestingAccount(baseAcc, from, balances, startTime, lockupPeriods, vestingPeriods, nil)
				nw.App.AccountKeeper.SetAccount(ctx, vestingAcc)
				return vestingAcc
			},
			false,
		},
		{
			"fail - locked & unvested",
			func(startTime time.Time) sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 33, 5)
				vestingAcc := types.NewClawbackVestingAccount(baseAcc, from, balances, ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				nw.App.AccountKeeper.SetAccount(ctx, vestingAcc)
				return vestingAcc
			},
			false,
		},
		{
			"success - unlocked & vested convert to base account",
			func(startTime time.Time) sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 33, 5)
				vestingPeriods := sdkvesting.Periods{{Length: 0, Amount: balances}}
				vestingAcc := types.NewClawbackVestingAccount(baseAcc, from, balances, startTime, nil, vestingPeriods, nil)
				nw.App.AccountKeeper.SetAccount(ctx, vestingAcc)
				return vestingAcc
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()
			startTime := ctx.BlockTime().Add(-5 * time.Second)

			acc := tc.malleate(startTime)

			msg := types.NewMsgConvertVestingAccount(acc.GetAddress())
			res, err := nw.App.VestingKeeper.ConvertVestingAccount(ctx, msg)

			if tc.expPass {
				require.NoError(t, err)
				require.NotNil(t, res)

				account := nw.App.AccountKeeper.GetAccount(ctx, acc.GetAddress())

				_, ok := account.(vestingexported.VestingAccount)
				require.False(t, ok)

				_, ok = account.(ethtypes.EthAccountI)
				require.True(t, ok)
			} else {
				require.Error(t, err)
				require.Nil(t, res)
			}
		})
	}
}

func TestConvertIntoVestingAccount(t *testing.T) {
	var (
		nw  *network.UnitTestNetwork
		ctx sdk.Context
	)

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
			addr,
			addr2,
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
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr, sdk.NewCoins(sdk.NewInt64Coin("NBND", 500)))
				require.NoError(t, err)
			},
			addr,
			addr2,
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
				baseAccount.AccountNumber = 33
				ethAccount := ethtypes.ProtoAccount().(*ethtypes.EthAccount)
				ethAccount.BaseAccount = baseAccount
				nw.App.AccountKeeper.SetAccount(ctx, ethAccount)
			},
			addr,
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
				baseAccount.AccountNumber = 44
				ethAccount := ethtypes.ProtoAccount().(*ethtypes.EthAccount)
				ethAccount.BaseAccount = baseAccount
				nw.App.AccountKeeper.SetAccount(ctx, ethAccount)
			},
			addr,
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
			addr,
			addr2,
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				baseAccount.AccountNumber = 44
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr2, balances)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			addr,
			addr2,
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				baseAccount.AccountNumber = 55
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr2, balances)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			addr2,
			addr2,
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				baseAccount.AccountNumber = 66
				funder := addr
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr2, balances)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			addr,
			addr2,
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				baseAccount.AccountNumber = 77
				funder := addr
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr2, balances)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			addr,
			addr2,
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(addr2)
				baseAccount.AccountNumber = 88
				funder := addr
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr2, balances)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
			},
			addr,
			addr2,
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
				baseAccount.AccountNumber = 99
				nw.App.AccountKeeper.SetAccount(ctx, authtypes.NewModuleAccount(baseAccount, "testmodule"))
			},
			addr,
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
		t.Run(tc.name, func(t *testing.T) {
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()

			tc.malleate()

			err := testutil.FundAccount(ctx, nw.App.BankKeeper, addr, balances)
			require.NoError(t, err)

			valAddr := sdk.ValAddress{}
			expBalanceBonded := math.ZeroInt()
			if tc.stake {
				vals, err := nw.App.StakingKeeper.GetAllValidators(ctx)
				require.NoError(t, err)
				require.Greater(t, len(vals), 0)
				valopAddress := vals[0].OperatorAddress
				valAddr, err = sdk.ValAddressFromBech32(valopAddress)
				require.NoError(t, err)
				expBalanceBonded = math.NewIntFromUint64(tc.expectDelegation)
			}

			msg := types.NewMsgConvertIntoVestingAccount(
				tc.from,
				tc.to,
				ctx.BlockTime().Add(-1*time.Second),
				tc.lockup,
				tc.vesting,
				tc.merge,
				tc.stake,
				valAddr,
			)
			res, err := nw.App.VestingKeeper.ConvertIntoVestingAccount(ctx, msg)

			expRes := &types.MsgConvertIntoVestingAccountResponse{}
			balanceSource := nw.App.BankKeeper.GetBalance(ctx, tc.from, "aISLM")
			balanceDest := nw.App.BankKeeper.GetBalance(ctx, tc.to, "aISLM")
			balanceBonded, err2 := nw.App.StakingKeeper.GetDelegatorBonded(ctx, tc.to)
			require.NoError(t, err2)

			if tc.expectPass {
				require.NoError(t, err)
				require.Equal(t, expRes, res)

				accI := nw.App.AccountKeeper.GetAccount(ctx, tc.to)
				require.NotNil(t, accI)
				require.IsType(t, &types.ClawbackVestingAccount{}, accI)
				require.Equal(t, sdk.NewInt64Coin("aISLM", 0), balanceSource)
				require.Equal(t, sdk.NewInt64Coin("aISLM", 1000+tc.expectExtraBalance), balanceDest)
				if tc.stake {
					require.True(t, balanceBonded.GT(math.ZeroInt()))
				}
				require.True(t, expBalanceBonded.Equal(balanceBonded))
			} else {
				require.Error(t, err)
			}
		})
	}
}
