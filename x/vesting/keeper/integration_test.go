package keeper_test

import (
	"math/big"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/encoding"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/testutil/integration/common/factory"
	haqqfactory "github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	"github.com/haqq-network/haqq/x/vesting/types"
)

type KeeperIntegrationTestSuite struct {
	suite.Suite

	network     *network.UnitTestNetwork
	factory     haqqfactory.TxFactory
	grpcHandler grpc.Handler
	keyring     keyring.Keyring
}

func TestKeeperIntegrationTestSuite(t *testing.T) {
	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Integration Test Suite")
}

// Initialize general error variable for easier handling in loops throughout this test suite.
var (
	numTestMsgs                     = 3
	vestingAccInitialBalance        = network.PrefundedAccountInitialBalance
	remainingAmtToPayFees           = math.NewInt(1e16)
	gasLimit                 uint64 = 500_000
	gasPrice                        = remainingAmtToPayFees.QuoRaw(int64(gasLimit))
	dest                            = utiltx.GenerateAddress()
	stakeDenom                      = utils.BaseDenom
	accountGasCoverage              = sdk.NewCoins(sdk.NewCoin(stakeDenom, remainingAmtToPayFees))
	amt                             = testutil.TestVestingSchedule.VestedCoinsPerPeriod[0].Amount
	cliff                           = testutil.TestVestingSchedule.CliffMonths
	cliffLength                     = testutil.TestVestingSchedule.CliffPeriodLength
	vestingAmtTotal                 = testutil.TestVestingSchedule.TotalVestingCoins
	vestingLength                   = testutil.TestVestingSchedule.VestingPeriodLength
	numLockupPeriods                = testutil.TestVestingSchedule.NumLockupPeriods
	periodsTotal                    = testutil.TestVestingSchedule.NumVestingPeriods
	lockup                          = testutil.TestVestingSchedule.LockupMonths
	lockupLength                    = testutil.TestVestingSchedule.LockupPeriodLength
	unlockedPerLockup               = testutil.TestVestingSchedule.UnlockedCoinsPerLockup
	unlockedPerLockupAmt            = unlockedPerLockup[0].Amount
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
	var (
		s                 *KeeperIntegrationTestSuite
		funder            keyring.Key
		vestingAccs       []keyring.Key
		clawbackAccount   *types.ClawbackVestingAccount
		unvested          sdk.Coins
		vested            sdk.Coins
		freeCoins         sdk.Coins
		twoThirdsOfVested sdk.Coins
		initialFreeCoins  sdk.Coins
	)

	BeforeEach(func() {
		s = new(KeeperIntegrationTestSuite)
		// create 5 prefunded accounts:
		keys := keyring.New(5)
		nw := network.NewUnitTestNetwork(
			network.WithPreFundedAccounts(keys.GetAllAccAddrs()...),
		)
		gh := grpc.NewIntegrationHandler(nw)
		tf := haqqfactory.New(nw, gh)

		s.network = nw
		s.factory = tf
		s.grpcHandler = gh
		s.keyring = keys

		// index 0 will be the funder
		// index 1-4 will be vesting accounts
		funder = keys.GetKey(0)
		vestingAccs = keys.GetKeys()[1:4]

		// Initialize all vesting accounts
		for _, account := range vestingAccs {
			// Create and fund periodic vesting account
			clawbackAccount = s.setupClawbackVestingAccount(account, funder, testutil.TestVestingSchedule.VestingPeriods, testutil.TestVestingSchedule.LockupPeriods)

			// Check if all tokens are unvested at vestingStart
			unvested = clawbackAccount.GetUnvestedOnly(s.network.GetContext().BlockTime())
			vested = clawbackAccount.GetVestedOnly(s.network.GetContext().BlockTime())
			Expect(vestingAmtTotal).To(Equal(unvested))
			Expect(vested.IsZero()).To(BeTrue())
		}

		initialFreeCoins = sdk.NewCoins(sdk.NewCoin(stakeDenom, vestingAccInitialBalance.Sub(remainingAmtToPayFees)))
	})

	Context("before first vesting period", func() {
		BeforeEach(func() {
			// Ensure no tokens are vested
			vested := clawbackAccount.GetVestedOnly(s.network.GetContext().BlockTime())
			unlocked := clawbackAccount.GetUnlockedOnly(s.network.GetContext().BlockTime())
			zeroCoins := sdk.NewCoins(sdk.NewCoin(stakeDenom, math.ZeroInt()))
			Expect(zeroCoins).To(Equal(vested))
			Expect(zeroCoins).To(Equal(unlocked))
		})

		It("cannot delegate unvested tokens", func() {
			err := s.factory.Delegate(
				vestingAccs[0].Priv,
				s.network.GetValidators()[0].OperatorAddress,
				sdk.NewCoin(stakeDenom, vestingAccInitialBalance.Add(math.NewInt(1))),
			)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("insufficient vested coins"))
		})

		It("can delegate spendable tokens", func() {
			// Try to delegate available spendable balance
			// All vesting tokens are unvested at this point
			// Only 1e16 aISLM is available for gas coverage
			err := s.factory.Delegate(
				vestingAccs[0].Priv,
				s.network.GetValidators()[0].OperatorAddress,
				sdk.NewCoin(stakeDenom, math.NewInt(1e15)),
			)
			// Must succeed as spendable balance is sufficient
			Expect(err).To(BeNil())
		})

		It("can transfer spendable tokens", func() {
			account := vestingAccs[0]
			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePrev := balRes.Balance

			sendAmt := vestingAccInitialBalance.Sub(remainingAmtToPayFees.MulRaw(2))
			spendableCoin := sdk.NewCoin(stakeDenom, sendAmt)
			coins := sdk.NewCoins(spendableCoin)
			msg := banktypes.NewMsgSend(account.AccAddr, dest.Bytes(), coins)
			res, err := s.factory.ExecuteCosmosTx(account.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice, Gas: &gasLimit})
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())
			Expect(s.network.NextBlock()).To(BeNil())

			// check final balance is as expected - transferred spendable tokens
			fees := gasPrice.Mul(math.NewInt(res.GasWanted))
			balRes, err = s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePost := balRes.Balance
			Expect(balancePost.Amount).To(Equal(balancePrev.Amount.Sub(fees).Sub(sendAmt)))

			balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
			Expect(err).To(BeNil())
			destBalance := balRes.Balance
			Expect(destBalance.Amount).To(Equal(sendAmt))
		})

		It("cannot transfer unvested tokens", func() {
			account := vestingAccs[0]
			coins := unvested.Add(sdk.NewCoin(stakeDenom, vestingAccInitialBalance.Sub(remainingAmtToPayFees)))
			msg := banktypes.NewMsgSend(account.AccAddr, dest.Bytes(), coins)
			_, err := s.factory.ExecuteCosmosTx(account.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("spendable balance"))
			Expect(err.Error()).To(ContainSubstring("is smaller than"))
		})

		It("can perform Ethereum tx with spendable balance", func() {
			account := vestingAccs[0]

			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePrev := balRes.Balance

			sendAmt := vestingAccInitialBalance.Sub(remainingAmtToPayFees.MulRaw(2))

			res, err := s.factory.ExecuteEthTx(account.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), Amount: sendAmt.BigInt()})
			Expect(err).To(BeNil())
			Expect(s.network.NextBlock()).To(BeNil())

			// check final balance is as expected - transferred spendable tokens
			fees := gasPrice.Mul(math.NewInt(res.GasWanted))
			balRes, err = s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePost := balRes.Balance
			Expect(balancePost.Amount).To(Equal(balancePrev.Amount.Sub(sendAmt).Sub(fees)))

			balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
			Expect(err).To(BeNil())
			destBalance := balRes.Balance
			Expect(destBalance.Amount).To(Equal(sendAmt))
		})

		It("cannot perform Ethereum tx with unvested balance", func() {
			account := vestingAccs[0]
			txAmount := vestingAccInitialBalance.Sub(remainingAmtToPayFees).Add(unvested.AmountOf(stakeDenom)).BigInt()

			res, err := s.factory.ExecuteEthTx(account.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), Amount: txAmount})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("clawback vesting account has insufficient unlocked tokens"))
			Expect(res.IsOK()).To(BeFalse())
		})
	})

	Context("after first vesting period and before lockup", func() {
		BeforeEach(func() {
			// Surpass cliff but none of lockup duration
			cliffDuration := time.Duration(cliffLength)
			Expect(s.network.NextBlockAfter(cliffDuration * time.Second)).To(BeNil())

			acc, err := s.grpcHandler.GetAccount(vestingAccs[0].AccAddr.String())
			Expect(err).To(BeNil())
			var ok bool
			clawbackAccount, ok = acc.(*types.ClawbackVestingAccount)
			Expect(ok).To(BeTrue())

			// Check if some, but not all tokens are vested
			vested = clawbackAccount.GetVestedOnly(s.network.GetContext().BlockTime())
			expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, amt.Mul(math.NewInt(cliff))))
			Expect(vestingAmtTotal).NotTo(Equal(vested))
			Expect(expVested).To(Equal(vested))

			// check the vested tokens are still locked
			freeCoins = clawbackAccount.GetVestedCoins(s.network.GetContext().BlockTime())
			Expect(freeCoins.IsZero()).To(BeTrue())

			twoThirdsOfVested = vested.Sub(vested.QuoInt(math.NewInt(3))...)

			qc := s.network.GetVestingClient()
			res, err := qc.Balances(s.network.GetContext(), &types.QueryBalancesRequest{Address: clawbackAccount.Address})
			Expect(err).To(BeNil())
			Expect(res.Vested).To(Equal(expVested))
			Expect(res.Unvested).To(Equal(vestingAmtTotal.Sub(expVested...)))
			// All coins from vesting schedule should be locked
			Expect(res.Locked).To(Equal(vestingAmtTotal))
		})

		It("can delegate vested locked tokens", func() {
			account := vestingAccs[0]
			// Verify that the total spendable coins should only be coins
			// not in the vesting schedule. Because all coins from the vesting
			// schedule are still locked
			res, err := s.grpcHandler.GetSpendableBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			spendablePre := res.Balance
			Expect(*spendablePre).To(Equal(initialFreeCoins[0]))

			// delegate the vested locked coins.
			err = s.factory.Delegate(account.Priv, s.network.GetValidators()[0].OperatorAddress, vested[0])
			Expect(err).To(BeNil(), "expected no error during delegation")
			Expect(s.network.NextBlock()).To(BeNil())

			// check spendable coins have only been reduced by the gas paid for the transaction to show that the delegated coins were taken from the locked but vested amount
			res, err = s.grpcHandler.GetSpendableBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			spendablePost := res.Balance
			Expect(*spendablePost).To(Equal(spendablePre.Sub(accountGasCoverage[0])))

			// check delegation was created successfully
			stkQuerier := s.network.GetStakingClient()
			delRes, err := stkQuerier.DelegatorDelegations(s.network.GetContext(), &stakingtypes.QueryDelegatorDelegationsRequest{DelegatorAddr: account.AccAddr.String()})
			Expect(err).To(BeNil())
			Expect(delRes.DelegationResponses).To(HaveLen(1))
			Expect(delRes.DelegationResponses[0].Balance.Amount).To(Equal(vested[0].Amount))
		})

		It("account with free balance - delegates the free balance amount. It is tracked as locked vested tokens for the spendable balance calculation", func() {
			account := vestingAccs[0]

			// vesting account has some initial balance
			coinsToDelegate := sdk.NewCoins(sdk.NewCoin(stakeDenom, math.NewInt(1e18)))
			// check that coins to delegate are greater than the locked up vested coins
			Expect(coinsToDelegate.IsAllGT(vested)).To(BeTrue())

			// the free coins delegated will be the delegatedCoins - lockedUp vested coins
			freeCoinsDelegated := coinsToDelegate.Sub(vested...)

			balRes, err := s.grpcHandler.GetAllBalances(account.AccAddr)
			Expect(err).To(BeNil())
			initialBalances := balRes.Balances
			Expect(initialBalances).To(Equal(testutil.TestVestingSchedule.TotalVestingCoins.Add(initialFreeCoins...)))
			// Verify that the total spendable coins should only be coins
			// not in the vesting schedule. Because all coins from the vesting
			// schedule are still locked up
			spRes, err := s.grpcHandler.GetSpendableBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			spendablePre := spRes.Balance
			Expect(*spendablePre).To(Equal(initialFreeCoins[0]))

			// delegate funds - the delegation amount will be tracked as locked up vested coins delegated + some free coins
			err = s.factory.Delegate(account.Priv, s.network.GetValidators()[0].OperatorAddress, coinsToDelegate[0])
			Expect(err).NotTo(HaveOccurred(), "expected no error during delegation")
			Expect(s.network.NextBlock()).To(BeNil())

			// check balances updated properly
			balRes, err = s.grpcHandler.GetAllBalances(account.AccAddr)
			Expect(err).To(BeNil())
			finalBalances := balRes.Balances
			Expect(finalBalances).To(Equal(initialBalances.Sub(coinsToDelegate...).Sub(accountGasCoverage...)))

			// the expected spendable balance will be
			// spendable = bank balances - (coins in vesting schedule - unlocked vested coins (0) - locked up vested coins delegated)
			expSpendable := finalBalances.Sub(testutil.TestVestingSchedule.TotalVestingCoins...).Add(vested...)

			// which should be equal to the initial freeCoins - freeCoins delegated
			Expect(expSpendable).To(Equal(initialFreeCoins.Sub(freeCoinsDelegated...).Sub(accountGasCoverage...)))

			// check spendable balance is updated properly
			res, err := s.grpcHandler.GetSpendableBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			spendablePost := res.Balance
			Expect(*spendablePost).To(Equal(expSpendable[0]))
		})

		It("cannot delegate unvested tokens in sequetial txs", func() {
			coinsToDelegate := sdk.NewCoin(stakeDenom, vestingAccInitialBalance.Sub(remainingAmtToPayFees.MulRaw(2))).Add(twoThirdsOfVested[0])
			err := s.factory.Delegate(vestingAccs[0].Priv, s.network.GetValidators()[0].OperatorAddress, coinsToDelegate)
			Expect(err).To(BeNil(), "error while executing the delegate message")
			Expect(s.network.NextBlock()).To(BeNil())

			err = s.factory.Delegate(vestingAccs[0].Priv, s.network.GetValidators()[0].OperatorAddress, twoThirdsOfVested[0])
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("cannot delegate unvested coins"))
		})

		It("cannot delegate then send tokens", func() {
			account := vestingAccs[0]
			coinsToDelegate := sdk.NewCoin(stakeDenom, vestingAccInitialBalance.Sub(remainingAmtToPayFees.MulRaw(2))).Add(twoThirdsOfVested[0])
			err := s.factory.Delegate(account.Priv, s.network.GetValidators()[0].OperatorAddress, coinsToDelegate)
			Expect(err).To(BeNil())
			Expect(s.network.NextBlock()).To(BeNil())

			msg := banktypes.NewMsgSend(account.AccAddr, dest.Bytes(), twoThirdsOfVested)
			_, err = s.factory.ExecuteCosmosTx(account.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("spendable balance"))
			Expect(err.Error()).To(ContainSubstring("is smaller than"))
		})

		It("cannot delegate more than the locked vested tokens", func() {
			coinsToDelegate := vested.Add(sdk.NewCoin(stakeDenom, vestingAccInitialBalance.Sub(remainingAmtToPayFees))).Add(sdk.NewCoin(stakeDenom, math.OneInt()))
			err := s.factory.Delegate(vestingAccs[0].Priv, s.network.GetValidators()[0].OperatorAddress, coinsToDelegate[0])
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("cannot delegate unvested coins"))
		})

		It("cannot delegate free tokens and then send locked/unvested tokens", func() {
			account := vestingAccs[0]
			// send some funds to the account to delegate
			coinsToDelegate := vested.Add(initialFreeCoins...).Sub(accountGasCoverage...)

			err := s.factory.Delegate(account.Priv, s.network.GetValidators()[0].OperatorAddress, coinsToDelegate[0])
			Expect(err).To(BeNil())
			Expect(s.network.NextBlock()).To(BeNil())

			sendCoins := twoThirdsOfVested

			msg := banktypes.NewMsgSend(account.AccAddr, dest.Bytes(), sendCoins)
			_, err = s.factory.ExecuteCosmosTx(account.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("spendable balance"))
			Expect(err.Error()).To(ContainSubstring("is smaller than"))
		})

		It("cannot transfer locked vested tokens", func() {
			msg := banktypes.NewMsgSend(vestingAccs[0].AccAddr, dest.Bytes(), vested.Add(sdk.NewCoin(stakeDenom, vestingAccInitialBalance.Sub(remainingAmtToPayFees))))
			_, err := s.factory.ExecuteCosmosTx(vestingAccs[0].Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("spendable balance"))
			Expect(err.Error()).To(ContainSubstring("is smaller than"))
		})

		It("can perform Ethereum tx with spendable balance", func() {
			account := vestingAccs[0]

			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePrev := balRes.Balance

			availableCoins := initialFreeCoins.Sub(accountGasCoverage...)
			txAmount := availableCoins[0].Amount
			res, err := s.factory.ExecuteEthTx(account.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), Amount: txAmount.BigInt()})
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())
			Expect(s.network.NextBlock()).To(BeNil())

			// check final balance is as expected - transferred spendable tokens
			fees := gasPrice.Mul(math.NewInt(res.GasWanted))
			balRes, err = s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePost := balRes.Balance
			Expect(balancePost.Amount).To(Equal(balancePrev.Amount.Sub(fees).Sub(txAmount)))

			balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
			Expect(err).To(BeNil())
			destBalance := balRes.Balance
			Expect(destBalance.Amount).To(Equal(txAmount))
		})

		It("cannot perform Ethereum tx with locked vested balance", func() {
			account := vestingAccs[0]
			txAmount := vestingAccInitialBalance.Add(vested.AmountOf(stakeDenom)).Sub(remainingAmtToPayFees)
			res, err := s.factory.ExecuteEthTx(account.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), Amount: txAmount.BigInt()})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("clawback vesting account has insufficient unlocked tokens to execute transaction"))
			Expect(res.IsOK()).To(BeFalse())
		})
	})

	Context("Between first and second lockup periods", func() {
		BeforeEach(func() {
			// Surpass first lockup
			vestDuration := time.Duration(testutil.TestVestingSchedule.LockupPeriodLength)
			Expect(s.network.NextBlockAfter(vestDuration * time.Second)).To(BeNil())

			// after first lockup period
			// half of total vesting tokens are unlocked
			// but only 12 vesting periods passed
			// Check if some, but not all tokens are vested and unlocked
			for _, account := range vestingAccs {
				acc, err := s.grpcHandler.GetAccount(account.AccAddr.String())
				Expect(err).To(BeNil())
				vestAcc, ok := acc.(*types.ClawbackVestingAccount)
				Expect(ok).To(BeTrue())

				vested = vestAcc.GetVestedOnly(s.network.GetContext().BlockTime())
				unlocked := vestAcc.GetUnlockedOnly(s.network.GetContext().BlockTime())
				freeCoins = vestAcc.GetVestedCoins(s.network.GetContext().BlockTime())
				expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, amt.Mul(math.NewInt(lockup))))
				expUnlockedVested := expVested

				Expect(vested).NotTo(Equal(vestingAmtTotal))
				Expect(vested).To(Equal(expVested))
				Expect(unlocked).To(Equal(unlockedPerLockup))
				Expect(freeCoins).To(Equal(expUnlockedVested))
			}
		})

		It("delegate unlocked vested tokens and spendable balance is updated properly", func() {
			account := vestingAccs[0]

			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balance := balRes.Balance
			// the returned balance should be the account's initial balance and
			// the total amount of the vesting schedule
			Expect(balance.Amount).To(Equal(initialFreeCoins.Add(vestingAmtTotal...)[0].Amount))

			spRes, err := s.grpcHandler.GetSpendableBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			// spendable balance should be the initial account balance + vested tokens
			initialSpendableBalance := spRes.Balance
			Expect(initialSpendableBalance.Amount).To(Equal(initialFreeCoins.Add(freeCoins...)[0].Amount))

			// can delegate vested tokens
			// fees paid is accountGasCoverage amount
			coinsToDelegate := freeCoins.Add(initialFreeCoins...).Sub(accountGasCoverage...)
			err = s.factory.Delegate(account.Priv, s.network.GetValidators()[0].OperatorAddress, coinsToDelegate[0])
			Expect(err).ToNot(HaveOccurred(), "expected no error during delegation")
			Expect(s.network.NextBlock()).To(BeNil())

			// spendable balance should be updated to be prevSpendableBalance - delegatedAmt - fees
			spRes, err = s.grpcHandler.GetSpendableBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			Expect(spRes.Balance.Amount.Int64()).To(Equal(int64(0)))

			// try to send coins - should error
			msg := banktypes.NewMsgSend(account.AccAddr, dest.Bytes(), vested)
			_, err = s.factory.ExecuteCosmosTx(account.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("spendable balance"))
			Expect(err.Error()).To(ContainSubstring("is smaller than"))
		})

		It("cannot delegate more than vested tokens", func() {
			account := vestingAccs[0]
			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balance := balRes.Balance
			// the returned balance should be the account's initial balance and
			// the total amount of the vesting schedule
			Expect(balance.Amount).To(Equal(initialFreeCoins.Add(vestingAmtTotal...)[0].Amount))

			spRes, err := s.grpcHandler.GetSpendableBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			// spendable balance should be the initial account balance + vested tokens
			initialSpendableBalance := spRes.Balance
			Expect(initialSpendableBalance.Amount).To(Equal(initialFreeCoins.Add(freeCoins...)[0].Amount))

			// cannot delegate more than vested tokens
			coinsToDelegate := freeCoins.Add(initialFreeCoins...).Add(sdk.NewCoin(stakeDenom, math.OneInt())).Sub(accountGasCoverage...)
			err = s.factory.Delegate(account.Priv, s.network.GetValidators()[0].OperatorAddress, coinsToDelegate[0])
			Expect(err).To(HaveOccurred(), "expected no error during delegation")
			Expect(err.Error()).To(ContainSubstring("cannot delegate unvested coins"))
		})

		It("should enable access to unlocked and vested EVM tokens (single-account, single-msg)", func() {
			account := vestingAccs[0]

			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePrev := balRes.Balance

			// the freeCoins are the unlocked vested coins
			txAmount := initialFreeCoins.Add(freeCoins...).Sub(accountGasCoverage...)[0].Amount
			res, err := s.factory.ExecuteEthTx(account.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), Amount: txAmount.BigInt()})
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())
			Expect(s.network.NextBlock()).To(BeNil())

			// check final balance is as expected - transferred spendable tokens
			fees := gasPrice.Mul(math.NewInt(res.GasWanted))
			balRes, err = s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePost := balRes.Balance
			Expect(balancePost.Amount).To(Equal(balancePrev.Amount.Sub(fees).Sub(txAmount)))

			balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
			Expect(err).To(BeNil())
			destBalance := balRes.Balance
			Expect(destBalance.Amount).To(Equal(txAmount))
		})

		It("should enable access to unlocked EVM tokens (single-account, multiple-msgs)", func() {
			account := vestingAccs[0]

			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePrev := balRes.Balance

			// Split the total unlocked amount into numTestMsgs equally sized tx's
			msgs := make([]sdk.Msg, numTestMsgs)
			// send all the account's spendable balance
			// initial_balance + unlocked in several messages
			totalSendAmt := initialFreeCoins.Add(freeCoins...)[0].Amount.Sub(remainingAmtToPayFees.MulRaw(2))
			txAmount := totalSendAmt.QuoRaw(int64(numTestMsgs))

			// update to the actual totalSendAmt to the sum of all sent txAmount
			// to avoid errors due to rounding
			totalSendAmt = math.ZeroInt()
			for i := 0; i < numTestMsgs; i++ {
				msg, err := s.factory.GenerateSignedMsgEthereumTx(account.Priv, evmtypes.EvmTxArgs{Nonce: uint64(i + 1), To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
				Expect(err).To(BeNil())
				msgs[i] = &msg
				totalSendAmt = totalSendAmt.Add(txAmount)
			}

			txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
			tx, err := utiltx.PrepareEthTx(txConfig, s.network.App, nil, msgs...)
			Expect(err).To(BeNil())

			txBytes, err := txConfig.TxEncoder()(tx)
			Expect(err).To(BeNil())

			res, err := s.network.BroadcastTxSync(txBytes)
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())
			Expect(s.network.NextBlock()).To(BeNil())

			// check final balance is as expected - transferred spendable tokens
			fees := gasPrice.Mul(math.NewInt(res.GasUsed))
			balRes, err = s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePost := balRes.Balance
			Expect(balancePost.Amount).To(Equal(balancePrev.Amount.Sub(fees).Sub(totalSendAmt)))

			balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
			Expect(err).To(BeNil())
			destBalance := balRes.Balance
			Expect(destBalance.Amount).To(Equal(totalSendAmt))
		})

		It("should enable access to unlocked EVM tokens (multi-account, single-tx)", func() {
			spRes, err := s.grpcHandler.GetSpendableBalance(vestingAccs[0].AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			spendableBalance := spRes.Balance
			// check that the spendable balance > than the initial free coins
			Expect(spendableBalance.Sub(initialFreeCoins[0]).IsPositive()).To(BeTrue())

			txAmount := spendableBalance.Amount.Sub(remainingAmtToPayFees)

			msgs := make([]sdk.Msg, len(vestingAccs))
			for i, grantee := range vestingAccs {
				msg, err := s.factory.GenerateSignedMsgEthereumTx(grantee.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
				Expect(err).To(BeNil())
				msgs[i] = &msg
			}

			txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
			tx, err := utiltx.PrepareEthTx(txConfig, s.network.App, nil, msgs...)
			Expect(err).To(BeNil())

			txBytes, err := txConfig.TxEncoder()(tx)
			Expect(err).To(BeNil())

			res, err := s.network.BroadcastTxSync(txBytes)
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())
			Expect(s.network.NextBlock()).To(BeNil())

			balRes, err := s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
			Expect(err).To(BeNil())
			destBalance := balRes.Balance
			Expect(destBalance.Amount).To(Equal(txAmount.MulRaw(int64(len(vestingAccs)))))
		})

		It("should enable access to unlocked EVM tokens (multi-account, multiple-msgs)", func() {
			spRes, err := s.grpcHandler.GetSpendableBalance(vestingAccs[0].AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			spendableBalance := spRes.Balance
			// check that the spendable balance > than the initial free coins
			Expect(spendableBalance.Sub(initialFreeCoins[0]).IsPositive()).To(BeTrue())

			amtSentByAcc := spendableBalance.Amount.Sub(remainingAmtToPayFees.MulRaw(int64(numTestMsgs)))
			txAmount := amtSentByAcc.QuoRaw(int64(numTestMsgs))

			msgs := []sdk.Msg{}
			for _, grantee := range vestingAccs {
				for i := 0; i < numTestMsgs; i++ {
					msg, err := s.factory.GenerateSignedMsgEthereumTx(grantee.Priv, evmtypes.EvmTxArgs{Nonce: uint64(i + 1), To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
					Expect(err).To(BeNil())
					msgs = append(msgs, &msg)
				}
			}

			txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
			tx, err := utiltx.PrepareEthTx(txConfig, s.network.App, nil, msgs...)
			Expect(err).To(BeNil())

			txBytes, err := txConfig.TxEncoder()(tx)
			Expect(err).To(BeNil())

			res, err := s.network.BroadcastTxSync(txBytes)
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())
			Expect(s.network.NextBlock()).To(BeNil())

			balRes, err := s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
			Expect(err).To(BeNil())
			destBalance := balRes.Balance
			Expect(destBalance.Amount).To(Equal(amtSentByAcc.MulRaw(int64(len(vestingAccs)))))
		})

		It("should not enable access to locked EVM tokens (single-account, single-msg)", func() {
			testAccount := vestingAccs[0]
			// Attempt to spend entire vesting balance
			txAmount := initialFreeCoins.Add(vestingAmtTotal...)[0].Amount.Sub(remainingAmtToPayFees)

			msg, err := s.factory.GenerateSignedMsgEthereumTx(testAccount.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
			Expect(err).To(BeNil())

			txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
			tx, err := utiltx.PrepareEthTx(txConfig, s.network.App, nil, &msg)
			Expect(err).To(BeNil())

			txBytes, err := txConfig.TxEncoder()(tx)
			Expect(err).To(BeNil())

			res, err := s.network.BroadcastTxSync(txBytes)
			Expect(err).To(BeNil())
			Expect(res.IsErr()).To(BeTrue())
			Expect(res.Log).To(ContainSubstring("clawback vesting account has insufficient unlocked tokens to execute transaction"))
		})

		It("should not enable access to locked EVM tokens (single-account, multiple-msgs)", func() {
			account := vestingAccs[0]
			msgs := make([]sdk.Msg, numTestMsgs+1)
			amt := vestingAccInitialBalance.Sub(remainingAmtToPayFees).Add(unlockedPerLockupAmt)
			txAmount := amt.QuoRaw(int64(numTestMsgs))

			// Add additional message that exceeds unlocked balance
			for i := 0; i < numTestMsgs+1; i++ {
				msg, err := s.factory.GenerateSignedMsgEthereumTx(account.Priv, evmtypes.EvmTxArgs{Nonce: uint64(i + 1), To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
				Expect(err).To(BeNil())
				msgs[i] = &msg
			}

			txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
			tx, err := utiltx.PrepareEthTx(txConfig, s.network.App, nil, msgs...)
			Expect(err).To(BeNil())

			txBytes, err := txConfig.TxEncoder()(tx)
			Expect(err).To(BeNil())

			res, err := s.network.BroadcastTxSync(txBytes)
			Expect(err).To(BeNil())
			Expect(res.IsErr()).To(BeTrue())
			Expect(res.Log).To(ContainSubstring("clawback vesting account has insufficient unlocked tokens to execute transaction"))
		})

		It("should not enable access to locked EVM tokens (multi-account, single-msg)", func() {
			numVestAccounts := len(vestingAccs)
			msgs := make([]sdk.Msg, numVestAccounts+1)
			txAmount := vestingAccInitialBalance.Sub(remainingAmtToPayFees).Add(unlockedPerLockupAmt)

			for i, account := range vestingAccs {
				msg, err := s.factory.GenerateSignedMsgEthereumTx(account.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
				Expect(err).To(BeNil())
				msgs[i] = &msg
			}
			// Add additional message that exceeds unlocked balance
			msg, err := s.factory.GenerateSignedMsgEthereumTx(vestingAccs[0].Priv, evmtypes.EvmTxArgs{Nonce: uint64(2), To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
			Expect(err).To(BeNil())
			msgs[numVestAccounts] = &msg

			txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
			tx, err := utiltx.PrepareEthTx(txConfig, s.network.App, nil, msgs...)
			Expect(err).To(BeNil())

			txBytes, err := txConfig.TxEncoder()(tx)
			Expect(err).To(BeNil())

			res, err := s.network.BroadcastTxSync(txBytes)
			Expect(err).To(BeNil())
			Expect(res.IsErr()).To(BeTrue())
			Expect(res.Log).To(ContainSubstring("clawback vesting account has insufficient unlocked tokens to execute transaction"))
		})

		It("should not enable access to locked EVM tokens (multi-account, multiple-msgs)", func() {
			msgs := []sdk.Msg{}
			amt := vestingAccInitialBalance.Sub(remainingAmtToPayFees).Add(unlockedPerLockupAmt)
			txAmount := amt.QuoRaw(int64(numTestMsgs))

			for _, account := range vestingAccs {
				for i := 0; i < numTestMsgs; i++ {
					msg, err := s.factory.GenerateSignedMsgEthereumTx(account.Priv, evmtypes.EvmTxArgs{Nonce: uint64(i + 1), To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
					Expect(err).To(BeNil())
					msgs = append(msgs, &msg)
				}
			}
			// Add additional message that exceeds unlocked balance
			msg, err := s.factory.GenerateSignedMsgEthereumTx(vestingAccs[0].Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
			Expect(err).To(BeNil())
			msgs = append(msgs, &msg)

			txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
			tx, err := utiltx.PrepareEthTx(txConfig, s.network.App, nil, msgs...)
			Expect(err).To(BeNil())

			txBytes, err := txConfig.TxEncoder()(tx)
			Expect(err).To(BeNil())

			res, err := s.network.BroadcastTxSync(txBytes)
			Expect(err).To(BeNil())
			Expect(res.IsErr()).To(BeTrue())
			Expect(res.Log).To(ContainSubstring("clawback vesting account has insufficient unlocked tokens to execute transaction"))
		})
		It("should not short-circuit with a normal account", func() {
			vestAcc := vestingAccs[0]
			normalAcc := funder

			txAmount := initialFreeCoins.Add(vestingAmtTotal...)[0].Amount.Sub(remainingAmtToPayFees)

			// Get message from a normal account to try to short-circuit the AnteHandler
			normAccMsg, err := s.factory.GenerateSignedMsgEthereumTx(normalAcc.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: big.NewInt(100_000)})
			Expect(err).To(BeNil())
			// Attempt to spend entire balance
			vestAccMsg, err := s.factory.GenerateSignedMsgEthereumTx(vestAcc.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
			Expect(err).To(BeNil())

			msgs := []sdk.Msg{&normAccMsg, &vestAccMsg}

			txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
			tx, err := utiltx.PrepareEthTx(txConfig, s.network.App, nil, msgs...)
			Expect(err).To(BeNil())

			txBytes, err := txConfig.TxEncoder()(tx)
			Expect(err).To(BeNil())

			res, err := s.network.BroadcastTxSync(txBytes)
			Expect(err).To(BeNil())
			Expect(res.IsErr()).To(BeTrue())
			Expect(res.Log).To(ContainSubstring("clawback vesting account has insufficient unlocked tokens to execute transaction"))
		})
	})

	Context("after first lockup and additional vest", func() {
		BeforeEach(func() {
			vestDuration := time.Duration(lockupLength + vestingLength)
			Expect(s.network.NextBlockAfter(vestDuration * time.Second)).To(BeNil())

			vested = clawbackAccount.GetVestedCoins(s.network.GetContext().BlockTime())
			expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, amt.Mul(math.NewInt(lockup+1))))

			unlocked := clawbackAccount.GetUnlockedOnly(s.network.GetContext().BlockTime())
			expUnlocked := unlockedPerLockup

			Expect(expVested).To(Equal(vested))
			Expect(expUnlocked).To(Equal(unlocked))
		})

		It("should enable access to unlocked EVM tokens", func() {
			account := vestingAccs[0]
			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePrev := balRes.Balance

			spRes, err := s.grpcHandler.GetSpendableBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			spendableBalance := spRes.Balance
			// check that the spendable balance > than the initial free coins
			Expect(spendableBalance.Sub(initialFreeCoins[0]).IsPositive()).To(BeTrue())

			txAmount := spendableBalance.Amount.Sub(remainingAmtToPayFees)

			res, err := s.factory.ExecuteEthTx(account.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())
			Expect(s.network.NextBlock()).To(BeNil())

			// check final balance is as expected - transferred spendable tokens
			fees := gasPrice.Mul(math.NewInt(res.GasUsed))
			balRes, err = s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePost := balRes.Balance
			Expect(balancePost.Amount).To(Equal(balancePrev.Amount.Sub(fees).Sub(txAmount)))

			balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
			Expect(err).To(BeNil())
			destBalance := balRes.Balance
			Expect(destBalance.Amount).To(Equal(txAmount))
		})

		It("should not enable access to locked EVM tokens", func() {
			account := vestingAccs[0]

			txAmount := initialFreeCoins.Add(vested...)[0].Amount.Sub(remainingAmtToPayFees).Add(math.OneInt())

			res, err := s.factory.ExecuteEthTx(account.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("insufficient funds"))
			Expect(res.IsErr()).To(BeTrue())
		})
	})

	Context("after half of vesting period and half lockups", func() {
		BeforeEach(func() {
			// Surpass half lockup duration
			passedLockups := numLockupPeriods / 2
			twoLockupsDuration := time.Duration(lockupLength * passedLockups)
			Expect(s.network.NextBlockAfter(twoLockupsDuration * time.Second)).To(BeNil())

			// Check if some, but not all tokens are vested
			unvested = clawbackAccount.GetVestingCoins(s.network.GetContext().BlockTime())
			vested = clawbackAccount.GetVestedOnly(s.network.GetContext().BlockTime())
			expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, amt.Mul(math.NewInt(lockup*passedLockups))))
			Expect(vestingAmtTotal).NotTo(Equal(vested))
			Expect(expVested).To(Equal(vested))
		})

		It("can delegate vested tokens", func() {
			account := vestingAccs[0]
			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePrev := balRes.Balance

			ok, vestedCoin := vested.Find(utils.BaseDenom)
			Expect(ok).To(BeTrue())
			// save some balance to pay fees
			delCoin := initialFreeCoins.Add(vestedCoin).Sub(accountGasCoverage...)[0]
			err = s.factory.Delegate(
				account.Priv,
				s.network.GetValidators()[0].OperatorAddress,
				delCoin,
			)
			Expect(err).To(BeNil())
			Expect(s.network.NextBlock()).To(BeNil())

			// check final balance is as expected - transferred spendable tokens
			balRes, err = s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePost := balRes.Balance
			// remaining balance should be less than prevBalance - delegated amount
			// cause should pay for fees too
			Expect(balancePost.Amount.LT(balancePrev.Amount.Sub(delCoin.Amount))).To(BeTrue())
		})

		It("cannot delegate unvested tokens", func() {
			ok, vestedCoin := vestingAmtTotal.Find(utils.BaseDenom)
			Expect(ok).To(BeTrue())
			delCoin := vestedCoin.Add(sdk.NewCoin(stakeDenom, vestingAccInitialBalance.Sub(remainingAmtToPayFees)))
			err := s.factory.Delegate(
				vestingAccs[0].Priv,
				s.network.GetValidators()[0].OperatorAddress,
				delCoin,
			)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("cannot delegate unvested coins"))
		})

		It("can transfer vested tokens", func() {
			account := vestingAccs[0]

			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePrev := balRes.Balance

			// save some balance to pay fees
			coins := initialFreeCoins.Add(vested...).Sub(accountGasCoverage...)
			sendAmt := coins[0].Amount

			msg := banktypes.NewMsgSend(account.AccAddr, dest.Bytes(), coins)
			res, err := s.factory.ExecuteCosmosTx(account.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())
			Expect(s.network.NextBlock()).To(BeNil())

			// check final balance is as expected - transferred spendable tokens
			fees := gasPrice.Mul(math.NewInt(res.GasWanted))
			balRes, err = s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePost := balRes.Balance
			Expect(balancePost.Amount).To(Equal(balancePrev.Amount.Sub(fees).Sub(sendAmt)))

			balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
			Expect(err).To(BeNil())
			destBalance := balRes.Balance
			Expect(destBalance.Amount).To(Equal(sendAmt))
		})
		It("cannot transfer unvested tokens", func() {
			account := vestingAccs[0]
			// save some balance to pay fees
			sendAmt := vestingAccInitialBalance.Sub(remainingAmtToPayFees)
			coins := vestingAmtTotal.Add(sdk.NewCoin(stakeDenom, sendAmt))

			msg := banktypes.NewMsgSend(account.AccAddr, dest.Bytes(), coins)
			_, err := s.factory.ExecuteCosmosTx(account.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("spendable balance"))
			Expect(err.Error()).To(ContainSubstring("is smaller than"))
		})
		It("can perform Ethereum tx with spendable balance", func() {
			account := vestingAccs[0]
			// save some balance to pay fees
			ok, vestedCoin := vested.Find(utils.BaseDenom)
			Expect(ok).To(BeTrue())
			txAmount := initialFreeCoins.Add(vestedCoin).Sub(accountGasCoverage...)[0].Amount

			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePrev := balRes.Balance

			res, err := s.factory.ExecuteEthTx(account.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())
			Expect(s.network.NextBlock()).To(BeNil())

			// check final balance is as expected - transferred spendable tokens
			fees := gasPrice.Mul(math.NewInt(res.GasUsed))
			balRes, err = s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePost := balRes.Balance
			Expect(balancePost.Amount).To(Equal(balancePrev.Amount.Sub(fees).Sub(txAmount)))

			balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
			Expect(err).To(BeNil())
			destBalance := balRes.Balance
			Expect(destBalance.Amount).To(Equal(txAmount))
		})
	})

	Context("after entire vesting period and all lockups", func() {
		BeforeEach(func() {
			// Surpass vest duration
			vestDuration := time.Duration(vestingLength * periodsTotal)
			Expect(s.network.NextBlockAfter(vestDuration * time.Second)).To(BeNil())

			// Check that all tokens are vested and unlocked
			unvested = clawbackAccount.GetVestingCoins(s.network.GetContext().BlockTime())
			vested = clawbackAccount.GetVestedOnly(s.network.GetContext().BlockTime())
			unlocked := clawbackAccount.GetUnlockedOnly(s.network.GetContext().BlockTime())
			unlockedVested := clawbackAccount.GetVestedCoins(s.network.GetContext().BlockTime())
			notSpendable := clawbackAccount.LockedCoins(s.network.GetContext().BlockTime())

			// all vested coins should be unlocked
			Expect(vested).To(Equal(unlockedVested))

			zeroCoins := sdk.NewCoins(sdk.NewCoin(stakeDenom, math.ZeroInt()))
			Expect(vestingAmtTotal).To(Equal(vested))
			Expect(vestingAmtTotal).To(Equal(unlocked))
			Expect(zeroCoins).To(Equal(notSpendable))
			Expect(zeroCoins).To(Equal(unvested))
		})

		It("can send entire balance", func() {
			account := vestingAccs[0]

			balRes, err := s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePrev := balRes.Balance

			txAmount := initialFreeCoins.Add(vestingAmtTotal...).Sub(accountGasCoverage...)[0].Amount

			res, err := s.factory.ExecuteEthTx(account.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())
			Expect(s.network.NextBlock()).To(BeNil())

			// check final balance is as expected - transferred spendable tokens
			fees := gasPrice.Mul(math.NewInt(res.GasUsed))
			balRes, err = s.grpcHandler.GetBalance(account.AccAddr, stakeDenom)
			Expect(err).To(BeNil())
			balancePost := balRes.Balance
			Expect(balancePost.Amount).To(Equal(balancePrev.Amount.Sub(fees).Sub(txAmount)))

			balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
			Expect(err).To(BeNil())
			destBalance := balRes.Balance
			Expect(destBalance.Amount).To(Equal(txAmount))
		})

		It("cannot exceed balance", func() {
			account := vestingAccs[0]

			txAmount := vestingAmtTotal.AmountOf(stakeDenom).Add(vestingAccInitialBalance).Mul(math.NewInt(2))
			res, err := s.factory.ExecuteEthTx(account.Priv, evmtypes.EvmTxArgs{To: &dest, GasPrice: gasPrice.BigInt(), GasLimit: gasLimit, Amount: txAmount.BigInt()})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("sender balance < tx cost"))
			Expect(res.IsErr()).To(BeTrue())
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
		s               *KeeperIntegrationTestSuite
		funder          keyring.Key
		vestingAcc      keyring.Key
		clawbackAccount *types.ClawbackVestingAccount
		vesting         sdk.Coins
		vested          sdk.Coins
		unlocked        sdk.Coins
		free            sdk.Coins
	)

	BeforeEach(func() {
		s = new(KeeperIntegrationTestSuite)
		// create 3 prefunded accounts:
		// index 0 will be the funder and
		// index 1 will be vesting account
		// index 2 will be extra account for other test cases
		keys := keyring.New(3)

		// don't send inflation and fees tokens to community pool
		// so we can check better when the claw backed tokens go to
		// the community pool
		customGen := network.CustomGenesisState{}
		// inflation custom genesis
		coinomicsGen := coinomicstypes.DefaultGenesisState()
		coinomicsGen.Params.EnableCoinomics = false
		customGen[coinomicstypes.ModuleName] = coinomicsGen
		// distribution custom genesis
		distrGen := distrtypes.DefaultGenesisState()
		distrGen.Params.CommunityTax = math.LegacyZeroDec()
		customGen[distrtypes.ModuleName] = distrGen

		nw := network.NewUnitTestNetwork(
			network.WithPreFundedAccounts(keys.GetAllAccAddrs()...),
			network.WithCustomGenesis(customGen),
		)
		gh := grpc.NewIntegrationHandler(nw)
		tf := haqqfactory.New(nw, gh)

		s.network = nw
		s.factory = tf
		s.grpcHandler = gh
		s.keyring = keys

		// index 0 will be the funder
		// index 1 will be vesting account
		funder = keys.GetKey(0)
		vestingAcc = keys.GetKey(1)

		// Create vesting account at vesting address
		clawbackAccount = s.setupClawbackVestingAccount(
			vestingAcc,
			funder,
			testutil.TestVestingSchedule.VestingPeriods,
			testutil.TestVestingSchedule.LockupPeriods,
		)

		// Check if all tokens are unvested and locked at vestingStart
		vesting = clawbackAccount.GetVestingCoins(s.network.GetContext().BlockTime())
		vested = clawbackAccount.GetVestedOnly(s.network.GetContext().BlockTime())
		unlocked = clawbackAccount.GetUnlockedOnly(s.network.GetContext().BlockTime())
		Expect(vesting).To(Equal(vestingAmtTotal), "expected difference vesting tokens")
		Expect(vested.IsZero()).To(BeTrue(), "expected no tokens to be vested")
		Expect(unlocked.IsZero()).To(BeTrue(), "expected no tokens to be unlocked")
	})

	It("should claw back unvested amount before cliff", func() {
		balRes, err := s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceFunder := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceGrantee := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
		Expect(err).To(BeNil())
		balanceDest := balRes.Balance

		// Perform clawback before cliff
		msg := types.NewMsgClawback(funder.AccAddr, vestingAcc.AccAddr, dest.Bytes())
		res, err := s.factory.ExecuteCosmosTx(funder.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
		Expect(err).To(BeNil())
		Expect(res.IsOK()).To(BeTrue())
		Expect(s.network.NextBlock()).To(BeNil())

		// All initial vesting amount goes to dest
		balRes, err = s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bF := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bG := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
		Expect(err).To(BeNil())
		bD := balRes.Balance

		// fees paid by funder
		fees := gasPrice.Mul(math.NewInt(res.GasWanted))

		Expect(bF.Amount).To(Equal(balanceFunder.Amount.Sub(fees)), "expected funder balance decrease due to fees")
		Expect(bG.Amount).To(Equal(balanceGrantee.Sub(vestingAmtTotal[0]).Amount), "expected all tokens to be clawed back")
		Expect(bD.Amount).To(Equal(balanceDest.Add(vestingAmtTotal[0]).Amount), "expected all tokens to be clawed back to the destination account")
	})

	It("should claw back any unvested amount after cliff before unlocking", func() {
		// Surpass cliff but not lockup duration
		cliffDuration := time.Duration(cliffLength)
		Expect(s.network.NextBlockAfter(cliffDuration * time.Second)).To(BeNil())
		blockTime := s.network.GetContext().BlockTime()

		// Check that all tokens are locked and some, but not all tokens are vested
		vested = clawbackAccount.GetVestedCoins(blockTime)
		unlocked = clawbackAccount.GetUnlockedOnly(blockTime)
		lockedUp := clawbackAccount.LockedCoins(blockTime)
		free = clawbackAccount.GetVestedCoins(blockTime)
		vesting = clawbackAccount.GetVestingCoins(blockTime)
		expVestedAmount := amt.Mul(math.NewInt(cliff))
		expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, expVestedAmount))
		unvested := vestingAmtTotal.Sub(vested...)

		Expect(expVested).To(Equal(vested))
		Expect(expVestedAmount.GT(math.NewInt(0))).To(BeTrue())
		Expect(free.IsZero()).To(BeTrue())
		Expect(lockedUp).To(Equal(vestingAmtTotal))
		Expect(vesting).To(Equal(unvested))

		balRes, err := s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceFunder := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceGrantee := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
		Expect(err).To(BeNil())
		balanceDest := balRes.Balance

		// Perform clawback
		msg := types.NewMsgClawback(funder.AccAddr, vestingAcc.AccAddr, dest.Bytes())
		res, err := s.factory.ExecuteCosmosTx(funder.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
		Expect(err).To(BeNil())
		Expect(res.IsOK()).To(BeTrue())
		Expect(s.network.NextBlock()).To(BeNil())

		// fees paid by funder
		fees := gasPrice.Mul(math.NewInt(res.GasWanted))

		balRes, err = s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bF := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bG := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
		Expect(err).To(BeNil())
		bD := balRes.Balance

		expClawback := clawbackAccount.GetVestingCoins(s.network.GetContext().BlockTime())
		Expect(expClawback).To(Equal(unvested))

		// Any unvested amount is clawed back
		Expect(bF.Amount).To(Equal(balanceFunder.Amount.Sub(fees)))
		Expect(bG.Amount).To(Equal(balanceGrantee.Sub(expClawback[0]).Amount))
		Expect(bD.Amount).To(Equal(balanceDest.Add(expClawback[0]).Amount))
	})

	It("should claw back any unvested amount after cliff and unlocking", func() {
		// Surpass lockup duration
		// A strict `if t < clawbackTime` comparison is used in ComputeClawback
		// so, we increment the duration with 1 for the free token calculation to match
		lockupDuration := time.Duration(lockupLength + 1)
		Expect(s.network.NextBlockAfter(lockupDuration * time.Second)).To(BeNil())

		// Check if some, but not all tokens are vested and unlocked
		vested = clawbackAccount.GetVestedOnly(s.network.GetContext().BlockTime())
		unlocked = clawbackAccount.GetUnlockedOnly(s.network.GetContext().BlockTime())
		free = clawbackAccount.GetVestedCoins(s.network.GetContext().BlockTime())
		vesting = clawbackAccount.GetVestingCoins(s.network.GetContext().BlockTime())
		expVestedAmount := amt.Mul(math.NewInt(lockup))
		expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, expVestedAmount))
		unvested := vestingAmtTotal.Sub(vested...)

		Expect(free).To(Equal(vested))
		Expect(expVested).To(Equal(vested))
		Expect(expVestedAmount.GT(math.NewInt(0))).To(BeTrue())
		Expect(vesting).To(Equal(unvested))

		balRes, err := s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceFunder := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceGrantee := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
		Expect(err).To(BeNil())
		balanceDest := balRes.Balance

		// Perform clawback
		msg := types.NewMsgClawback(funder.AccAddr, vestingAcc.AccAddr, dest.Bytes())
		res, err := s.factory.ExecuteCosmosTx(funder.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
		Expect(err).To(BeNil())
		Expect(res.IsOK()).To(BeTrue())
		Expect(s.network.NextBlock()).To(BeNil())

		// fees paid by funder
		fees := gasPrice.Mul(math.NewInt(res.GasWanted))

		balRes, err = s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bF := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bG := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
		Expect(err).To(BeNil())
		bD := balRes.Balance

		// Any unvested amount is clawed back
		Expect(bF.Amount).To(Equal(balanceFunder.Amount.Sub(fees)))
		Expect(bG.Amount).To(Equal(balanceGrantee.Sub(vesting[0]).Amount))
		Expect(bD.Amount).To(Equal(balanceDest.Add(vesting[0]).Amount))
	})

	It("should not claw back any amount after vesting periods end", func() {
		// Surpass vesting periods
		vestingDuration := time.Duration(periodsTotal*vestingLength + 1)
		Expect(s.network.NextBlockAfter(vestingDuration * time.Second)).To(BeNil())

		// Check if some, but not all tokens are vested and unlocked
		vested = clawbackAccount.GetVestedOnly(s.network.GetContext().BlockTime())
		unlocked = clawbackAccount.GetUnlockedOnly(s.network.GetContext().BlockTime())
		free = clawbackAccount.GetVestedCoins(s.network.GetContext().BlockTime())
		vesting = clawbackAccount.GetVestingCoins(s.network.GetContext().BlockTime())

		expVested := sdk.NewCoins(sdk.NewCoin(stakeDenom, amt.Mul(math.NewInt(periodsTotal))))
		unvested := vestingAmtTotal.Sub(vested...)

		Expect(free).To(Equal(vested))
		Expect(expVested).To(Equal(vested))
		Expect(expVested).To(Equal(vestingAmtTotal))
		Expect(unlocked).To(Equal(vestingAmtTotal))
		Expect(vesting).To(Equal(unvested))
		Expect(vesting.IsZero()).To(BeTrue())

		balRes, err := s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceFunder := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceGrantee := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
		Expect(err).To(BeNil())
		balanceDest := balRes.Balance

		// Perform clawback
		msg := types.NewMsgClawback(funder.AccAddr, vestingAcc.AccAddr, dest.Bytes())
		res, err := s.factory.ExecuteCosmosTx(funder.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
		Expect(err).To(BeNil())
		Expect(res.IsOK()).To(BeTrue())
		Expect(s.network.NextBlock()).To(BeNil())

		// fees paid by funder
		fees := gasPrice.Mul(math.NewInt(res.GasWanted))

		balRes, err = s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bF := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bG := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(dest.Bytes(), stakeDenom)
		Expect(err).To(BeNil())
		bD := balRes.Balance

		// No amount is clawed back
		Expect(bF.Amount).To(Equal(balanceFunder.Amount.Sub(fees)))
		Expect(bG.Amount).To(Equal(balanceGrantee.Amount))
		Expect(bD.Amount).To(Equal(balanceDest.Amount))
	})

	It("should update vesting funder and claw back unvested amount before cliff", func() {
		newFunder := s.keyring.GetKey(2)

		balRes, err := s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceFunder := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(newFunder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceNewFunder := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceGrantee := balRes.Balance

		// Update clawback vesting account funder
		updateFunderMsg := types.NewMsgUpdateVestingFunder(funder.AccAddr, newFunder.AccAddr, vestingAcc.AccAddr)
		txRes, err := s.factory.ExecuteCosmosTx(funder.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{updateFunderMsg}, GasPrice: &gasPrice})
		Expect(err).To(BeNil())
		Expect(txRes.IsOK()).To(BeTrue())
		Expect(s.network.NextBlock()).To(BeNil())

		funderFees := gasPrice.Mul(math.NewInt(txRes.GasWanted))

		// Perform clawback before cliff - funds should go to new funder (no dest address defined)
		msg := types.NewMsgClawback(newFunder.AccAddr, vestingAcc.AccAddr, sdk.AccAddress([]byte{}))
		txRes, err = s.factory.ExecuteCosmosTx(newFunder.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
		Expect(err).To(BeNil())
		Expect(txRes.IsOK()).To(BeTrue())
		Expect(s.network.NextBlock()).To(BeNil())

		newFunderFees := gasPrice.Mul(math.NewInt(txRes.GasWanted))

		// All initial vesting amount goes to funder
		balRes, err = s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bF := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(newFunder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bNewF := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bG := balRes.Balance

		// Original funder balance should not change
		Expect(bF.Amount).To(Equal(balanceFunder.Amount.Sub(funderFees)))
		// New funder should get the vested tokens
		Expect(bNewF.Amount).To(Equal(balanceNewFunder.Add(vestingAmtTotal[0]).Amount.Sub(newFunderFees)))
		Expect(bG.Amount).To(Equal(balanceGrantee.Sub(vestingAmtTotal[0]).Amount))
	})

	It("should update vesting funder and first funder cannot claw back unvested before cliff", func() {
		newFunder := s.keyring.GetKey(2)

		balRes, err := s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceFunder := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(newFunder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceNewFunder := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		balanceGrantee := balRes.Balance

		// Update clawback vesting account funder
		updateFunderMsg := types.NewMsgUpdateVestingFunder(funder.AccAddr, newFunder.AccAddr, vestingAcc.AccAddr)
		txRes, err := s.factory.ExecuteCosmosTx(funder.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{updateFunderMsg}, GasPrice: &gasPrice})
		Expect(err).To(BeNil())
		Expect(txRes.IsOK()).To(BeTrue())
		Expect(s.network.NextBlock()).To(BeNil())

		// Original funder tries to perform clawback before cliff - is not the current funder
		msg := types.NewMsgClawback(funder.AccAddr, vestingAcc.AccAddr, sdk.AccAddress([]byte{}))
		_, err = s.factory.ExecuteCosmosTx(funder.Priv, factory.CosmosTxArgs{Msgs: []sdk.Msg{msg}, GasPrice: &gasPrice})
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("clawback can only be requested by original funder"))
		Expect(s.network.NextBlock()).To(BeNil())

		fees := gasPrice.Mul(math.NewInt(txRes.GasWanted))

		// All balances should remain the same
		balRes, err = s.grpcHandler.GetBalance(funder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bF := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(newFunder.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bNewF := balRes.Balance
		balRes, err = s.grpcHandler.GetBalance(vestingAcc.AccAddr, stakeDenom)
		Expect(err).To(BeNil())
		bG := balRes.Balance

		Expect(bF.Amount).To(Equal(balanceFunder.Amount.Sub(fees)))
		Expect(balanceNewFunder).To(Equal(bNewF))
		Expect(balanceGrantee).To(Equal(bG))
	})
})
