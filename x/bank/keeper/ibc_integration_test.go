package keeper_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/app"
	ibctesting "github.com/haqq-network/haqq/ibc/testing"
	teststypes "github.com/haqq-network/haqq/types/tests"
	haqqbankkeeper "github.com/haqq-network/haqq/x/bank/keeper"
	"github.com/haqq-network/haqq/x/erc20/types"
)

var _ = Describe("Check balance of IBC tokens registered as ERC20", Ordered, func() {
	var (
		erc20Symbol      = "CTKN"
		sender, receiver string
		haqqDenom        string
		receiverAcc      sdk.AccAddress
		senderAcc        sdk.AccAddress
		amount           int64 = 10
		pair             *types.TokenPair
		bankQueryClient  banktypes.QueryClient
	)

	// Metadata to register OSMO with a Token Pair for testing
	osmoMeta := banktypes.Metadata{
		Description: "IBC Coin for IBC Osmosis Chain",
		Base:        teststypes.UosmoIbcdenom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    teststypes.UosmoDenomtrace.BaseDenom,
				Exponent: 0,
			},
		},
		Name:    teststypes.UosmoIbcdenom,
		Symbol:  erc20Symbol,
		Display: teststypes.UosmoDenomtrace.BaseDenom,
	}

	BeforeEach(func() {
		s.suiteIBCTesting = true
		s.SetupTest()
		s.suiteIBCTesting = false
	})

	Describe("registered coin", func() {
		BeforeEach(func() {
			receiver = s.IBCOsmosisChain.SenderAccount.GetAddress().String()
			sender = s.HaqqChain.SenderAccount.GetAddress().String()
			receiverAcc = sdk.MustAccAddressFromBech32("haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq")
			senderAcc = sdk.MustAccAddressFromBech32(sender)
			haqqDenom = s.HaqqChain.App.(*app.Haqq).StakingKeeper.BondDenom(s.HaqqChain.GetContext())

			erc20params := types.DefaultParams()
			erc20params.EnableErc20 = false
			err := s.app.Erc20Keeper.SetParams(s.HaqqChain.GetContext(), erc20params)
			s.Require().NoError(err)

			// Send from osmosis to Haqq
			s.SendAndReceiveMessage(s.pathOsmosisHaqq, s.IBCOsmosisChain, "uosmo", amount, receiver, sender, 1, "")
			s.HaqqChain.Coordinator.CommitBlock(s.HaqqChain)
			erc20params.EnableErc20 = true
			err = s.app.Erc20Keeper.SetParams(s.HaqqChain.GetContext(), erc20params)
			s.Require().NoError(err)

			// Register uosmo pair
			pair, err = s.app.Erc20Keeper.RegisterCoin(s.HaqqChain.GetContext(), osmoMeta)
			s.Require().NoError(err)

			bankQueryHelper := baseapp.NewQueryServerTestHelper(s.HaqqChain.GetContext(), s.app.InterfaceRegistry())
			wrappedBankKeeper := haqqbankkeeper.NewWrappedBaseKeeper(s.app.BankKeeper, s.app.Erc20Keeper, s.app.AccountKeeper)
			banktypes.RegisterQueryServer(bankQueryHelper, wrappedBankKeeper)
			bankQueryClient = banktypes.NewQueryClient(bankQueryHelper)
		})

		Describe("Get Balance of a given denom", func() {
			It("internal - should change after conversion", func() {
				uosmoInternalBalanceBeforeConversion := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
				s.Require().Equal(amount, uosmoInternalBalanceBeforeConversion.Amount.Int64())

				// Convert ibc vouchers to erc20 tokens
				msgConvertCoin := types.NewMsgConvertCoin(
					sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
					common.BytesToAddress(senderAcc.Bytes()),
					senderAcc,
				)

				err := msgConvertCoin.ValidateBasic()
				s.Require().NoError(err)

				_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.HaqqChain.GetContext()), msgConvertCoin)
				s.Require().NoError(err)

				s.HaqqChain.Coordinator.CommitBlock()

				// Check balance after conversion
				uosmoInternalBalanceAfterConversion := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
				s.Require().Equal(amount, uosmoInternalBalanceBeforeConversion.Amount.Int64())
				// should be less than before conversion for the amount converted
				s.Require().Equal(uosmoInternalBalanceBeforeConversion.Amount.Int64()-amount, uosmoInternalBalanceAfterConversion.Amount.Int64())
			})
			It("grpc - should be the same after conversion", func() {
				uosmoGrpcBalanceBeforeConversion, err := bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
					Address: sender,
					Denom:   teststypes.UosmoIbcdenom,
				})
				s.Require().NoError(err)
				s.Require().Equal(amount, uosmoGrpcBalanceBeforeConversion.Balance.Amount.Int64())

				// Convert ibc vouchers to erc20 tokens
				msgConvertCoin := types.NewMsgConvertCoin(
					sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
					common.BytesToAddress(senderAcc.Bytes()),
					senderAcc,
				)

				err = msgConvertCoin.ValidateBasic()
				s.Require().NoError(err)

				_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.HaqqChain.GetContext()), msgConvertCoin)
				s.Require().NoError(err)

				s.HaqqChain.Coordinator.CommitBlock()

				// Check balance after conversion
				uosmoGrpcBalanceAfterConversion, err := bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
					Address: sender,
					Denom:   teststypes.UosmoIbcdenom,
				})
				s.Require().NoError(err)
				s.Require().Equal(uosmoGrpcBalanceBeforeConversion.Balance.Amount.Int64(), uosmoGrpcBalanceAfterConversion.Balance.Amount.Int64())
			})
			It("grpc - should behave like internal if ERC20 is disabled", func() {
				uosmoGrpcBalanceBeforeConversion, err := bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
					Address: sender,
					Denom:   teststypes.UosmoIbcdenom,
				})
				s.Require().NoError(err)
				s.Require().Equal(amount, uosmoGrpcBalanceBeforeConversion.Balance.Amount.Int64())

				// Convert ibc vouchers to erc20 tokens
				msgConvertCoin := types.NewMsgConvertCoin(
					sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
					common.BytesToAddress(senderAcc.Bytes()),
					senderAcc,
				)

				err = msgConvertCoin.ValidateBasic()
				s.Require().NoError(err)

				_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.HaqqChain.GetContext()), msgConvertCoin)
				s.Require().NoError(err)

				s.HaqqChain.Coordinator.CommitBlock()

				erc20params := types.DefaultParams()
				erc20params.EnableErc20 = false
				err = s.app.Erc20Keeper.SetParams(s.HaqqChain.GetContext(), erc20params)
				s.Require().NoError(err)

				// Check balance after conversion
				uosmoGrpcBalanceAfterConversion, err := bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
					Address: sender,
					Denom:   teststypes.UosmoIbcdenom,
				})
				s.Require().NoError(err)
				// should be less than before conversion for the amount converted
				s.Require().Equal(uosmoGrpcBalanceBeforeConversion.Balance.Amount.Int64()-amount, uosmoGrpcBalanceAfterConversion.Balance.Amount.Int64())
			})
		})

		Describe("Get All Balances", func() {
			It("internal - should change after conversion", func() {
				internalAllBalancesBeforeConversion := s.app.BankKeeper.GetAllBalances(s.HaqqChain.GetContext(), senderAcc)
				// Should contain 2 denoms
				s.Require().Equal(2, len(internalAllBalancesBeforeConversion))
				found, uosmoBefore := internalAllBalancesBeforeConversion.Find(teststypes.UosmoIbcdenom)
				s.Require().True(found)
				s.Require().Equal(amount, uosmoBefore.Amount.Int64())

				// Convert ibc vouchers to erc20 tokens
				msgConvertCoin := types.NewMsgConvertCoin(
					sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
					common.BytesToAddress(senderAcc.Bytes()),
					senderAcc,
				)

				err := msgConvertCoin.ValidateBasic()
				s.Require().NoError(err)

				_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.HaqqChain.GetContext()), msgConvertCoin)
				s.Require().NoError(err)

				s.HaqqChain.Coordinator.CommitBlock()

				// Check balance after conversion
				internalAllBalancesAfterConversion := s.app.BankKeeper.GetAllBalances(s.HaqqChain.GetContext(), senderAcc)
				// Should contain 1 denom
				s.Require().Equal(1, len(internalAllBalancesAfterConversion))
				found, _ = internalAllBalancesAfterConversion.Find(teststypes.UosmoIbcdenom)
				s.Require().False(found)
			})
			It("grpc - should be the same after conversion", func() {
				grpcAllBalancesBeforeConversion, err := bankQueryClient.AllBalances(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryAllBalancesRequest{
					Address: sender,
				})
				s.Require().NoError(err)
				s.Require().Equal(2, len(grpcAllBalancesBeforeConversion.Balances))
				found, uosmoBefore := grpcAllBalancesBeforeConversion.Balances.Find(teststypes.UosmoIbcdenom)
				s.Require().True(found)
				s.Require().Equal(amount, uosmoBefore.Amount.Int64())

				// Convert ibc vouchers to erc20 tokens
				msgConvertCoin := types.NewMsgConvertCoin(
					sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
					common.BytesToAddress(senderAcc.Bytes()),
					senderAcc,
				)

				err = msgConvertCoin.ValidateBasic()
				s.Require().NoError(err)

				_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.HaqqChain.GetContext()), msgConvertCoin)
				s.Require().NoError(err)

				s.HaqqChain.Coordinator.CommitBlock()

				// Check balance after conversion
				grpcAllBalancesAfterConversion, err := bankQueryClient.AllBalances(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryAllBalancesRequest{
					Address: sender,
				})
				s.Require().NoError(err)
				s.Require().Equal(2, len(grpcAllBalancesAfterConversion.Balances))
				found, uosmoAfter := grpcAllBalancesAfterConversion.Balances.Find(teststypes.UosmoIbcdenom)
				s.Require().True(found)
				s.Require().Equal(uosmoBefore.Amount.Int64(), uosmoAfter.Amount.Int64())
			})
			It("grpc - should behave like internal if ERC20 is disabled", func() {
				grpcAllBalancesBeforeConversion, err := bankQueryClient.AllBalances(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryAllBalancesRequest{
					Address: sender,
				})
				s.Require().NoError(err)
				s.Require().Equal(2, len(grpcAllBalancesBeforeConversion.Balances))
				found, uosmoBefore := grpcAllBalancesBeforeConversion.Balances.Find(teststypes.UosmoIbcdenom)
				s.Require().True(found)
				s.Require().Equal(amount, uosmoBefore.Amount.Int64())

				// Convert ibc vouchers to erc20 tokens
				msgConvertCoin := types.NewMsgConvertCoin(
					sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
					common.BytesToAddress(senderAcc.Bytes()),
					senderAcc,
				)

				err = msgConvertCoin.ValidateBasic()
				s.Require().NoError(err)

				_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.HaqqChain.GetContext()), msgConvertCoin)
				s.Require().NoError(err)

				s.HaqqChain.Coordinator.CommitBlock()

				erc20params := types.DefaultParams()
				erc20params.EnableErc20 = false
				err = s.app.Erc20Keeper.SetParams(s.HaqqChain.GetContext(), erc20params)
				s.Require().NoError(err)

				// Check balance after conversion
				grpcAllBalancesAfterConversion, err := bankQueryClient.AllBalances(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryAllBalancesRequest{
					Address: sender,
				})
				s.Require().NoError(err)
				s.Require().Equal(1, len(grpcAllBalancesAfterConversion.Balances))
				found, _ = grpcAllBalancesAfterConversion.Balances.Find(teststypes.UosmoIbcdenom)
				s.Require().False(found)
			})
		})

		Describe("Get Spendable Balances", func() {
			It("internal - should change after conversion", func() {
				internalSpendableCoinsBeforeConversion := s.app.BankKeeper.SpendableCoins(s.HaqqChain.GetContext(), senderAcc)
				// Should contain 2 denoms
				s.Require().Equal(2, len(internalSpendableCoinsBeforeConversion))
				found, uosmoBefore := internalSpendableCoinsBeforeConversion.Find(teststypes.UosmoIbcdenom)
				s.Require().True(found)
				s.Require().Equal(amount, uosmoBefore.Amount.Int64())

				// Convert ibc vouchers to erc20 tokens
				msgConvertCoin := types.NewMsgConvertCoin(
					sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
					common.BytesToAddress(senderAcc.Bytes()),
					senderAcc,
				)

				err := msgConvertCoin.ValidateBasic()
				s.Require().NoError(err)

				_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.HaqqChain.GetContext()), msgConvertCoin)
				s.Require().NoError(err)

				s.HaqqChain.Coordinator.CommitBlock()

				// Check balance after conversion
				internalSpendableCoinsAfterConversion := s.app.BankKeeper.SpendableCoins(s.HaqqChain.GetContext(), senderAcc)
				// Should contain 1 denom
				s.Require().Equal(1, len(internalSpendableCoinsAfterConversion))
				found, _ = internalSpendableCoinsAfterConversion.Find(teststypes.UosmoIbcdenom)
				s.Require().False(found)
			})
			It("grpc - should be the same after conversion", func() {
				grpcSpendableBalancesBeforeConversion, err := bankQueryClient.SpendableBalances(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QuerySpendableBalancesRequest{
					Address: sender,
				})
				s.Require().NoError(err)
				s.Require().Equal(2, len(grpcSpendableBalancesBeforeConversion.Balances))
				found, uosmoBefore := grpcSpendableBalancesBeforeConversion.Balances.Find(teststypes.UosmoIbcdenom)
				s.Require().True(found)
				s.Require().Equal(amount, uosmoBefore.Amount.Int64())

				// Convert ibc vouchers to erc20 tokens
				msgConvertCoin := types.NewMsgConvertCoin(
					sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
					common.BytesToAddress(senderAcc.Bytes()),
					senderAcc,
				)

				err = msgConvertCoin.ValidateBasic()
				s.Require().NoError(err)

				_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.HaqqChain.GetContext()), msgConvertCoin)
				s.Require().NoError(err)

				s.HaqqChain.Coordinator.CommitBlock()

				// Check balance after conversion
				grpcSpendableBalancesAfterConversion, err := bankQueryClient.SpendableBalances(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QuerySpendableBalancesRequest{
					Address: sender,
				})
				s.Require().NoError(err)
				s.Require().Equal(2, len(grpcSpendableBalancesAfterConversion.Balances))
				found, uosmoAfter := grpcSpendableBalancesAfterConversion.Balances.Find(teststypes.UosmoIbcdenom)
				s.Require().True(found)
				s.Require().Equal(uosmoBefore.Amount.Int64(), uosmoAfter.Amount.Int64())
			})
			It("grpc - should behave like internal if ERC20 is disabled", func() {
				grpcSpendableBalancesBeforeConversion, err := bankQueryClient.SpendableBalances(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QuerySpendableBalancesRequest{
					Address: sender,
				})
				s.Require().NoError(err)
				s.Require().Equal(2, len(grpcSpendableBalancesBeforeConversion.Balances))
				found, uosmoBefore := grpcSpendableBalancesBeforeConversion.Balances.Find(teststypes.UosmoIbcdenom)
				s.Require().True(found)
				s.Require().Equal(amount, uosmoBefore.Amount.Int64())

				// Convert ibc vouchers to erc20 tokens
				msgConvertCoin := types.NewMsgConvertCoin(
					sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
					common.BytesToAddress(senderAcc.Bytes()),
					senderAcc,
				)

				err = msgConvertCoin.ValidateBasic()
				s.Require().NoError(err)

				_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.HaqqChain.GetContext()), msgConvertCoin)
				s.Require().NoError(err)

				s.HaqqChain.Coordinator.CommitBlock()

				erc20params := types.DefaultParams()
				erc20params.EnableErc20 = false
				err = s.app.Erc20Keeper.SetParams(s.HaqqChain.GetContext(), erc20params)
				s.Require().NoError(err)

				// Check balance after conversion
				grpcSpendableBalancesAfterConversion, err := bankQueryClient.SpendableBalances(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QuerySpendableBalancesRequest{
					Address: sender,
				})
				s.Require().NoError(err)
				s.Require().Equal(1, len(grpcSpendableBalancesAfterConversion.Balances))
				found, _ = grpcSpendableBalancesAfterConversion.Balances.Find(teststypes.UosmoIbcdenom)
				s.Require().False(found)
			})
		})

		Describe("Transfer coins - ERC20 is Disabled", func() {
			BeforeEach(func() {
				erc20params := types.DefaultParams()
				erc20params.EnableErc20 = false
				err := s.app.Erc20Keeper.SetParams(s.HaqqChain.GetContext(), erc20params)
				s.Require().NoError(err)
			})

			It("should send unconverted coins on native layer", func() {
				uosmoSenderBalanceBefore := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
				s.Require().Equal(amount, uosmoSenderBalanceBefore.Amount.Int64())

				aislmSenderBalanceBefore := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
				s.Require().False(aislmSenderBalanceBefore.IsZero())

				uosmoReceiverBalanceBefore := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
				s.Require().True(uosmoReceiverBalanceBefore.IsZero())

				aislmReceiverBalanceBefore := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
				s.Require().True(aislmReceiverBalanceBefore.IsZero())

				// Transfer Coins via DeliverTx
				fiveUosmo := sdk.NewCoin(teststypes.UosmoIbcdenom, sdk.NewIntFromUint64(5))
				fiveIslm, err := sdk.ParseCoinNormalized("5000000000000000000aISLM")
				s.Require().NoError(err)

				transferCoins := sdk.NewCoins(fiveUosmo, fiveIslm)
				bankTransferMsg := banktypes.NewMsgSend(senderAcc, receiverAcc, transferCoins)
				_, err = ibctesting.SendMsgs(s.HaqqChain, ibctesting.DefaultFeeAmt, bankTransferMsg)
				s.Require().NoError(err) // message committed

				uosmoSenderBalanceAfter := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
				s.Require().Equal(amount-fiveUosmo.Amount.Int64(), uosmoSenderBalanceAfter.Amount.Int64())

				aislmSenderBalanceAfter := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
				s.Require().True(aislmSenderBalanceBefore.Sub(fiveIslm).IsGTE(aislmSenderBalanceAfter))

				uosmoReceiverBalanceAfter := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
				s.Require().Equal(fiveUosmo.Amount.Int64(), uosmoReceiverBalanceAfter.Amount.Int64())

				aislmReceiverBalanceAfter := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
				s.Require().True(aislmReceiverBalanceBefore.Add(fiveIslm).IsGTE(aislmReceiverBalanceAfter))
				s.Require().True(aislmReceiverBalanceAfter.IsGTE(aislmReceiverBalanceBefore.Add(fiveIslm)))
			})
			It("should fail on sending of unconverted coins on native layer", func() {
				uosmoSenderBalanceBefore := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
				s.Require().Equal(amount, uosmoSenderBalanceBefore.Amount.Int64())

				aislmSenderBalanceBefore := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
				s.Require().False(aislmSenderBalanceBefore.IsZero())

				uosmoReceiverBalanceBefore := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
				s.Require().True(uosmoReceiverBalanceBefore.IsZero())

				aislmReceiverBalanceBefore := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
				s.Require().True(aislmReceiverBalanceBefore.IsZero())

				// Transfer Coins via DeliverTx
				fiveUosmo := sdk.NewCoin(teststypes.UosmoIbcdenom, sdk.NewIntFromUint64(115))
				fiveIslm, err := sdk.ParseCoinNormalized("5000000000000000000aISLM")
				s.Require().NoError(err)

				transferCoins := sdk.NewCoins(fiveUosmo, fiveIslm)
				bankTransferMsg := banktypes.NewMsgSend(senderAcc, receiverAcc, transferCoins)

				s.HaqqChain.Coordinator.UpdateTimeForChain(s.HaqqChain)
				fee := sdk.Coins{sdk.NewInt64Coin(haqqDenom, ibctesting.DefaultFeeAmt)}

				_, _, err = ibctesting.SignAndDeliver(
					s.HaqqChain.T,
					s.HaqqChain.TxConfig,
					s.HaqqChain.App.GetBaseApp(),
					[]sdk.Msg{bankTransferMsg},
					fee,
					s.HaqqChain.ChainID,
					[]uint64{s.HaqqChain.SenderAccount.GetAccountNumber()},
					[]uint64{s.HaqqChain.SenderAccount.GetSequence()},
					false, s.HaqqChain.SenderPrivKey,
				)
				s.Require().Error(err)
				// NextBlock calls app.Commit()
				s.HaqqChain.NextBlock()

				uosmoSenderBalanceAfter := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
				s.Require().Equal(amount, uosmoSenderBalanceAfter.Amount.Int64())

				aislmSenderBalanceAfter := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
				s.Require().True(aislmSenderBalanceBefore.Sub(fiveIslm).IsLTE(aislmSenderBalanceAfter))

				uosmoReceiverBalanceAfter := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
				s.Require().True(uosmoReceiverBalanceAfter.IsZero())

				aislmReceiverBalanceAfter := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
				s.Require().True(aislmReceiverBalanceBefore.IsGTE(aislmReceiverBalanceAfter))
				s.Require().True(aislmReceiverBalanceAfter.IsGTE(aislmReceiverBalanceBefore))
			})
		})

		Describe("Transfer coins - ERC20 is Enabled", func() {
			BeforeEach(func() {
				erc20params := types.DefaultParams()
				erc20params.EnableErc20 = true
				err := s.app.Erc20Keeper.SetParams(s.HaqqChain.GetContext(), erc20params)
				s.Require().NoError(err)
			})

			Describe("TokenPair is Enabled", func() {
				var err error
				var (
					uosmoSenderBalanceBefore, aislmSenderBalanceBefore             sdk.Coin
					uosmoSenderGrpcBalanceBefore, aislmSenderGrpcBalanceBefore     *banktypes.QueryBalanceResponse
					uosmoReceiverBalanceBefore, aislmReceiverBalanceBefore         sdk.Coin
					uosmoReceiverGrpcBalanceBefore, aislmReceiverGrpcBalanceBefore *banktypes.QueryBalanceResponse
				)

				var (
					uosmoSenderBalanceAfter, aislmSenderBalanceAfter             sdk.Coin
					uosmoSenderGrpcBalanceAfter, aislmSenderGrpcBalanceAfter     *banktypes.QueryBalanceResponse
					uosmoReceiverBalanceAfter, aislmReceiverBalanceAfter         sdk.Coin
					uosmoReceiverGrpcBalanceAfter, aislmReceiverGrpcBalanceAfter *banktypes.QueryBalanceResponse
				)

				// prepare coins for transfers
				fourUosmo := sdk.NewCoin(teststypes.UosmoIbcdenom, sdk.NewInt(4))
				sixUosmo := sdk.NewCoin(teststypes.UosmoIbcdenom, sdk.NewInt(6))
				fiveIslm, err := sdk.ParseCoinNormalized("5000000000000000000aISLM")
				s.Require().NoError(err)

				Describe("All coins are on native layer", func() {
					BeforeEach(func() {
						uosmoSenderBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
						s.Require().Equal(amount, uosmoSenderBalanceBefore.Amount.Int64())
						aislmSenderBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
						s.Require().False(aislmSenderBalanceBefore.IsZero())

						uosmoSenderGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().Equal(amount, uosmoSenderGrpcBalanceBefore.Balance.Amount.Int64())
						aislmSenderGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().False(aislmSenderGrpcBalanceBefore.Balance.IsZero())

						uosmoReceiverBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoReceiverBalanceBefore.IsZero())
						aislmReceiverBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
						s.Require().True(aislmReceiverBalanceBefore.IsZero())

						uosmoReceiverGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().True(uosmoReceiverGrpcBalanceBefore.Balance.IsZero())
						aislmReceiverGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmReceiverGrpcBalanceBefore.Balance.IsZero())
					})

					It("should convert all coins and transfer on EVM layer", func() {
						// At this point
						// sender has 10 uosmo (all on SDK layer) and some aislm tokens
						// receiver has zero balance for both uosmo and aislm

						// Transfer Coins
						fiveUosmo := sdk.NewCoin(teststypes.UosmoIbcdenom, sdk.NewInt(5))
						transferCoins := sdk.NewCoins(fiveUosmo, fiveIslm)
						bankTransferMsg := banktypes.NewMsgSend(senderAcc, receiverAcc, transferCoins)
						_, err = ibctesting.SendMsgs(s.HaqqChain, ibctesting.DefaultFeeAmt, bankTransferMsg)
						s.Require().NoError(err) // message committed

						// Check the results

						// Sender must have zero uosmo on SDK layer and 5 uosmo on EVM
						uosmoSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoSenderBalanceAfter.IsZero())
						uosmoSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().Equal(uosmoSenderGrpcBalanceBefore.Balance.Sub(fiveUosmo).Amount.Int64(), uosmoSenderGrpcBalanceAfter.Balance.Amount.Int64())
						// Sender must have less aislm tokens than before on transfer amount and fee
						aislmSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
						s.Require().True(aislmSenderBalanceBefore.Sub(fiveIslm).IsGTE(aislmSenderBalanceAfter))
						aislmSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmSenderGrpcBalanceBefore.Balance.Sub(fiveIslm).IsGTE(*aislmSenderGrpcBalanceAfter.Balance))

						// Receiver must have zero uosmo on SDK layer and 5 uosmo on EVM
						uosmoReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoReceiverBalanceAfter.IsZero())
						uosmoReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().Equal(uosmoReceiverGrpcBalanceBefore.Balance.Add(fiveUosmo).Amount.Int64(), uosmoReceiverGrpcBalanceAfter.Balance.Amount.Int64())
						// Receiver must have 5 ISLM
						aislmReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
						s.Require().True(aislmReceiverBalanceBefore.Add(fiveIslm).IsGTE(aislmReceiverBalanceAfter))
						s.Require().True(aislmReceiverBalanceAfter.IsGTE(aislmReceiverBalanceBefore.Add(fiveIslm)))
						aislmReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmReceiverGrpcBalanceBefore.Balance.Add(fiveIslm).IsGTE(*aislmReceiverGrpcBalanceAfter.Balance))
						s.Require().True(aislmReceiverGrpcBalanceAfter.Balance.IsGTE(aislmReceiverGrpcBalanceBefore.Balance.Add(fiveIslm)))
					})

					It("insufficient funds - should skip conversion and fail", func() {
						// At this point
						// sender has 10 uosmo (all on SDK layer) and some aislm tokens
						// receiver has zero balance for both uosmo and aislm

						// Transfer Coins
						fiveHundredUosmo := sdk.NewCoin(teststypes.UosmoIbcdenom, sdk.NewInt(500))
						transferCoins := sdk.NewCoins(fiveHundredUosmo, fiveIslm)
						bankTransferMsg := banktypes.NewMsgSend(senderAcc, receiverAcc, transferCoins)

						s.HaqqChain.Coordinator.UpdateTimeForChain(s.HaqqChain)
						fee := sdk.NewInt64Coin(haqqDenom, ibctesting.DefaultFeeAmt)
						_, _, err = ibctesting.SignAndDeliver(
							s.HaqqChain.T,
							s.HaqqChain.TxConfig,
							s.HaqqChain.App.GetBaseApp(),
							[]sdk.Msg{bankTransferMsg},
							sdk.Coins{fee},
							s.HaqqChain.ChainID,
							[]uint64{s.HaqqChain.SenderAccount.GetAccountNumber()},
							[]uint64{s.HaqqChain.SenderAccount.GetSequence()},
							false, s.HaqqChain.SenderPrivKey,
						)
						s.Require().Error(err)
						// NextBlock calls app.Commit()
						s.HaqqChain.NextBlock()

						// Check the results

						// Sender must have the same uosmo balance on both SDK and EVM layers as before
						uosmoSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
						s.Require().Equal(amount, uosmoSenderBalanceAfter.Amount.Int64())
						uosmoSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().True(uosmoSenderGrpcBalanceBefore.Balance.IsGTE(*uosmoSenderGrpcBalanceAfter.Balance))
						s.Require().True(uosmoSenderGrpcBalanceAfter.Balance.IsGTE(*uosmoSenderGrpcBalanceBefore.Balance))
						// Sender must have the same aislm balance (minus fee) on both SDK and EVM layers as before
						aislmSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
						s.Require().True(aislmSenderBalanceBefore.Sub(fiveIslm).IsLT(aislmSenderBalanceAfter))
						aislmSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmSenderGrpcBalanceBefore.Balance.Sub(fee).IsGTE(*aislmSenderGrpcBalanceAfter.Balance))
						s.Require().True(aislmSenderGrpcBalanceAfter.Balance.IsGTE((*aislmSenderGrpcBalanceBefore.Balance).Sub(fee)))

						// Receiver must have the same uosmo balance on both SDK and EVM layers as before
						uosmoReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoReceiverBalanceAfter.IsZero())
						uosmoReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().True(uosmoReceiverGrpcBalanceAfter.Balance.IsZero())
						// Receiver must have the same aislm balance on both SDK and EVM layers as before
						aislmReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
						s.Require().True(aislmReceiverBalanceAfter.IsZero())
						aislmReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmReceiverGrpcBalanceAfter.Balance.IsZero())
					})
				})

				Describe("part of coins is on native layer", func() {
					BeforeEach(func() {
						// Convert ibc vouchers to erc20 tokens
						msgConvertCoin := types.NewMsgConvertCoin(fourUosmo, common.BytesToAddress(senderAcc.Bytes()), senderAcc)

						err := msgConvertCoin.ValidateBasic()
						s.Require().NoError(err)

						_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.HaqqChain.GetContext()), msgConvertCoin)
						s.Require().NoError(err)

						s.HaqqChain.Coordinator.CommitBlock()

						// Get the initial balances after convertion
						uosmoSenderBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
						s.Require().Equal(sixUosmo.Amount.Int64(), uosmoSenderBalanceBefore.Amount.Int64())
						aislmSenderBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
						s.Require().False(aislmSenderBalanceBefore.IsZero())

						uosmoSenderGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().Equal(amount, uosmoSenderGrpcBalanceBefore.Balance.Amount.Int64())
						aislmSenderGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().False(aislmSenderGrpcBalanceBefore.Balance.IsZero())

						uosmoReceiverBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoReceiverBalanceBefore.IsZero())
						aislmReceiverBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
						s.Require().True(aislmReceiverBalanceBefore.IsZero())

						uosmoReceiverGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().True(uosmoReceiverGrpcBalanceBefore.Balance.IsZero())
						aislmReceiverGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmReceiverGrpcBalanceBefore.Balance.IsZero())
					})

					It("should convert unconverted coins and transfer on EVM layer", func() {
						// At this point
						// sender has 10 uosmo: 6 on SDK layer, 4 on EVM layer
						// receiver has zero balance for both uosmo and aislm

						// Now we ganna transfer 4 uosmo to another account

						// Transfer Coins via DeliverTx
						transferCoins := sdk.NewCoins(fourUosmo, fiveIslm)
						bankTransferMsg := banktypes.NewMsgSend(senderAcc, receiverAcc, transferCoins)
						_, err = ibctesting.SendMsgs(s.HaqqChain, ibctesting.DefaultFeeAmt, bankTransferMsg)
						s.Require().NoError(err) // message committed

						// Check the results

						// Sender must have zero uosmo on SDK layer and 6 uosmo on EVM
						uosmoSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoSenderBalanceAfter.IsZero())
						uosmoSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().Equal(uosmoSenderGrpcBalanceBefore.Balance.Sub(fourUosmo).Amount.Int64(), uosmoSenderGrpcBalanceAfter.Balance.Amount.Int64())
						// Sender must have less aislm tokens than before on transfer amount and fee
						aislmSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
						s.Require().True(aislmSenderBalanceBefore.Sub(fiveIslm).IsGTE(aislmSenderBalanceAfter))
						aislmSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmSenderGrpcBalanceBefore.Balance.Sub(fiveIslm).IsGTE(*aislmSenderGrpcBalanceAfter.Balance))

						// Receiver must have zero uosmo on SDK layer and 4 uosmo on EVM
						uosmoReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoReceiverBalanceAfter.IsZero())
						uosmoReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().Equal(uosmoReceiverGrpcBalanceBefore.Balance.Add(fourUosmo).Amount.Int64(), uosmoReceiverGrpcBalanceAfter.Balance.Amount.Int64())
						// Receiver must have 5 ISLM
						aislmReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
						s.Require().True(aislmReceiverBalanceBefore.Add(fiveIslm).IsGTE(aislmReceiverBalanceAfter))
						s.Require().True(aislmReceiverBalanceAfter.IsGTE(aislmReceiverBalanceBefore.Add(fiveIslm)))
						aislmReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmReceiverGrpcBalanceBefore.Balance.Add(fiveIslm).IsGTE(*aislmReceiverGrpcBalanceAfter.Balance))
						s.Require().True(aislmReceiverGrpcBalanceAfter.Balance.IsGTE(aislmReceiverGrpcBalanceBefore.Balance.Add(fiveIslm)))
					})
					It("insufficient funds - should skip conversion and fail", func() {
						// At this point
						// sender has 10 uosmo: 6 on SDK layer, 4 on EVM layer
						// receiver has zero balance for both uosmo and aislm

						// Now we ganna transfer 500 uosmo to another account and expect fail

						// Transfer Coins
						fiveHundredUosmo := sdk.NewCoin(teststypes.UosmoIbcdenom, sdk.NewInt(500))
						transferCoins := sdk.NewCoins(fiveHundredUosmo, fiveIslm)
						bankTransferMsg := banktypes.NewMsgSend(senderAcc, receiverAcc, transferCoins)

						s.HaqqChain.Coordinator.UpdateTimeForChain(s.HaqqChain)
						fee := sdk.NewInt64Coin(haqqDenom, ibctesting.DefaultFeeAmt)
						_, _, err = ibctesting.SignAndDeliver(
							s.HaqqChain.T,
							s.HaqqChain.TxConfig,
							s.HaqqChain.App.GetBaseApp(),
							[]sdk.Msg{bankTransferMsg},
							sdk.Coins{fee},
							s.HaqqChain.ChainID,
							[]uint64{s.HaqqChain.SenderAccount.GetAccountNumber()},
							[]uint64{s.HaqqChain.SenderAccount.GetSequence()},
							false, s.HaqqChain.SenderPrivKey,
						)
						s.Require().Error(err)
						// NextBlock calls app.Commit()
						s.HaqqChain.NextBlock()

						// Check the results

						// Sender must have the same uosmo balance on both SDK and EVM layers as before
						uosmoSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
						s.Require().Equal(sixUosmo.Amount.Int64(), uosmoSenderBalanceAfter.Amount.Int64())
						uosmoSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().True(uosmoSenderGrpcBalanceBefore.Balance.IsGTE(*uosmoSenderGrpcBalanceAfter.Balance))
						s.Require().True(uosmoSenderGrpcBalanceAfter.Balance.IsGTE(*uosmoSenderGrpcBalanceBefore.Balance))
						// Sender must have the same aislm balance (minus fee) on both SDK and EVM layers as before
						aislmSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
						s.Require().True(aislmSenderBalanceBefore.Sub(fiveIslm).IsLT(aislmSenderBalanceAfter))
						aislmSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmSenderGrpcBalanceBefore.Balance.Sub(fee).IsGTE(*aislmSenderGrpcBalanceAfter.Balance))
						s.Require().True(aislmSenderGrpcBalanceAfter.Balance.IsGTE((*aislmSenderGrpcBalanceBefore.Balance).Sub(fee)))

						// Receiver must have the same uosmo balance on both SDK and EVM layers as before
						uosmoReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoReceiverBalanceAfter.IsZero())
						uosmoReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().True(uosmoReceiverGrpcBalanceAfter.Balance.IsZero())
						// Receiver must have the same aislm balance on both SDK and EVM layers as before
						aislmReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
						s.Require().True(aislmReceiverBalanceAfter.IsZero())
						aislmReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmReceiverGrpcBalanceAfter.Balance.IsZero())
					})
				})

				Describe("all coins are on EVM layer", func() {
					BeforeEach(func() {
						// Convert ibc vouchers to erc20 tokens
						msgConvertCoin := types.NewMsgConvertCoin(fourUosmo.Add(sixUosmo), common.BytesToAddress(senderAcc.Bytes()), senderAcc)

						err := msgConvertCoin.ValidateBasic()
						s.Require().NoError(err)

						_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.HaqqChain.GetContext()), msgConvertCoin)
						s.Require().NoError(err)

						s.HaqqChain.Coordinator.CommitBlock()

						// Get the initial balances after convertion
						uosmoSenderBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoSenderBalanceBefore.IsZero())
						aislmSenderBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
						s.Require().False(aislmSenderBalanceBefore.IsZero())

						uosmoSenderGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().Equal(amount, uosmoSenderGrpcBalanceBefore.Balance.Amount.Int64())
						aislmSenderGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().False(aislmSenderGrpcBalanceBefore.Balance.IsZero())

						uosmoReceiverBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoReceiverBalanceBefore.IsZero())
						aislmReceiverBalanceBefore = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
						s.Require().True(aislmReceiverBalanceBefore.IsZero())

						uosmoReceiverGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().True(uosmoReceiverGrpcBalanceBefore.Balance.IsZero())
						aislmReceiverGrpcBalanceBefore, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmReceiverGrpcBalanceBefore.Balance.IsZero())
					})

					It("should skip conversion and transfer on EVM layer", func() {
						// At this point
						// sender has 10 uosmo: 6 on SDK layer, 4 on EVM layer
						// receiver has zero balance for both uosmo and aislm

						// Now we ganna transfer 4 uosmo to another account

						// Transfer Coins via DeliverTx
						transferCoins := sdk.NewCoins(fourUosmo, fiveIslm)
						bankTransferMsg := banktypes.NewMsgSend(senderAcc, receiverAcc, transferCoins)
						_, err = ibctesting.SendMsgs(s.HaqqChain, ibctesting.DefaultFeeAmt, bankTransferMsg)
						s.Require().NoError(err) // message committed

						// Check the results

						// Sender must have zero uosmo on SDK layer and 6 uosmo on EVM
						uosmoSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoSenderBalanceAfter.IsZero())
						uosmoSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().Equal(uosmoSenderGrpcBalanceBefore.Balance.Sub(fourUosmo).Amount.Int64(), uosmoSenderGrpcBalanceAfter.Balance.Amount.Int64())
						// Sender must have less aislm tokens than before on transfer amount and fee
						aislmSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
						s.Require().True(aislmSenderBalanceBefore.Sub(fiveIslm).IsGTE(aislmSenderBalanceAfter))
						aislmSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmSenderGrpcBalanceBefore.Balance.Sub(fiveIslm).IsGTE(*aislmSenderGrpcBalanceAfter.Balance))

						// Receiver must have zero uosmo on SDK layer and 4 uosmo on EVM
						uosmoReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoReceiverBalanceAfter.IsZero())
						uosmoReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().Equal(uosmoReceiverGrpcBalanceBefore.Balance.Add(fourUosmo).Amount.Int64(), uosmoReceiverGrpcBalanceAfter.Balance.Amount.Int64())
						// Receiver must have 5 ISLM
						aislmReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
						s.Require().True(aislmReceiverBalanceBefore.Add(fiveIslm).IsGTE(aislmReceiverBalanceAfter))
						s.Require().True(aislmReceiverBalanceAfter.IsGTE(aislmReceiverBalanceBefore.Add(fiveIslm)))
						aislmReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmReceiverGrpcBalanceBefore.Balance.Add(fiveIslm).IsGTE(*aislmReceiverGrpcBalanceAfter.Balance))
						s.Require().True(aislmReceiverGrpcBalanceAfter.Balance.IsGTE(aislmReceiverGrpcBalanceBefore.Balance.Add(fiveIslm)))
					})
					It("insufficient funds - should skip conversion and fail", func() {
						// At this point
						// sender has 10 uosmo: 6 on SDK layer, 4 on EVM layer
						// receiver has zero balance for both uosmo and aislm

						// Now we ganna transfer 500 uosmo to another account and expect fail

						// Transfer Coins
						fiveHundredUosmo := sdk.NewCoin(teststypes.UosmoIbcdenom, sdk.NewInt(500))
						transferCoins := sdk.NewCoins(fiveHundredUosmo, fiveIslm)
						bankTransferMsg := banktypes.NewMsgSend(senderAcc, receiverAcc, transferCoins)

						s.HaqqChain.Coordinator.UpdateTimeForChain(s.HaqqChain)
						fee := sdk.NewInt64Coin(haqqDenom, ibctesting.DefaultFeeAmt)
						_, _, err = ibctesting.SignAndDeliver(
							s.HaqqChain.T,
							s.HaqqChain.TxConfig,
							s.HaqqChain.App.GetBaseApp(),
							[]sdk.Msg{bankTransferMsg},
							sdk.Coins{fee},
							s.HaqqChain.ChainID,
							[]uint64{s.HaqqChain.SenderAccount.GetAccountNumber()},
							[]uint64{s.HaqqChain.SenderAccount.GetSequence()},
							false, s.HaqqChain.SenderPrivKey,
						)
						s.Require().Error(err)
						// NextBlock calls app.Commit()
						s.HaqqChain.NextBlock()

						// Check the results

						// Sender must have the same uosmo balance on both SDK and EVM layers as before
						uosmoSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
						s.Require().Equal(uosmoSenderBalanceBefore.Amount.Int64(), uosmoSenderBalanceAfter.Amount.Int64())
						uosmoSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().True(uosmoSenderGrpcBalanceBefore.Balance.IsGTE(*uosmoSenderGrpcBalanceAfter.Balance))
						s.Require().True(uosmoSenderGrpcBalanceAfter.Balance.IsGTE(*uosmoSenderGrpcBalanceBefore.Balance))
						// Sender must have the same aislm balance (minus fee) on both SDK and EVM layers as before
						aislmSenderBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), senderAcc, haqqDenom)
						s.Require().True(aislmSenderBalanceBefore.Sub(fiveIslm).IsLT(aislmSenderBalanceAfter))
						aislmSenderGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: senderAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmSenderGrpcBalanceBefore.Balance.Sub(fee).IsGTE(*aislmSenderGrpcBalanceAfter.Balance))
						s.Require().True(aislmSenderGrpcBalanceAfter.Balance.IsGTE((*aislmSenderGrpcBalanceBefore.Balance).Sub(fee)))

						// Receiver must have the same uosmo balance on both SDK and EVM layers as before
						uosmoReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
						s.Require().True(uosmoReceiverBalanceAfter.IsZero())
						uosmoReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   teststypes.UosmoIbcdenom,
						})
						s.Require().NoError(err)
						s.Require().True(uosmoReceiverGrpcBalanceAfter.Balance.IsZero())
						// Receiver must have the same aislm balance on both SDK and EVM layers as before
						aislmReceiverBalanceAfter = s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), receiverAcc, haqqDenom)
						s.Require().True(aislmReceiverBalanceAfter.IsZero())
						aislmReceiverGrpcBalanceAfter, err = bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
							Address: receiverAcc.String(),
							Denom:   haqqDenom,
						})
						s.Require().NoError(err)
						s.Require().True(aislmReceiverGrpcBalanceAfter.Balance.IsZero())
					})
				})
			})

			Describe("TokenPair is Disabled", func() {
				// TODO cover this case
			})
		})
	})
})
