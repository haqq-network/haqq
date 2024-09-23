package keeper_test

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	"github.com/haqq-network/haqq/testutil"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	"github.com/haqq-network/haqq/x/vesting/types"
)

func TestKeeperIntegrationTestSuite(t *testing.T) {
	s = new(KeeperTestSuite)
	s.SetT(t)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

// TestClawbackAccount is a struct to store all relevant information that is corresponding
// to a clawback vesting account.
type TestClawbackAccount struct {
	privKey         *ethsecp256k1.PrivKey
	address         sdk.AccAddress
	clawbackAccount *types.ClawbackVestingAccount
}

// Initialize general error variable for easier handling in loops throughout this test suite.
var (
	err                error
	stakeDenom         = utils.BaseDenom
	accountGasCoverage = sdk.NewCoins(sdk.NewCoin(stakeDenom, math.NewInt(1e16)))
	amt                = testutil.TestVestingSchedule.VestedCoinsPerPeriod[0].Amount
	cliff              = testutil.TestVestingSchedule.CliffMonths
	cliffLength        = testutil.TestVestingSchedule.CliffPeriodLength
	vestingAmtTotal    = testutil.TestVestingSchedule.TotalVestingCoins
	vestingLength      = testutil.TestVestingSchedule.VestingPeriodLength
	numLockupPeriods   = testutil.TestVestingSchedule.NumLockupPeriods
	periodsTotal       = testutil.TestVestingSchedule.NumVestingPeriods
	lockup             = testutil.TestVestingSchedule.LockupMonths
	unlockedPerLockup  = testutil.TestVestingSchedule.UnlockedCoinsPerLockup
)

// Clawback vesting with Cliff and Lock. In this case the cliff is reached
// before the lockup period is reached to represent the scenario in which an
// employee starts before mainnet launch (periodsCliff < lockupPeriod)
//
// Example:
// 21/10 Employee joins Haqq and vesting starts
// 22/03 Mainnet launch
// 22/09 Cliff ends
// 23/02 Lock ends
var _ = Describe("Clawback Vesting Accounts", Ordered, func() {
	// Create test accounts with private keys for signing
	numTestAccounts := 4
	testAccounts := make([]TestClawbackAccount, numTestAccounts)
	for i := range testAccounts {
		address, privKey := utiltx.NewAddrKey()
		testAccounts[i] = TestClawbackAccount{
			privKey: privKey,
			address: address.Bytes(),
		}
	}
	numTestMsgs := 3

	var (
		clawbackAccount *types.ClawbackVestingAccount
		unvested        sdk.Coins
		vested          sdk.Coins
		// freeCoins are unlocked vested coins of the vesting schedule
		freeCoins         sdk.Coins
		twoThirdsOfVested sdk.Coins
	)

	dest := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	funder := sdk.AccAddress(utiltx.GenerateAddress().Bytes())

	BeforeEach(func() {
		s.SetupTest()

		// Initialize all test accounts
		for i, account := range testAccounts {
			// Create and fund periodic vesting account
			vestingStart := s.ctx.BlockTime()
			baseAccount := authtypes.NewBaseAccountWithAddress(account.address)
			clawbackAccount = types.NewClawbackVestingAccount(
				baseAccount,
				funder,
				vestingAmtTotal,
				vestingStart,
				testutil.TestVestingSchedule.LockupPeriods,
				testutil.TestVestingSchedule.VestingPeriods,
				nil,
			)

			err := testutil.FundAccount(s.ctx, s.app.BankKeeper, account.address, vestingAmtTotal)
			Expect(err).To(BeNil())
			acc := s.app.AccountKeeper.NewAccount(s.ctx, clawbackAccount)
			s.app.AccountKeeper.SetAccount(s.ctx, acc)

			// Check if all tokens are unvested at vestingStart
			unvested = clawbackAccount.GetVestingCoins(s.ctx.BlockTime())
			vested = clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
			Expect(vestingAmtTotal).To(Equal(unvested))
			Expect(vested.IsZero()).To(BeTrue())

			// Grant gas stipend to cover EVM fees
			err = testutil.FundAccount(s.ctx, s.app.BankKeeper, clawbackAccount.GetAddress(), accountGasCoverage)
			Expect(err).To(BeNil())
			granteeBalance := s.app.BankKeeper.GetBalance(s.ctx, account.address, stakeDenom)
			Expect(granteeBalance).To(Equal(accountGasCoverage[0].Add(vestingAmtTotal[0])))

			// Update testAccounts clawbackAccount reference
			testAccounts[i].clawbackAccount = clawbackAccount
		}
	})

	Context("before first vesting period", func() {
		BeforeEach(func() {
			// Add a commit to instantiate blocks
			s.Commit()

			// Ensure no tokens are vested
			vested := clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
			unlocked := clawbackAccount.GetUnlockedCoins(s.ctx.BlockTime())
			zeroCoins := sdk.NewCoins(sdk.NewCoin(stakeDenom, math.ZeroInt()))
			Expect(zeroCoins).To(Equal(vested))
			Expect(zeroCoins).To(Equal(unlocked))
		})

		It("cannot delegate tokens", func() {
			_, err := testutil.Delegate(s.ctx, s.app, testAccounts[0].privKey, accountGasCoverage.Add(sdk.NewCoin(stakeDenom, math.NewInt(1)))[0], s.validator)
			Expect(err).ToNot(BeNil())
		})

		It("can delegate spendable tokens", func() {
			// Try to delegate available spendable balance
			// All vesting tokens are unvested at this point
			// Only 1e16 aISLM is available for gas coverage
			_, err := testutil.Delegate(s.ctx, s.app, testAccounts[0].privKey, sdk.NewCoin(stakeDenom, math.NewInt(1e15)), s.validator)
			Expect(err).ToNot(BeNil())
		})

		It("can transfer spendable tokens", func() {
			account := testAccounts[0]
			// Fund account with new spendable tokens
			err := testutil.FundAccount(s.ctx, s.app.BankKeeper, account.address, unvested)
			Expect(err).To(BeNil())

			err = s.app.BankKeeper.SendCoins(
				s.ctx,
				account.address,
				dest,
				unvested,
			)
			Expect(err).To(BeNil())
		})

		It("cannot transfer unvested tokens", func() {
			err := s.app.BankKeeper.SendCoins(
				s.ctx,
				clawbackAccount.GetAddress(),
				dest,
				unvested,
			)
			Expect(err).ToNot(BeNil())
		})

		It("can perform Ethereum tx with spendable balance", func() {
			account := testAccounts[0]
			// Fund account with new spendable tokens
			coins := testutil.TestVestingSchedule.UnlockedCoinsPerLockup
			err := testutil.FundAccount(s.ctx, s.app.BankKeeper, account.address, coins)
			Expect(err).To(BeNil())

			txAmount := coins.AmountOf(stakeDenom).BigInt()
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, txAmount, 0)
			Expect(err).To(BeNil())

			assertEthSucceeds([]TestClawbackAccount{account}, funder, dest, coins.AmountOf(stakeDenom), stakeDenom, msg)
		})

		It("cannot perform Ethereum tx with unvested balance", func() {
			account := testAccounts[0]
			unlockedCoins := testutil.TestVestingSchedule.UnlockedCoinsPerLockup
			txAmount := unlockedCoins.AmountOf(stakeDenom).BigInt()
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, txAmount, 0)
			Expect(err).To(BeNil())

			assertEthFails(msg)
		})
	})

	Context("after first vesting period and before lockup", func() {
		BeforeEach(func() {
			// Surpass cliff but none of lockup duration
			cliffDuration := time.Duration(testutil.TestVestingSchedule.CliffPeriodLength)
			s.CommitAfter(cliffDuration * time.Second)

			// Check if some, but not all tokens are vested
			vested = clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
			expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, amt.Mul(math.NewInt(testutil.TestVestingSchedule.CliffMonths))))
			Expect(vested).NotTo(Equal(vestingAmtTotal))
			Expect(vested).To(Equal(expVested))

			// check the vested tokens are still locked
			freeCoins = clawbackAccount.GetUnlockedVestedCoins(s.ctx.BlockTime())
			Expect(freeCoins).To(Equal(sdk.Coins{}))

			twoThirdsOfVested = vested.Sub(vested.QuoInt(math.NewInt(3))...)

			res, err := s.app.VestingKeeper.Balances(s.ctx, &types.QueryBalancesRequest{Address: clawbackAccount.Address})
			Expect(err).To(BeNil())
			Expect(res.Vested).To(Equal(expVested))
			Expect(res.Unvested).To(Equal(vestingAmtTotal.Sub(expVested...)))
			// All coins from vesting schedule should be locked
			Expect(res.Locked).To(Equal(vestingAmtTotal))
		})

		It("can delegate vested locked tokens", func() {
			testAccount := testAccounts[0]
			// Verify that the total spendable coins should only be coins
			// not in the vesting schedule. Because all coins from the vesting
			// schedule are still locked
			spendablePre := s.app.BankKeeper.SpendableCoins(s.ctx, testAccount.address)
			Expect(spendablePre).To(Equal(accountGasCoverage))

			// delegate the vested locked coins.
			_, err := testutil.Delegate(s.ctx, s.app, testAccount.privKey, vested[0], s.validator)
			Expect(err).To(BeNil(), "expected no error during delegation")

			// check spendable coins have only been reduced by the gas paid for the transaction to show that the delegated coins were taken from the locked but vested amount
			spendablePost := s.app.BankKeeper.SpendableCoins(s.ctx, testAccount.address)
			Expect(spendablePost).To(Equal(spendablePre.Sub(accountGasCoverage...)))

			// check delegation was created successfully
			stkQuerier := stakingkeeper.Querier{Keeper: s.app.StakingKeeper.Keeper}
			delRes, err := stkQuerier.DelegatorDelegations(s.ctx, &stakingtypes.QueryDelegatorDelegationsRequest{DelegatorAddr: testAccount.clawbackAccount.Address})
			Expect(err).To(BeNil())
			Expect(delRes.DelegationResponses).To(HaveLen(1))
			Expect(delRes.DelegationResponses[0].Balance.Amount).To(Equal(vested[0].Amount))
		})

		It("account with free balance - delegates the free balance amount. It is tracked as locked vested tokens for the spendable balance calculation", func() {
			testAccount := testAccounts[0]

			// send some funds to the account to delegate
			coinsToDelegate := sdk.NewCoins(sdk.NewCoin(stakeDenom, math.NewInt(1e18)))
			// check that coins to delegate are greater than the locked up vested coins
			Expect(coinsToDelegate.IsAllGT(vested)).To(BeTrue())

			err = testutil.FundAccount(s.ctx, s.app.BankKeeper, testAccount.address, coinsToDelegate)
			Expect(err).To(BeNil())

			// the free coins delegated will be the delegatedCoins - lockedUp vested coins
			freeCoinsDelegated := coinsToDelegate.Sub(vested...)

			initialBalances := s.app.BankKeeper.GetAllBalances(s.ctx, testAccount.address)
			Expect(initialBalances).To(Equal(testutil.TestVestingSchedule.TotalVestingCoins.Add(coinsToDelegate...).Add(accountGasCoverage...)))
			// Verify that the total spendable coins should only be coins
			// not in the vesting schedule. Because all coins from the vesting
			// schedule are still locked up
			spendablePre := s.app.BankKeeper.SpendableCoins(s.ctx, testAccount.address)
			Expect(spendablePre).To(Equal(accountGasCoverage.Add(coinsToDelegate...)))

			// delegate funds - the delegation amount will be tracked as locked up vested coins delegated + some free coins
			res, err := testutil.Delegate(s.ctx, s.app, testAccount.privKey, coinsToDelegate[0], s.validator)
			Expect(err).NotTo(HaveOccurred(), "expected no error during delegation")
			Expect(res.IsOK()).To(BeTrue())

			// check balances updated properly
			finalBalances := s.app.BankKeeper.GetAllBalances(s.ctx, testAccount.address)
			Expect(finalBalances).To(Equal(initialBalances.Sub(coinsToDelegate...).Sub(accountGasCoverage...)))

			// the expected spendable balance will be
			// spendable = bank balances - (coins in vesting schedule - unlocked vested coins (0) - locked up vested coins delegated)
			expSpendable := finalBalances.Sub(testutil.TestVestingSchedule.TotalVestingCoins...).Add(vested...)

			// which should be equal to the initial freeCoins - freeCoins delegated
			Expect(expSpendable).To(Equal(coinsToDelegate.Sub(freeCoinsDelegated...)))

			// check spendable balance is updated properly
			spendablePost := s.app.BankKeeper.SpendableCoins(s.ctx, testAccount.address)
			Expect(spendablePost).To(Equal(expSpendable))
		})

		It("can delegate tokens from account balance (free tokens) + locked vested tokens", func() {
			testAccount := testAccounts[0]

			// send some funds to the account to delegate
			amt := sdk.NewCoins(sdk.NewCoin(stakeDenom, math.NewInt(1e18)))
			err = testutil.FundAccount(s.ctx, s.app.BankKeeper, testAccount.address, amt)
			Expect(err).To(BeNil())

			// Verify that the total spendable coins should only be coins
			// not in the vesting schedule. Because all coins from the vesting
			// schedule are still locked
			spendablePre := s.app.BankKeeper.SpendableCoins(s.ctx, testAccount.address)
			Expect(spendablePre).To(Equal(accountGasCoverage.Add(amt...)))

			// delegate some tokens from the account balance + locked vested coins
			coinsToDelegate := amt.Add(vested...)

			res, err := testutil.Delegate(s.ctx, s.app, testAccount.privKey, coinsToDelegate[0], s.validator)
			Expect(err).NotTo(HaveOccurred(), "expected no error during delegation")
			Expect(res.IsOK()).To(BeTrue())

			// check spendable balance is updated properly
			spendablePost := s.app.BankKeeper.SpendableCoins(s.ctx, testAccount.address)
			Expect(spendablePost).To(Equal(spendablePre.Sub(amt...).Sub(accountGasCoverage...)))
		})

		It("cannot delegate unvested tokens in sequetial txs", func() {
			_, err := testutil.Delegate(s.ctx, s.app, testAccounts[0].privKey, twoThirdsOfVested[0], s.validator)
			Expect(err).To(BeNil(), "error while executing the delegate message")
			_, err = testutil.Delegate(s.ctx, s.app, testAccounts[0].privKey, twoThirdsOfVested[0], s.validator)
			Expect(err).ToNot(BeNil())
		})

		It("cannot delegate then send tokens", func() {
			_, err := testutil.Delegate(s.ctx, s.app, testAccounts[0].privKey, twoThirdsOfVested[0], s.validator)
			Expect(err).To(BeNil())

			err = s.app.BankKeeper.SendCoins(
				s.ctx,
				clawbackAccount.GetAddress(),
				dest,
				twoThirdsOfVested,
			)
			Expect(err).ToNot(BeNil())
		})

		It("cannot delegate more than the locked vested tokens", func() {
			_, err := testutil.Delegate(s.ctx, s.app, testAccounts[0].privKey, vested[0].Add(sdk.NewCoin(stakeDenom, math.NewInt(1))), s.validator)
			Expect(err).ToNot(BeNil())
		})

		It("cannot delegate free tokens and then send locked/unvested tokens", func() {
			// send some funds to the account to delegate
			coinsToDelegate := sdk.NewCoins(sdk.NewCoin(stakeDenom, math.NewInt(1e18)))
			err = testutil.FundAccount(s.ctx, s.app.BankKeeper, testAccounts[0].address, coinsToDelegate)
			Expect(err).To(BeNil())

			_, err := testutil.Delegate(s.ctx, s.app, testAccounts[0].privKey, coinsToDelegate[0], s.validator)
			Expect(err).To(BeNil())

			err = s.app.BankKeeper.SendCoins(
				s.ctx,
				clawbackAccount.GetAddress(),
				dest,
				twoThirdsOfVested,
			)
			Expect(err).ToNot(BeNil())
		})

		It("cannot transfer locked vested tokens", func() {
			err := s.app.BankKeeper.SendCoins(
				s.ctx,
				clawbackAccount.GetAddress(),
				dest,
				vested,
			)
			Expect(err).ToNot(BeNil())
		})

		It("can perform Ethereum tx with spendable balance", func() {
			account := testAccounts[0]
			coins := testutil.TestVestingSchedule.UnlockedCoinsPerLockup
			// Fund account with new spendable tokens
			err := testutil.FundAccount(s.ctx, s.app.BankKeeper, account.address, coins)
			Expect(err).To(BeNil())

			txAmount := coins.AmountOf(stakeDenom)
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, txAmount.BigInt(), 0)
			Expect(err).To(BeNil())

			assertEthSucceeds([]TestClawbackAccount{account}, funder, dest, txAmount, stakeDenom, msg)
		})

		It("cannot perform Ethereum tx with locked vested balance", func() {
			account := testAccounts[0]
			txAmount := vested.AmountOf(stakeDenom).BigInt()
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, txAmount, 0)
			Expect(err).To(BeNil())
			assertEthFails(msg)
		})
	})

	Context("Between first and second lockup periods", func() {
		BeforeEach(func() {
			// Surpass first lockup
			vestDuration := time.Duration(testutil.TestVestingSchedule.LockupPeriodLength)
			s.CommitAfter(vestDuration * time.Second)

			// after first lockup period
			// half of total vesting tokens are unlocked
			// but only 12 vesting periods passed
			// Check if some, but not all tokens are vested and unlocked
			for _, account := range testAccounts {
				vested = account.clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
				unlocked := account.clawbackAccount.GetUnlockedCoins(s.ctx.BlockTime())
				freeCoins = account.clawbackAccount.GetUnlockedVestedCoins(s.ctx.BlockTime())

				expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, amt.Mul(math.NewInt(lockup))))
				expUnlockedVested := expVested

				Expect(vested).NotTo(Equal(vestingAmtTotal))
				Expect(vested).To(Equal(expVested))
				Expect(unlocked).To(Equal(unlockedPerLockup))
				Expect(freeCoins).To(Equal(expUnlockedVested))
			}
		})

		It("delegate unlocked vested tokens and spendable balance is updated properly", func() {
			account := testAccounts[0]
			balance := s.app.BankKeeper.GetBalance(s.ctx, account.address, stakeDenom)
			// the returned balance should be the account's initial balance and
			// the total amount of the vesting schedule
			Expect(balance.Amount).To(Equal(accountGasCoverage.Add(vestingAmtTotal...)[0].Amount))

			spReq := &banktypes.QuerySpendableBalanceByDenomRequest{Address: account.address.String(), Denom: stakeDenom}
			spRes, err := s.app.BankKeeper.SpendableBalanceByDenom(s.ctx, spReq)
			Expect(err).To(BeNil())
			// spendable balance should be the initial account balance + vested tokens
			initialSpendableBalance := spRes.Balance
			Expect(initialSpendableBalance.Amount).To(Equal(accountGasCoverage.Add(freeCoins...)[0].Amount))

			// can delegate vested tokens
			// fees paid is accountGasCoverage amount
			res, err := testutil.Delegate(s.ctx, s.app, account.privKey, freeCoins[0], s.validator)
			Expect(err).ToNot(HaveOccurred(), "expected no error during delegation")
			Expect(res.Code).To(BeZero(), "expected delegation to succeed")

			// spendable balance should be updated to be prevSpendableBalance - delegatedAmt - fees
			spRes, err = s.app.BankKeeper.SpendableBalanceByDenom(s.ctx, spReq)
			Expect(err).To(BeNil())
			Expect(spRes.Balance.Amount.Int64()).To(Equal(int64(0)))

			// try to send coins - should error
			err = s.app.BankKeeper.SendCoins(s.ctx, account.address, funder, vested)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("spendable balance"))
			Expect(err.Error()).To(ContainSubstring("is smaller than"))
		})

		It("cannot delegate more than vested tokens", func() {
			account := testAccounts[0]
			balance := s.app.BankKeeper.GetBalance(s.ctx, account.address, stakeDenom)
			// the returned balance should be the account's initial balance and
			// the total amount of the vesting schedule
			Expect(balance.Amount).To(Equal(accountGasCoverage.Add(vestingAmtTotal...)[0].Amount))

			spReq := &banktypes.QuerySpendableBalanceByDenomRequest{Address: account.address.String(), Denom: stakeDenom}
			spRes, err := s.app.BankKeeper.SpendableBalanceByDenom(s.ctx, spReq)
			Expect(err).To(BeNil())
			// spendable balance should be the initial account balance + vested tokens
			initialSpendableBalance := spRes.Balance
			Expect(initialSpendableBalance.Amount).To(Equal(accountGasCoverage.Add(freeCoins...)[0].Amount))

			// cannot delegate more than vested tokens
			_, err = testutil.Delegate(s.ctx, s.app, account.privKey, freeCoins[0].Add(sdk.NewCoin(stakeDenom, math.NewInt(1))), s.validator)
			Expect(err).To(HaveOccurred(), "expected no error during delegation")
			Expect(err.Error()).To(ContainSubstring("cannot delegate unvested coins"))
		})

		It("should enable access to unlocked and vested EVM tokens (single-account, single-msg)", func() {
			account := testAccounts[0]
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, vested[0].Amount.BigInt(), 0)
			Expect(err).To(BeNil())

			assertEthSucceeds([]TestClawbackAccount{account}, funder, dest, vested[0].Amount, stakeDenom, msg)
		})

		It("should enable access to unlocked EVM tokens (single-account, multiple-msgs)", func() {
			account := testAccounts[0]

			// Split the total unlocked amount into numTestMsgs equally sized tx's
			msgs := make([]sdk.Msg, numTestMsgs)
			txAmount := vested[0].Amount.QuoRaw(int64(numTestMsgs)).BigInt()

			for i := 0; i < numTestMsgs; i++ {
				msgs[i], err = utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, txAmount, i)
				Expect(err).To(BeNil())
			}

			assertEthSucceeds([]TestClawbackAccount{account}, funder, dest, vested[0].Amount, stakeDenom, msgs...)
		})

		It("should enable access to unlocked EVM tokens (multi-account, single-msg)", func() {
			txAmount := vested[0].Amount.BigInt()

			msgs := make([]sdk.Msg, numTestAccounts)
			for i, grantee := range testAccounts {
				msgs[i], err = utiltx.CreateEthTx(s.ctx, s.app, grantee.privKey, grantee.address, dest, txAmount, 0)
				Expect(err).To(BeNil())
			}

			assertEthSucceeds(testAccounts, funder, dest, vested[0].Amount, stakeDenom, msgs...)
		})

		It("should enable access to unlocked EVM tokens (multi-account, multiple-msgs)", func() {
			msgs := []sdk.Msg{}
			txAmount := vested[0].Amount.QuoRaw(int64(numTestMsgs)).BigInt()

			for _, grantee := range testAccounts {
				for j := 0; j < numTestMsgs; j++ {
					addedMsg, err := utiltx.CreateEthTx(s.ctx, s.app, grantee.privKey, grantee.address, dest, txAmount, j)
					Expect(err).To(BeNil())
					msgs = append(msgs, addedMsg)
				}
			}

			assertEthSucceeds(testAccounts, funder, dest, vested[0].Amount, stakeDenom, msgs...)
		})

		It("should not enable access to locked EVM tokens (single-account, single-msg)", func() {
			testAccount := testAccounts[0]
			// Attempt to spend entire vesting balance
			txAmount := vestingAmtTotal.AmountOf(stakeDenom).BigInt()
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, testAccount.privKey, testAccount.address, dest, txAmount, 0)
			Expect(err).To(BeNil())
			assertEthFails(msg)
		})

		It("should not enable access to locked EVM tokens (single-account, multiple-msgs)", func() {
			msgs := make([]sdk.Msg, numTestMsgs+1)
			txAmount := vested[0].Amount.QuoRaw(int64(numTestMsgs)).BigInt()
			testAccount := testAccounts[0]

			// Add additional message that exceeds unlocked balance
			for i := 0; i < numTestMsgs+1; i++ {
				msgs[i], err = utiltx.CreateEthTx(s.ctx, s.app, testAccount.privKey, testAccount.address, dest, txAmount, i)
				Expect(err).To(BeNil())
			}
			assertEthFails(msgs...)
		})

		It("should not enable access to locked EVM tokens (multi-account, single-msg)", func() {
			msgs := make([]sdk.Msg, numTestAccounts+1)
			txAmount := vested[0].Amount.BigInt()

			for i, account := range testAccounts {
				msgs[i], err = utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, txAmount, 0)
				Expect(err).To(BeNil())
			}
			// Add additional message that exceeds unlocked balance
			msgs[numTestAccounts], err = utiltx.CreateEthTx(s.ctx, s.app, testAccounts[0].privKey, testAccounts[0].address, dest, txAmount, 1)
			Expect(err).To(BeNil())
			assertEthFails(msgs...)
		})

		It("should not enable access to locked EVM tokens (multi-account, multiple-msgs)", func() {
			msgs := []sdk.Msg{}
			txAmount := vested[0].Amount.QuoRaw(int64(numTestMsgs)).BigInt()
			var addedMsg sdk.Msg

			for _, account := range testAccounts {
				for j := 0; j < numTestMsgs; j++ {
					addedMsg, err = utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, txAmount, j)
					msgs = append(msgs, addedMsg)
				}
			}
			// Add additional message that exceeds unlocked balance
			addedMsg, err = utiltx.CreateEthTx(s.ctx, s.app, testAccounts[0].privKey, testAccounts[0].address, dest, txAmount, numTestMsgs)
			Expect(err).To(BeNil())
			msgs = append(msgs, addedMsg)
			assertEthFails(msgs...)
		})

		It("should not short-circuit with a normal account", func() {
			account := testAccounts[0]
			address, privKey := utiltx.NewAccAddressAndKey()
			txAmount := vestingAmtTotal.AmountOf(stakeDenom).BigInt()

			// Fund a normal account to try to short-circuit the AnteHandler
			err = testutil.FundAccount(s.ctx, s.app.BankKeeper, address, vestingAmtTotal.MulInt(math.NewInt(2)))
			Expect(err).To(BeNil())
			normalAccMsg, err := utiltx.CreateEthTx(s.ctx, s.app, privKey, address, dest, txAmount, 0)
			Expect(err).To(BeNil())

			// Attempt to spend entire balance
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, txAmount, 0)
			Expect(err).To(BeNil())
			err = validateEthVestingTransactionDecorator(normalAccMsg, msg)
			Expect(err).ToNot(BeNil())

			_, err = testutil.DeliverEthTx(s.app, nil, msg)
			Expect(err).ToNot(BeNil())
		})
	})

	Context("after first lockup and additional vest", func() {
		BeforeEach(func() {
			vestDuration := time.Duration(testutil.TestVestingSchedule.LockupPeriodLength + vestingLength)
			s.CommitAfter(vestDuration * time.Second)

			// after first lockup period
			// half of total vesting tokens are unlocked
			// now only 13 vesting periods passed

			vested = clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
			expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, amt.Mul(math.NewInt(lockup+1))))

			unlocked := clawbackAccount.GetUnlockedCoins(s.ctx.BlockTime())
			expUnlocked := unlockedPerLockup

			Expect(expVested).To(Equal(vested))
			Expect(expUnlocked).To(Equal(unlocked))
		})

		It("should enable access to unlocked EVM tokens", func() {
			testAccount := testAccounts[0]

			txAmount := vested[0].Amount.BigInt()
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, testAccount.privKey, testAccount.address, dest, txAmount, 0)
			Expect(err).To(BeNil())

			assertEthSucceeds([]TestClawbackAccount{testAccount}, funder, dest, vested[0].Amount, stakeDenom, msg)
		})

		It("should not enable access to unvested EVM tokens", func() {
			testAccount := testAccounts[0]

			txAmount := vested[0].Amount.Add(amt).BigInt()
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, testAccount.privKey, testAccount.address, dest, txAmount, 0)
			Expect(err).To(BeNil())

			assertEthFails(msg)
		})
	})

	Context("after half of vesting period and both lockups", func() {
		BeforeEach(func() {
			// Surpass lockup duration
			lockupDuration := time.Duration(testutil.TestVestingSchedule.LockupPeriodLength * numLockupPeriods)
			s.CommitAfter(lockupDuration * time.Second)
			// after two lockup period
			// total vesting tokens are unlocked
			// and 24/48 vesting periods passed

			// Check if some, but not all tokens are vested
			unvested = clawbackAccount.GetVestingCoins(s.ctx.BlockTime())
			vested = clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
			expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, amt.Mul(math.NewInt(lockup*numLockupPeriods))))
			Expect(vestingAmtTotal).NotTo(Equal(vested))
			Expect(expVested).To(Equal(vested))
		})

		It("can delegate vested tokens", func() {
			_, err := testutil.Delegate(s.ctx, s.app, testAccounts[0].privKey, vested[0], s.validator)
			Expect(err).To(BeNil())
		})

		It("cannot delegate unvested tokens", func() {
			_, err := testutil.Delegate(s.ctx, s.app, testAccounts[0].privKey, vestingAmtTotal[0], s.validator)
			Expect(err).ToNot(BeNil())
		})

		It("can transfer vested tokens", func() {
			err := s.app.BankKeeper.SendCoins(
				s.ctx,
				clawbackAccount.GetAddress(),
				sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
				vested,
			)
			Expect(err).To(BeNil())
		})

		It("cannot transfer unvested tokens", func() {
			err := s.app.BankKeeper.SendCoins(
				s.ctx,
				clawbackAccount.GetAddress(),
				sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
				vestingAmtTotal,
			)
			Expect(err).ToNot(BeNil())
		})

		It("can perform Ethereum tx with spendable balance", func() {
			account := testAccounts[0]
			txAmount := vested.AmountOf(stakeDenom).BigInt()
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, txAmount, 0)
			Expect(err).To(BeNil())
			assertEthSucceeds([]TestClawbackAccount{account}, funder, dest, vested.AmountOf(stakeDenom), stakeDenom, msg)
		})
	})

	Context("after entire vesting period and both lockups", func() {
		BeforeEach(func() {
			// Surpass vest duration
			vestDuration := time.Duration(vestingLength * periodsTotal)
			s.CommitAfter(vestDuration * time.Second)

			// Check that all tokens are vested and unlocked
			unvested = clawbackAccount.GetVestingCoins(s.ctx.BlockTime())
			vested = clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
			unlocked := clawbackAccount.GetUnlockedCoins(s.ctx.BlockTime())
			unlockedVested := clawbackAccount.GetUnlockedVestedCoins(s.ctx.BlockTime())
			notSpendable := clawbackAccount.LockedCoins(s.ctx.BlockTime())

			// all vested coins should be unlocked
			Expect(vested).To(Equal(unlockedVested))

			zeroCoins := sdk.NewCoins(sdk.NewCoin(stakeDenom, math.ZeroInt()))
			Expect(vestingAmtTotal).To(Equal(vested))
			Expect(vestingAmtTotal).To(Equal(unlocked))
			Expect(zeroCoins).To(Equal(notSpendable))
			Expect(zeroCoins).To(Equal(unvested))
		})

		It("can send entire balance", func() {
			account := testAccounts[0]
			txAmount := vestingAmtTotal.AmountOf(stakeDenom)
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, txAmount.BigInt(), 0)
			Expect(err).To(BeNil())
			assertEthSucceeds([]TestClawbackAccount{account}, funder, dest, txAmount, stakeDenom, msg)
		})

		It("cannot exceed balance", func() {
			account := testAccounts[0]
			txAmount := vestingAmtTotal.AmountOf(stakeDenom).Mul(math.NewInt(2))
			msg, err := utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, txAmount.BigInt(), 0)
			Expect(err).To(BeNil())
			assertEthFails(msg)
		})

		It("should short-circuit with zero balance", func() {
			account := testAccounts[0]
			balance := s.app.BankKeeper.GetBalance(s.ctx, account.address, stakeDenom)

			// Drain account balance
			err := s.app.BankKeeper.SendCoins(s.ctx, account.address, dest, sdk.NewCoins(balance))
			Expect(err).To(BeNil())

			msg, err := utiltx.CreateEthTx(s.ctx, s.app, account.privKey, account.address, dest, big.NewInt(0), 0)
			Expect(err).To(BeNil())
			err = validateEthVestingTransactionDecorator(msg)
			Expect(err).ToNot(BeNil())
			Expect(strings.Contains(err.Error(), "no balance")).To(BeTrue())
		})
	})
})

