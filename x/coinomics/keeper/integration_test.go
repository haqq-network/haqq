package keeper_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	"github.com/haqq-network/haqq/testutil"
)

func setupCoinomicsParams(s *KeeperTestSuite, rewardCoefficient sdk.Dec) {
	coinomicsParams := s.app.CoinomicsKeeper.GetParams(s.ctx)
	coinomicsParams.RewardCoefficient = rewardCoefficient
	s.app.CoinomicsKeeper.SetParams(s.ctx, coinomicsParams)
}

func isLeapYear(ctx sdk.Context) bool {
	year := ctx.BlockTime().Year()
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

var _ = Describe("Coinomics", Ordered, func() {
	BeforeEach(func() {
		s.SetupTest()

		// set coinomics params
		rewardCoefficient := sdk.NewDecWithPrec(78, 1) // 7.8%
		setupCoinomicsParams(s, rewardCoefficient)

		// set distribution module params
		distributionParams := s.app.DistrKeeper.GetParams(s.ctx)
		distributionParams.CommunityTax = sdk.NewDecWithPrec(10, 2)
		distributionParams.BaseProposerReward = sdk.NewDecWithPrec(1, 2)
		distributionParams.BonusProposerReward = sdk.NewDecWithPrec(4, 2)
		distributionParams.WithdrawAddrEnabled = true

		s.app.DistrKeeper.SetParams(s.ctx, distributionParams)
	})

	Describe("Check coinomics on regular year", func() {
		Context("with coinomics disabled", func() {
			BeforeEach(func() {
				params := s.app.CoinomicsKeeper.GetParams(s.ctx)
				params.EnableCoinomics = false

				s.app.CoinomicsKeeper.SetParams(s.ctx, params)
			})

			It("should not mint coins when coinomics is disabled", func() {
				currentSupply := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				currentBlock := s.ctx.BlockHeight()

				startExpectedSupply := sdk.NewCoin(denomMint, math.NewIntWithDecimal(20_000_000_000, 18))

				Expect(startExpectedSupply.Amount).To(Equal(currentSupply.Amount))

				s.Commit(100)

				currentBlockAfterCommins := s.ctx.BlockHeight()

				Expect(currentBlock + 100).To(Equal(currentBlockAfterCommins))

				currentSupply = s.app.BankKeeper.GetSupply(s.ctx, denomMint)

				Expect(startExpectedSupply.Amount).To(Equal(currentSupply.Amount))
			})
		})

		Context("with coinomics enabled", func() {
			BeforeEach(func() {
				params := s.app.CoinomicsKeeper.GetParams(s.ctx)
				params.EnableCoinomics = true

				s.app.CoinomicsKeeper.SetParams(s.ctx, params)
			})

			It("check mint calculations on regular year", func() {
				totalSupplyOnStart := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				startExpectedSupply := sdk.NewCoin(denomMint, math.NewIntWithDecimal(20_000_000_000, 18))

				Expect(startExpectedSupply.Amount).To(Equal(totalSupplyOnStart.Amount))

				accKey, err := ethsecp256k1.GenerateKey()
				s.Require().NoError(err)
				addr := sdk.AccAddress(accKey.PubKey().Address())

				fundAmount := sdk.TokensFromConsensusPower(100_000_000, sdk.DefaultPowerReduction)

				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, addr, sdk.NewCoins(sdk.NewCoin(denomMint, fundAmount)))
				s.Require().NoError(err)

				totalSupplyAfterFund := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				expectedSupply := sdk.NewCoin(denomMint, math.NewIntWithDecimal(20_100_000_000, 18))

				Expect(totalSupplyAfterFund.Amount).To(Equal(expectedSupply.Amount))

				s.Commit(1)

				isLeapYear := isLeapYear(s.ctx)
				Expect(isLeapYear).To(Equal(false))

				// delegation
				delAmount := sdk.TokensFromConsensusPower(10_000_000, sdk.DefaultPowerReduction)
				delCoin := sdk.NewCoin(denomMint, delAmount)

				_, err = testutil.Delegate(s.ctx, s.app, accKey, delCoin, s.validator)
				s.Require().NoError(err)

				totalBonded := s.app.StakingKeeper.TotalBondedTokens(s.ctx)
				expectedTotalBonded := sdk.NewCoin(denomMint, math.NewIntWithDecimal(10_000_001, 18))
				Expect(totalBonded).To(Equal(expectedTotalBonded.Amount))

				totalSupplyBeforeMint10Blocks := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				heightBeforeMint := s.ctx.BlockHeight()
				PrevTSBeforeMint := s.app.CoinomicsKeeper.GetPrevBlockTS(s.ctx)

				// mint blocks
				s.Commit(10)

				PrevTSAfter10Blocks := s.app.CoinomicsKeeper.GetPrevBlockTS(s.ctx)

				// check commit height is changed and prev block ts is changed
				Expect(s.ctx.BlockHeight()).To(Equal(heightBeforeMint + 10))
				Expect(PrevTSAfter10Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(10 * 6 * 1000))))

				// check mint amount with 7.8% coefficient
				totalSupplyAfterMint10Blocks := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				diff7_8Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				Expect(diff7_8Coefficient10blocks.Amount).To(Equal(sdk.NewInt(1484018413245226480)))

				// change params
				rewardCoefficient := sdk.NewDecWithPrec(10, 1) // 10%
				setupCoinomicsParams(s, rewardCoefficient)

				totalSupplyBeforeMint10Blocks = s.app.BankKeeper.GetSupply(s.ctx, denomMint)

				// mint blocks
				s.Commit(10)

				PrevTSAfter20Blocks := s.app.CoinomicsKeeper.GetPrevBlockTS(s.ctx)

				// check commit height is changed and prev block ts is changed
				Expect(s.ctx.BlockHeight()).To(Equal(heightBeforeMint + 20))
				Expect(PrevTSAfter20Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(20 * 6 * 1000))))

				// check mint amount with 10.0% coefficient
				totalSupplyAfterMint10Blocks = s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				diff10_0Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				Expect(diff10_0Coefficient10blocks.Amount).To(Equal(sdk.NewInt(190258770928875190)))
			})

			It("check mint calculations for leap year", func() {
				totalSupplyOnStart := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				startExpectedSupply := sdk.NewCoin(denomMint, math.NewIntWithDecimal(20_000_000_000, 18))

				Expect(startExpectedSupply.Amount).To(Equal(totalSupplyOnStart.Amount))

				accKey, err := ethsecp256k1.GenerateKey()
				s.Require().NoError(err)
				addr := sdk.AccAddress(accKey.PubKey().Address())

				fundAmount := sdk.TokensFromConsensusPower(100_000_000, sdk.DefaultPowerReduction)

				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, addr, sdk.NewCoins(sdk.NewCoin(denomMint, fundAmount)))
				s.Require().NoError(err)

				totalSupplyAfterFund := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				expectedSupply := sdk.NewCoin(denomMint, math.NewIntWithDecimal(20_100_000_000, 18))

				Expect(totalSupplyAfterFund.Amount).To(Equal(expectedSupply.Amount))

				s.CommitLeapYear()

				isLeapYear := isLeapYear(s.ctx)
				Expect(isLeapYear).To(Equal(true))

				// delegation
				delAmount := sdk.TokensFromConsensusPower(10_000_000, sdk.DefaultPowerReduction)
				delCoin := sdk.NewCoin(denomMint, delAmount)

				_, err = testutil.Delegate(s.ctx, s.app, accKey, delCoin, s.validator)
				s.Require().NoError(err)

				totalBonded := s.app.StakingKeeper.TotalBondedTokens(s.ctx)
				expectedTotalBonded := sdk.NewCoin(denomMint, math.NewIntWithDecimal(10_000_001, 18))
				Expect(totalBonded).To(Equal(expectedTotalBonded.Amount))

				totalSupplyBeforeMint10Blocks := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				heightBeforeMint := s.ctx.BlockHeight()
				PrevTSBeforeMint := s.app.CoinomicsKeeper.GetPrevBlockTS(s.ctx)

				// mint blocks
				s.Commit(10)

				PrevTSAfter10Blocks := s.app.CoinomicsKeeper.GetPrevBlockTS(s.ctx)

				// check commit height is changed and prev block ts is changed
				Expect(s.ctx.BlockHeight()).To(Equal(heightBeforeMint + 10))
				Expect(PrevTSAfter10Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(10 * 6 * 1000))))

				// check mint amount with 7.8% coefficient
				totalSupplyAfterMint10Blocks := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				diff7_8Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				Expect(diff7_8Coefficient10blocks.Amount).To(Equal(sdk.NewInt(1479963718122957010)))

				// change params
				rewardCoefficient := sdk.NewDecWithPrec(10, 1) // 10%
				setupCoinomicsParams(s, rewardCoefficient)

				totalSupplyBeforeMint10Blocks = s.app.BankKeeper.GetSupply(s.ctx, denomMint)

				// mint blocks
				s.Commit(10)

				PrevTSAfter20Blocks := s.app.CoinomicsKeeper.GetPrevBlockTS(s.ctx)

				// check commit height is changed and prev block ts is changed
				Expect(s.ctx.BlockHeight()).To(Equal(heightBeforeMint + 20))
				Expect(PrevTSAfter20Blocks).To(Equal(PrevTSBeforeMint.Add(math.NewInt(20 * 6 * 1000))))

				// check mint amount with 10.0% coefficient
				totalSupplyAfterMint10Blocks = s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				diff10_0Coefficient10blocks := totalSupplyAfterMint10Blocks.Sub(totalSupplyBeforeMint10Blocks)
				Expect(diff10_0Coefficient10blocks.Amount).To(Equal(sdk.NewInt(189738938220891920)))
			})

			It("check max supply limit", func() {
				setupCoinomicsParams(s, sdk.NewDecWithPrec(15_000_000_000, 0)) // 15_000_000_000 %

				totalSupplyOnStart := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				startExpectedSupply := sdk.NewCoin(denomMint, math.NewIntWithDecimal(20_000_000_000, 18))

				Expect(startExpectedSupply.Amount).To(Equal(totalSupplyOnStart.Amount))

				accKey, err := ethsecp256k1.GenerateKey()
				s.Require().NoError(err)
				addr := sdk.AccAddress(accKey.PubKey().Address())

				fundAmount := sdk.TokensFromConsensusPower(100_000_000, sdk.DefaultPowerReduction)

				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, addr, sdk.NewCoins(sdk.NewCoin(denomMint, fundAmount)))
				s.Require().NoError(err)

				totalSupplyAfterFund := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				expectedSupply := sdk.NewCoin(denomMint, math.NewIntWithDecimal(20_100_000_000, 18))

				Expect(totalSupplyAfterFund.Amount).To(Equal(expectedSupply.Amount))

				s.Commit(1)

				// delegation
				delAmount := sdk.TokensFromConsensusPower(90_000_000, sdk.DefaultPowerReduction)
				delCoin := sdk.NewCoin(denomMint, delAmount)

				_, err = testutil.Delegate(s.ctx, s.app, accKey, delCoin, s.validator)
				s.Require().NoError(err)

				totalBonded := s.app.StakingKeeper.TotalBondedTokens(s.ctx)
				expectedTotalBonded := sdk.NewCoin(denomMint, math.NewIntWithDecimal(90_000_001, 18))
				Expect(totalBonded).To(Equal(expectedTotalBonded.Amount))

				// mint blocks
				s.Commit(10)

				maxSupply := s.app.CoinomicsKeeper.GetMaxSupply(s.ctx)
				paramsAfterCommits := s.app.CoinomicsKeeper.GetParams(s.ctx)
				totalSupplyAfterCommits := s.app.BankKeeper.GetSupply(s.ctx, denomMint)

				Expect(maxSupply.Amount).To(Not(Equal(totalSupplyAfterCommits.Amount)))
				Expect(paramsAfterCommits.EnableCoinomics).To(Equal(true))

				// mint blocks
				s.Commit(60)

				totalSupplyAfterCommits = s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				paramsAfterCommits = s.app.CoinomicsKeeper.GetParams(s.ctx)

				Expect(maxSupply.Amount).To(Equal(totalSupplyAfterCommits.Amount))
				Expect(paramsAfterCommits.EnableCoinomics).To(Equal(false))

				// double check
				s.Commit(10)

				totalSupplyAfterCommits = s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				Expect(maxSupply.Amount).To(Equal(totalSupplyAfterCommits.Amount))

				// check: not mint if coinomics enables back
				params := s.app.CoinomicsKeeper.GetParams(s.ctx)
				params.EnableCoinomics = true

				s.app.CoinomicsKeeper.SetParams(s.ctx, params)

				s.Commit(10)

				totalSupplyAfterCommits = s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				paramsAfterCommits = s.app.CoinomicsKeeper.GetParams(s.ctx)

				Expect(maxSupply.Amount).To(Equal(totalSupplyAfterCommits.Amount))
				Expect(paramsAfterCommits.EnableCoinomics).To(Equal(false))
			})
		})
	})
})
