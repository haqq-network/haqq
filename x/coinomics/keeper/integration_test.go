package keeper_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	"github.com/haqq-network/haqq/testutil"
)

func setupCoinomicsParams(it *IntegrationTestSuite, rewardCoefficient math.LegacyDec) {
	coinomicsParams := it.network.App.CoinomicsKeeper.GetParams(it.network.GetContext())
	coinomicsParams.RewardCoefficient = rewardCoefficient
	it.network.App.CoinomicsKeeper.SetParams(it.network.GetContext(), coinomicsParams)
}

func isLeapYear(ctx sdk.Context) bool {
	year := ctx.BlockTime().Year()
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

var _ = Describe("Coinomics", Ordered, func() {
	BeforeEach(func() {
		it.SetupTest()

		// set coinomics params
		rewardCoefficient := math.LegacyNewDecWithPrec(78, 1) // 7.8%
		setupCoinomicsParams(it, rewardCoefficient)

		// set distribution module params
		distributionParams, err := it.network.App.DistrKeeper.Params.Get(it.network.GetContext())
		Expect(err).To(BeNil())
		distributionParams.CommunityTax = math.LegacyNewDecWithPrec(10, 2)
		distributionParams.BaseProposerReward = math.LegacyNewDecWithPrec(1, 2)
		distributionParams.BonusProposerReward = math.LegacyNewDecWithPrec(4, 2)
		distributionParams.WithdrawAddrEnabled = true
		err = it.network.App.DistrKeeper.Params.Set(it.network.GetContext(), distributionParams)
		Expect(err).To(BeNil())
	})

	Describe("Check coinomics on regular year", func() {
		Context("with coinomics disabled", func() {
			BeforeEach(func() {
				params := it.network.App.CoinomicsKeeper.GetParams(it.network.GetContext())
				params.EnableCoinomics = false
				it.network.App.CoinomicsKeeper.SetParams(it.network.GetContext(), params)
			})

			It("should not mint coins when coinomics is disabled", func() {
				params := it.network.App.CoinomicsKeeper.GetParams(it.network.GetContext())
				Expect(params.EnableCoinomics).To(Equal(false))

				err := it.CommitNumBlocks(1)
				Expect(err).To(BeNil())

				params2 := it.network.App.CoinomicsKeeper.GetParams(it.network.GetContext())
				Expect(params2.EnableCoinomics).To(Equal(false))

				startSupply := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				currentBlock := it.network.GetContext().BlockHeight()

				err = it.CommitNumBlocks(100)
				Expect(err).To(BeNil())

				currentBlockAfterCommits := it.network.GetContext().BlockHeight()
				Expect(currentBlock + 100).To(Equal(currentBlockAfterCommits))

				currentSupply := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				Expect(startSupply.Amount.String()).To(Equal(currentSupply.Amount.String()))
			})
		})

		Context("with coinomics enabled", func() {
			BeforeEach(func() {
				params := it.network.App.CoinomicsKeeper.GetParams(it.network.GetContext())
				params.EnableCoinomics = true
				it.network.App.CoinomicsKeeper.SetParams(it.network.GetContext(), params)
			})

			It("check mint calculations on regular year", func() {
				totalSupplyOnStart := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				startExpectedSupply := sdk.NewCoin(it.denom, math.NewIntWithDecimal(20_000_000_000, 18))
				Expect(startExpectedSupply.Amount).To(Equal(totalSupplyOnStart.Amount))

				accKey, err := ethsecp256k1.GenerateKey()
				Expect(err).To(BeNil())
				addr := sdk.AccAddress(accKey.PubKey().Address())

				fundAmount := sdk.TokensFromConsensusPower(100_000_000, sdk.DefaultPowerReduction)
				err = testutil.FundAccount(it.network.GetContext(), it.network.App.BankKeeper, addr, sdk.NewCoins(sdk.NewCoin(it.denom, fundAmount)))
				Expect(err).To(BeNil())

				totalSupplyAfterFund := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				expectedSupply := sdk.NewCoin(it.denom, math.NewIntWithDecimal(20_100_000_000, 18))
				Expect(totalSupplyAfterFund.Amount).To(Equal(expectedSupply.Amount))

				err = it.CommitNumBlocks(1)
				Expect(err).To(BeNil())

				isLeapYear := isLeapYear(it.network.GetContext())
				Expect(isLeapYear).To(Equal(false))

				// delegation
				delAmount := sdk.TokensFromConsensusPower(10_000_000, sdk.DefaultPowerReduction)
				delCoin := sdk.NewCoin(it.denom, delAmount)
				_, err = testutil.Delegate(it.network.GetContext(), it.network.App, accKey, delCoin, it.network.GetValidators()[0])
				Expect(err).To(BeNil())

				totalBonded, err := it.network.App.StakingKeeper.TotalBondedTokens(it.network.GetContext())
				Expect(err).To(BeNil())

				expectedTotalBonded := sdk.NewCoin(it.denom, math.NewIntWithDecimal(10_000_001, 18))
				Expect(totalBonded).To(Equal(expectedTotalBonded.Amount))
				totalSupplyBeforeMint10Blocks := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				heightBeforeMint := it.network.GetContext().BlockHeight()
				PrevTSBeforeMint := it.network.App.CoinomicsKeeper.GetPrevBlockTS(it.network.GetContext())

				// mint blocks
				err = it.CommitNumBlocks(10)
				Expect(err).To(BeNil())

				PrevTSAfter10Blocks := it.network.App.CoinomicsKeeper.GetPrevBlockTS(it.network.GetContext())

				// check commit height is changed and prev block ts is changed
				Expect(it.network.GetContext().BlockHeight()).To(Equal(heightBeforeMint + 10))
				Expect(PrevTSAfter10Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(10 * 6 * 1000))))

				// check mint amount with 7.8% coefficient
				totalSupplyAfterMint10Blocks := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				diff7_8Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				Expect(diff7_8Coefficient10blocks.Amount).To(Equal(math.NewInt(1484018413245226480)))

				// change params
				rewardCoefficient := math.LegacyNewDecWithPrec(10, 1) // 10%
				setupCoinomicsParams(it, rewardCoefficient)

				totalSupplyBeforeMint10Blocks = it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)

				// mint blocks
				err = it.CommitNumBlocks(10)
				Expect(err).To(BeNil())

				PrevTSAfter20Blocks := it.network.App.CoinomicsKeeper.GetPrevBlockTS(it.network.GetContext())

				// check commit height is changed and prev block ts is changed
				Expect(it.network.GetContext().BlockHeight()).To(Equal(heightBeforeMint + 20))
				Expect(PrevTSAfter20Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(20 * 6 * 1000))))

				// check mint amount with 10.0% coefficient
				totalSupplyAfterMint10Blocks = it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				diff10_0Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				Expect(diff10_0Coefficient10blocks.Amount).To(Equal(math.NewInt(190258770928875190)))
			})

			It("check mint calculations for leap year", func() {
				totalSupplyOnStart := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				startExpectedSupply := sdk.NewCoin(it.denom, math.NewIntWithDecimal(20_000_000_000, 18))
				Expect(startExpectedSupply.Amount).To(Equal(totalSupplyOnStart.Amount))

				accKey, err := ethsecp256k1.GenerateKey()
				Expect(err).To(BeNil())
				addr := sdk.AccAddress(accKey.PubKey().Address())

				fundAmount := sdk.TokensFromConsensusPower(100_000_000, sdk.DefaultPowerReduction)
				err = testutil.FundAccount(it.network.GetContext(), it.network.App.BankKeeper, addr, sdk.NewCoins(sdk.NewCoin(it.denom, fundAmount)))
				Expect(err).To(BeNil())

				totalSupplyAfterFund := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				expectedSupply := sdk.NewCoin(it.denom, math.NewIntWithDecimal(20_100_000_000, 18))
				Expect(totalSupplyAfterFund.Amount).To(Equal(expectedSupply.Amount))

				leapYaerDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				dateDiff := leapYaerDate.Sub(it.network.GetContext().BlockTime())
				err = it.network.NextBlockAfter(dateDiff)
				Expect(err).To(BeNil())

				isLeapYear := isLeapYear(it.network.GetContext())
				Expect(isLeapYear).To(Equal(true))

				// delegation
				delAmount := sdk.TokensFromConsensusPower(10_000_000, sdk.DefaultPowerReduction)
				delCoin := sdk.NewCoin(it.denom, delAmount)
				_, err = testutil.Delegate(it.network.GetContext(), it.network.App, accKey, delCoin, it.network.GetValidators()[0])
				Expect(err).To(BeNil())

				totalBonded, err := it.network.App.StakingKeeper.TotalBondedTokens(it.network.GetContext())
				Expect(err).To(BeNil())
				expectedTotalBonded := sdk.NewCoin(it.denom, math.NewIntWithDecimal(10_000_001, 18))
				Expect(totalBonded).To(Equal(expectedTotalBonded.Amount))

				totalSupplyBeforeMint10Blocks := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				heightBeforeMint := it.network.GetContext().BlockHeight()
				PrevTSBeforeMint := it.network.App.CoinomicsKeeper.GetPrevBlockTS(it.network.GetContext())

				// mint blocks
				err = it.CommitNumBlocks(10)
				Expect(err).To(BeNil())

				PrevTSAfter10Blocks := it.network.App.CoinomicsKeeper.GetPrevBlockTS(it.network.GetContext())

				// check commit height is changed and prev block ts is changed
				Expect(it.network.GetContext().BlockHeight()).To(Equal(heightBeforeMint + 10))
				Expect(PrevTSAfter10Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(10 * 6 * 1000))))

				// check mint amount with 7.8% coefficient
				totalSupplyAfterMint10Blocks := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				diff7_8Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				Expect(diff7_8Coefficient10blocks.Amount).To(Equal(math.NewInt(1479963718122957010)))

				// change params
				rewardCoefficient := math.LegacyNewDecWithPrec(10, 1) // 10%
				setupCoinomicsParams(it, rewardCoefficient)

				totalSupplyBeforeMint10Blocks = it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)

				// mint blocks
				err = it.CommitNumBlocks(10)
				Expect(err).To(BeNil())

				PrevTSAfter20Blocks := it.network.App.CoinomicsKeeper.GetPrevBlockTS(it.network.GetContext())

				// check commit height is changed and prev block ts is changed
				Expect(it.network.GetContext().BlockHeight()).To(Equal(heightBeforeMint + 20))
				Expect(PrevTSAfter20Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(20 * 6 * 1000))))

				// check mint amount with 10.0% coefficient
				totalSupplyAfterMint10Blocks = it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				diff10_0Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				Expect(diff10_0Coefficient10blocks.Amount).To(Equal(math.NewInt(189738938220891920)))
			})

			It("check max supply limit", func() {
				setupCoinomicsParams(it, math.LegacyNewDecWithPrec(15_000_000_000, 0)) // 15_000_000_000 %

				totalSupplyOnStart := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				startExpectedSupply := sdk.NewCoin(it.denom, math.NewIntWithDecimal(20_000_000_000, 18))
				Expect(startExpectedSupply.Amount).To(Equal(totalSupplyOnStart.Amount))

				accKey, err := ethsecp256k1.GenerateKey()
				Expect(err).To(BeNil())
				addr := sdk.AccAddress(accKey.PubKey().Address())

				fundAmount := sdk.TokensFromConsensusPower(100_000_000, sdk.DefaultPowerReduction)
				err = testutil.FundAccount(it.network.GetContext(), it.network.App.BankKeeper, addr, sdk.NewCoins(sdk.NewCoin(it.denom, fundAmount)))
				Expect(err).To(BeNil())

				totalSupplyAfterFund := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				expectedSupply := sdk.NewCoin(it.denom, math.NewIntWithDecimal(20_100_000_000, 18))
				Expect(totalSupplyAfterFund.Amount).To(Equal(expectedSupply.Amount))

				err = it.CommitNumBlocks(1)
				Expect(err).To(BeNil())

				// delegation
				delAmount := sdk.TokensFromConsensusPower(90_000_000, sdk.DefaultPowerReduction)
				delCoin := sdk.NewCoin(it.denom, delAmount)

				_, err = testutil.Delegate(it.network.GetContext(), it.network.App, accKey, delCoin, it.network.GetValidators()[0])
				Expect(err).To(BeNil())

				totalBonded, err := it.network.App.StakingKeeper.TotalBondedTokens(it.network.GetContext())
				Expect(err).To(BeNil())
				expectedTotalBonded := sdk.NewCoin(it.denom, math.NewIntWithDecimal(90_000_001, 18))
				Expect(totalBonded).To(Equal(expectedTotalBonded.Amount))

				// mint blocks
				err = it.CommitNumBlocks(10)
				Expect(err).To(BeNil())

				maxSupply := it.network.App.CoinomicsKeeper.GetMaxSupply(it.network.GetContext())
				paramsAfterCommits := it.network.App.CoinomicsKeeper.GetParams(it.network.GetContext())
				totalSupplyAfterCommits := it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				Expect(maxSupply.Amount).To(Not(Equal(totalSupplyAfterCommits.Amount)))
				Expect(paramsAfterCommits.EnableCoinomics).To(Equal(true))

				// mint blocks
				err = it.CommitNumBlocks(60)
				Expect(err).To(BeNil())

				totalSupplyAfterCommits = it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				paramsAfterCommits = it.network.App.CoinomicsKeeper.GetParams(it.network.GetContext())
				Expect(maxSupply.Amount).To(Equal(totalSupplyAfterCommits.Amount))
				Expect(paramsAfterCommits.EnableCoinomics).To(Equal(false))

				// double check
				err = it.CommitNumBlocks(10)
				Expect(err).To(BeNil())

				totalSupplyAfterCommits = it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				Expect(maxSupply.Amount).To(Equal(totalSupplyAfterCommits.Amount))

				// check: not mint if coinomics enables back
				params := it.network.App.CoinomicsKeeper.GetParams(it.network.GetContext())
				params.EnableCoinomics = true
				it.network.App.CoinomicsKeeper.SetParams(it.network.GetContext(), params)

				err = it.CommitNumBlocks(10)
				Expect(err).To(BeNil())

				totalSupplyAfterCommits = it.network.App.BankKeeper.GetSupply(it.network.GetContext(), it.denom)
				paramsAfterCommits = it.network.App.CoinomicsKeeper.GetParams(it.network.GetContext())
				Expect(maxSupply.Amount).To(Equal(totalSupplyAfterCommits.Amount))
				Expect(paramsAfterCommits.EnableCoinomics).To(Equal(false))
			})
		})
	})
})