// Example:
// 21/10 Employee joins Haqq and vesting starts
// 22/03 Mainnet launch
// 22/09 Cliff ends
// 23/02 Lock ends
var _ = Describe("Clawback Vesting Accounts - claw back tokens", func() {
	var (
		clawbackAccount *types.ClawbackVestingAccount
		vesting         sdk.Coins
		vested          sdk.Coins
		unlocked        sdk.Coins
		free            sdk.Coins
		isClawback      bool
	)

	vestingAddr := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	funder := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	dest := sdk.AccAddress(utiltx.GenerateAddress().Bytes())

	BeforeEach(func() {
		s.SetupTest()

		vestingStart := s.ctx.BlockTime()

		// Initialize account at vesting address by funding it with tokens
		// and then send them over to the vesting funder
		err := testutil.FundAccount(s.ctx, s.app.BankKeeper, vestingAddr, vestingAmtTotal)
		Expect(err).ToNot(HaveOccurred(), "failed to fund target account")
		err = s.app.BankKeeper.SendCoins(s.ctx, vestingAddr, funder, vestingAmtTotal)
		Expect(err).ToNot(HaveOccurred(), "failed to send coins to funder")

		// Send some tokens to the vesting account to cover tx fees
		err = testutil.FundAccount(s.ctx, s.app.BankKeeper, vestingAddr, accountGasCoverage)
		Expect(err).ToNot(HaveOccurred(), "failed to fund target account")

		balanceFunder := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		balanceGrantee := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)
		balanceDest := s.app.BankKeeper.GetBalance(s.ctx, dest, stakeDenom)
		Expect(balanceFunder).To(Equal(vestingAmtTotal[0]), "expected different funder balance")
		Expect(balanceGrantee.Amount).To(Equal(accountGasCoverage[0].Amount))
		Expect(balanceDest.IsZero()).To(BeTrue(), "expected destination balance to be zero")

		msg := types.NewMsgConvertIntoVestingAccount(
			funder,
			vestingAddr,
			vestingStart,
			testutil.TestVestingSchedule.LockupPeriods,
			testutil.TestVestingSchedule.VestingPeriods,
			true, false, nil,
		)
		_, err = s.app.VestingKeeper.ConvertIntoVestingAccount(sdk.WrapSDKContext(s.ctx), msg)
		Expect(err).ToNot(HaveOccurred(), "expected creating clawback vesting account to succeed")
		acc := s.app.AccountKeeper.GetAccount(s.ctx, vestingAddr)
		clawbackAccount, isClawback = acc.(*types.ClawbackVestingAccount)
		Expect(isClawback).To(BeTrue(), "expected account to be clawback vesting account")

		// Check if all tokens are unvested and locked at vestingStart
		vesting = clawbackAccount.GetVestingCoins(s.ctx.BlockTime())
		vested = clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
		unlocked = clawbackAccount.GetUnlockedCoins(s.ctx.BlockTime())
		Expect(vesting).To(Equal(vestingAmtTotal), "expected difference vesting tokens")
		Expect(vested.IsZero()).To(BeTrue(), "expected no tokens to be vested")
		Expect(unlocked.IsZero()).To(BeTrue(), "expected no tokens to be unlocked")
		bF := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		balanceGrantee = s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)
		balanceDest = s.app.BankKeeper.GetBalance(s.ctx, dest, stakeDenom)

		Expect(bF.IsZero()).To(BeTrue(), "expected funder balance to be zero")
		Expect(balanceGrantee).To(Equal(vestingAmtTotal.Add(accountGasCoverage...)[0]), "expected all tokens to be locked")
		Expect(balanceDest.IsZero()).To(BeTrue(), "expected no tokens to be unlocked")
	})

	It("should claw back unvested amount before cliff", func() {
		ctx := sdk.WrapSDKContext(s.ctx)
		balanceFunder := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		balanceDest := s.app.BankKeeper.GetBalance(s.ctx, dest, stakeDenom)

		// Perform clawback before cliff
		msg := types.NewMsgClawback(funder, vestingAddr, dest)
		_, err := s.app.VestingKeeper.Clawback(ctx, msg)
		Expect(err).To(BeNil())

		// All initial vesting amount goes to dest
		bF := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		bG := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)
		bD := s.app.BankKeeper.GetBalance(s.ctx, dest, stakeDenom)

		Expect(bF).To(Equal(balanceFunder), "expected funder balance to be unchanged")
		Expect(bG.Amount).To(Equal(accountGasCoverage[0].Amount), "expected all tokens to be clawed back")
		Expect(bD).To(Equal(balanceDest.Add(vestingAmtTotal[0])), "expected all tokens to be clawed back to the destination account")
	})

	It("should claw back any unvested amount after cliff before unlocking", func() {
		// Surpass cliff but not lockup duration
		cliffDuration := time.Duration(cliffLength)
		s.CommitAfter(cliffDuration * time.Second)

		// Check that all tokens are locked and some, but not all tokens are vested
		vested = clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
		unlocked = clawbackAccount.GetUnlockedCoins(s.ctx.BlockTime())
		free = clawbackAccount.GetUnlockedVestedCoins(s.ctx.BlockTime())
		vesting = clawbackAccount.GetVestingCoins(s.ctx.BlockTime())
		expVestedAmount := amt.Mul(math.NewInt(cliff))
		expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, expVestedAmount))

		Expect(expVested).To(Equal(vested))
		Expect(expVestedAmount.GT(math.NewInt(0))).To(BeTrue())
		Expect(free.IsZero()).To(BeTrue())
		Expect(vesting).To(Equal(vestingAmtTotal.Sub(expVested...)))

		balanceFunder := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		balanceGrantee := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)
		balanceDest := s.app.BankKeeper.GetBalance(s.ctx, dest, stakeDenom)
		// Perform clawback
		msg := types.NewMsgClawback(funder, vestingAddr, dest)
		ctx := sdk.WrapSDKContext(s.ctx)
		_, err = s.app.VestingKeeper.Clawback(ctx, msg)
		Expect(err).To(BeNil())

		bF := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		bG := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)
		bD := s.app.BankKeeper.GetBalance(s.ctx, dest, stakeDenom)

		expClawback := clawbackAccount.GetVestingCoins(s.ctx.BlockTime())

		// Any unvested amount is clawed back
		Expect(balanceFunder).To(Equal(bF))
		Expect(balanceGrantee.Sub(expClawback[0]).Amount.Uint64()).To(Equal(bG.Amount.Uint64()))
		Expect(balanceDest.Add(expClawback[0]).Amount.Uint64()).To(Equal(bD.Amount.Uint64()))
	})

	It("should claw back any unvested amount after cliff and unlocking", func() {
		// Surpass lockup duration
		// A strict `if t < clawbackTime` comparison is used in ComputeClawback
		// so, we increment the duration with 1 for the free token calculation to match
		lockupDuration := time.Duration(testutil.TestVestingSchedule.LockupPeriodLength + 1)
		s.CommitAfter(lockupDuration * time.Second)

		// Check if some, but not all tokens are vested and unlocked
		vested = clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
		unlocked = clawbackAccount.GetUnlockedCoins(s.ctx.BlockTime())
		free = clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
		vesting = clawbackAccount.GetVestingCoins(s.ctx.BlockTime())
		expVestedAmount := amt.Mul(math.NewInt(lockup))
		expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, expVestedAmount))
		unvested := vestingAmtTotal.Sub(vested...)

		Expect(free).To(Equal(vested))
		Expect(expVested).To(Equal(vested))
		Expect(expVestedAmount.GT(math.NewInt(0))).To(BeTrue())
		Expect(vesting).To(Equal(unvested))

		balanceFunder := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		balanceGrantee := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)
		balanceDest := s.app.BankKeeper.GetBalance(s.ctx, dest, stakeDenom)
		// Perform clawback
		msg := types.NewMsgClawback(funder, vestingAddr, dest)
		ctx := sdk.WrapSDKContext(s.ctx)
		_, err = s.app.VestingKeeper.Clawback(ctx, msg)
		Expect(err).To(BeNil())

		bF := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		bG := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)
		bD := s.app.BankKeeper.GetBalance(s.ctx, dest, stakeDenom)

		// Any unvested amount is clawed back
		Expect(balanceFunder).To(Equal(bF))
		Expect(balanceGrantee.Sub(vesting[0]).Amount.Uint64()).To(Equal(bG.Amount.Uint64()))
		Expect(balanceDest.Add(vesting[0]).Amount.Uint64()).To(Equal(bD.Amount.Uint64()))
	})

	It("should not claw back any amount after vesting periods end", func() {
		// Surpass vesting periods
		vestingDuration := time.Duration(periodsTotal*vestingLength + 1)
		s.CommitAfter(vestingDuration * time.Second)

		// Check if some, but not all tokens are vested and unlocked
		vested = clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
		unlocked = clawbackAccount.GetUnlockedCoins(s.ctx.BlockTime())
		free = clawbackAccount.GetVestedCoins(s.ctx.BlockTime())
		vesting = clawbackAccount.GetVestingCoins(s.ctx.BlockTime())

		expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, amt.Mul(math.NewInt(periodsTotal))))
		unvested := vestingAmtTotal.Sub(vested...)

		Expect(free).To(Equal(vested))
		Expect(expVested).To(Equal(vested))
		Expect(expVested).To(Equal(vestingAmtTotal))
		Expect(unlocked).To(Equal(vestingAmtTotal))
		Expect(vesting).To(Equal(unvested))
		Expect(vesting.IsZero()).To(BeTrue())

		balanceFunder := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		balanceGrantee := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)
		balanceDest := s.app.BankKeeper.GetBalance(s.ctx, dest, stakeDenom)
		// Perform clawback
		msg := types.NewMsgClawback(funder, vestingAddr, dest)
		ctx := sdk.WrapSDKContext(s.ctx)
		res, err := s.app.VestingKeeper.Clawback(ctx, msg)
		Expect(err).To(BeNil(), "expected no error during clawback")
		Expect(res).ToNot(BeNil(), "expected response not to be nil")

		bF := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		bG := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)
		bD := s.app.BankKeeper.GetBalance(s.ctx, dest, stakeDenom)

		// No amount is clawed back
		Expect(balanceFunder).To(Equal(bF))
		Expect(balanceGrantee).To(Equal(bG))
		Expect(balanceDest).To(Equal(bD))
	})

	It("should update vesting funder and claw back unvested amount before cliff", func() {
		ctx := sdk.WrapSDKContext(s.ctx)
		newFunder := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
		balanceFunder := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		balanceNewFunder := s.app.BankKeeper.GetBalance(s.ctx, newFunder, stakeDenom)
		balanceGrantee := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)

		// Update clawback vesting account funder
		updateFunderMsg := types.NewMsgUpdateVestingFunder(funder, newFunder, vestingAddr)
		_, err := s.app.VestingKeeper.UpdateVestingFunder(ctx, updateFunderMsg)
		Expect(err).To(BeNil())

		// Perform clawback before cliff - funds should go to new funder (no dest address defined)
		msg := types.NewMsgClawback(newFunder, vestingAddr, sdk.AccAddress([]byte{}))
		_, err = s.app.VestingKeeper.Clawback(ctx, msg)
		Expect(err).To(BeNil())

		// All initial vesting amount goes to funder
		bF := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		bNewF := s.app.BankKeeper.GetBalance(s.ctx, newFunder, stakeDenom)
		bG := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)

		// Original funder balance should not change
		Expect(bF).To(Equal(balanceFunder))
		// New funder should get the vested tokens
		Expect(balanceNewFunder.Add(vestingAmtTotal[0]).Amount.Uint64()).To(Equal(bNewF.Amount.Uint64()))
		Expect(balanceGrantee.Sub(vestingAmtTotal[0]).Amount.Uint64()).To(Equal(bG.Amount.Uint64()))
	})

	It("should update vesting funder and first funder cannot claw back unvested before cliff", func() {
		ctx := sdk.WrapSDKContext(s.ctx)
		newFunder := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
		balanceFunder := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		balanceNewFunder := s.app.BankKeeper.GetBalance(s.ctx, newFunder, stakeDenom)
		balanceGrantee := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)

		// Update clawback vesting account funder
		updateFunderMsg := types.NewMsgUpdateVestingFunder(funder, newFunder, vestingAddr)
		_, err := s.app.VestingKeeper.UpdateVestingFunder(ctx, updateFunderMsg)
		Expect(err).To(BeNil())

		// Original funder tries to perform clawback before cliff - is not the current funder
		msg := types.NewMsgClawback(funder, vestingAddr, sdk.AccAddress([]byte{}))
		_, err = s.app.VestingKeeper.Clawback(ctx, msg)
		Expect(err).NotTo(BeNil())

		// All balances should remain the same
		bF := s.app.BankKeeper.GetBalance(s.ctx, funder, stakeDenom)
		bNewF := s.app.BankKeeper.GetBalance(s.ctx, newFunder, stakeDenom)
		bG := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, stakeDenom)

		Expect(bF).To(Equal(balanceFunder))
		Expect(balanceNewFunder).To(Equal(bNewF))
		Expect(balanceGrantee).To(Equal(bG))
	})
})

