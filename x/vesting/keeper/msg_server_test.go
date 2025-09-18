package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	ethtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	"github.com/haqq-network/haqq/x/vesting/types"
)

var (
	vestAmount      = int64(1000)
	balances        = sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, vestAmount))
	delegationCoins = sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1e18))
	quarter         = sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 250))
	addr3           = sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	addr4           = sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	funder          = sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	vestingAddr     = sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	lockupPeriods   = sdkvesting.Periods{{Length: 5000, Amount: balances}}
	vestingPeriods  = sdkvesting.Periods{
		{Length: 2000, Amount: quarter},
		{Length: 2000, Amount: quarter},
		{Length: 2000, Amount: quarter},
		{Length: 2000, Amount: quarter},
	}
)

func TestMsgCreateClawbackVestingAccount(t *testing.T) {
	var (
		ctx sdk.Context
		nw  *network.UnitTestNetwork
	)

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
				vestingStart := ctx.BlockTime()
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				clawbackAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, vestingAddr, balances)
				require.NoError(t, err)
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
				vestingStart := ctx.BlockTime()
				funder := sdk.AccAddress(types.ModuleName)
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				clawbackAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, vestingAddr, balances)
				require.NoError(t, err)
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				clawbackAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, vestingAddr, balances)
				require.NoError(t, err)
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
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()

			tc.malleate()

			err := testutil.FundAccount(ctx, nw.App.BankKeeper, tc.funder, balances)
			require.NoError(t, err, "failed to fund funder account")

			msg := types.NewMsgCreateClawbackVestingAccount(
				tc.funder,
				tc.vestingAddr,
				ctx.BlockTime().Add(-1*time.Second),
				tc.lockup,
				tc.vesting,
				tc.merge,
			)
			res, err := nw.App.VestingKeeper.CreateClawbackVestingAccount(ctx, msg)

			expRes := &types.MsgCreateClawbackVestingAccountResponse{}
			balanceFunder := nw.App.BankKeeper.GetBalance(ctx, tc.funder, utils.BaseDenom)
			balanceVestingAddr := nw.App.BankKeeper.GetBalance(ctx, tc.vestingAddr, utils.BaseDenom)
			spendableBalanceVestingAddr := nw.App.BankKeeper.SpendableCoin(ctx, tc.vestingAddr, utils.BaseDenom)

			if tc.expPass {
				require.NoError(t, err, tc.name)
				require.Equal(t, expRes, res)

				accI := nw.App.AccountKeeper.GetAccount(ctx, tc.vestingAddr)
				require.NotNil(t, accI)
				vestAcc, ok := accI.(*types.ClawbackVestingAccount)
				require.True(t, ok)

				require.Equal(t, sdk.NewInt64Coin(utils.BaseDenom, 0), balanceFunder)
				require.Equal(t, sdk.NewInt64Coin(utils.BaseDenom, vestAmount+tc.expectExtraBalance), balanceVestingAddr)
				// require.Equal(t, tc.expDelegatedFree, vestAcc.DelegatedFree) // TODO cover it
				require.Empty(t, vestAcc.DelegatedVesting)
				require.True(t, spendableBalanceVestingAddr.Amount.IsZero())
			} else {
				require.Error(t, err, tc.name)
				require.ErrorContains(t, err, tc.errContains)
			}
		})
	}
}

