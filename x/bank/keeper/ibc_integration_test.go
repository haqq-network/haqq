package keeper_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	teststypes "github.com/haqq-network/haqq/types/tests"
	haqqbankkeeper "github.com/haqq-network/haqq/x/bank/keeper"
	"github.com/haqq-network/haqq/x/erc20/types"
)

var _ = Describe("Check balance of IBC tokens registered as ERC20", Ordered, func() {
	var (
		erc20Symbol      = "CTKN"
		sender, receiver string
		// receiverAcc      sdk.AccAddress
		senderAcc       sdk.AccAddress
		amount          int64 = 10
		pair            *types.TokenPair
		bankQueryClient banktypes.QueryClient
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
			// receiverAcc = sdk.MustAccAddressFromBech32(receiver)
			senderAcc = sdk.MustAccAddressFromBech32(sender)

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
			wrappedBankKeeper := haqqbankkeeper.NewWrappedBaseKeeper(s.app.BankKeeper, s.app.Erc20Keeper)
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
	})
})
