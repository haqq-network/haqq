package v196_test

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"slices"
	"sort"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/haqq-network/haqq/app"
	v196 "github.com/haqq-network/haqq/app/upgrades/v1.9.6"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	ethtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// Real mainnet accounts from HackerAccounts, each set up to cover one path.
const (
	// operates a validator (Dexedrine on mainnet)
	hackerOperatorBech = "haqq1vxz7aufkv685l3wgdspgfp5wd83jje29gwltte"
	// turned into a staked vesting account by a third party
	hackerVestingBech = "haqq1xe6ddj7wlutkvjh7lf090k0jp7ytkndlv5x6l3"
	// no account at all, only a ucDAO escrow (three such accounts on mainnet)
	hackerEscrowOnlyBech = "haqq1qfwsr7gz0nx7tr8fjwkn0eshndz8kgfvlgzmph"
	// balance, ucDAO escrow, delegation, unbonding, grants
	hackerPlainBech = "haqq10kewh3awzg4k9cg689h92nykfpzwyuszl5q6mw"
)

type scenario struct {
	nw *network.UnitTestNetwork

	hackerOperator, hackerVesting, hackerEscrowOnly, hackerPlain sdk.AccAddress

	// innocent delegates to the hacker's validator and is the hacker's withdraw
	// address before the upgrade; nothing of theirs may change.
	innocent sdk.AccAddress
	// funder is a third party that funds and grants to the hacker accounts.
	funder sdk.AccAddress

	valHacker, valOther sdk.ValAddress
}

func islm(n int64) sdk.Coin {
	return sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(n).Mul(sdkmath.NewInt(1e18)))
}

func haqqCoin(n int64) sdk.Coin {
	return sdk.NewCoin(ethiqtypes.BaseDenom, sdkmath.NewInt(n).Mul(sdkmath.NewInt(1e18)))
}

func keepersOf(a *app.Haqq) v196.Keepers {
	return v196.Keepers{
		AccountKeeper:  a.AccountKeeper,
		BankKeeper:     a.BankKeeper,
		StakingKeeper:  a.StakingKeeper,
		DistrKeeper:    a.DistrKeeper,
		AuthzKeeper:    a.AuthzKeeper,
		FeeGrantKeeper: a.FeeGrantKeeper,
		DaoKeeper:      a.DaoKeeper,
	}
}

