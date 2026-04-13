package ucdao_test

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/testutil"
	"github.com/haqq-network/haqq/precompiles/ucdao"
	safecontracts "github.com/haqq-network/haqq/precompiles/testutil/contracts/safe"
	commonfactory "github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	testutiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
)

func TestUCDAOSafeIntegrationSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ucDAO Gnosis Safe integration suite")
}

// ucdaoSafeSuite holds shared integration test dependencies (no precompile required).
type ucdaoSafeSuite struct {
	network     *network.UnitTestNetwork
	factory     factory.TxFactory
	grpcHandler grpc.Handler
}

func (s *ucdaoSafeSuite) setupNetwork(kr keyring.Keyring) {
	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(kr.GetAllAccAddrs()...),
	)
	gh := grpc.NewIntegrationHandler(nw)
	s.network = nw
	s.factory = factory.New(nw, gh)
	s.grpcHandler = gh
}

// Phase 1 (native MsgFund + Safe preparation):
// - Two new owner keys, 1500 ISLM each; deploy Gnosis Safe 1-of-2; fund Safe with 1000 ISLM from owners.
// - Third account with 1001 ISLM signs MsgFund: 1000 ISLM into ucDAO for depositor = third account.
//
// Note: MsgFund credits ucDAO to depositor. We Fund liquid from account 3, then
// MsgTransferOwnershipWithAmount(account3→Safe) moves both aISLM and liquid ucDAO balances to Safe.
//
// Phase 2: Safe execTransaction → ucDAO precompile convertToHaqq(500 ISLM), signed by owner1 (threshold 1).
// Expected on success: Safe bank 1000 ISLM; Safe ucDAO 1500 aISLM and no liquid; aHAQQ mint = ethiq curve for 500 ISLM.
// Until precompile allows holder=Safe when tx origin is EOA, exec returns success=false (documents the bug).
var _ = Describe("ucDAO with Gnosis Safe (phase 1)", Ordered, func() {
	var (
		kr               keyring.Keyring
		s                *ucdaoSafeSuite
		deployer         keyring.Key
		safeOwnerOne     keyring.Key
		safeOwnerTwo     keyring.Key
		gnosisSafe       evmtypes.CompiledContract
		gnosisSafeAddr   common.Address
		proxyFactory     evmtypes.CompiledContract
		proxyFactoryAddr common.Address
		funder           keyring.Key
	)

	BeforeAll(func() {
		kr = keyring.New(1)
		s = new(ucdaoSafeSuite)
		s.setupNetwork(kr)
		deployer = kr.GetKey(0)

		oneAddr, onePriv := testutiltx.NewAddrKey()
		safeOwnerOne = keyring.Key{Addr: oneAddr, AccAddr: sdk.AccAddress(oneAddr.Bytes()), Priv: onePriv}
		twoAddr, twoPriv := testutiltx.NewAddrKey()
		safeOwnerTwo = keyring.Key{Addr: twoAddr, AccAddr: sdk.AccAddress(twoAddr.Bytes()), Priv: twoPriv}

		fund1500 := sdkmath.NewInt(1500).MulRaw(1e18)
		Expect(s.network.FundAccountWithBaseDenom(safeOwnerOne.AccAddr, fund1500)).To(Succeed())
		Expect(s.network.FundAccountWithBaseDenom(safeOwnerTwo.AccAddr, fund1500)).To(Succeed())
		Expect(s.network.NextBlock()).To(Succeed())

		var err error
		gnosisSafe, err = safecontracts.LoadGnosisSafeContract()
		Expect(err).NotTo(HaveOccurred())
		proxyFactory, err = safecontracts.LoadGnosisSafeProxyFactoryContract()
		Expect(err).NotTo(HaveOccurred())

		gnosisSafeAddr, err = s.factory.DeployContract(
			deployer.Priv,
			evmtypes.EvmTxArgs{},
			factory.ContractDeploymentData{Contract: gnosisSafe},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(gnosisSafeAddr).NotTo(Equal(common.Address{}))

		proxyFactoryAddr, err = s.factory.DeployContract(
			deployer.Priv,
			evmtypes.EvmTxArgs{},
			factory.ContractDeploymentData{Contract: proxyFactory},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(proxyFactoryAddr).NotTo(Equal(common.Address{}))
		Expect(s.network.NextBlock()).To(Succeed())
	})

	It("prepares Safe, checks bank balances, then MsgFund from third account", func() {
		transferToSafe := sdkmath.NewInt(500).MulRaw(1e18)
		expectedSafeBank := sdkmath.NewInt(1000).MulRaw(1e18)
		expectedOwnerTwoBank := sdkmath.NewInt(1000).MulRaw(1e18)
		oneIslm := sdkmath.NewInt(1).MulRaw(1e18)
		fundAmount := sdkmath.NewInt(1000).MulRaw(1e18)
		gasPrice := sdkmath.NewInt(1_000_000_000)
		gasLimit := uint64(400_000)

		safeSetupData, err := gnosisSafe.ABI.Pack(
			"setup",
			[]common.Address{safeOwnerOne.Addr, safeOwnerTwo.Addr},
			big.NewInt(1),
			common.Address{},
			[]byte{},
			common.Address{},
			common.Address{},
			big.NewInt(0),
			common.Address{},
		)
		Expect(err).NotTo(HaveOccurred())

		createProxyRes, err := s.factory.ExecuteContractCall(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &proxyFactoryAddr},
			factory.CallArgs{
				ContractABI: proxyFactory.ABI,
				MethodName:  "createProxy",
				Args:        []interface{}{gnosisSafeAddr, safeSetupData},
			},
		)
		Expect(err).NotTo(HaveOccurred())
		ethRes, err := s.factory.GetEvmTransactionResponseFromTxResult(createProxyRes)
		Expect(err).NotTo(HaveOccurred())

		proxyCreationEvent := proxyFactory.ABI.Events["ProxyCreation"]
		var proxyCreationLog *evmtypes.Log
		for i := range ethRes.Logs {
			l := ethRes.Logs[i]
			if len(l.Topics) == 0 {
				continue
			}
			if l.Topics[0] != proxyCreationEvent.ID.String() {
				continue
			}
			if common.HexToAddress(l.Address) != proxyFactoryAddr {
				continue
			}
			proxyCreationLog = l
			break
		}
		Expect(proxyCreationLog).NotTo(BeNil())

		eventInputs, err := proxyFactory.ABI.Events["ProxyCreation"].Inputs.Unpack(proxyCreationLog.Data)
		Expect(err).NotTo(HaveOccurred())
		safeWalletAddr, ok := eventInputs[0].(common.Address)
		Expect(ok).To(BeTrue())
		Expect(safeWalletAddr).NotTo(Equal(common.Address{}))
		Expect(s.network.NextBlock()).To(Succeed())

		_, thresholdRes, err := s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &safeWalletAddr},
			factory.CallArgs{ContractABI: gnosisSafe.ABI, MethodName: "getThreshold"},
			testutil.LogCheckArgs{ExpPass: true},
		)
		Expect(err).NotTo(HaveOccurred())
		thrOut, err := gnosisSafe.ABI.Methods["getThreshold"].Outputs.Unpack(thresholdRes.Ret)
		Expect(err).NotTo(HaveOccurred())
		Expect(thrOut[0]).To(Equal(big.NewInt(1)))

		safeWalletAccAddr := sdk.AccAddress(safeWalletAddr.Bytes())
		ctx := s.network.GetContext()
		coins500 := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, transferToSafe))
		Expect(s.network.App.BankKeeper.SendCoins(ctx, safeOwnerOne.AccAddr, safeWalletAccAddr, coins500)).To(Succeed())
		Expect(s.network.App.BankKeeper.SendCoins(ctx, safeOwnerTwo.AccAddr, safeWalletAccAddr, coins500)).To(Succeed())

		safeBank, err := s.grpcHandler.GetBalance(safeWalletAccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(safeBank.Balance.Amount).To(Equal(expectedSafeBank))

		ownerTwoBank, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(ownerTwoBank.Balance.Amount).To(Equal(expectedOwnerTwoBank))

		ownerOneBank, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		n999 := sdkmath.NewInt(999).MulRaw(1e18)
		n1000 := sdkmath.NewInt(1000).MulRaw(1e18)
		amt := ownerOneBank.Balance.Amount
		Expect(amt.GT(n999)).To(BeTrue(), "first owner should keep more than 999 ISLM after gas")
		Expect(amt.LT(n1000)).To(BeTrue(), "first owner should keep less than 1000 ISLM after paying gas for deployment and transfer")

		// Third account: 1001 ISLM for 1000 ISLM Fund + fees
		fAddr, fPriv := testutiltx.NewAddrKey()
		funder = keyring.Key{Addr: fAddr, AccAddr: sdk.AccAddress(fAddr.Bytes()), Priv: fPriv}
		fund101 := sdkmath.NewInt(1001).MulRaw(1e18)
		Expect(s.network.FundAccountWithBaseDenom(funder.AccAddr, fund101)).To(Succeed())
		Expect(s.network.NextBlock()).To(Succeed())

		fundMsg := ucdaotypes.NewMsgFund(sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, fundAmount)), funder.AccAddr)
		res, err := s.factory.CommitCosmosTx(funder.Priv, commonfactory.CosmosTxArgs{
			Msgs:     []sdk.Msg{fundMsg},
			GasPrice: &gasPrice,
			Gas:      &gasLimit,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(res.IsOK()).To(BeTrue(), res.Log)
		Expect(s.network.NextBlock()).To(Succeed())

		funderBankAfterFund, err := s.grpcHandler.GetBalance(funder.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(funderBankAfterFund.Balance.Amount.LT(oneIslm)).To(BeTrue(),
			"third account bank balance should be under 1 ISLM after MsgFund (before vesting grant)")

		ucdaoClient := s.network.GetUCDAOClient()
		funderUcdao, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: funder.AccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(funderUcdao.Balances.AmountOf(utils.BaseDenom)).To(Equal(fundAmount),
			"ucDAO balance for the depositor (third account) should equal funded aISLM")

		safeUcdao, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: safeWalletAccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(safeUcdao.Balances.IsZero()).To(BeTrue(),
			"Safe cannot sign MsgFund with depositor=self; ucDAO balance for Safe stays zero until authz or another flow")

		// Fourth account funds clawback vesting on account 3: fully vested after 1s, locked for 1 year (single lockup period).
		fourAddr, fourPriv := testutiltx.NewAddrKey()
		vestingGranter := keyring.Key{Addr: fourAddr, AccAddr: sdk.AccAddress(fourAddr.Bytes()), Priv: fourPriv}
		fundGranterAmt := sdkmath.NewInt(1001).MulRaw(1e18)
		Expect(s.network.FundAccountWithBaseDenom(vestingGranter.AccAddr, fundGranterAmt)).To(Succeed())
		Expect(s.network.NextBlock()).To(Succeed())

		ctx2 := s.network.GetContext()
		startTime := ctx2.BlockTime()
		coin1000 := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000).MulRaw(1e18))
		oneYearSec := int64(365 * 24 * 3600)
		lockupPeriods := sdkvesting.Periods{{Length: oneYearSec, Amount: sdk.NewCoins(coin1000)}}
		vestingPeriods := sdkvesting.Periods{{Length: 1, Amount: sdk.NewCoins(coin1000)}}
		var emptyValAddr sdk.ValAddress
		convertMsg := vestingtypes.NewMsgConvertIntoVestingAccount(
			vestingGranter.AccAddr,
			funder.AccAddr,
			startTime,
			lockupPeriods,
			vestingPeriods,
			false,
			false,
			emptyValAddr,
		)
		resVest, err := s.factory.CommitCosmosTx(vestingGranter.Priv, commonfactory.CosmosTxArgs{
			Msgs:     []sdk.Msg{convertMsg},
			GasPrice: &gasPrice,
			Gas:      &gasLimit,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resVest.IsOK()).To(BeTrue(), resVest.Log)
		Expect(s.network.NextBlock()).To(Succeed())

		granterBank, err := s.grpcHandler.GetBalance(vestingGranter.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(granterBank.Balance.Amount.LT(oneIslm)).To(BeTrue(),
			"fourth account should have under 1 ISLM after vesting grant and fees")

		thousandIslm := sdkmath.NewInt(1000).MulRaw(1e18)
		funderTotalBank, err := s.grpcHandler.GetBalance(funder.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(funderTotalBank.Balance.Amount.GT(thousandIslm)).To(BeTrue(),
			"third account total bank balance should exceed 1000 ISLM (locked grant + leftover from before)")

		funderSpendable, err := s.grpcHandler.GetSpendableBalance(funder.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(funderSpendable.Balance.Amount.LT(oneIslm)).To(BeTrue(),
			"third account spendable aISLM should stay under 1 ISLM while grant is locked")

		// Liquidate 1000 aISLM of locked vested coins into liquid token (aLIQUID0 on first liquidation).
		liquidMsg := liquidvestingtypes.NewMsgLiquidate(funder.AccAddr, funder.AccAddr, coin1000)
		resLiq, err := s.factory.CommitCosmosTx(funder.Priv, commonfactory.CosmosTxArgs{
			Msgs:     []sdk.Msg{liquidMsg},
			GasPrice: &gasPrice,
			Gas:      &gasLimit,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resLiq.IsOK()).To(BeTrue(), resLiq.Log)
		Expect(s.network.NextBlock()).To(Succeed())

		funderAISLM, err := s.grpcHandler.GetBalance(funder.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(funderAISLM.Balance.Amount.LT(oneIslm)).To(BeTrue(),
			"third account aISLM bank balance should be under 1 ISLM after liquidation")

		liquidDenom := liquidvestingtypes.DenomBaseNameFromID(0)
		funderLiquid, err := s.grpcHandler.GetBalance(funder.AccAddr, liquidDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(funderLiquid.Balance.Amount).To(Equal(thousandIslm),
			"liquid vesting token balance should match 1000 ISLM liquidated")

		// MsgFund credits ucDAO to depositor only. After funding liquid aLIQUID on account 3, move both
		// aISLM and liquid ucDAO positions to Safe in one MsgTransferOwnershipWithAmount.
		fundLiquidMsg := ucdaotypes.NewMsgFund(sdk.NewCoins(sdk.NewCoin(liquidDenom, thousandIslm)), funder.AccAddr)
		resFundLiq, err := s.factory.CommitCosmosTx(funder.Priv, commonfactory.CosmosTxArgs{
			Msgs:     []sdk.Msg{fundLiquidMsg},
			GasPrice: &gasPrice,
			Gas:      &gasLimit,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resFundLiq.IsOK()).To(BeTrue(), resFundLiq.Log)
		Expect(s.network.NextBlock()).To(Succeed())

		transferToSafeMsg := ucdaotypes.NewMsgTransferOwnershipWithAmount(
			funder.AccAddr,
			safeWalletAccAddr,
			sdk.NewCoins(
				sdk.NewCoin(utils.BaseDenom, fundAmount),
				sdk.NewCoin(liquidDenom, thousandIslm),
			),
		)
		resXfer, err := s.factory.CommitCosmosTx(funder.Priv, commonfactory.CosmosTxArgs{
			Msgs:     []sdk.Msg{transferToSafeMsg},
			GasPrice: &gasPrice,
			Gas:      &gasLimit,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resXfer.IsOK()).To(BeTrue(), resXfer.Log)
		Expect(s.network.NextBlock()).To(Succeed())

		safeUcdaoAfter, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: safeWalletAccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(safeUcdaoAfter.Balances.AmountOf(utils.BaseDenom)).To(Equal(fundAmount),
			"Safe ucDAO: 1000 aISLM")
		Expect(safeUcdaoAfter.Balances.AmountOf(liquidDenom)).To(Equal(thousandIslm),
			"Safe ucDAO: 1000 liquid vesting (aLIQUID0)")

		funderUcdaoAfter, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: funder.AccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(funderUcdaoAfter.Balances.IsZero()).To(BeTrue(),
			"account 3 should have no ucDAO balance after transferring both positions to Safe")

		// Safe (1-of-2, owner1 signs) calls ucDAO precompile convertToHaqq: burn 500 ISLM worth from Safe's ucDAO
		// position (sender/receiver = Safe). Inner call reverts while precompile requires origin==sender (EOA vs Safe).
		fiveHundredIslm := sdkmath.NewInt(500).MulRaw(1e18)
		expectedSafeUcdaoIslm := sdkmath.NewInt(1500).MulRaw(1e18)

		_, nonceRes, err := s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &safeWalletAddr},
			factory.CallArgs{ContractABI: gnosisSafe.ABI, MethodName: "nonce"},
			testutil.LogCheckArgs{ExpPass: true},
		)
		Expect(err).NotTo(HaveOccurred())
		nonceOut, err := gnosisSafe.ABI.Methods["nonce"].Outputs.Unpack(nonceRes.Ret)
		Expect(err).NotTo(HaveOccurred())
		nonceBI, ok := nonceOut[0].(*big.Int)
		Expect(ok).To(BeTrue())
		Expect(nonceBI.Sign()).To(Equal(0), "first Safe exec must use nonce 0 (no prior execTransaction)")

		ctxEthiq := s.network.GetContext()
		expectedHaqqMint, err := s.network.App.EthiqKeeper.CalculateHaqqCoinsToMint(ctxEthiq, fiveHundredIslm)
		Expect(err).NotTo(HaveOccurred())

		ucdaoPc, err := ucdao.NewPrecompile(s.network.App.DaoKeeper)
		Expect(err).NotTo(HaveOccurred())
		convertCallData, err := ucdaoPc.ABI.Pack(ucdao.ConvertToHaqqMethod, safeWalletAddr, safeWalletAddr, fiveHundredIslm.BigInt())
		Expect(err).NotTo(HaveOccurred())

		ucdaoPrecompileAddr := common.HexToAddress(evmtypes.UcdaoPrecompileAddress)
		// getTransactionHash args must match execTransaction (except signatures); last arg is Safe nonce.
		getTxHashArgs := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "getTransactionHash",
			Args: []interface{}{
				ucdaoPrecompileAddr,
				big.NewInt(0),
				convertCallData,
				uint8(0), // Operation.Call
				big.NewInt(400_000),
				big.NewInt(0),
				big.NewInt(0),
				common.Address{},
				common.Address{},
				big.NewInt(0),
			},
		}
		_, txHashRes, err := s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &safeWalletAddr},
			getTxHashArgs,
			testutil.LogCheckArgs{ExpPass: true},
		)
		Expect(err).NotTo(HaveOccurred())
		txHashOutputs, err := gnosisSafe.ABI.Methods["getTransactionHash"].Outputs.Unpack(txHashRes.Ret)
		Expect(err).NotTo(HaveOccurred())
		Expect(txHashOutputs).To(HaveLen(1))
		txHash, ok := txHashOutputs[0].([32]byte)
		Expect(ok).To(BeTrue())
		Expect(txHash).NotTo(Equal([32]byte{}))

		signature := make([]byte, 65)
		copy(signature[12:32], safeOwnerOne.Addr.Bytes())
		signature[64] = 1

		execTxArgs := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "execTransaction",
			Args: []interface{}{
				ucdaoPrecompileAddr,
				big.NewInt(0),
				convertCallData,
				uint8(0),
				big.NewInt(400_000),
				big.NewInt(0),
				big.NewInt(0),
				common.Address{},
				common.Address{},
				signature,
			},
		}
		execTxRes, err := s.factory.ExecuteContractCall(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &safeWalletAddr},
			execTxArgs,
		)
		Expect(err).NotTo(HaveOccurred())
		execRes, err := s.factory.GetEvmTransactionResponseFromTxResult(execTxRes)
		Expect(err).NotTo(HaveOccurred())
		execOutputs, err := gnosisSafe.ABI.Methods["execTransaction"].Outputs.Unpack(execRes.Ret)
		Expect(err).NotTo(HaveOccurred())
		Expect(execOutputs).To(HaveLen(1))
		execSuccess, ok := execOutputs[0].(bool)
		Expect(ok).To(BeTrue())
		Expect(execSuccess).To(BeTrue(), "Safe execTransaction should succeed once precompile accepts contract caller as holder")

		executionSuccessEvent := gnosisSafe.ABI.Events["ExecutionSuccess"]
		executionSuccessFound := false
		for i := range execRes.Logs {
			l := execRes.Logs[i]
			if len(l.Topics) == 0 {
				continue
			}
			if l.Topics[0] != executionSuccessEvent.ID.String() {
				continue
			}
			if common.HexToAddress(l.Address) != safeWalletAddr {
				continue
			}
			executionSuccessFound = true
			break
		}
		Expect(executionSuccessFound).To(BeTrue(), "inner call success should emit ExecutionSuccess on Safe")

		Expect(s.network.NextBlock()).To(Succeed())

		ownerOneAfter, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(ownerOneAfter.Balance.Amount.GT(n999)).To(BeTrue(),
			"first Safe owner bank balance should stay above 999 ISLM after Safe exec (gas only)")

		ownerTwoAfter, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(ownerTwoAfter.Balance.Amount).To(Equal(expectedOwnerTwoBank),
			"second owner bank balance should remain 1000 ISLM")

		safeBankAfter, err := s.grpcHandler.GetBalance(safeWalletAccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(safeBankAfter.Balance.Amount).To(Equal(expectedSafeBank),
			"Safe bank ISLM should stay 1000 (conversion uses ucDAO escrow, not Safe bank)")

		safeUcdaoFinal, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: safeWalletAccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(safeUcdaoFinal.Balances.AmountOf(utils.BaseDenom)).To(Equal(expectedSafeUcdaoIslm),
			"Safe ucDAO: 1500 aISLM equivalent after redeeming liquid and burning 500 ISLM")
		Expect(safeUcdaoFinal.Balances.AmountOf(liquidDenom)).To(Equal(sdkmath.ZeroInt()),
			"liquid ucDAO denoms should be fully redeemed/converted")

		safeHaqqAfter, err := s.grpcHandler.GetBalance(safeWalletAccAddr, ethiqtypes.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(safeHaqqAfter.Balance.Amount).To(Equal(expectedHaqqMint),
			"aHAQQ mint for 500 ISLM follows ethiq pricing (not necessarily 1:1 display HAQQ)")

		fmt.Fprintf(os.Stderr,
			"\n[DEBUG] Safe ucDAO AllBalances (after convertToHaqq via Safe): %s | address=%s\n",
			safeUcdaoFinal.Balances.String(), safeWalletAccAddr.String())
	})
})
