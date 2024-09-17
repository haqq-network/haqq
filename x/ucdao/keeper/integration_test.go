package keeper_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ucdao/types"
)

var _ = Describe("United Contributors DAO", func() {
	var (
		daoModuleAcc                   authtypes.ModuleAccountI
		fundMsg                        *types.MsgFund
		transferOwnershipMsg           *types.MsgTransferOwnership
		transferOwnershipWithRatioMsg  *types.MsgTransferOwnershipWithRatio
		transferOwnershipWithAmountMsg *types.MsgTransferOwnershipWithAmount
		err                            error
	)

	oneHundred, _ := sdk.NewIntFromString("100000000000000000000")
	oneHundredIslm := sdk.NewCoin(utils.BaseDenom, oneHundred)
	oneIslm := sdk.NewInt64Coin(utils.BaseDenom, 1000000000000000000)
	twoIslm := sdk.NewInt64Coin(utils.BaseDenom, 2000000000000000000)
	halfIslm := sdk.NewInt64Coin(utils.BaseDenom, 500000000000000000)
	threeInvalid := sdk.NewInt64Coin("invalid", 3000000000000000000)
	fiveLiquid1 := sdk.NewInt64Coin("aLIQUID1", 5000000000000000000)
	twopointfiveLiquid1 := sdk.NewInt64Coin("aLIQUID1", 2500000000000000000)
	sevenLiquid75 := sdk.NewInt64Coin("aLIQUID75", 7000000000000000000)
	nineLiquidInvalid := sdk.NewInt64Coin("aLIQUID", 9000000000000000000)
	gasPrice := sdkmath.NewInt(1000000000)

	DescribeTableSubtree("Fund transactions", func(signMode signing.SignMode) {
		Context("with invalid denom", func() {
			BeforeEach(func() {
				s.SetupTest()

				daoModuleAcc = s.app.AccountKeeper.GetModuleAccount(s.ctx, types.ModuleName)

				daoBalanceBefore := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), utils.BaseDenom)
				Expect(daoBalanceBefore.IsZero()).To(BeTrue(), "dao account should have no balance")

				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, s.address, sdk.NewCoins(oneHundredIslm, threeInvalid, fiveLiquid1, sevenLiquid75, nineLiquidInvalid))
				s.Require().NoError(err)

				s.Commit()
			})

			Context("correct aLIQUID with invalid", func() {
				It("should fail", func() {
					// Check balances before TX
					daoBankBalanceBefore := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
					Expect(daoBankBalanceBefore.IsZero()).To(BeTrue(), "dao account should have no balance")

					daoTotalBalanceBefore, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceBefore, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					// TX Process
					fundMsg = types.NewMsgFund(
						sdk.NewCoins(threeInvalid, fiveLiquid1),
						s.address,
					)
					_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
					Expect(err).NotTo(BeNil(), "transaction should have failed")
					Expect(
						strings.Contains(err.Error(),
							"denom invalid is not allowed"),
					).To(BeTrue(), err.Error())
					s.Commit()

					// Check balances after TX
					daoBankBalanceAfter := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
					Expect(daoBankBalanceAfter.IsZero()).To(BeTrue(), "dao account should have no balance")

					daoTotalBalanceAfter, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceAfter.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceAfter, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceAfter.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")
				})
			})

			Context("correct aISLM with invalid", func() {
				It("should fail", func() {
					// Check balances before TX
					daoBankBalanceBefore := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
					Expect(daoBankBalanceBefore.IsZero()).To(BeTrue(), "dao account should have no balance")

					daoTotalBalanceBefore, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceBefore, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					// TX Process
					fundMsg = types.NewMsgFund(
						sdk.NewCoins(oneIslm, nineLiquidInvalid),
						s.address,
					)
					_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
					Expect(err).NotTo(BeNil(), "transaction should have failed")
					Expect(
						strings.Contains(err.Error(),
							"denom aLIQUID is not allowed"),
					).To(BeTrue(), err.Error())
					s.Commit()

					// Check balances after TX
					daoBankBalanceAfter := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
					Expect(daoBankBalanceAfter.IsZero()).To(BeTrue(), "dao account should have no balance")

					daoTotalBalanceAfter, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceAfter.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceAfter, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceAfter.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")
				})
			})
		})

		Context("with correct denoms", func() {
			BeforeEach(func() {
				s.SetupTest()

				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, s.address, sdk.NewCoins(oneHundredIslm, threeInvalid, fiveLiquid1, sevenLiquid75, nineLiquidInvalid))
				s.Require().NoError(err)

				s.Commit()
			})

			Context("correct standalone aLIQUID", func() {
				It("should pass", func() {
					// Check balances before TX
					daoBankBalanceBefore := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), sevenLiquid75.Denom)
					Expect(daoBankBalanceBefore.IsZero()).To(BeTrue(), "dao account should have no balance")

					daoTotalBalanceBefore, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceBefore, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					// TX Process
					fundMsg = types.NewMsgFund(
						sdk.NewCoins(sevenLiquid75),
						s.address,
					)
					_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
					Expect(err).To(BeNil(), "transaction should have succeed")
					s.Commit()

					// Check balances after TX
					daoBankBalanceAfter := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), sevenLiquid75.Denom)
					Expect(daoBankBalanceAfter.Amount.String()).To(Equal(sevenLiquid75.Amount.String()), "dao account should have received the funds")

					daoTotalBalanceAfter, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					ok, amount := daoTotalBalanceAfter.TotalBalance.Find(sevenLiquid75.Denom)
					Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
					Expect(amount.String()).To(Equal(sevenLiquid75.String()), "dao total balance should have received the funds")

					daoAddressBalanceAfter, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					okAcc, amountAcc := daoAddressBalanceAfter.Balances.Find(sevenLiquid75.Denom)
					Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
					Expect(amountAcc.String()).To(Equal(sevenLiquid75.String()), "dao address balance should have received the funds")
				})
			})

			Context("correct standalone aISLM", func() {
				It("should pass", func() {
					// Check balances before TX
					daoBankBalanceBefore := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
					Expect(daoBankBalanceBefore.IsZero()).To(BeTrue(), "dao account should have no balance")

					daoTotalBalanceBefore, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceBefore, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					// TX Process
					fundMsg = types.NewMsgFund(
						sdk.NewCoins(oneIslm),
						s.address,
					)
					_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
					Expect(err).To(BeNil(), "transaction should have succeed")
					s.Commit()

					// Check balances after TX
					daoBankBalanceAfter := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
					Expect(daoBankBalanceAfter.Amount.String()).To(Equal(oneIslm.Amount.String()), "dao account should have received the funds")

					daoTotalBalanceAfter, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					ok, amount := daoTotalBalanceAfter.TotalBalance.Find(oneIslm.Denom)
					Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
					Expect(amount.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")

					daoAddressBalanceAfter, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					okAcc, amountAcc := daoAddressBalanceAfter.Balances.Find(oneIslm.Denom)
					Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
					Expect(amountAcc.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				})
			})

			Context("correct aISLM and aLIQUID", func() {
				It("should pass", func() {
					// Check balances before TX
					daoBankIslmBalanceBefore := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
					Expect(daoBankIslmBalanceBefore.IsZero()).To(BeTrue(), "dao account should have no balance")

					daoTotalIslmBalanceBefore, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalIslmBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressIslmBalanceBefore, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressIslmBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					daoBankLiquidBalanceBefore := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
					Expect(daoBankLiquidBalanceBefore.IsZero()).To(BeTrue(), "dao account should have no balance")

					daoTotalLiquidBalanceBefore, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalLiquidBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressLiquidBalanceBefore, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressLiquidBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					// TX Process
					fundMsg = types.NewMsgFund(
						sdk.NewCoins(oneIslm, fiveLiquid1),
						s.address,
					)
					_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
					Expect(err).To(BeNil(), "transaction should have succeed")
					s.Commit()

					// Check balances after TX
					daoBankIslmBalanceAfter := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
					Expect(daoBankIslmBalanceAfter.Amount.String()).To(Equal(oneIslm.Amount.String()), "dao account should have received the funds")

					daoTotalIslmBalanceAfter, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					ok, amount := daoTotalIslmBalanceAfter.TotalBalance.Find(oneIslm.Denom)
					Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
					Expect(amount.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")

					daoAddressIslmBalanceAfter, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					okAcc, amountAcc := daoAddressIslmBalanceAfter.Balances.Find(oneIslm.Denom)
					Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
					Expect(amountAcc.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")

					daoBankLiquidBalanceAfter := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
					Expect(daoBankLiquidBalanceAfter.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "dao account should have received the funds")

					daoTotalLiquidBalanceAfter, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					ok, amountLiq := daoTotalLiquidBalanceAfter.TotalBalance.Find(fiveLiquid1.Denom)
					Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
					Expect(amountLiq.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

					daoAddressLiquidBalanceAfter, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
					Expect(err).To(BeNil(), "query should have succeed")
					okAcc, amountLiqAcc := daoAddressLiquidBalanceAfter.Balances.Find(fiveLiquid1.Denom)
					Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
					Expect(amountLiqAcc.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")
				})
			})
		})
	},
		Entry("Direct sign mode", signing.SignMode_SIGN_MODE_DIRECT),
		Entry("Legacy Amino JSON sign mode", signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON),
	)

	DescribeTableSubtree("Transfer Ownership transactions", func(signMode signing.SignMode) {
		newOwnerPriv, err := ethsecp256k1.GenerateKey()
		newOwnerAddr := sdk.AccAddress(newOwnerPriv.PubKey().Address().Bytes())

		Context("basic validation", func() {
			BeforeEach(func() {
				s.SetupTest()

				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, s.address, sdk.NewCoins(oneHundredIslm))
				s.Require().NoError(err)

				s.Commit()
			})

			It("should fail - invalid owner address", func() {
				// TX Process
				transferOwnershipMsg = &types.MsgTransferOwnership{
					Owner:    "haqq1",
					NewOwner: newOwnerAddr.String(),
				}

				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipMsg)
				Expect(err).NotTo(BeNil(), "transaction should fail")
				Expect(err.Error()).To(ContainSubstring("invalid owner address"))
				s.Commit()
			})

			It("should fail - invalid new owner address", func() {
				// TX Process
				transferOwnershipMsg = &types.MsgTransferOwnership{
					Owner:    s.address.String(),
					NewOwner: "haqq1",
				}

				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipMsg)
				Expect(err).NotTo(BeNil(), "transaction should fail")
				Expect(err.Error()).To(ContainSubstring("invalid new owner address"))
				s.Commit()
			})
		})

		Context("with non-member as owner", func() {
			BeforeEach(func() {
				s.SetupTest()

				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, s.address, sdk.NewCoins(oneHundredIslm))
				s.Require().NoError(err)

				s.Commit()
			})

			It("should fail - not eligible NewMsgTransferOwnership", func() {
				daoTotalBalanceBefore, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoAddressBalanceBefore, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

				// TX Process
				transferOwnershipMsg = types.NewMsgTransferOwnership(s.address, newOwnerAddr)

				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipMsg)
				Expect(err).NotTo(BeNil(), "transaction should fail")
				Expect(err.Error()).To(ContainSubstring("not eligible"), "error message should be correct")
				s.Commit()
			})
			It("should fail - not eligible NewMsgTransferOwnershipWithRatio", func() {
				daoTotalBalanceBefore, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoAddressBalanceBefore, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

				// TX Process — 50% ownership
				transferOwnershipWithRatioMsg = types.NewMsgTransferOwnershipWithRatio(s.address, newOwnerAddr, sdkmath.LegacyNewDecWithPrec(5, 1))

				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipWithRatioMsg)
				Expect(err).NotTo(BeNil(), "transaction should fail")
				Expect(err.Error()).To(ContainSubstring("not eligible"), "error message should be correct")
				s.Commit()
			})
			It("should fail - not eligible NewMsgTransferOwnershipWithAmount", func() {
				daoTotalBalanceBefore, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoAddressBalanceBefore, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

				// TX Process — 1000 aISLM ownership
				transferOwnershipWithAmountMsg = types.NewMsgTransferOwnershipWithAmount(s.address, newOwnerAddr, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdk.NewInt(1000))))

				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipWithAmountMsg)
				Expect(err).NotTo(BeNil(), "transaction should fail")
				Expect(err.Error()).To(ContainSubstring("not eligible"), "error message should be correct")
				s.Commit()
			})
		})

		Context("with member as owner and non-member as new owner", func() {
			BeforeEach(func() {
				s.SetupTest()

				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, s.address, sdk.NewCoins(oneHundredIslm, fiveLiquid1))
				s.Require().NoError(err)

				s.Commit()
			})

			It("successfully transferred - NewMsgTransferOwnership", func() {
				daoModuleAddressBankBalanceBeforeFund := s.app.BankKeeper.GetAllBalances(s.ctx, daoModuleAcc.GetAddress())
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "dao module account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.address,
				)
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
				Expect(err).To(BeNil(), "transaction should have succeed")
				s.Commit()

				// Check balances after funding TX
				daoModuleAddressBankBalanceAfterFundIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "dao account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "dao account should have received the funds")

				daoTotalBalanceAfterFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipMsg = types.NewMsgTransferOwnership(s.address, newOwnerAddr)
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipMsg)
				Expect(err).To(BeNil(), "transaction should succeed")
				s.Commit()

				// Checks after transfer ownership
				// Module bank balance shouldn't change
				daoModuleAddressBankBalanceAfterTransferIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferIslm.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()))
				daoModuleAddressBankBalanceAfterTransferLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferLiquid.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()))

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should become empty
				daoOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeTrue())

				// All tokens should be transferred to new owner
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				okNewAcc, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(okNewAcc).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(oneIslm.String()))
				okNewAcc, liquidNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(okNewAcc).To(BeTrue())
				Expect(liquidNewAccAmount.String()).To(Equal(fiveLiquid1.String()))
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(oneIslm.String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(fiveLiquid1.String()))
			})

			It("successfully transferred - NewMsgTransferOwnershipWithRatio", func() {
				daoModuleAddressBankBalanceBeforeFund := s.app.BankKeeper.GetAllBalances(s.ctx, daoModuleAcc.GetAddress())
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "dao module account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.address,
				)
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
				Expect(err).To(BeNil(), "transaction should have succeed")
				s.Commit()

				// Check balances after funding TX
				daoModuleAddressBankBalanceAfterFundIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "dao account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "dao account should have received the funds")

				daoTotalBalanceAfterFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithRatioMsg = types.NewMsgTransferOwnershipWithRatio(s.address, newOwnerAddr, sdkmath.LegacyNewDecWithPrec(5, 1))
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipWithRatioMsg)
				Expect(err).To(BeNil(), "transaction should succeed")
				s.Commit()

				// Checks after transfer ownership
				// Module bank balance shouldn't change
				daoModuleAddressBankBalanceAfterTransferIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferIslm.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()))
				daoModuleAddressBankBalanceAfterTransferLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferLiquid.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()))

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should become lower by 50%
				daoOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, halfIslmownerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(halfIslmownerAmount.String()).To(Equal(halfIslm.String()))
				found, halfLiquidOwnerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(halfLiquidOwnerAmount.String()).To(Equal(twopointfiveLiquid1.String()))

				// 50% of all tokens should be transferred to new owner
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(halfIslm.String()))
				found, liquidNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(liquidNewAccAmount.String()).To(Equal(twopointfiveLiquid1.String()))
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(halfIslm.String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(twopointfiveLiquid1.String()))
			})

			It("should fail - NewMsgTransferOwnershipWithRatio - invalid ratio", func() {
				daoModuleAddressBankBalanceBeforeFund := s.app.BankKeeper.GetAllBalances(s.ctx, daoModuleAcc.GetAddress())
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "dao module account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.address,
				)
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
				Expect(err).To(BeNil(), "transaction should have succeed")
				s.Commit()

				// Check balances after funding TX
				daoModuleAddressBankBalanceAfterFundIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "dao account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "dao account should have received the funds")

				daoTotalBalanceAfterFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithRatioMsg = types.NewMsgTransferOwnershipWithRatio(s.address, newOwnerAddr, sdkmath.LegacyNewDecWithPrec(15, 1))
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipWithRatioMsg)
				Expect(err).NotTo(BeNil(), "transaction should fail")
				Expect(err.Error()).To(ContainSubstring("invalid ratio"), "transaction should fail")
				s.Commit()
			})

			It("successfully transferred - NewMsgTransferOwnershipWithAmount - all aISLM, aLIQUID intact", func() {
				daoModuleAddressBankBalanceBeforeFund := s.app.BankKeeper.GetAllBalances(s.ctx, daoModuleAcc.GetAddress())
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "dao module account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.address,
				)
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
				Expect(err).To(BeNil(), "transaction should have succeed")
				s.Commit()

				// Check balances after funding TX
				daoModuleAddressBankBalanceAfterFundIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "dao account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "dao account should have received the funds")

				daoTotalBalanceAfterFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithAmountMsg = types.NewMsgTransferOwnershipWithAmount(s.address, newOwnerAddr, sdk.NewCoins(oneIslm))
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipWithAmountMsg)
				Expect(err).To(BeNil(), "transaction should succeed")
				s.Commit()

				// Checks after transfer ownership
				// Module bank balance shouldn't change
				daoModuleAddressBankBalanceAfterTransferIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferIslm.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()))
				daoModuleAddressBankBalanceAfterTransferLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferLiquid.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()))

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should contain only liquid coins
				daoOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, _ := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeFalse())
				found, intactLiquidOwnerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(intactLiquidOwnerAmount.String()).To(Equal(fiveLiquid1.String()))

				// Old owner internal dao balance should contain only islm
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(oneIslm.String()))
				found, _ = daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeFalse())
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(oneIslm.String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(sdk.NewCoin(fiveLiquid1.Denom, sdk.ZeroInt()).String()))
			})

			It("successfully transferred - NewMsgTransferOwnershipWithAmount - part aISLM, aLIQUID intact", func() {
				daoModuleAddressBankBalanceBeforeFund := s.app.BankKeeper.GetAllBalances(s.ctx, daoModuleAcc.GetAddress())
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "dao module account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.address,
				)
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
				Expect(err).To(BeNil(), "transaction should have succeed")
				s.Commit()

				// Check balances after funding TX
				daoModuleAddressBankBalanceAfterFundIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "dao account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "dao account should have received the funds")

				daoTotalBalanceAfterFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithAmountMsg = types.NewMsgTransferOwnershipWithAmount(s.address, newOwnerAddr, sdk.NewCoins(halfIslm))
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipWithAmountMsg)
				Expect(err).To(BeNil(), "transaction should succeed")
				s.Commit()

				// Checks after transfer ownership
				// Module bank balance shouldn't change
				daoModuleAddressBankBalanceAfterTransferIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferIslm.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()))
				daoModuleAddressBankBalanceAfterTransferLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferLiquid.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()))

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should become lower by 50%
				daoOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, halfIslmownerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(halfIslmownerAmount.String()).To(Equal(halfIslm.String()))
				found, halfLiquidOwnerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(halfLiquidOwnerAmount.String()).To(Equal(fiveLiquid1.String()))

				// 50% of all tokens should be transferred to new owner
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(halfIslm.String()))
				found, _ = daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeFalse())
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(halfIslm.String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(sdk.NewCoin(fiveLiquid1.Denom, sdk.ZeroInt()).String()))
			})

			It("successfully transferred - NewMsgTransferOwnershipWithAmount - part aISLM, part aLIQUID", func() {
				daoModuleAddressBankBalanceBeforeFund := s.app.BankKeeper.GetAllBalances(s.ctx, daoModuleAcc.GetAddress())
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "dao module account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.address,
				)
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
				Expect(err).To(BeNil(), "transaction should have succeed")
				s.Commit()

				// Check balances after funding TX
				daoModuleAddressBankBalanceAfterFundIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "dao account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "dao account should have received the funds")

				daoTotalBalanceAfterFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithAmountMsg = types.NewMsgTransferOwnershipWithAmount(s.address, newOwnerAddr, sdk.NewCoins(halfIslm, twopointfiveLiquid1))
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipWithAmountMsg)
				Expect(err).To(BeNil(), "transaction should succeed")
				s.Commit()

				// Checks after transfer ownership
				// Module bank balance shouldn't change
				daoModuleAddressBankBalanceAfterTransferIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferIslm.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()))
				daoModuleAddressBankBalanceAfterTransferLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferLiquid.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()))

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should become lower by 50%
				daoOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, halfIslmownerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(halfIslmownerAmount.String()).To(Equal(halfIslm.String()))
				found, halfLiquidOwnerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(halfLiquidOwnerAmount.String()).To(Equal(twopointfiveLiquid1.String()))

				// 50% of all tokens should be transferred to new owner
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(halfIslm.String()))
				found, liquidNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(liquidNewAccAmount.String()).To(Equal(twopointfiveLiquid1.String()))
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(halfIslm.String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(twopointfiveLiquid1.String()))
			})

			It("should fail - NewMsgTransferOwnershipWithAmount - insufficient funds", func() {
				daoModuleAddressBankBalanceBeforeFund := s.app.BankKeeper.GetAllBalances(s.ctx, daoModuleAcc.GetAddress())
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "dao module account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.address,
				)
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
				Expect(err).To(BeNil(), "transaction should have succeed")
				s.Commit()

				// Check balances after funding TX
				daoModuleAddressBankBalanceAfterFundIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "dao account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "dao account should have received the funds")

				daoTotalBalanceAfterFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithAmountMsg = types.NewMsgTransferOwnershipWithAmount(s.address, newOwnerAddr, sdk.NewCoins(twoIslm))
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipWithAmountMsg)
				Expect(err).NotTo(BeNil(), "transaction should fail")
				Expect(err.Error()).To(ContainSubstring("insufficient funds"), "transaction should fail")
				s.Commit()
			})
		})

		Context("with both owner and new owner as members", func() {
			BeforeEach(func() {
				s.SetupTest()

				// Fund owner account
				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, s.address, sdk.NewCoins(oneHundredIslm, fiveLiquid1))
				s.Require().NoError(err)

				// Fund new owner account
				err = testutil.FundAccount(s.ctx, s.app.BankKeeper, newOwnerAddr, sdk.NewCoins(oneHundredIslm))
				s.Require().NoError(err)

				s.Commit()
			})

			It("successfully transferred - NewMsgTransferOwnership", func() {
				daoModuleAddressBankBalanceBeforeFund := s.app.BankKeeper.GetAllBalances(s.ctx, daoModuleAcc.GetAddress())
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "dao module account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund by owner TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.address,
				)
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, fundMsg)
				Expect(err).To(BeNil(), "transaction should have succeed")
				s.Commit()

				// Fund by new owner TX Process
				twoIslm := oneIslm.Add(oneIslm)
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(twoIslm),
					newOwnerAddr,
				)
				_, err = testutil.DeliverTx(s.ctx, s.app, newOwnerPriv, &gasPrice, signMode, fundMsg)
				Expect(err).To(BeNil(), "transaction should have succeed")
				s.Commit()

				// Check balances after funding TX
				daoModuleAddressBankBalanceAfterFundIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.Add(twoIslm.Amount).String()))
				daoModuleAddressBankBalanceAfterFundLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()))

				daoTotalBalanceAfterFund, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.Add(twoIslm).String()))
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()))

				daoOwnerAddressAllBalancesAfterFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse())
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue())
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()))
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue())
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()))

				daoOwnerAddressIslmBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()))
				daoOwnerAddressLiquidBalanceAfterFund, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: s.address.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()))

				daoNewOwnerAddressAllBalancesAfterFund, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse())
				okNewAcc, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterFund.Balances.Find(twoIslm.Denom)
				Expect(okNewAcc).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(twoIslm.String()))

				// Transfer TX Process
				transferOwnershipMsg = types.NewMsgTransferOwnership(s.address, newOwnerAddr)
				_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, signMode, transferOwnershipMsg)
				Expect(err).To(BeNil(), "transaction should succeed")
				s.Commit()

				// Checks after transfer ownership
				// Module bank balance shouldn't change
				daoModuleAddressBankBalanceAfterTransferIslm := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferIslm.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()))
				daoModuleAddressBankBalanceAfterTransferLiquid := s.app.BankKeeper.GetBalance(s.ctx, daoModuleAcc.GetAddress(), fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterTransferLiquid.Amount.String()).To(Equal(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()))

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.queryClient.TotalBalance(s.ctx, &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should become empty
				daoOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: s.address.String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeTrue())

				// All tokens should be transferred to new owner
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.queryClient.AllBalances(s.ctx, &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				okNewAcc, islmNewAccAmount = daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(okNewAcc).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(oneIslm.Add(twoIslm).String()))
				okNewAcc, liquidNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(okNewAcc).To(BeTrue())
				Expect(liquidNewAccAmount.String()).To(Equal(fiveLiquid1.String()))
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(oneIslm.Add(twoIslm).String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.queryClient.Balance(s.ctx, &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(fiveLiquid1.String()))
			})
		})
	},
		Entry("Direct sign mode", signing.SignMode_SIGN_MODE_DIRECT),
		Entry("Legacy Amino JSON sign mode", signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON),
	)
})