func setupScenario(t *testing.T) scenario {
	t.Helper()

	kr := keyring.New(3)
	s := scenario{
		hackerOperator:   sdk.MustAccAddressFromBech32(hackerOperatorBech),
		hackerVesting:    sdk.MustAccAddressFromBech32(hackerVestingBech),
		hackerEscrowOnly: sdk.MustAccAddressFromBech32(hackerEscrowOnlyBech),
		hackerPlain:      sdk.MustAccAddressFromBech32(hackerPlainBech),
		innocent:         kr.GetAccAddr(0),
		funder:           kr.GetAccAddr(1),
	}
	otherOperator := kr.GetAccAddr(2)
	s.valHacker = sdk.ValAddress(s.hackerOperator)
	s.valOther = sdk.ValAddress(otherOperator)

	s.nw = network.NewUnitTestNetwork(
		network.WithAmountOfValidators(2),
		network.WithValidatorOperators([]sdk.AccAddress{s.hackerOperator, otherOperator}),
		network.WithPreFundedAccounts(s.innocent, s.funder),
		network.WithOtherDenoms([]string{ethiqtypes.BaseDenom}),
	)
	ctx := s.nw.GetContext()
	a := s.nw.App
	bk, sk, dk := a.BankKeeper, a.StakingKeeper, a.DistrKeeper

	send := func(to sdk.AccAddress, coins ...sdk.Coin) {
		require.NoError(t, bk.SendCoins(ctx, s.funder, to, sdk.NewCoins(coins...)))
	}
	delegate := func(del sdk.AccAddress, val sdk.ValAddress, amt sdk.Coin) {
		validator, err := sk.GetValidator(ctx, val)
		require.NoError(t, err)
		_, err = sk.Delegate(ctx, del, amt.Amount, stakingtypes.Unbonded, validator, true)
		require.NoError(t, err)
	}

	send(s.hackerOperator, islm(10))
	send(s.hackerPlain, islm(1000), haqqCoin(50))
	send(s.hackerVesting, islm(1)) // an existing EthAccount, as on mainnet

	// The hacker's validator: own stake, an innocent delegator, aHAQQ rewards and
	// commission, and a withdraw address pointing at somebody else.
	delegate(s.hackerOperator, s.valHacker, islm(5))
	delegate(s.innocent, s.valHacker, islm(5))
	rewards := sdk.NewCoins(haqqCoin(1000))
	require.NoError(t, bk.MintCoins(ctx, ethiqtypes.ModuleName, rewards))
	require.NoError(t, bk.SendCoinsFromModuleToModule(ctx, ethiqtypes.ModuleName, distrtypes.ModuleName, rewards))
	validator, err := sk.GetValidator(ctx, s.valHacker)
	require.NoError(t, err)
	require.NoError(t, dk.AllocateTokensToValidator(ctx, validator, sdk.NewDecCoinsFromCoins(rewards...)))
	require.NoError(t, dk.SetDelegatorWithdrawAddr(ctx, s.hackerOperator, s.innocent))

	// A third party turns a frozen account into a clawback vesting account, stakes
	// the vested part and locks more on top (MsgConvertIntoVestingAccount needs
	// no signature from the target).
	// Started a day ago: the first grant is vested (so it can be staked) but
	// locked for a year.
	year := int64(365 * 24 * 60 * 60)
	start := ctx.BlockTime().Add(-24 * time.Hour)
	_, err = a.VestingKeeper.ConvertIntoVestingAccount(ctx, &vestingtypes.MsgConvertIntoVestingAccount{
		FromAddress:      s.funder.String(),
		ToAddress:        s.hackerVesting.String(),
		StartTime:        start,
		LockupPeriods:    sdkvesting.Periods{{Length: year, Amount: sdk.NewCoins(islm(100))}},
		VestingPeriods:   sdkvesting.Periods{{Length: 1, Amount: sdk.NewCoins(islm(100))}},
		Merge:            true,
		Stake:            true,
		ValidatorAddress: s.valOther.String(),
	})
	require.NoError(t, err)
	_, err = a.VestingKeeper.ConvertIntoVestingAccount(ctx, &vestingtypes.MsgConvertIntoVestingAccount{
		FromAddress:    s.funder.String(),
		ToAddress:      s.hackerVesting.String(),
		StartTime:      start,
		LockupPeriods:  sdkvesting.Periods{{Length: year, Amount: sdk.NewCoins(islm(40))}},
		VestingPeriods: sdkvesting.Periods{{Length: year, Amount: sdk.NewCoins(islm(40))}},
		Merge:          true,
	})
	require.NoError(t, err)
	_, isVesting := a.AccountKeeper.GetAccount(ctx, s.hackerVesting).(*vestingtypes.ClawbackVestingAccount)
	require.True(t, isVesting)

	// ucDAO: a tracked Fund, untracked direct transfers to the escrow (the
	// counter drift) and an ownership transfer to an account that does not exist.
	require.NoError(t, a.DaoKeeper.Fund(ctx, sdk.NewCoins(islm(300)), s.hackerPlain))
	send(ucdaotypes.GetEscrowAddress(s.hackerPlain), islm(7), haqqCoin(3))
	_, err = a.DaoKeeper.TransferOwnership(ctx, s.hackerPlain, s.hackerEscrowOnly, sdk.NewCoins(islm(100)))
	require.NoError(t, err)
	require.Nil(t, a.AccountKeeper.GetAccount(ctx, s.hackerEscrowOnly))

	// A delegation and an unbonding in progress.
	delegate(s.hackerPlain, s.valOther, islm(20))
	valOther, err := sk.GetValidator(ctx, s.valOther)
	require.NoError(t, err)
	shares, err := valOther.SharesFromTokens(islm(5).Amount)
	require.NoError(t, err)
	_, _, err = sk.Undelegate(ctx, s.hackerPlain, s.valOther, shares)
	require.NoError(t, err)

	// Grants in both directions, plus unrelated ones that must survive.
	expiration := ctx.BlockTime().Add(time.Hour)
	ak := a.AuthzKeeper
	require.NoError(t, ak.SaveGrant(ctx, s.innocent, s.hackerPlain, &ucdaotypes.TransferOwnershipAuthorization{}, &expiration))
	require.NoError(t, ak.SaveGrant(ctx, s.hackerOperator, s.innocent, authz.NewGenericAuthorization(sdk.MsgTypeURL(&banktypes.MsgSend{})), nil))
	require.NoError(t, ak.SaveGrant(ctx, s.funder, s.innocent, authz.NewGenericAuthorization(sdk.MsgTypeURL(&banktypes.MsgSend{})), nil))

	fk := a.FeeGrantKeeper
	require.NoError(t, fk.GrantAllowance(ctx, s.hackerPlain, s.innocent, &feegrant.BasicAllowance{}))
	require.NoError(t, fk.GrantAllowance(ctx, s.funder, s.hackerVesting, &feegrant.BasicAllowance{}))
	require.NoError(t, fk.GrantAllowance(ctx, s.funder, s.innocent, &feegrant.BasicAllowance{}))

	return s
}

