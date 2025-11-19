package keeper_test

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	"github.com/haqq-network/haqq/testutil"
)

func setupCoinomicsParams(s *KeeperTestSuite, rewardCoefficient math.LegacyDec) {
	coinomicsParams := s.network.App.CoinomicsKeeper.GetParams(s.network.GetContext())
	coinomicsParams.RewardCoefficient = rewardCoefficient
	s.network.App.CoinomicsKeeper.SetParams(s.network.GetContext(), coinomicsParams)
}

func isLeapYear(ctx sdk.Context) bool {
	year := ctx.BlockTime().Year()
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

var _ = Describe("Coinomics", Ordered, func() {
	var startExpectedSupply sdk.Coin

	BeforeEach(func() {
		s.SetupTest()

		// set coinomics params
		rewardCoefficient := math.LegacyNewDecWithPrec(78, 1) // 7.8%
		setupCoinomicsParams(s, rewardCoefficient)

		// set distribution module params
		distributionParams, err := s.network.App.DistrKeeper.Params.Get(s.network.GetContext())
		s.Require().NoError(err)
		distributionParams.CommunityTax = math.LegacyNewDecWithPrec(10, 2)
		distributionParams.BaseProposerReward = math.LegacyNewDecWithPrec(1, 2)
		distributionParams.BonusProposerReward = math.LegacyNewDecWithPrec(4, 2)
		distributionParams.WithdrawAddrEnabled = true

		err = s.network.App.DistrKeeper.Params.Set(s.network.GetContext(), distributionParams)
		s.Require().NoError(err)

		// We've minted in genesis block balances for 2 accounts plus 3 coins for validators
		startExpectedSupply = sdk.NewCoin(denomMint, math.NewIntWithDecimal(200_003, 18))
	})

	Describe("Check coinomics on regular year", func() {
		Context("with coinomics disabled", func() {
			BeforeEach(func() {
				params := s.network.App.CoinomicsKeeper.GetParams(s.network.GetContext())
				params.EnableCoinomics = false

				s.network.App.CoinomicsKeeper.SetParams(s.network.GetContext(), params)
			})

			It("should not mint coins when coinomics is disabled", func() {
				currentSupply := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				currentBlock := s.network.GetContext().BlockHeight()

				Expect(startExpectedSupply.Amount.String()).To(Equal(currentSupply.Amount.String()))

				Expect(s.network.NextNBlocks(100)).To(BeNil())

				currentBlockAfterCommins := s.network.GetContext().BlockHeight()

				Expect(currentBlock + 100).To(Equal(currentBlockAfterCommins))

				currentSupply = s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)

				Expect(startExpectedSupply.Amount).To(Equal(currentSupply.Amount))
			})
		})

		Context("with coinomics enabled", func() {
			BeforeEach(func() {
				params := s.network.App.CoinomicsKeeper.GetParams(s.network.GetContext())
				params.EnableCoinomics = true

				s.network.App.CoinomicsKeeper.SetParams(s.network.GetContext(), params)
			})

			It("check mint calculations on regular year", func() {
				totalSupplyOnStart := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)

				Expect(startExpectedSupply.Amount).To(Equal(totalSupplyOnStart.Amount))

				accKey, err := ethsecp256k1.GenerateKey()
				s.Require().NoError(err)
				addr := sdk.AccAddress(accKey.PubKey().Address())
				fundAmount := sdk.TokensFromConsensusPower(100_000_000, sdk.DefaultPowerReduction)

				err = testutil.FundAccount(s.network.GetContext(), s.network.App.BankKeeper, addr, sdk.NewCoins(sdk.NewCoin(denomMint, fundAmount)))
				s.Require().NoError(err)

				totalSupplyAfterFund := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				expectedSupply := sdk.NewCoin(denomMint, math.NewIntWithDecimal(100_200_003, 18))

				Expect(totalSupplyAfterFund.Amount).To(Equal(expectedSupply.Amount))

				Expect(s.network.NextBlock()).To(BeNil())

				isLeapYear := isLeapYear(s.network.GetContext())
				Expect(isLeapYear).To(Equal(false))

				// delegation
				delAmount := sdk.TokensFromConsensusPower(10_000_000, sdk.DefaultPowerReduction)
				delCoin := sdk.NewCoin(denomMint, delAmount)

				err = s.factory.Delegate(accKey, s.network.GetValidators()[0].OperatorAddress, delCoin)
				s.Require().NoError(err)

				totalBonded, err := s.network.App.StakingKeeper.TotalBondedTokens(s.network.GetContext())
				s.Require().NoError(err)
				expectedTotalBonded := sdk.NewCoin(denomMint, math.NewIntWithDecimal(10_000_003, 18))
				Expect(totalBonded.String()).To(Equal(expectedTotalBonded.Amount.String()))

				totalSupplyBeforeMint10Blocks := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				heightBeforeMint := s.network.GetContext().BlockHeight()
				PrevTSBeforeMint := s.network.App.CoinomicsKeeper.GetPrevBlockTS(s.network.GetContext())

				// mint blocks
				Expect(s.network.NextNBlocks(10)).To(BeNil())

				PrevTSAfter10Blocks := s.network.App.CoinomicsKeeper.GetPrevBlockTS(s.network.GetContext())

				// check commit height is changed and prev block ts is changed
				Expect(s.network.GetContext().BlockHeight()).To(Equal(heightBeforeMint + 10))
				// testutils uses 1 second block time
				Expect(PrevTSAfter10Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(10 * 1 * 1000))))

				// check mint amount with 7.8% coefficient
				totalSupplyAfterMint10Blocks := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				diff7_8Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				// 10000003 * 7.8 / 100 * 1000 / 31536000000 * 10 = 0.247336451676113240
				Expect(diff7_8Coefficient10blocks.Amount.String()).To(Equal(math.NewInt(247336451676113240).String()))
				// change params
				rewardCoefficient := math.LegacyNewDecWithPrec(10, 1) // 10%
				setupCoinomicsParams(s, rewardCoefficient)

				totalSupplyBeforeMint10Blocks = s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)

				// mint blocks
				Expect(s.network.NextNBlocks(10)).To(BeNil())

				PrevTSAfter20Blocks := s.network.App.CoinomicsKeeper.GetPrevBlockTS(s.network.GetContext())

				// check commit height is changed and prev block ts is changed
				Expect(s.network.GetContext().BlockHeight()).To(Equal(heightBeforeMint + 20))
				// testutils uses 1 second block time
				Expect(PrevTSAfter20Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(20 * 1 * 1000))))

				// check mint amount with 10.0% coefficient
				totalSupplyAfterMint10Blocks = s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				diff10_0Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				// 10000003 * 10 / 100 * 1000 / 31536000000 * 10 = 0.31709801496937600
				Expect(diff10_0Coefficient10blocks.Amount.String()).To(Equal(math.NewInt(31709801496937600).String()))
			})

			It("check mint calculations for leap year", func() {
				totalSupplyOnStart := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)

				Expect(startExpectedSupply.Amount).To(Equal(totalSupplyOnStart.Amount))

				accKey, err := ethsecp256k1.GenerateKey()
				s.Require().NoError(err)
				addr := sdk.AccAddress(accKey.PubKey().Address())

				fundAmount := sdk.TokensFromConsensusPower(100_000_000, sdk.DefaultPowerReduction)

				err = testutil.FundAccount(s.network.GetContext(), s.network.App.BankKeeper, addr, sdk.NewCoins(sdk.NewCoin(denomMint, fundAmount)))
				s.Require().NoError(err)

				totalSupplyAfterFund := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				expectedSupply := sdk.NewCoin(denomMint, math.NewIntWithDecimal(100_200_003, 18))

				Expect(totalSupplyAfterFund.Amount).To(Equal(expectedSupply.Amount))

				// Commit leap year block
				timeDiff := s.leapYearTime.Sub(s.network.GetContext().BlockTime())
				Expect(s.network.NextBlockAfter(timeDiff)).To(BeNil())

				isLeapYear := isLeapYear(s.network.GetContext())
				Expect(isLeapYear).To(Equal(true))

				// delegation
				delAmount := sdk.TokensFromConsensusPower(10_000_000, sdk.DefaultPowerReduction)
				delCoin := sdk.NewCoin(denomMint, delAmount)

				err = s.factory.Delegate(accKey, s.network.GetValidators()[0].OperatorAddress, delCoin)
				s.Require().NoError(err)

				totalBonded, err := s.network.App.StakingKeeper.TotalBondedTokens(s.network.GetContext())
				s.Require().NoError(err)
				expectedTotalBonded := sdk.NewCoin(denomMint, math.NewIntWithDecimal(10_000_003, 18))
				Expect(totalBonded).To(Equal(expectedTotalBonded.Amount))

				totalSupplyBeforeMint10Blocks := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				heightBeforeMint := s.network.GetContext().BlockHeight()
				PrevTSBeforeMint := s.network.App.CoinomicsKeeper.GetPrevBlockTS(s.network.GetContext())

				// mint blocks
				Expect(s.network.NextNBlocks(10)).To(BeNil())

				PrevTSAfter10Blocks := s.network.App.CoinomicsKeeper.GetPrevBlockTS(s.network.GetContext())

				// check commit height is changed and prev block ts is changed
				Expect(s.network.GetContext().BlockHeight()).To(Equal(heightBeforeMint + 10))
				// testutils uses 1 second block time
				Expect(PrevTSAfter10Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(10 * 1 * 1000))))

				// check mint amount with 7.8% coefficient
				totalSupplyAfterMint10Blocks := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				diff7_8Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				// 10000003 * 7.8 / 100 * 1000 / 31622400000 * 10 = 0.246660669020578510
				Expect(diff7_8Coefficient10blocks.Amount.String()).To(Equal(math.NewInt(246660669020578510).String()))

				// change params
				rewardCoefficient := math.LegacyNewDecWithPrec(10, 1) // 10%
				setupCoinomicsParams(s, rewardCoefficient)

				totalSupplyBeforeMint10Blocks = s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)

				// mint blocks
				Expect(s.network.NextNBlocks(10)).To(BeNil())

				PrevTSAfter20Blocks := s.network.App.CoinomicsKeeper.GetPrevBlockTS(s.network.GetContext())

				// check commit height is changed and prev block ts is changed
				Expect(s.network.GetContext().BlockHeight()).To(Equal(heightBeforeMint + 20))
				// testutils uses 1 second block time
				Expect(PrevTSAfter20Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(20 * 1 * 1000))))

				// check mint amount with 10.0% coefficient
				totalSupplyAfterMint10Blocks = s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				diff10_0Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				// 10000003 * 10 / 100 * 1000 / 31622400000 * 10 = 0.31623162694945960
				Expect(diff10_0Coefficient10blocks.Amount.String()).To(Equal(math.NewInt(31623162694945960).String()))
			})

			It("check max supply limit", func() {
				setupCoinomicsParams(s, math.LegacyNewDecWithPrec(150_000_000_000, 0)) // 150_000_000_000 %

				totalSupplyOnStart := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)

				Expect(startExpectedSupply.Amount).To(Equal(totalSupplyOnStart.Amount))

				accKey, err := ethsecp256k1.GenerateKey()
				s.Require().NoError(err)
				addr := sdk.AccAddress(accKey.PubKey().Address())

				fundAmount := sdk.TokensFromConsensusPower(100_000_000, sdk.DefaultPowerReduction)

				err = testutil.FundAccount(s.network.GetContext(), s.network.App.BankKeeper, addr, sdk.NewCoins(sdk.NewCoin(denomMint, fundAmount)))
				s.Require().NoError(err)

				totalSupplyAfterFund := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				expectedSupply := sdk.NewCoin(denomMint, math.NewIntWithDecimal(100_200_003, 18))

				Expect(totalSupplyAfterFund.Amount).To(Equal(expectedSupply.Amount))

				Expect(s.network.NextBlock()).To(BeNil())

				// delegation
				delAmount := sdk.TokensFromConsensusPower(90_000_000, sdk.DefaultPowerReduction)
				delCoin := sdk.NewCoin(denomMint, delAmount)

				err = s.factory.Delegate(accKey, s.network.GetValidators()[0].OperatorAddress, delCoin)
				s.Require().NoError(err)

				totalBonded, err := s.network.App.StakingKeeper.TotalBondedTokens(s.network.GetContext())
				s.Require().NoError(err)
				expectedTotalBonded := sdk.NewCoin(denomMint, math.NewIntWithDecimal(90_000_003, 18))
				Expect(totalBonded).To(Equal(expectedTotalBonded.Amount))

				// mint blocks
				Expect(s.network.NextNBlocks(10)).To(BeNil())

				maxSupply := s.network.App.CoinomicsKeeper.GetMaxSupply(s.network.GetContext())
				paramsAfterCommits := s.network.App.CoinomicsKeeper.GetParams(s.network.GetContext())
				totalSupplyAfterCommits := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)

				Expect(maxSupply.Amount).To(Not(Equal(totalSupplyAfterCommits.Amount)))
				Expect(paramsAfterCommits.EnableCoinomics).To(Equal(true))

				// mint blocks
				Expect(s.network.NextNBlocks(60)).To(BeNil())

				totalSupplyAfterCommits = s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				paramsAfterCommits = s.network.App.CoinomicsKeeper.GetParams(s.network.GetContext())

				Expect(maxSupply.Amount.String()).To(Equal(totalSupplyAfterCommits.Amount.String()))
				Expect(paramsAfterCommits.EnableCoinomics).To(Equal(false))

				// double check
				Expect(s.network.NextNBlocks(10)).To(BeNil())

				totalSupplyAfterCommits = s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				Expect(maxSupply.Amount).To(Equal(totalSupplyAfterCommits.Amount))

				// check: not mint if coinomics enables back
				params := s.network.App.CoinomicsKeeper.GetParams(s.network.GetContext())
				params.EnableCoinomics = true

				s.network.App.CoinomicsKeeper.SetParams(s.network.GetContext(), params)

				Expect(s.network.NextNBlocks(10)).To(BeNil())

				totalSupplyAfterCommits = s.network.App.BankKeeper.GetSupply(s.network.GetContext(), denomMint)
				paramsAfterCommits = s.network.App.CoinomicsKeeper.GetParams(s.network.GetContext())

				Expect(maxSupply.Amount).To(Equal(totalSupplyAfterCommits.Amount))
				Expect(paramsAfterCommits.EnableCoinomics).To(Equal(false))
			})
		})
	})
})