// Testing that smart contracts cannot be converted to clawback vesting accounts
//
// NOTE: For smart contracts, it is not possible to directly call keeper methods
// or send SDK transactions. They go exclusively through the EVM, which is tested
// in the precompiles package.
// The test here is just confirming the expected behavior on the module level.
var _ = Describe("Clawback Vesting Account - Smart contract", func() {
	var (
		contractAddr common.Address
		contract     evmtypes.CompiledContract
		err          error
	)

	BeforeEach(func() {
		s.SetupTest()

		contract = contracts.ERC20MinterBurnerDecimalsContract
		contractAddr, err = testutil.DeployContract(
			s.ctx,
			s.app,
			s.priv,
			s.queryClientEvm,
			contract,
			"Test", "TTT", uint8(18),
		)
		Expect(err).ToNot(HaveOccurred(), "failed to deploy contract")
	})

	It("should not convert a smart contract to a clawback vesting account", func() {
		msgConv := types.NewMsgConvertIntoVestingAccount(
			s.address.Bytes(),
			contractAddr.Bytes(),
			s.ctx.BlockTime(),
			testutil.TestVestingSchedule.LockupPeriods,
			testutil.TestVestingSchedule.VestingPeriods,
			true,
			false,
			nil,
		)
		_, err := s.app.VestingKeeper.ConvertIntoVestingAccount(s.ctx, msgConv)
		Expect(err).To(HaveOccurred(), "expected error")
		Expect(err.Error()).To(ContainSubstring(
			fmt.Sprintf(
				"account %s is a contract account and cannot be converted in a clawback vesting account",
				sdk.AccAddress(contractAddr.Bytes()).String()),
		))
		// Check that the account was not converted
		acc := s.app.AccountKeeper.GetAccount(s.ctx, contractAddr.Bytes())
		Expect(acc).ToNot(BeNil(), "smart contract should be found")
		_, ok := acc.(*types.ClawbackVestingAccount)
		Expect(ok).To(BeFalse(), "account should not be a clawback vesting account")
		// Check that the contract code was not deleted
		//
		// NOTE: When it was possible to create clawback vesting accounts for smart contracts,
		// the contract code was deleted from the EVM state. This checks that this is not the case.
		res, err := s.app.EvmKeeper.Code(s.ctx, &evmtypes.QueryCodeRequest{Address: contractAddr.String()})
		Expect(err).ToNot(HaveOccurred(), "failed to query contract code")
		Expect(res.Code).ToNot(BeEmpty(), "contract code should not be empty")
	})
})

