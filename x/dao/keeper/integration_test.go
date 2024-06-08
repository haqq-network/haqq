package keeper_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/dao/types"
)

var _ = Describe("Feemarket", func() {
	var (
		daoModuleAcc authtypes.ModuleAccountI
		fundMsg      *types.MsgFund
		err          error
	)

	Describe("Performing Cosmos transactions", func() {
		oneHundred, _ := sdk.NewIntFromString("100000000000000000000")
		oneHundredIslm := sdk.NewCoin(utils.BaseDenom, oneHundred)
		oneIslm := sdk.NewInt64Coin(utils.BaseDenom, 1000000000000000000)
		threeInvalid := sdk.NewInt64Coin("invalid", 3000000000000000000)
		fiveLiquid1 := sdk.NewInt64Coin("aLIQUID1", 5000000000000000000)
		sevenLiquid75 := sdk.NewInt64Coin("aLIQUID75", 7000000000000000000)
		nineLiquidInvalid := sdk.NewInt64Coin("aLIQUID", 9000000000000000000)
		gasPrice := sdkmath.NewInt(1000000000)

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
					_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, fundMsg)
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
					_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, fundMsg)
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
					_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, fundMsg)
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
					_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, fundMsg)
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
					_, err = testutil.DeliverTx(s.ctx, s.app, s.priv, &gasPrice, fundMsg)
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
	})
})