// hackers returns the scenario's frozen accounts.
func (s scenario) hackers() []sdk.AccAddress {
	return []sdk.AccAddress{s.hackerOperator, s.hackerVesting, s.hackerEscrowOnly, s.hackerPlain}
}

// requireCleared checks that nothing burnable is left on the scenario accounts
// and that nothing else was touched.
func (s scenario) requireCleared(t *testing.T, ctx sdk.Context, innocentBefore sdk.Coins, accNum, seq uint64) {
	t.Helper()
	a := s.nw.App

	for _, addr := range s.hackers() {
		for _, denom := range v196.BurnDenoms {
			require.True(t, a.BankKeeper.GetBalance(ctx, addr, denom).IsZero(), "%s %s", addr, denom)
			require.True(t, a.BankKeeper.GetBalance(ctx, ucdaotypes.GetEscrowAddress(addr), denom).IsZero(), "%s escrow %s", addr, denom)
		}
		dels, err := a.StakingKeeper.GetAllDelegatorDelegations(ctx, addr)
		require.NoError(t, err)
		require.Empty(t, dels, addr)
		ubds, err := a.StakingKeeper.GetAllUnbondingDelegations(ctx, addr)
		require.NoError(t, err)
		require.Empty(t, ubds, addr)
		withdrawAddr, err := a.DistrKeeper.GetDelegatorWithdrawAddr(ctx, addr)
		require.NoError(t, err)
		require.Equal(t, addr, withdrawAddr)
		require.False(t, a.DaoKeeper.IsHolder(ctx, addr), addr)
	}

	// The vesting account is a plain EOA again, with number and sequence kept.
	acc, ok := a.AccountKeeper.GetAccount(ctx, s.hackerVesting).(*ethtypes.EthAccount)
	require.True(t, ok)
	require.Equal(t, accNum, acc.GetAccountNumber())
	require.Equal(t, seq, acc.GetSequence())
	require.Equal(t, ethtypes.AccountTypeEOA, acc.Type())

	// The innocent withdraw address received none of the hacker's rewards, and
	// the innocent delegation on the hacker's validator is untouched.
	require.Equal(t, innocentBefore, a.BankKeeper.GetAllBalances(ctx, s.innocent))
	del, err := a.StakingKeeper.GetDelegation(ctx, s.innocent, s.valHacker)
	require.NoError(t, err)
	require.True(t, del.Shares.IsPositive())

	// Grants and allowances of the hacker accounts are gone, unrelated ones stay.
	var grants [][2]string
	a.AuthzKeeper.IterateGrants(ctx, func(granter, grantee sdk.AccAddress, _ authz.Grant) bool {
		grants = append(grants, [2]string{granter.String(), grantee.String()})
		return false
	})
	require.Equal(t, [][2]string{{s.innocent.String(), s.funder.String()}}, grants)

	var allowances [][2]string
	require.NoError(t, a.FeeGrantKeeper.IterateAllFeeAllowances(ctx, func(g feegrant.Grant) bool {
		allowances = append(allowances, [2]string{g.Granter, g.Grantee})
		return false
	}))
	require.Equal(t, [][2]string{{s.funder.String(), s.innocent.String()}}, allowances)
}

