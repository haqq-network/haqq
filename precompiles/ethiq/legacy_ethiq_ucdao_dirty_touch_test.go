package ethiq_test

import (
	"math/big"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/ethiq"
	"github.com/haqq-network/haqq/precompiles/testutil"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	commonfactory "github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

// TestEthiqUcdaoApplicationBurnPreservesDirtyEscrow verifies the EVM commit boundary,
// not just the keeper result. The forwarder first sends 1 wei to the UCDAO escrow,
// making it dirty in StateDB, and then calls mintHaqqByApplication in the same EVM
// transaction. Commit must retain both the touch and the application burn.
func (s *PrecompileTestSuite) TestEthiqUcdaoApplicationBurnPreservesDirtyEscrow() {
	owner := s.keyring.GetKey(0)
	ownerEscrow := ucdaotypes.GetEscrowAddress(owner.AccAddr)
	ownerEscrowEVM := common.BytesToAddress(ownerEscrow.Bytes())
	burnAmount := math.NewInt(60).Mul(math.NewInt(1e18))
	fundAmount := math.NewInt(100).Mul(math.NewInt(1e18))

	fundMsg := ucdaotypes.NewMsgFund(
		sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmount)),
		owner.AccAddr,
	)
	fundRes, err := s.factory.CommitCosmosTx(owner.Priv, commonfactory.CosmosTxArgs{
		Msgs: []sdk.Msg{fundMsg},
	})
	s.Require().NoError(err)
	s.Require().True(fundRes.IsOK(), "MsgFund failed: %s", fundRes.Log)
	s.Require().NoError(s.network.NextBlock())

	forwarderContract, err := contracts.LoadUcdaoForwarderContract()
	s.Require().NoError(err)
	forwarderAddr, err := s.factory.DeployContract(
		owner.Priv,
		evmtypes.EvmTxArgs{},
		factory.ContractDeploymentData{Contract: forwarderContract},
	)
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	ownerBech32 := owner.AccAddr.String()
	waitlistItem := ethiqtypes.ApplicationListItem{
		FromAddress:                ownerBech32,
		ToAddress:                  ownerBech32,
		FundSource:                 ethiqtypes.SourceOfFunds_SOURCE_OF_FUNDS_UCDAO,
		IslmAmount:                 burnAmount.String(),
		IslmAccumulatedBurntAmount: "0",
	}
	_, err = waitlistItem.AsBurnApplication()
	s.Require().NoError(err)
	appID, restoreWaitlist := ethiqtypes.PushRegisteredApplicationForIntegrationTest(waitlistItem)
	defer restoreWaitlist()

	expiration := s.network.GetContext().BlockTime().Add(time.Hour)
	s.Require().NoError(s.network.App.AuthzKeeper.SaveGrant(
		s.network.GetContext(),
		forwarderAddr.Bytes(),
		owner.AccAddr,
		&ethiqtypes.MintHaqqByApplicationIDAuthorization{ApplicationsList: []uint64{appID}},
		&expiration,
	))

	beforeEscrow := s.network.App.BankKeeper.GetBalance(
		s.network.GetContext(),
		ownerEscrow,
		utils.BaseDenom,
	).Amount
	s.Require().Equal(fundAmount, beforeEscrow)

	mintCalldata, err := s.precompile.ABI.Pack(
		ethiq.MintHaqqByApplication,
		owner.Addr,
		new(big.Int).SetUint64(appID),
	)
	s.Require().NoError(err)

	res, evmRes, err := s.factory.CallContractAndCheckLogs(
		owner.Priv,
		evmtypes.EvmTxArgs{
			To:       &forwarderAddr,
			GasLimit: 1_500_000,
			Amount:   big.NewInt(1),
		},
		factory.CallArgs{
			ContractABI: forwarderContract.ABI,
			MethodName:  "touchAndForward",
			Args:        []interface{}{ownerEscrowEVM, s.precompile.Address(), mintCalldata},
		},
		testutil.LogCheckArgs{ABIEvents: s.precompile.Events}.
			WithExpEvents(ethiq.EventTypeMintHaqqByApplication).
			WithExpPass(true),
	)
	s.Require().NoError(err)
	s.Require().True(res.IsOK(), "touchAndForward failed: %s", res.Log)
	s.Require().False(evmRes.Failed(), "touchAndForward VM error: %s", evmRes.VmError)
	s.Require().NoError(s.network.NextBlock())

	afterEscrow := s.network.App.BankKeeper.GetBalance(
		s.network.GetContext(),
		ownerEscrow,
		utils.BaseDenom,
	).Amount
	touchDust := math.OneInt()
	s.Require().Equal(
		beforeEscrow.Add(touchDust).Sub(burnAmount),
		afterEscrow,
		"dirty escrow must retain the 1 wei touch and the exact application burn after EVM commit",
	)
	s.Require().True(
		s.network.App.EthiqKeeper.IsApplicationExecuted(s.network.GetContext(), appID),
		"application must be marked executed",
	)

	haqqBalance := s.network.App.BankKeeper.GetBalance(
		s.network.GetContext(),
		owner.AccAddr,
		ethiqtypes.BaseDenom,
	).Amount
	s.Require().True(haqqBalance.IsPositive(), "owner must receive minted aHAQQ")
}