func TestMsgClawback(t *testing.T) {
	var (
		ctx sdk.Context
		nw  *network.UnitTestNetwork
	)
	now := time.Now()
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
			name: "fail - account does not exist",
			malleate: func() {
				vestingAddr = sdk.AccAddress(utiltx.GenerateAddress().Bytes())
			},
			funder:      funder,
			vestingAddr: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			startTime:   now,
			expPass:     false,
			errContains: "does not exist",
		},
		{
			name:         "fail - no clawback account",
			malleate:     func() {},
			funder:       funder,
			vestingAddr:  vestingAddr,
			clawbackDest: addr3,
			startTime:    now,
			expPass:      false,
			errContains:  types.ErrNotSubjectToClawback.Error(),
		},
		{
			name: "fail - wrong account type",
			malleate: func() {
				// create a base vesting account instead of a clawback vesting account at the vesting address
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				baseAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				acc, err := sdkvesting.NewBaseVestingAccount(baseAccount, balances, 500000)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, acc)
			},
			funder:       funder,
			vestingAddr:  vestingAddr,
			clawbackDest: addr3,
			startTime:    now,
			expPass:      false,
			errContains:  types.ErrNotSubjectToClawback.Error(),
		},
		{
			name:         "fail - wrong funder",
			malleate:     func() {},
			funder:       addr3,
			vestingAddr:  vestingAddr,
			clawbackDest: addr3,
			startTime:    now,
			expPass:      false,
			errContains:  "clawback can only be requested by original funder",
		},
		{
			name:         "fail - clawback destination is blocked",
			malleate:     func() {},
			funder:       funder,
			vestingAddr:  vestingAddr,
			clawbackDest: authtypes.NewModuleAddress("transfer"),
			startTime:    now,
			expPass:      false,
			errContains:  "is a blocked address and not allowed to receive funds",
		},
		{
			name:        "pass - before start time",
			malleate:    func() {},
			funder:      funder,
			vestingAddr: vestingAddr,
			startTime:   now.Add(time.Hour),
			expPass:     true,
		},
		{
			name:         "pass - with clawback destination",
			malleate:     func() {},
			funder:       funder,
			vestingAddr:  vestingAddr,
			clawbackDest: addr3,
			startTime:    now,
			expPass:      true,
		},
		{
			name:        "pass - without clawback destination",
			malleate:    func() {},
			funder:      funder,
			vestingAddr: vestingAddr,
			startTime:   now,
			expPass:     true,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			// reset
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()
			vestingAddr = tc.vestingAddr

			// Initiate and fund the funder account
			err := testutil.FundAccount(ctx, nw.App.BankKeeper, funder, balances)
			require.NoError(t, err, "failed to fund funder account")

			// Create Clawback Vesting Account
			createMsg := types.NewMsgCreateClawbackVestingAccount(funder, vestingAddr, tc.startTime, lockupPeriods, vestingPeriods, false)
			createRes, err := nw.App.VestingKeeper.CreateClawbackVestingAccount(ctx, createMsg)
			require.NoError(t, err)
			require.NotNil(t, createRes)

			balanceVestingAcc := nw.App.BankKeeper.GetBalance(ctx, vestingAddr, utils.BaseDenom)
			require.Equal(t, balanceVestingAcc, sdk.NewInt64Coin(utils.BaseDenom, 1000))

			tc.malleate()

			// Perform clawback
			msg := types.NewMsgClawback(tc.funder, vestingAddr, tc.clawbackDest)
			res, err := nw.App.VestingKeeper.Clawback(ctx, msg)

			balanceVestingAcc = nw.App.BankKeeper.GetBalance(ctx, vestingAddr, utils.BaseDenom)
			balanceClaw := nw.App.BankKeeper.GetBalance(ctx, tc.clawbackDest, utils.BaseDenom)
			if len(tc.clawbackDest) == 0 {
				balanceClaw = nw.App.BankKeeper.GetBalance(ctx, tc.funder, utils.BaseDenom)
			}

			if tc.expPass {
				require.NoError(t, err)

				expRes := &types.MsgClawbackResponse{}
				require.Equal(t, expRes, res, "expected full balances to be clawed back")
				require.Equal(t, sdk.NewInt64Coin(utils.BaseDenom, 0), balanceVestingAcc)
				require.Equal(t, balances[0], balanceClaw)
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errContains)
				require.Nil(t, res)
			}
		})
	}
}