func TestHackerAccounts(t *testing.T) {
	require.Len(t, v196.HackerAccounts, 50)
	require.True(t, sort.StringsAreSorted(v196.HackerAccounts), "HackerAccounts must stay in sort.Strings order")

	nw := network.NewUnitTestNetwork()
	blocked := nw.App.BlockedAddrs()
	for i, bech := range v196.HackerAccounts {
		if i > 0 {
			require.NotEqual(t, v196.HackerAccounts[i-1], bech, "duplicate")
		}
		addr, err := sdk.AccAddressFromBech32(bech)
		require.NoError(t, err, bech)
		require.Equal(t, bech, addr.String(), "non-canonical bech32")
		// Rewards and commission are paid out to the account itself; bank refuses
		// to pay a blocked address and would fail the upgrade.
		require.False(t, blocked[bech], "%s is a bank-blocked address", bech)
	}

	for _, bech := range []string{hackerOperatorBech, hackerVestingBech, hackerEscrowOnlyBech, hackerPlainBech} {
		require.True(t, slices.Contains(v196.HackerAccounts, bech), bech)
	}
}

func TestBurnHackerFunds(t *testing.T) {
	s := setupScenario(t)
	ctx := s.nw.GetContext()
	a := s.nw.App

	vestingAcc := a.AccountKeeper.GetAccount(ctx, s.hackerVesting)
	innocentBefore := a.BankKeeper.GetAllBalances(ctx, s.innocent)
	supplyBefore := sdk.NewCoins(a.BankKeeper.GetSupply(ctx, utils.BaseDenom), a.BankKeeper.GetSupply(ctx, ethiqtypes.BaseDenom))
	feePoolBefore, err := a.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	ethiqBurnedBefore := a.EthiqKeeper.GetTotalBurnedAmount(ctx)
	require.Equal(t, islm(300).Amount, a.DaoKeeper.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount)

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, v196.BurnHackerFunds(ctx, keepersOf(a), v196.HackerAccounts))

	s.requireCleared(t, ctx, innocentBefore, vestingAcc.GetAccountNumber(), vestingAcc.GetSequence())

	// The counter tracked 300, the escrows held 307: the subtraction saturates.
	require.True(t, a.DaoKeeper.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount.IsZero())

	// Burned, not moved: supply dropped by at least everything placed on the
	// hacker accounts, and neither the community pool nor ethiq's burn counter
	// moved. The handler itself checks the drop is exact.
	supplyAfter := sdk.NewCoins(a.BankKeeper.GetSupply(ctx, utils.BaseDenom), a.BankKeeper.GetSupply(ctx, ethiqtypes.BaseDenom))
	burned := supplyBefore.Sub(supplyAfter...)
	// 10 + 1000 + 1 + 100 + 40 + 7 ISLM put on the accounts
	require.True(t, burned.AmountOf(utils.BaseDenom).GTE(islm(1158).Amount), burned.String())
	// 50 + 3 HAQQ on balance and escrow, plus the operator's commission and rewards
	require.True(t, burned.AmountOf(ethiqtypes.BaseDenom).GT(haqqCoin(53).Amount), burned.String())

	feePoolAfter, err := a.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, feePoolBefore.CommunityPool, feePoolAfter.CommunityPool)
	require.Equal(t, ethiqBurnedBefore, a.EthiqKeeper.GetTotalBurnedAmount(ctx))

	var events int
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == v196.EventTypeHackerFundsBurned {
			events++
		}
	}
	require.Equal(t, len(v196.HackerAccounts), events)
}

