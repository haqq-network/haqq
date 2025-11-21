package keeper_test

import (
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ucdao/types"
)

var gasLimit uint64 = 400_000

var _ = Describe("United Contributors DAO", func() {
	var (
		fundMsg                        *types.MsgFund
		transferOwnershipMsg           *types.MsgTransferOwnership
		transferOwnershipWithRatioMsg  *types.MsgTransferOwnershipWithRatio
		transferOwnershipWithAmountMsg *types.MsgTransferOwnershipWithAmount
		err                            error
	)

	oneHundred, _ := sdkmath.NewIntFromString("100000000000000000000")
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

	// TODO Add signMode testing
	DescribeTableSubtree("Fund transactions", func(_ signing.SignMode) {
		Context("with invalid denom", func() {
			BeforeEach(func() {
				s.SetupTest()

				escrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoBalanceBefore := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, utils.BaseDenom)
				Expect(daoBalanceBefore.IsZero()).To(BeTrue(), "escrow account should have no balance")

				err = testutil.FundAccount(s.network.GetContext(), s.network.App.BankKeeper, s.keyring.GetAccAddr(0), sdk.NewCoins(oneHundredIslm, threeInvalid, fiveLiquid1, sevenLiquid75, nineLiquidInvalid))
				s.Require().NoError(err)

				s.Require().NoError(s.network.NextBlock())
			})

			Context("correct aLIQUID with invalid", func() {
				It("should fail", func() {
					// Check balances before TX
					escrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankBalanceBefore := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, fiveLiquid1.Denom)
					Expect(daoBankBalanceBefore.IsZero()).To(BeTrue(), "escrow account should have no balance")

					daoTotalBalanceBefore, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceBefore, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					// TX Process
					fundMsg = types.NewMsgFund(
						sdk.NewCoins(threeInvalid, fiveLiquid1),
						s.keyring.GetAccAddr(0),
					)
					res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
					Expect(err).To(BeNil())
					Expect(res.IsOK()).To(BeFalse(), "transaction should have failed")
					Expect(res.Log).To(ContainSubstring("denom invalid is not allowed"), "transaction should fail")
					s.Require().NoError(s.network.NextBlock())

					// Check balances after TX
					escrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankBalanceAfter := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, fiveLiquid1.Denom)
					Expect(daoBankBalanceAfter.IsZero()).To(BeTrue(), "escrow account should have no balance")

					daoTotalBalanceAfter, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceAfter.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceAfter, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceAfter.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")
				})
			})

			Context("correct aISLM with invalid", func() {
				It("should fail", func() {
					// Check balances before TX
					escrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankBalanceBefore := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, oneIslm.Denom)
					Expect(daoBankBalanceBefore.IsZero()).To(BeTrue(), "escrow account should have no balance")

					daoTotalBalanceBefore, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceBefore, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					// TX Process
					fundMsg = types.NewMsgFund(
						sdk.NewCoins(oneIslm, nineLiquidInvalid),
						s.keyring.GetAccAddr(0),
					)
					res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
					Expect(err).To(BeNil())
					Expect(res.IsOK()).To(BeFalse(), "transaction should have failed")
					Expect(res.Log).To(ContainSubstring("denom aLIQUID is not allowed"), "transaction should fail")
					s.Require().NoError(s.network.NextBlock())

					// Check balances after TX
					escrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankBalanceAfter := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, oneIslm.Denom)
					Expect(daoBankBalanceAfter.IsZero()).To(BeTrue(), "escrow account should have no balance")

					daoTotalBalanceAfter, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceAfter.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceAfter, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceAfter.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")
				})
			})
		})

		Context("with correct denoms", func() {
			BeforeEach(func() {
				s.SetupTest()

				err = testutil.FundAccount(s.network.GetContext(), s.network.App.BankKeeper, s.keyring.GetAccAddr(0), sdk.NewCoins(oneHundredIslm, threeInvalid, fiveLiquid1, sevenLiquid75, nineLiquidInvalid))
				s.Require().NoError(err)

				s.Require().NoError(s.network.NextBlock())
			})

			Context("correct standalone aLIQUID", func() {
				It("should pass", func() {
					// Check balances before TX
					escrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankBalanceBefore := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, sevenLiquid75.Denom)
					Expect(daoBankBalanceBefore.IsZero()).To(BeTrue(), "escrow account should have no balance")

					daoTotalBalanceBefore, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceBefore, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					// TX Process
					fundMsg = types.NewMsgFund(
						sdk.NewCoins(sevenLiquid75),
						s.keyring.GetAccAddr(0),
					)
					res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
					Expect(err).To(BeNil(), "transaction should have succeed")
					Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
					s.Require().NoError(s.network.NextBlock())

					// Check balances after TX
					escrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankBalanceAfter := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, sevenLiquid75.Denom)
					Expect(daoBankBalanceAfter.Amount.String()).To(Equal(sevenLiquid75.Amount.String()), "escrow account should have received the funds")

					daoTotalBalanceAfter, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					ok, amount := daoTotalBalanceAfter.TotalBalance.Find(sevenLiquid75.Denom)
					Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
					Expect(amount.String()).To(Equal(sevenLiquid75.String()), "dao total balance should have received the funds")

					daoAddressBalanceAfter, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
					Expect(err).To(BeNil(), "query should have succeed")
					okAcc, amountAcc := daoAddressBalanceAfter.Balances.Find(sevenLiquid75.Denom)
					Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
					Expect(amountAcc.String()).To(Equal(sevenLiquid75.String()), "dao address balance should have received the funds")
				})
			})

			Context("correct standalone aISLM", func() {
				It("should pass", func() {
					// Check balances before TX
					escrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankBalanceBefore := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, oneIslm.Denom)
					Expect(daoBankBalanceBefore.IsZero()).To(BeTrue(), "escrow account should have no balance")

					daoTotalBalanceBefore, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressBalanceBefore, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					// TX Process
					fundMsg = types.NewMsgFund(
						sdk.NewCoins(oneIslm),
						s.keyring.GetAccAddr(0),
					)
					res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
					Expect(err).To(BeNil(), "transaction should have succeed")
					Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
					s.Require().NoError(s.network.NextBlock())

					// Check balances after TX
					escrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankBalanceAfter := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, oneIslm.Denom)
					Expect(daoBankBalanceAfter.Amount.String()).To(Equal(oneIslm.Amount.String()), "escrow account should have received the funds")

					daoTotalBalanceAfter, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					ok, amount := daoTotalBalanceAfter.TotalBalance.Find(oneIslm.Denom)
					Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
					Expect(amount.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")

					daoAddressBalanceAfter, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
					Expect(err).To(BeNil(), "query should have succeed")
					okAcc, amountAcc := daoAddressBalanceAfter.Balances.Find(oneIslm.Denom)
					Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
					Expect(amountAcc.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				})
			})

			Context("correct aISLM and aLIQUID", func() {
				It("should pass", func() {
					// Check balances before TX
					escrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankIslmBalanceBefore := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, oneIslm.Denom)
					Expect(daoBankIslmBalanceBefore.IsZero()).To(BeTrue(), "escrow account should have no balance")

					daoTotalIslmBalanceBefore, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalIslmBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressIslmBalanceBefore, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressIslmBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					escrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankLiquidBalanceBefore := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, fiveLiquid1.Denom)
					Expect(daoBankLiquidBalanceBefore.IsZero()).To(BeTrue(), "escrow account should have no balance")

					daoTotalLiquidBalanceBefore, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoTotalLiquidBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

					daoAddressLiquidBalanceBefore, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
					Expect(err).To(BeNil(), "query should have succeed")
					Expect(daoAddressLiquidBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

					// TX Process
					fundMsg = types.NewMsgFund(
						sdk.NewCoins(oneIslm, fiveLiquid1),
						s.keyring.GetAccAddr(0),
					)
					res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
					Expect(err).To(BeNil(), "transaction should have succeed")
					Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
					s.Require().NoError(s.network.NextBlock())

					// Check balances after TX
					escrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankIslmBalanceAfter := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, oneIslm.Denom)
					Expect(daoBankIslmBalanceAfter.Amount.String()).To(Equal(oneIslm.Amount.String()), "escrow account should have received the funds")

					daoTotalIslmBalanceAfter, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					ok, amount := daoTotalIslmBalanceAfter.TotalBalance.Find(oneIslm.Denom)
					Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
					Expect(amount.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")

					daoAddressIslmBalanceAfter, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
					Expect(err).To(BeNil(), "query should have succeed")
					okAcc, amountAcc := daoAddressIslmBalanceAfter.Balances.Find(oneIslm.Denom)
					Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
					Expect(amountAcc.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")

					escrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
					daoBankLiquidBalanceAfter := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), escrowAddr, fiveLiquid1.Denom)
					Expect(daoBankLiquidBalanceAfter.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "escrow account should have received the funds")

					daoTotalLiquidBalanceAfter, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
					Expect(err).To(BeNil(), "query should have succeed")
					ok, amountLiq := daoTotalLiquidBalanceAfter.TotalBalance.Find(fiveLiquid1.Denom)
					Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
					Expect(amountLiq.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

					daoAddressLiquidBalanceAfter, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
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

	// TODO Add signMode testing
	DescribeTableSubtree("Transfer Ownership transactions", func(_ signing.SignMode) {
		newOwnerPriv, err := ethsecp256k1.GenerateKey()
		newOwnerAddr := sdk.AccAddress(newOwnerPriv.PubKey().Address().Bytes())

		Context("basic validation", func() {
			BeforeEach(func() {
				s.SetupTest()

				err = testutil.FundAccount(s.network.GetContext(), s.network.App.BankKeeper, s.keyring.GetAccAddr(0), sdk.NewCoins(oneHundredIslm))
				s.Require().NoError(err)

				s.Require().NoError(s.network.NextBlock())
			})

			It("should fail - invalid owner address", func() {
				// TX Process
				transferOwnershipMsg = &types.MsgTransferOwnership{
					Owner:    "haqq1",
					NewOwner: newOwnerAddr.String(),
				}

				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeFalse(), "transaction should have failed")
				Expect(res.Log).To(ContainSubstring("invalid owner address"))
				s.Require().NoError(s.network.NextBlock())
			})

			It("should fail - invalid new owner address", func() {
				// TX Process
				transferOwnershipMsg = &types.MsgTransferOwnership{
					Owner:    s.keyring.GetAccAddr(0).String(),
					NewOwner: "haqq1",
				}

				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeFalse(), "transaction should have failed")
				Expect(res.Log).To(ContainSubstring("invalid new owner address"))
				s.Require().NoError(s.network.NextBlock())
			})
		})

		Context("with non-member as owner", func() {
			BeforeEach(func() {
				s.SetupTest()

				err = testutil.FundAccount(s.network.GetContext(), s.network.App.BankKeeper, s.keyring.GetAccAddr(0), sdk.NewCoins(oneHundredIslm))
				s.Require().NoError(err)

				s.Require().NoError(s.network.NextBlock())
			})

			It("should fail - not eligible NewMsgTransferOwnership", func() {
				daoTotalBalanceBefore, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoAddressBalanceBefore, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

				// TX Process
				transferOwnershipMsg = types.NewMsgTransferOwnership(s.keyring.GetAccAddr(0), newOwnerAddr)

				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeFalse(), "transaction should have failed")
				Expect(res.Log).To(ContainSubstring("not eligible"), "error message should be correct")
				s.Require().NoError(s.network.NextBlock())
			})
			It("should fail - not eligible NewMsgTransferOwnershipWithRatio", func() {
				daoTotalBalanceBefore, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoAddressBalanceBefore, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

				// TX Process — 50% ownership
				transferOwnershipWithRatioMsg = types.NewMsgTransferOwnershipWithRatio(s.keyring.GetAccAddr(0), newOwnerAddr, sdkmath.LegacyNewDecWithPrec(5, 1))

				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipWithRatioMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeFalse(), "transaction should have failed")
				Expect(res.Log).To(ContainSubstring("not eligible"), "error message should be correct")
				s.Require().NoError(s.network.NextBlock())
			})
			It("should fail - not eligible NewMsgTransferOwnershipWithAmount", func() {
				daoTotalBalanceBefore, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBefore.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoAddressBalanceBefore, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoAddressBalanceBefore.Balances.IsZero()).To(BeTrue(), "dao address balance should be empty")

				// TX Process — 1000 aISLM ownership
				transferOwnershipWithAmountMsg = types.NewMsgTransferOwnershipWithAmount(s.keyring.GetAccAddr(0), newOwnerAddr, sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000))))

				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipWithAmountMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeFalse(), "transaction should have failed")
				Expect(res.Log).To(ContainSubstring("not eligible"), "error message should be correct")
				s.Require().NoError(s.network.NextBlock())
			})
		})

		Context("with member as owner and non-member as new owner", func() {
			BeforeEach(func() {
				s.SetupTest()

				err = testutil.FundAccount(s.network.GetContext(), s.network.App.BankKeeper, s.keyring.GetAccAddr(0), sdk.NewCoins(oneHundredIslm, fiveLiquid1))
				s.Require().NoError(err)

				s.Require().NoError(s.network.NextBlock())
			})

			It("successfully transferred - NewMsgTransferOwnership", func() {
				ownerEscrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceBeforeFund := s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), ownerEscrowAddr)
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "owner escrow account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.keyring.GetAccAddr(0),
				)
				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Check balances after funding TX
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceAfterFundIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "owner escrow account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "owner escrow account should have received the funds")

				daoTotalBalanceAfterFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipMsg = types.NewMsgTransferOwnership(s.keyring.GetAccAddr(0), newOwnerAddr)
				res, err = s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Checks after transfer ownership
				// Total escrow balance shouldn't change (sum of all escrow addresses)
				// Owner escrow should be empty, new owner escrow should have all the funds
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				newOwnerEscrowAddr := types.GetEscrowAddress(newOwnerAddr)
				ownerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(ownerEscrowBalanceAfterTransferIslm.IsZero()).To(BeTrue(), "owner escrow should be empty after full transfer")
				newOwnerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, oneIslm.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "new owner escrow should have all aISLM")
				ownerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(ownerEscrowBalanceAfterTransferLiquid.IsZero()).To(BeTrue(), "owner escrow should be empty after full transfer")
				newOwnerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, fiveLiquid1.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "new owner escrow should have all aLIQUID")

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should become empty
				daoOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeTrue())

				// All tokens should be transferred to new owner
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				okNewAcc, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(okNewAcc).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(oneIslm.String()))
				okNewAcc, liquidNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(okNewAcc).To(BeTrue())
				Expect(liquidNewAccAmount.String()).To(Equal(fiveLiquid1.String()))
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(oneIslm.String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(fiveLiquid1.String()))
			})

			It("successfully transferred - NewMsgTransferOwnershipWithRatio", func() {
				ownerEscrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceBeforeFund := s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), ownerEscrowAddr)
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "owner escrow account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.keyring.GetAccAddr(0),
				)
				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Check balances after funding TX
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceAfterFundIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "owner escrow account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "owner escrow account should have received the funds")

				daoTotalBalanceAfterFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithRatioMsg = types.NewMsgTransferOwnershipWithRatio(s.keyring.GetAccAddr(0), newOwnerAddr, sdkmath.LegacyNewDecWithPrec(5, 1))
				res, err = s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipWithRatioMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Checks after transfer ownership
				// Total escrow balance shouldn't change (sum of all escrow addresses)
				// Owner escrow should have 50%, new owner escrow should have 50%
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				newOwnerEscrowAddr := types.GetEscrowAddress(newOwnerAddr)
				ownerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(ownerEscrowBalanceAfterTransferIslm.Amount.String()).To(Equal(halfIslm.Amount.String()), "owner escrow should have 50% after transfer")
				newOwnerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, oneIslm.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferIslm.Amount.String()).To(Equal(halfIslm.Amount.String()), "new owner escrow should have 50%")
				ownerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(ownerEscrowBalanceAfterTransferLiquid.Amount.String()).To(Equal(twopointfiveLiquid1.Amount.String()), "owner escrow should have 50% after transfer")
				newOwnerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, fiveLiquid1.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferLiquid.Amount.String()).To(Equal(twopointfiveLiquid1.Amount.String()), "new owner escrow should have 50%")

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should become lower by 50%
				daoOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, halfIslmownerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(halfIslmownerAmount.String()).To(Equal(halfIslm.String()))
				found, halfLiquidOwnerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(halfLiquidOwnerAmount.String()).To(Equal(twopointfiveLiquid1.String()))

				// 50% of all tokens should be transferred to new owner
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(halfIslm.String()))
				found, liquidNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(liquidNewAccAmount.String()).To(Equal(twopointfiveLiquid1.String()))
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(halfIslm.String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(twopointfiveLiquid1.String()))
			})

			It("should fail - NewMsgTransferOwnershipWithRatio - invalid ratio", func() {
				ownerEscrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceBeforeFund := s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), ownerEscrowAddr)
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "owner escrow account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.keyring.GetAccAddr(0),
				)
				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Check balances after funding TX
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceAfterFundIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "owner escrow account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "owner escrow account should have received the funds")

				daoTotalBalanceAfterFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithRatioMsg = types.NewMsgTransferOwnershipWithRatio(s.keyring.GetAccAddr(0), newOwnerAddr, sdkmath.LegacyNewDecWithPrec(15, 1))
				res, err = s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipWithRatioMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeFalse(), "transaction should have failed")
				Expect(res.Log).To(ContainSubstring("invalid ratio"), "transaction should fail")
				s.Require().NoError(s.network.NextBlock())
			})

			It("successfully transferred - NewMsgTransferOwnershipWithAmount - all aISLM, aLIQUID intact", func() {
				ownerEscrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceBeforeFund := s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), ownerEscrowAddr)
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "owner escrow account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.keyring.GetAccAddr(0),
				)
				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Check balances after funding TX
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceAfterFundIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "owner escrow account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "owner escrow account should have received the funds")

				daoTotalBalanceAfterFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithAmountMsg = types.NewMsgTransferOwnershipWithAmount(s.keyring.GetAccAddr(0), newOwnerAddr, sdk.NewCoins(oneIslm))
				res, err = s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipWithAmountMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Checks after transfer ownership
				// Total escrow balance shouldn't change (sum of all escrow addresses)
				// Owner escrow: no aISLM (all transferred), all aLIQUID (intact)
				// New owner escrow: all aISLM, no aLIQUID
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				newOwnerEscrowAddr := types.GetEscrowAddress(newOwnerAddr)
				ownerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(ownerEscrowBalanceAfterTransferIslm.IsZero()).To(BeTrue(), "owner escrow should have no aISLM after transfer")
				newOwnerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, oneIslm.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "new owner escrow should have all aISLM")
				ownerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(ownerEscrowBalanceAfterTransferLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "owner escrow should have all aLIQUID intact")
				newOwnerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, fiveLiquid1.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferLiquid.IsZero()).To(BeTrue(), "new owner escrow should have no aLIQUID")

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should contain only liquid coins
				daoOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, _ := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeFalse())
				found, intactLiquidOwnerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(intactLiquidOwnerAmount.String()).To(Equal(fiveLiquid1.String()))

				// Old owner internal dao balance should contain only islm
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(oneIslm.String()))
				found, _ = daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeFalse())
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(oneIslm.String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(sdk.NewCoin(fiveLiquid1.Denom, sdkmath.ZeroInt()).String()))
			})

			It("successfully transferred - NewMsgTransferOwnershipWithAmount - part aISLM, aLIQUID intact", func() {
				ownerEscrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceBeforeFund := s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), ownerEscrowAddr)
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "owner escrow account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.keyring.GetAccAddr(0),
				)
				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Check balances after funding TX
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceAfterFundIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "owner escrow account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "owner escrow account should have received the funds")

				daoTotalBalanceAfterFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithAmountMsg = types.NewMsgTransferOwnershipWithAmount(s.keyring.GetAccAddr(0), newOwnerAddr, sdk.NewCoins(halfIslm))
				res, err = s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipWithAmountMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Checks after transfer ownership
				// Total escrow balance shouldn't change (sum of all escrow addresses)
				// Owner escrow: halfIslm (50% of aISLM transferred), all fiveLiquid1 (intact)
				// New owner escrow: halfIslm, no aLIQUID
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				newOwnerEscrowAddr := types.GetEscrowAddress(newOwnerAddr)
				ownerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(ownerEscrowBalanceAfterTransferIslm.Amount.String()).To(Equal(halfIslm.Amount.String()), "owner escrow should have 50% aISLM after transfer")
				newOwnerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, oneIslm.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferIslm.Amount.String()).To(Equal(halfIslm.Amount.String()), "new owner escrow should have 50% aISLM")
				ownerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(ownerEscrowBalanceAfterTransferLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "owner escrow should have all aLIQUID intact")
				newOwnerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, fiveLiquid1.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferLiquid.IsZero()).To(BeTrue(), "new owner escrow should have no aLIQUID")

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should become lower by 50%
				daoOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, halfIslmownerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(halfIslmownerAmount.String()).To(Equal(halfIslm.String()))
				found, halfLiquidOwnerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(halfLiquidOwnerAmount.String()).To(Equal(fiveLiquid1.String()))

				// 50% of all tokens should be transferred to new owner
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(halfIslm.String()))
				found, _ = daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeFalse())
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(halfIslm.String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(sdk.NewCoin(fiveLiquid1.Denom, sdkmath.ZeroInt()).String()))
			})

			It("successfully transferred - NewMsgTransferOwnershipWithAmount - part aISLM, part aLIQUID", func() {
				ownerEscrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceBeforeFund := s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), ownerEscrowAddr)
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "owner escrow account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.keyring.GetAccAddr(0),
				)
				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Check balances after funding TX
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceAfterFundIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "owner escrow account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "owner escrow account should have received the funds")

				daoTotalBalanceAfterFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithAmountMsg = types.NewMsgTransferOwnershipWithAmount(s.keyring.GetAccAddr(0), newOwnerAddr, sdk.NewCoins(halfIslm, twopointfiveLiquid1))
				res, err = s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipWithAmountMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Checks after transfer ownership
				// Total escrow balance shouldn't change (sum of all escrow addresses)
				// Owner escrow should have 50%, new owner escrow should have 50%
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				newOwnerEscrowAddr := types.GetEscrowAddress(newOwnerAddr)
				ownerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(ownerEscrowBalanceAfterTransferIslm.Amount.String()).To(Equal(halfIslm.Amount.String()), "owner escrow should have 50% after transfer")
				newOwnerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, oneIslm.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferIslm.Amount.String()).To(Equal(halfIslm.Amount.String()), "new owner escrow should have 50%")
				ownerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(ownerEscrowBalanceAfterTransferLiquid.Amount.String()).To(Equal(twopointfiveLiquid1.Amount.String()), "owner escrow should have 50% after transfer")
				newOwnerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, fiveLiquid1.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferLiquid.Amount.String()).To(Equal(twopointfiveLiquid1.Amount.String()), "new owner escrow should have 50%")

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should become lower by 50%
				daoOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, halfIslmownerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(halfIslmownerAmount.String()).To(Equal(halfIslm.String()))
				found, halfLiquidOwnerAmount := daoOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(halfLiquidOwnerAmount.String()).To(Equal(twopointfiveLiquid1.String()))

				// 50% of all tokens should be transferred to new owner
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				found, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(found).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(halfIslm.String()))
				found, liquidNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(found).To(BeTrue())
				Expect(liquidNewAccAmount.String()).To(Equal(twopointfiveLiquid1.String()))
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(halfIslm.String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(twopointfiveLiquid1.String()))
			})

			It("should fail - NewMsgTransferOwnershipWithAmount - insufficient funds", func() {
				ownerEscrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceBeforeFund := s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), ownerEscrowAddr)
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "owner escrow account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.keyring.GetAccAddr(0),
				)
				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Check balances after funding TX
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceAfterFundIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "owner escrow account should have received the funds")
				daoModuleAddressBankBalanceAfterFundLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(daoModuleAddressBankBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "owner escrow account should have received the funds")

				daoTotalBalanceAfterFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse(), "dao total balance should not be empty")
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.String()), "dao total balance should have received the funds")
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue(), "dao total balance should have received the funds")
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()), "dao total balance should have received the funds")

				daoOwnerAddressAllBalancesAfterFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse(), "dao account balance should not be empty")
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue(), "dao address balance should have received the funds")
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				daoOwnerAddressIslmBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()), "dao address balance should have received the funds")
				daoOwnerAddressLiquidBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()), "dao address balance should have received the funds")

				// Store new owner address balances before transfer ownership
				daoNewOwnerAddressAllBalancesBeforeTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoNewOwnerAddressAllBalancesBeforeTransfer.Balances.IsZero()).To(BeTrue(), "dao new owner address balance should be empty")

				// Transfer TX Process
				transferOwnershipWithAmountMsg = types.NewMsgTransferOwnershipWithAmount(s.keyring.GetAccAddr(0), newOwnerAddr, sdk.NewCoins(twoIslm))
				res, err = s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipWithAmountMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeFalse(), "transaction should have failed")
				Expect(res.Log).To(ContainSubstring("insufficient funds"), "transaction should fail")
				s.Require().NoError(s.network.NextBlock())
			})
		})

		Context("with both owner and new owner as members", func() {
			BeforeEach(func() {
				s.SetupTest()

				// Fund owner account
				err = testutil.FundAccount(s.network.GetContext(), s.network.App.BankKeeper, s.keyring.GetAccAddr(0), sdk.NewCoins(oneHundredIslm, fiveLiquid1))
				s.Require().NoError(err)

				// Fund new owner account
				err = testutil.FundAccount(s.network.GetContext(), s.network.App.BankKeeper, newOwnerAddr, sdk.NewCoins(oneHundredIslm))
				s.Require().NoError(err)

				s.Require().NoError(s.network.NextBlock())
			})

			It("successfully transferred - NewMsgTransferOwnership", func() {
				ownerEscrowAddr := types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				daoModuleAddressBankBalanceBeforeFund := s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), ownerEscrowAddr)
				Expect(daoModuleAddressBankBalanceBeforeFund.IsZero()).To(BeTrue(), "owner escrow account bank balance should be empty")

				daoTotalBalanceBeforeFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoTotalBalanceBeforeFund.TotalBalance.IsZero()).To(BeTrue(), "dao total balance should be empty")

				daoOwnerAddressBalanceBeforeFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil(), "query should have succeed")
				Expect(daoOwnerAddressBalanceBeforeFund.Balances.IsZero()).To(BeTrue(), "dao module address balance should be empty")

				// Fund by owner TX Process
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(oneIslm, fiveLiquid1),
					s.keyring.GetAccAddr(0),
				)
				res, err := s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Fund by new owner TX Process
				twoIslm := oneIslm.Add(oneIslm)
				fundMsg = types.NewMsgFund(
					sdk.NewCoins(twoIslm),
					newOwnerAddr,
				)
				res, err = s.factory.CommitCosmosTx(newOwnerPriv, factory.CosmosTxArgs{Msgs: []sdk.Msg{fundMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Check balances after funding TX
				// Check individual escrow addresses
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				newOwnerEscrowAddr := types.GetEscrowAddress(newOwnerAddr)
				ownerEscrowBalanceAfterFundIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(ownerEscrowBalanceAfterFundIslm.Amount.String()).To(Equal(oneIslm.Amount.String()), "owner escrow should have oneIslm")
				newOwnerEscrowBalanceAfterFundIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, oneIslm.Denom)
				Expect(newOwnerEscrowBalanceAfterFundIslm.Amount.String()).To(Equal(twoIslm.Amount.String()), "new owner escrow should have twoIslm")
				ownerEscrowBalanceAfterFundLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(ownerEscrowBalanceAfterFundLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "owner escrow should have fiveLiquid1")

				daoTotalBalanceAfterFund, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterFund.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterFund.String()).To(Equal(oneIslm.Add(twoIslm).String()))
				ok, liquidAmountAfterFund := daoTotalBalanceAfterFund.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterFund.String()).To(Equal(fiveLiquid1.String()))

				daoOwnerAddressAllBalancesAfterFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse())
				okAcc, islmAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(oneIslm.Denom)
				Expect(okAcc).To(BeTrue())
				Expect(islmAccAmount.String()).To(Equal(oneIslm.String()))
				okAcc, liquidAccAmount := daoOwnerAddressAllBalancesAfterFund.Balances.Find(fiveLiquid1.Denom)
				Expect(okAcc).To(BeTrue())
				Expect(liquidAccAmount.String()).To(Equal(fiveLiquid1.String()))

				daoOwnerAddressIslmBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressIslmBalanceAfterFund.Balance.String()).To(Equal(oneIslm.String()))
				daoOwnerAddressLiquidBalanceAfterFund, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: s.keyring.GetAccAddr(0).String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressLiquidBalanceAfterFund.Balance.String()).To(Equal(fiveLiquid1.String()))

				daoNewOwnerAddressAllBalancesAfterFund, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterFund.Balances.IsZero()).To(BeFalse())
				okNewAcc, islmNewAccAmount := daoNewOwnerAddressAllBalancesAfterFund.Balances.Find(twoIslm.Denom)
				Expect(okNewAcc).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(twoIslm.String()))

				// Transfer TX Process
				transferOwnershipMsg = types.NewMsgTransferOwnership(s.keyring.GetAccAddr(0), newOwnerAddr)
				res, err = s.factory.CommitCosmosTx(s.keyring.GetPrivKey(0), factory.CosmosTxArgs{Msgs: []sdk.Msg{transferOwnershipMsg}, GasPrice: &gasPrice, Gas: &gasLimit})
				Expect(err).To(BeNil())
				Expect(res.IsOK()).To(BeTrue(), "transaction should have succeed")
				s.Require().NoError(s.network.NextBlock())

				// Checks after transfer ownership
				// Total escrow balance shouldn't change (sum of all escrow addresses)
				// Owner escrow should be empty, new owner escrow should have all funds
				ownerEscrowAddr = types.GetEscrowAddress(s.keyring.GetAccAddr(0))
				newOwnerEscrowAddr = types.GetEscrowAddress(newOwnerAddr)
				ownerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, oneIslm.Denom)
				Expect(ownerEscrowBalanceAfterTransferIslm.IsZero()).To(BeTrue(), "owner escrow should be empty after full transfer")
				newOwnerEscrowBalanceAfterTransferIslm := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, oneIslm.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferIslm.Amount.String()).To(Equal(oneIslm.Add(twoIslm).Amount.String()), "new owner escrow should have all aISLM")
				ownerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), ownerEscrowAddr, fiveLiquid1.Denom)
				Expect(ownerEscrowBalanceAfterTransferLiquid.IsZero()).To(BeTrue(), "owner escrow should be empty after full transfer")
				newOwnerEscrowBalanceAfterTransferLiquid := s.network.App.BankKeeper.GetBalance(s.network.GetContext(), newOwnerEscrowAddr, fiveLiquid1.Denom)
				Expect(newOwnerEscrowBalanceAfterTransferLiquid.Amount.String()).To(Equal(fiveLiquid1.Amount.String()), "new owner escrow should have all aLIQUID")

				// Module internal total balance shouldn't change
				daoTotalBalanceAfterTransfer, err := s.network.GetUCDAOClient().TotalBalance(s.network.GetContext(), &types.QueryTotalBalanceRequest{})
				Expect(err).To(BeNil())
				Expect(daoTotalBalanceAfterTransfer.TotalBalance.IsZero()).To(BeFalse())
				ok, islmAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(oneIslm.Denom)
				Expect(ok).To(BeTrue())
				Expect(islmAmountAfterTransfer.String()).To(Equal(islmAmountAfterFund.String()))
				ok, liquidAmountAfterTransfer := daoTotalBalanceAfterTransfer.TotalBalance.Find(fiveLiquid1.Denom)
				Expect(ok).To(BeTrue())
				Expect(liquidAmountAfterTransfer.String()).To(Equal(liquidAmountAfterFund.String()))

				// Old owner internal dao balance should become empty
				daoOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: s.keyring.GetAccAddr(0).String()})
				Expect(err).To(BeNil())
				Expect(daoOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeTrue())

				// All tokens should be transferred to new owner
				daoNewOwnerAddressAllBalancesAfterTransfer, err := s.network.GetUCDAOClient().AllBalances(s.network.GetContext(), &types.QueryAllBalancesRequest{Address: newOwnerAddr.String()})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressAllBalancesAfterTransfer.Balances.IsZero()).To(BeFalse())
				okNewAcc, islmNewAccAmount = daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(oneIslm.Denom)
				Expect(okNewAcc).To(BeTrue())
				Expect(islmNewAccAmount.String()).To(Equal(oneIslm.Add(twoIslm).String()))
				okNewAcc, liquidNewAccAmount := daoNewOwnerAddressAllBalancesAfterTransfer.Balances.Find(fiveLiquid1.Denom)
				Expect(okNewAcc).To(BeTrue())
				Expect(liquidNewAccAmount.String()).To(Equal(fiveLiquid1.String()))
				daoNewOwnerAddressIslmBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: oneIslm.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressIslmBalanceAfterTransfer.Balance.String()).To(Equal(oneIslm.Add(twoIslm).String()))
				daoNewOwnerAddressLiquidBalanceAfterTransfer, err := s.network.GetUCDAOClient().Balance(s.network.GetContext(), &types.QueryBalanceRequest{Address: newOwnerAddr.String(), Denom: fiveLiquid1.Denom})
				Expect(err).To(BeNil())
				Expect(daoNewOwnerAddressLiquidBalanceAfterTransfer.Balance.String()).To(Equal(fiveLiquid1.String()))
			})
		})
	},
		Entry("Direct sign mode", signing.SignMode_SIGN_MODE_DIRECT),
		Entry("Legacy Amino JSON sign mode", signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON),
	)
})