func TestMsgUpdateVestingFunder(t *testing.T) {
	var (
		ctx sdk.Context
		nw  *network.UnitTestNetwork
	)
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
			vestingAcc:  sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			newFunder:   newFunder,
			expPass:     false,
			errContains: "does not exist",
		},
		{
			name: "fail - wrong account type",
			malleate: func() {
				baseAccount := authtypes.NewBaseAccountWithAddress(addr4)
				baseAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				acc, err := sdkvesting.NewBaseVestingAccount(baseAccount, balances, 500000)
				require.NoError(t, err)
				nw.App.AccountKeeper.SetAccount(ctx, acc)
			},
			funder:      funder,
			vestingAcc:  vestingAddr,
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
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			// reset
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()
			startTime := ctx.BlockTime()

			// Set funder
			err := testutil.FundAccount(ctx, nw.App.BankKeeper, funder, balances)
			require.NoError(t, err)

			// Create Clawback Vesting Account
			createMsg := types.NewMsgCreateClawbackVestingAccount(funder, vestingAddr, startTime, lockupPeriods, vestingPeriods, false)
			createRes, err := nw.App.VestingKeeper.CreateClawbackVestingAccount(ctx, createMsg)
			require.NoError(t, err)
			require.NotNil(t, createRes)

			balanceVestingAcc := nw.App.BankKeeper.GetBalance(ctx, vestingAddr, utils.BaseDenom)
			require.Equal(t, balanceVestingAcc, sdk.NewInt64Coin(utils.BaseDenom, 1000))

			tc.malleate()

			// Perform Vesting account update
			msg := types.NewMsgUpdateVestingFunder(tc.funder, tc.newFunder, tc.vestingAcc)
			res, err := nw.App.VestingKeeper.UpdateVestingFunder(ctx, msg)

			expRes := &types.MsgUpdateVestingFunderResponse{}

			if tc.expPass {
				// get the updated vesting account
				vestingAcc := nw.App.AccountKeeper.GetAccount(ctx, tc.vestingAcc)
				va, ok := vestingAcc.(*types.ClawbackVestingAccount)
				require.True(t, ok, "vesting account could not be casted to ClawbackVestingAccount")

				require.NoError(t, err)
				require.Equal(t, expRes, res)
				require.Equal(t, va.FunderAddress, tc.newFunder.String())
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errContains)
				require.Nil(t, res)
			}
		})
	}
}

func TestClawbackVestingAccountStore(t *testing.T) {
	nw := network.NewUnitTestNetwork()
	ctx := nw.GetContext()

	// Create and set clawback vesting account
	vestingStart := ctx.BlockTime()
	funder := sdk.AccAddress(types.ModuleName)
	addr := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	baseAccount := authtypes.NewBaseAccountWithAddress(addr)
	baseAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
	acc := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
	nw.App.AccountKeeper.SetAccount(ctx, acc)

	acc2 := nw.App.AccountKeeper.GetAccount(ctx, acc.GetAddress())
	require.IsType(t, &types.ClawbackVestingAccount{}, acc2)
	require.Equal(t, acc.String(), acc2.String())
}

