package keeper_test

import (
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/contracts"
	teststypes "github.com/haqq-network/haqq/types/tests"
	"github.com/haqq-network/haqq/utils"
	haqqbankkeeper "github.com/haqq-network/haqq/x/bank/keeper"
)

var _ = Describe("Native coins from IBC", Ordered, func() {
	var (
		amount          int64 = 10
		bankQueryClient banktypes.QueryClient
	)

	BeforeEach(func() {
		s.suiteIBCTesting = true
		s.SetupTest()
		s.suiteIBCTesting = false

		bankQueryHelper := baseapp.NewQueryServerTestHelper(s.HaqqChain.GetContext(), s.app.InterfaceRegistry())
		wrappedBankKeeper := haqqbankkeeper.NewWrappedBaseKeeper(s.app.BankKeeper, s.app.EvmKeeper, s.app.Erc20Keeper, s.app.AccountKeeper)
		banktypes.RegisterQueryServer(bankQueryHelper, wrappedBankKeeper)
		bankQueryClient = banktypes.NewQueryClient(bankQueryHelper)
	})

	It("Is native from source chain - should transfer and register pair and deploy a precompile", func() {
		osmosisAddress := s.IBCOsmosisChain.SenderAccount.GetAddress().String()
		haqqAddress := s.HaqqChain.SenderAccount.GetAddress().String()
		haqqAccount := sdk.MustAccAddressFromBech32(haqqAddress)

		// Precompile should not be available before IBC
		uosmoContractAddr, err := utils.GetIBCDenomAddress(teststypes.UosmoIbcdenom)
		s.Require().NoError(err)

		params := s.app.EvmKeeper.GetParams(s.HaqqChain.GetContext())
		s.Require().False(s.app.EvmKeeper.IsAvailableStaticPrecompile(&params, uosmoContractAddr))

		// Check receiver's balance for IBC before transfer. Should be zero
		ibcOsmoBalanceBefore := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), haqqAccount, teststypes.UosmoIbcdenom)
		s.Require().Equal(int64(0), ibcOsmoBalanceBefore.Amount.Int64())
		s.HaqqChain.Coordinator.CommitBlock()

		// Send uosmo from osmosis to haqq
		s.SendAndReceiveMessage(s.pathOsmosisHaqq, s.IBCOsmosisChain, "uosmo", amount, osmosisAddress, haqqAddress, 1, "")
		s.HaqqChain.Coordinator.CommitBlock()

		// Check IBC uosmo coin balance - should be equals to amount sent
		ibcOsmoBalanceAfter := s.app.BankKeeper.GetBalance(s.HaqqChain.GetContext(), haqqAccount, teststypes.UosmoIbcdenom)
		s.Require().Equal(amount, ibcOsmoBalanceAfter.Amount.Int64())

		// Pair should be registered now and precompile available
		pairID := s.app.Erc20Keeper.GetTokenPairID(s.HaqqChain.GetContext(), teststypes.UosmoIbcdenom)
		_, found := s.app.Erc20Keeper.GetTokenPair(s.HaqqChain.GetContext(), pairID)
		s.Require().True(found)
		activeDynamicPrecompiles := s.app.Erc20Keeper.GetParams(s.HaqqChain.GetContext()).DynamicPrecompiles
		s.Require().Contains(activeDynamicPrecompiles, uosmoContractAddr.String())

		// Check GRPC Balance request
		uosmoGrpcBalanceAfter, err := bankQueryClient.Balance(sdk.WrapSDKContext(s.HaqqChain.GetContext()), &banktypes.QueryBalanceRequest{
			Address: haqqAddress,
			Denom:   teststypes.UosmoIbcdenom,
		})
		s.Require().NoError(err)
		s.Require().Equal(amount, uosmoGrpcBalanceAfter.Balance.Amount.Int64())

		// Check EVM balance request
		erc20contractAbi := contracts.ERC20MinterBurnerDecimalsContract.ABI
		uosmoErc20BalanceAfter := s.app.Erc20Keeper.BalanceOf(s.HaqqChain.GetContext(), erc20contractAbi, uosmoContractAddr, common.BytesToAddress(haqqAccount.Bytes()))
		s.Require().Equal(amount, uosmoErc20BalanceAfter.Int64())
	})
})
