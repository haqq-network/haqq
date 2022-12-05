package keeper_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	// "github.com/cosmos/cosmos-sdk/x/staking/types"
	// "github.com/evmos/ethermint/crypto/ethsecp256k1"
	// "github.com/haqq-network/haqq/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Coinomics", Ordered, func() {
	BeforeEach(func() {
		s.SetupTest()

		// make eras shorter
		coinomicsParams := s.app.CoinomicsKeeper.GetParams(s.ctx)
		coinomicsParams.BlocksPerEra = 100
		s.app.CoinomicsKeeper.SetParams(s.ctx, coinomicsParams)

		// change distribution module params
		distributionParams := s.app.DistrKeeper.GetParams(s.ctx)
		distributionParams.CommunityTax = sdk.NewDecWithPrec(10, 2)
		distributionParams.BaseProposerReward = sdk.NewDecWithPrec(1, 2)
		distributionParams.BonusProposerReward = sdk.NewDecWithPrec(4, 2)
		distributionParams.WithdrawAddrEnabled = true

		s.app.DistrKeeper.SetParams(s.ctx, distributionParams)
	})

	Describe("Commiting a block", func() {
		Context("with coinomics disabled", func() {
			BeforeEach(func() {
				params := s.app.CoinomicsKeeper.GetParams(s.ctx)
				params.EnableCoinomics = false

				s.app.CoinomicsKeeper.SetParams(s.ctx, params)
			})

			It("check totals in all eras", func() {
				currentSupply := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
				currentEra := s.app.CoinomicsKeeper.GetEra(s.ctx)

				start_expectedSupply := sdk.NewCoin(denomMint, sdk.NewIntWithDecimal(20_000_000_000, 18))

				Expect(start_expectedSupply.Amount).To(Equal(currentSupply.Amount))
				Expect(currentEra).To(Equal(uint64(0)))

				for era := 1; era <= 50; era++ {
					s.Commit(100)

					currentSupply = s.app.BankKeeper.GetSupply(s.ctx, denomMint)
					currentEra = s.app.CoinomicsKeeper.GetEra(s.ctx)

					Expect(start_expectedSupply.Amount).To(Equal(currentSupply.Amount))
					Expect(currentEra).To(Equal(uint64(0)))
				}
			})
		})

		Context("with coinomics enabled", func() {
			BeforeEach(func() {
				params := s.app.CoinomicsKeeper.GetParams(s.ctx)
				params.EnableCoinomics = true

				s.app.CoinomicsKeeper.SetParams(s.ctx, params)
			})

			Context("check all eras mint", func() {
				It("should allocate all eras correct", func() {
					currentSupply := s.app.BankKeeper.GetSupply(s.ctx, denomMint)
					currentEra := s.app.CoinomicsKeeper.GetEra(s.ctx)

					start_expectedSupply := sdk.NewCoin(denomMint, sdk.NewIntWithDecimal(20_000_000_000, 18))

					Expect(start_expectedSupply.Amount).To(Equal(currentSupply.Amount))
					Expect(currentEra).To(Equal(uint64(0)))

					expectedSupplys := []string{
						"24333436136376723087884929700",
						"28450200465934610021375612900",
						"32361126579014602608191762000",
						"36076506386440595565667103600",
						"39606117203495288875268678100",
						"42959247479697247519390173900",
						"46144721242089108231305594900",
						"49170921316361375907625244900",
						"52045811386920030200128912400",
						"54776956953950751778007396500",
						"57371545242629937276991956400",
						"59836404116875163501027288300",
						"62178020047408128413860853600",
						"64402555181414445081052740600",
						"66515863558720445914885033300",
						"68523506517161146707025711300",
						"70430767327679812459559355400",
						"72242665097672544924466317300",
						"73963967979165640766127931100",
						"75599205716584081815706464200",
						"77152681567131600812806070700",
						"78628483625151743860050696800",
						"80030495580270879754933091600",
						"81362406937634058855071366700",
						"82627722727129079000202728000",
						"83829772727149348138077521300",
						"84971720227168603819058574900",
						"86056570352186896715990575800",
						"87087177970954274968075976700",
						"88066255208783284307557107500",
						"88996378584720843180064181800",
						"89879995791861524108945902400",
						"90719432138645170991383536900",
						"91516896668089635529699289700",
						"92274487971061876841099254900",
						"92994199708885506086929221800",
						"93677925859817953870467690300",
						"94327465703203779264829235400",
						"94944528554420313389472703300",
						"95530738263076020807883997800",
						"96087637486298942855374727500",
						"96616691748360718800490920800",
						"97119293297319405948351304400",
						"97596764768830158738818668800",
						"98050362666765373889762665000",
						"98481280669803828283159461400",
						"98890652772690359956886417900",
						"99279556270432565046927026600",
						"99649014593287659882465604900",
						"100000000000000000000000000000",
						"100000000000000000000000000000",
					}

					for era := 1; era <= len(expectedSupplys); era++ {
						s.Commit(100)

						currentSupply = s.app.BankKeeper.GetSupply(s.ctx, denomMint)
						currentEra = s.app.CoinomicsKeeper.GetEra(s.ctx)

						era_expectedSupplyBigInt, _ := big.NewInt(0).SetString(expectedSupplys[era-1], 0)
						era_expectedSupply := sdk.NewCoin(denomMint, sdk.NewIntFromBigInt(era_expectedSupplyBigInt))

						Expect(era_expectedSupply.Amount).To(Equal(currentSupply.Amount))
						Expect(currentEra).To(Equal(uint64(era)))
					}
				})
				It("should allocate funds to the evergreen pool", func() {
					balanceCommunityPool := s.app.DistrKeeper.GetFeePoolCommunityCoins(s.ctx)
					Expect(balanceCommunityPool.IsZero()).To(BeTrue())

					// skip some blocks
					s.Commit(10)

					currentEra := s.app.CoinomicsKeeper.GetEra(s.ctx)
					Expect(currentEra).To(Equal(uint64(1)))

					balanceCommunityPool = s.app.DistrKeeper.GetFeePoolCommunityCoins(s.ctx)
					Expect(balanceCommunityPool.IsZero()).ToNot(BeTrue())
				})
				// It("delegations is works", func() {
				// 	s.Commit(1)

				// 	accKey, err := ethsecp256k1.GenerateKey() // _accPrivKey
				// 	s.Require().NoError(err)
				// 	addr := sdk.AccAddress(accKey.PubKey().Address())

				// 	fundAmount := sdk.TokensFromConsensusPower(100_000_000, sdk.DefaultPowerReduction)

				// 	err = testutil.FundAccount(s.ctx, s.app.BankKeeper, addr, sdk.NewCoins(sdk.NewCoin(denomMint, fundAmount)))
				// 	s.Require().NoError(err)

				// 	s.Commit(1)

				// 	// delegation
				// 	delAmount := sdk.TokensFromConsensusPower(11_000_000, sdk.DefaultPowerReduction)
				// 	delCoin := sdk.NewCoin(denomMint, delAmount)

				// 	_, err = testutil.Delegate(s.ctx, s.app, accKey, delCoin, s.validator)
				// 	s.Require().NoError(err)

				// 	// println("suite.validator #start# bonded tokens: ", s.validator.GetBondedTokens().String())
				// 	// println("suite.validator #start# get tokens: ", s.validator.GetTokens().String())

				// 	s.Commit(1)

				// 	validators := s.app.StakingKeeper.GetValidators(s.ctx, 2)

				// 	for i, v := range validators {
				// 		println("validator info", i, "status: ", v.Status.String())

				// 		if v.Status == types.Bonded {
				// 			println("Bonded!")
				// 			_, err = testutil.Delegate(s.ctx, s.app, accKey, delCoin, v)
				// 			s.Require().NoError(err)
				// 			s.Commit(1)

				// 			// println("suite.validator #start# bonded tokens: ", v.GetBondedTokens().String())
				// 			// println("suite.validator #start# get tokens: ", v.GetTokens().String())
				// 		}
				// 	}

				// 	s.Commit(10)

				// 	delegations := s.app.StakingKeeper.GetAllDelegations(s.ctx)

				// 	for i, del := range delegations {
				// 		println("delegation #", i, "deladdr: ", del.DelegatorAddress, "valaddr: ", del.ValidatorAddress, "shares: ", del.Shares.String())

				// 		endingPeriod := s.app.DistrKeeper.IncrementValidatorPeriod(s.ctx, s.validator)
				// 		rewards := s.app.DistrKeeper.CalculateDelegationRewards(s.ctx, s.validator, del, endingPeriod)
				// 		println("rewards: ", rewards.String())
				// 	}
				// })
			})
		})
	})
})