func TestConvertVestingAccount(t *testing.T) {
	var (
		ctx sdk.Context
		nw  *network.UnitTestNetwork
	)
	now := time.Now()
	startTime := now.Add(-5 * time.Second)
	testCases := []struct {
		name     string
		malleate func() sdk.AccountI
		expPass  bool
	}{
		{
			"fail - no account found",
			func() sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				return baseAcc
			},
			false,
		},
		{
			"fail - not a vesting account",
			func() sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				baseAcc.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				nw.App.AccountKeeper.SetAccount(ctx, baseAcc)
				return baseAcc
			},
			false,
		},
		{
			"fail - unlocked & unvested",
			func() sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				baseAcc.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
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
			func() sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				vestingPeriods := sdkvesting.Periods{{Length: 0, Amount: balances}}
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				baseAcc.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				vestingAcc := types.NewClawbackVestingAccount(baseAcc, from, balances, startTime, lockupPeriods, vestingPeriods, nil)
				nw.App.AccountKeeper.SetAccount(ctx, vestingAcc)
				return vestingAcc
			},
			false,
		},
		{
			"fail - locked & unvested",
			func() sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				baseAcc.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				vestingAcc := types.NewClawbackVestingAccount(baseAcc, from, balances, ctx.BlockTime(), lockupPeriods, vestingPeriods, nil)
				nw.App.AccountKeeper.SetAccount(ctx, vestingAcc)
				return vestingAcc
			},
			false,
		},
		{
			"success - unlocked & vested convert to base account",
			func() sdk.AccountI {
				from, priv := utiltx.NewAccAddressAndKey()
				baseAcc := authtypes.NewBaseAccount(from, priv.PubKey(), 1, 5)
				baseAcc.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				vestingPeriods := sdkvesting.Periods{{Length: 0, Amount: balances}}
				vestingAcc := types.NewClawbackVestingAccount(baseAcc, from, balances, startTime, nil, vestingPeriods, nil)
				nw.App.AccountKeeper.SetAccount(ctx, vestingAcc)
				return vestingAcc
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()

			acc := tc.malleate()

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
		ctx sdk.Context
		nw  *network.UnitTestNetwork
	)

	testCases := []struct {
		name               string
		malleate           func()
		funder             sdk.AccAddress
		vestingAddr        sdk.AccAddress
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
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, funder, sdk.NewCoins(sdk.NewInt64Coin("NBND", 500)))
				require.NoError(t, err)
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
				ethAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				ethAccount.CodeHash = common.BytesToHash(evmtypes.EmptyCodeHash).Hex()
				nw.App.AccountKeeper.SetAccount(ctx, ethAccount)
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
				ethAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				ethAccount.CodeHash = common.BytesToHash(evmtypes.EmptyCodeHash).Hex()
				nw.App.AccountKeeper.SetAccount(ctx, ethAccount)
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				clawbackAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, vestingAddr, balances)
				require.NoError(t, err)
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				funder := sdk.AccAddress(types.ModuleName)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				clawbackAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, vestingAddr, balances)
				require.NoError(t, err)
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				clawbackAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, vestingAddr, balances)
				require.NoError(t, err)
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				clawbackAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, vestingAddr, balances)
				require.NoError(t, err)
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
				vestingStart := ctx.BlockTime()
				baseAccount := authtypes.NewBaseAccountWithAddress(vestingAddr)
				clawbackAccount := types.NewClawbackVestingAccount(baseAccount, funder, balances, vestingStart, lockupPeriods, vestingPeriods, nil)
				clawbackAccount.AccountNumber = nw.App.AccountKeeper.NextAccountNumber(ctx)
				nw.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
				err := testutil.FundAccount(ctx, nw.App.BankKeeper, vestingAddr, balances)
				require.NoError(t, err)
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
				baseAccount := authtypes.NewBaseAccountWithAddress(addr4)
				nw.App.AccountKeeper.SetAccount(ctx, authtypes.NewModuleAccount(baseAccount, "testmodule"))
			},
			funder,
			addr4,
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
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			// reset
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()

			tc.malleate()

			err := testutil.FundAccount(ctx, nw.App.BankKeeper, funder, balances)
			require.NoError(t, err)

			valAddr := sdk.ValAddress{}
			expBalanceBonded := sdkmath.ZeroInt()
			if tc.stake {
				vals, err := nw.App.StakingKeeper.GetAllValidators(ctx)
				require.NoError(t, err)
				require.Greater(t, len(vals), 0)
				valopAddress := vals[0].OperatorAddress
				valAddr, err = sdk.ValAddressFromBech32(valopAddress)
				require.NoError(t, err)
				expBalanceBonded = sdkmath.NewIntFromUint64(tc.expectDelegation)
			}

			msg := types.NewMsgConvertIntoVestingAccount(
				tc.funder,
				tc.vestingAddr,
				ctx.BlockTime().Add(-1*time.Second),
				tc.lockup,
				tc.vesting,
				tc.merge,
				tc.stake,
				valAddr,
			)
			res, err := nw.App.VestingKeeper.ConvertIntoVestingAccount(ctx, msg)
			require.NoError(t, err)

			expResponse := &types.MsgConvertIntoVestingAccountResponse{}
			balanceSource := nw.App.BankKeeper.GetBalance(ctx, tc.funder, utils.BaseDenom)
			balanceDest := nw.App.BankKeeper.GetBalance(ctx, tc.vestingAddr, utils.BaseDenom)
			balanceBonded, err := nw.App.StakingKeeper.GetDelegatorBonded(ctx, tc.vestingAddr)
			require.NoError(t, err)

			if tc.expectPass {
				require.NoError(t, err, tc.name)
				require.Equal(t, expResponse, res)

				accI := nw.App.AccountKeeper.GetAccount(ctx, tc.vestingAddr)
				require.NotNil(t, accI)
				require.IsType(t, &types.ClawbackVestingAccount{}, accI)
				require.Equal(t, sdk.NewInt64Coin(utils.BaseDenom, 0), balanceSource)
				require.Equal(t, sdk.NewInt64Coin(utils.BaseDenom, 1000+tc.expectExtraBalance), balanceDest)
				if tc.stake {
					require.True(t, balanceBonded.GT(sdkmath.ZeroInt()))
				}
				require.True(t, expBalanceBonded.Equal(balanceBonded))
			} else {
				require.Error(t, err, tc.name)
			}
		})
	}
}