// Trying to replicate the faulty behavior in MsgCreateClawbackVestingAccount,
// that was disclosed as a potential attack vector in relation to the Barberry
// security patch.
//
// It was possible to fund a clawback vesting account with negative amounts.
// Avoiding this requires an additional validation of the amount in the
// MsgFundVestingAccount's ValidateBasic method.
var _ = Describe("Clawback Vesting Account - Barberry bug", func() {
	var (
		// coinsNoNegAmount is a Coins struct with a positive and a negative amount of the same
		// denomination.
		coinsNoNegAmount = sdk.Coins{
			sdk.Coin{Denom: utils.BaseDenom, Amount: math.NewInt(1e18)},
		}
		// coinsWithNegAmount is a Coins struct with a positive and a negative amount of the same
		// denomination.
		coinsWithNegAmount = sdk.Coins{
			sdk.Coin{Denom: utils.BaseDenom, Amount: math.NewInt(1e18)},
			sdk.Coin{Denom: utils.BaseDenom, Amount: math.NewInt(-1e18)},
		}
		// coinsWithZeroAmount is a Coins struct with a positive and a zero amount of the same
		// denomination.
		coinsWithZeroAmount = sdk.Coins{
			sdk.Coin{Denom: utils.BaseDenom, Amount: math.NewInt(1e18)},
			sdk.Coin{Denom: utils.BaseDenom, Amount: math.NewInt(0)},
		}
		// emptyCoins is an Coins struct
		emptyCoins = sdk.Coins{}
		// funder and funderPriv are the address and private key of the account funding the vesting account
		funder, funderPriv = utiltx.NewAccAddressAndKey()
		// gasPrice is the gas price to be used in the transactions executed by the vesting account so that
		// the transaction fees can be deducted from the expected account balance
		gasPrice = math.NewInt(1e9)
		// vestingAddr and vestingPriv are the address and private key of the vesting account to be created
		vestingAddr, _ = utiltx.NewAccAddressAndKey()
		// vestingLength is a period of time in seconds to be used for the creation of the vesting
		// account.
		vestingLength = int64(60 * 60 * 24 * 30) // 30 days in seconds
	)

	BeforeEach(func() {
		s.SetupTest()

		// Initialize the account at the vesting address and the funder accounts by funding them
		fundedCoins := sdk.Coins{{Denom: utils.BaseDenom, Amount: math.NewInt(2e18)}} // fund more than what is sent to the vesting account for transaction fees
		err = testutil.FundAccount(s.ctx, s.app.BankKeeper, funder, fundedCoins)
		Expect(err).ToNot(HaveOccurred(), "failed to fund account")
	})

	Context("when funding a clawback vesting account", func() {
		testcases := []struct {
			name         string
			lockupCoins  sdk.Coins
			vestingCoins sdk.Coins
			expError     bool
			errContains  string
		}{
			{
				name:        "pass - positive amounts for the lockup period",
				lockupCoins: coinsNoNegAmount,
				expError:    false,
			},
			{
				name:         "pass - positive amounts for the vesting period",
				vestingCoins: coinsNoNegAmount,
				expError:     false,
			},
			{
				name:         "pass - positive amounts for both the lockup and vesting periods",
				lockupCoins:  coinsNoNegAmount,
				vestingCoins: coinsNoNegAmount,
				expError:     false,
			},
			{
				name:        "fail - negative amounts for the lockup period",
				lockupCoins: coinsWithNegAmount,
				expError:    true,
				errContains: errortypes.ErrInvalidCoins.Wrap(coinsWithNegAmount.String()).Error(),
			},
			{
				name:         "fail - negative amounts for the vesting period",
				vestingCoins: coinsWithNegAmount,
				expError:     true,
				errContains:  "invalid coins: invalid request",
			},
			{
				name:        "fail - zero amount for the lockup period",
				lockupCoins: coinsWithZeroAmount,
				expError:    true,
				errContains: errortypes.ErrInvalidCoins.Wrap(coinsWithZeroAmount.String()).Error(),
			},
			{
				name:         "fail - zero amount for the vesting period",
				vestingCoins: coinsWithZeroAmount,
				expError:     true,
				errContains:  "invalid coins: invalid request",
			},
			{
				name:         "fail - empty amount for both the lockup and vesting periods",
				lockupCoins:  emptyCoins,
				vestingCoins: emptyCoins,
				expError:     true,
				errContains:  "vesting and/or lockup schedules must be present",
			},
		}
		for _, tc := range testcases {
			It(tc.name, func() {
				var (
					lockupPeriods  sdkvesting.Periods
					vestingPeriods sdkvesting.Periods
				)
				if !tc.lockupCoins.Empty() {
					lockupPeriods = sdkvesting.Periods{
						sdkvesting.Period{Length: vestingLength, Amount: tc.lockupCoins},
					}
				}
				if !tc.vestingCoins.Empty() {
					vestingPeriods = sdkvesting.Periods{
						sdkvesting.Period{Length: vestingLength, Amount: tc.vestingCoins},
					}
				}

				// Create a clawback vesting account
				msgCreate := types.NewMsgCreateClawbackVestingAccount(
					funder,
					vestingAddr,
					s.ctx.BlockTime(),
					lockupPeriods,
					vestingPeriods,
					false,
				)
				// Deliver transaction with message
				res, err := testutil.DeliverTx(s.ctx, s.app, funderPriv, &gasPrice, signing.SignMode_SIGN_MODE_DIRECT, msgCreate)
				// Get account at the new address
				acc := s.app.AccountKeeper.GetAccount(s.ctx, vestingAddr)
				if tc.expError {
					Expect(err).To(HaveOccurred(), "expected funding the vesting account to have failed")
					Expect(err.Error()).To(ContainSubstring(tc.errContains), "expected funding the vesting account to have failed")
					Expect(acc).To(BeNil(), "expected clawback vesting account to not have been created")
				} else {
					Expect(acc).NotTo(BeNil(), "expected clawback vesting account should have been created")
					vacc, _ := acc.(*types.ClawbackVestingAccount)
					Expect(err).ToNot(HaveOccurred(), "failed to fund clawback vesting account")
					Expect(res.Code).To(Equal(uint32(0)), "failed to fund clawback vesting account")
					Expect(vacc.LockupPeriods).ToNot(BeEmpty(), "vesting account should have been funded")
					// Check that the vesting account has the correct balance
					balance := s.app.BankKeeper.GetBalance(s.ctx, vestingAddr, utils.BaseDenom)
					expBalance := int64(1e18) // vestingCoins
					Expect(balance.Amount.Int64()).To(Equal(expBalance), "vesting account has incorrect balance")
				}
			})
		}
	})
})