func TestUpgradeHandlerEndToEnd(t *testing.T) {
	s := setupScenario(t)
	ctx := s.nw.GetContext()
	a := s.nw.App

	vestingAcc := a.AccountKeeper.GetAccount(ctx, s.hackerVesting)
	innocentBefore := a.BankKeeper.GetAllBalances(ctx, s.innocent)

	// The plan must be due on the very next block: a block below the plan height
	// on a binary that already has the handler halts ("BINARY UPDATED BEFORE
	// TRIGGER"), which is the guard against starting the new binary early.
	plan := upgradetypes.Plan{Name: v196.UpgradeName, Height: ctx.BlockHeight()}
	require.NoError(t, a.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, s.nw.NextBlock()) // PreBlocker runs the handler, then the block commits

	ctx = s.nw.GetContext()
	done, err := a.UpgradeKeeper.GetDoneHeight(ctx, v196.UpgradeName)
	require.NoError(t, err)
	require.Equal(t, plan.Height, done)

	s.requireCleared(t, ctx, innocentBefore, vestingAcc.GetAccountNumber(), vestingAcc.GetSequence())
}

func TestBurnHackerFundsDeterministic(t *testing.T) {
	s := setupScenario(t)
	ctx := s.nw.GetContext()
	a := s.nw.App

	reversed := slices.Clone(v196.HackerAccounts)
	slices.Reverse(reversed)

	ctxA, _ := ctx.CacheContext()
	ctxB, _ := ctx.CacheContext()
	ctxA = ctxA.WithEventManager(sdk.NewEventManager())
	ctxB = ctxB.WithEventManager(sdk.NewEventManager())
	require.NoError(t, v196.BurnHackerFunds(ctxA, keepersOf(a), v196.HackerAccounts))
	require.NoError(t, v196.BurnHackerFunds(ctxB, keepersOf(a), reversed))

	hashA := stateHash(t, a, ctxA)
	require.Equal(t, hashA, stateHash(t, a, ctxB), "input order must not change the resulting state")
	require.NotEqual(t, stateHash(t, a, ctx), hashA, "the burn must change the state")
	require.Equal(t, ctxA.EventManager().Events(), ctxB.EventManager().Events())
}

func TestBurnHackerFundsFailsOnLeftovers(t *testing.T) {
	s := setupScenario(t)
	ctx := s.nw.GetContext()
	a := s.nw.App

	// A redelegation record cannot be removed by undelegating; the handler must
	// refuse to finish rather than leave the account half-processed.
	valOther, err := a.StakingKeeper.GetValidator(ctx, s.valOther)
	require.NoError(t, err)
	shares, err := valOther.SharesFromTokens(islm(5).Amount)
	require.NoError(t, err)
	_, err = a.StakingKeeper.BeginRedelegation(ctx, s.hackerPlain, s.valOther, s.valHacker, shares)
	require.NoError(t, err)

	err = v196.BurnHackerFunds(ctx, keepersOf(a), v196.HackerAccounts)
	require.ErrorContains(t, err, "redelegations left")
}

func TestBurnHackerFundsInput(t *testing.T) {
	nw := network.NewUnitTestNetwork()
	ctx := nw.GetContext()
	k := keepersOf(nw.App)

	// Nothing on the accounts: a no-op that still passes every check.
	require.NoError(t, v196.BurnHackerFunds(ctx, k, v196.HackerAccounts))

	dup := []string{hackerPlainBech, hackerOperatorBech, hackerPlainBech}
	require.ErrorContains(t, v196.BurnHackerFunds(ctx, k, dup), "duplicate account")
	require.Error(t, v196.BurnHackerFunds(ctx, k, []string{"haqq1invalid"}))
}

// stateHash hashes every store the burn can touch.
func stateHash(t *testing.T, a *app.Haqq, ctx sdk.Context) []byte {
	t.Helper()

	h := sha256.New()
	for _, name := range []string{
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey, distrtypes.StoreKey,
		slashingtypes.StoreKey, authzkeeper.StoreKey, feegrant.StoreKey, ucdaotypes.StoreKey,
	} {
		key := a.GetKey(name)
		require.NotNil(t, key, name)

		it := ctx.KVStore(key).Iterator(nil, nil)
		for ; it.Valid(); it.Next() {
			writeLenPrefixed(h, it.Key())
			writeLenPrefixed(h, it.Value())
		}
		require.NoError(t, it.Close())
	}

	return h.Sum(nil)
}

func writeLenPrefixed(h hash.Hash, bz []byte) {
	var l [8]byte
	binary.BigEndian.PutUint64(l[:], uint64(len(bz)))
	h.Write(l[:])
	h.Write(bz)
}