var _ = Describe("Clawback Vesting Accounts - Track delegations", func() {
	// Create test accounts with private keys for signing
	numTestAccounts := 4
	testAccounts := make([]TestClawbackAccount, numTestAccounts)
	for i := range testAccounts {
		address, privKey := utiltx.NewAddrKey()
		testAccounts[i] = TestClawbackAccount{
			privKey: privKey,
			address: address.Bytes(),
		}
	}

	var (
		clawbackAccount *types.ClawbackVestingAccount
		isClawback      bool
	)

	// Prepare custom test amounts
	initialFreeBalance := sdk.NewCoins(sdk.NewCoin(stakeDenom, math.NewInt(1e18).MulRaw(10)))

	BeforeEach(func() {
		s.SetupTest()

		// Initialize all test accounts
		for _, account := range testAccounts {
			err := testutil.FundAccount(s.ctx, s.app.BankKeeper, account.address, initialFreeBalance)
			Expect(err).To(BeNil())
			accBal := s.app.BankKeeper.GetBalance(s.ctx, account.address, stakeDenom)
			Expect(accBal).To(Equal(initialFreeBalance[0]))
		}

		// Add a commit to instantiate blocks
		s.Commit()
	})

	It("Has delegation before conversion and free spendable coins", func() {
		vacc := testAccounts[0]
		vaccAddr := vacc.address
		vaccPrivKey := vacc.privKey

		// Add coins to be delegated
		err := testutil.FundAccount(s.ctx, s.app.BankKeeper, vaccAddr, initialFreeBalance)
		Expect(err).To(BeNil())
		s.Commit()

		// Should be doubled initialFreeBalance
		vaccBal := s.app.BankKeeper.GetBalance(s.ctx, vaccAddr, stakeDenom)
		Expect(vaccBal).To(Equal(initialFreeBalance.Add(initialFreeBalance...)[0]))

		// should be equal to account balance
		vaccSpendableCoins := s.app.BankKeeper.SpendableCoins(s.ctx, vaccAddr)
		Expect(vaccSpendableCoins).ToNot(BeNil())
		Expect(vaccSpendableCoins[0]).To(Equal(vaccBal))

		// delegate half of the balance
		_, err = testutil.Delegate(s.ctx, s.app, vaccPrivKey, initialFreeBalance[0], s.validator)
		Expect(err).To(BeNil())
		s.Commit()

		// Check delegation
		vaccBalAfterDelegation := s.app.BankKeeper.GetBalance(s.ctx, vaccAddr, stakeDenom)
		Expect(vaccBalAfterDelegation.IsLT(initialFreeBalance[0])).To(BeTrue())

		bondedAmt := s.app.StakingKeeper.GetDelegatorBonded(s.ctx, vaccAddr)
		unbondingAmt := s.app.StakingKeeper.GetDelegatorUnbonding(s.ctx, vaccAddr)
		delegatedAmt := bondedAmt.Add(unbondingAmt)
		Expect(delegatedAmt).To(Equal(initialFreeBalance[0].Amount))

		vaccSpendableCoinsAfterDelegation := s.app.BankKeeper.SpendableCoins(s.ctx, vaccAddr)
		Expect(vaccSpendableCoinsAfterDelegation).ToNot(BeNil())
		Expect(vaccSpendableCoinsAfterDelegation[0].IsLT(initialFreeBalance[0])).To(BeTrue())

		// test vesting account creation
		vaccBalBeforeVesting := s.app.BankKeeper.GetBalance(s.ctx, vaccAddr, stakeDenom)
		Expect(vaccBalBeforeVesting).To(Equal(vaccBalAfterDelegation))

		vaccSpendableCoinsBeforeVesting := s.app.BankKeeper.SpendableCoins(s.ctx, vaccAddr)
		Expect(vaccSpendableCoinsBeforeVesting).To(Equal(vaccSpendableCoinsAfterDelegation))

		vestingStart := s.ctx.BlockTime()
		msg := types.NewMsgConvertIntoVestingAccount(
			testAccounts[1].address,
			vaccAddr,
			vestingStart,
			testutil.TestVestingSchedule.LockupPeriods,
			testutil.TestVestingSchedule.VestingPeriods,
			true, false, nil,
		)
		_, err = testutil.DeliverTx(s.ctx, s.app, testAccounts[1].privKey, nil, signing.SignMode_SIGN_MODE_DIRECT, msg)
		Expect(err).ToNot(HaveOccurred(), "expected creating clawback vesting account to succeed")
		s.Commit()

		funderAccBalAfterVesting := s.app.BankKeeper.GetBalance(s.ctx, testAccounts[1].address, stakeDenom)
		Expect(funderAccBalAfterVesting.IsLT(initialFreeBalance[0].Sub(vestingAmtTotal[0]))).To(BeTrue())

		vaccBalAfterVesting := s.app.BankKeeper.GetBalance(s.ctx, vaccAddr, stakeDenom)
		Expect(vaccBalAfterVesting).To(Equal(vaccBalBeforeVesting.Add(vestingAmtTotal[0])))

		vaccSpendableCoinsAfterVesting := s.app.BankKeeper.SpendableCoins(s.ctx, vaccAddr)
		Expect(vaccSpendableCoinsAfterVesting).To(Equal(vaccSpendableCoinsBeforeVesting))

		acc := s.app.AccountKeeper.GetAccount(s.ctx, testAccounts[0].address)
		clawbackAccount, isClawback = acc.(*types.ClawbackVestingAccount)
		Expect(isClawback).To(BeTrue(), "expected account to be clawback vesting account")

		Expect(clawbackAccount.DelegatedVesting).To(BeNil())
		Expect(clawbackAccount.DelegatedFree).ToNot(BeNil())
		Expect(clawbackAccount.DelegatedFree[0]).To(Equal(initialFreeBalance[0]))

		// test spendable coins after vesting cliff
		s.CommitAfter(time.Duration(cliffLength)*time.Second + 1)

		vaccBalAfterCliff := s.app.BankKeeper.GetBalance(s.ctx, vaccAddr, stakeDenom)
		Expect(vaccBalAfterCliff).To(Equal(vaccBalAfterVesting))

		vaccSpendableCoinsAfterCliff := s.app.BankKeeper.SpendableCoins(s.ctx, vaccAddr)
		Expect(vaccSpendableCoinsAfterCliff).To(Equal(vaccSpendableCoinsAfterVesting.Add(clawbackAccount.GetLockedUpVestedCoins(s.ctx.BlockTime())...)))
	})
})
