package ucdao_test

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"testing"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/testutil"
	multisendcontracts "github.com/haqq-network/haqq/precompiles/testutil/contracts/multisend"
	safecontracts "github.com/haqq-network/haqq/precompiles/testutil/contracts/safe"
	"github.com/haqq-network/haqq/precompiles/ucdao"
	commonfactory "github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	testutiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

func TestUCDAOSafeIntegrationSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ucDAO Gnosis Safe integration suite")
}

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

var _ = Describe("ucDAO with Gnosis Safe (phase 1)", Ordered, func() {
	var (
		kr               keyring.Keyring
		s                *ucdaoSafeSuite
		deployer         keyring.Key
		safeOwnerOne     keyring.Key
		safeOwnerTwo     keyring.Key
		gnosisSafe       evmtypes.CompiledContract
		gnosisSafeAddr   common.Address
		multiSend        evmtypes.CompiledContract
		multiSendAddr    common.Address
		proxyFactory     evmtypes.CompiledContract
		proxyFactoryAddr common.Address
	)

	// ucdaoPositionHolder selects whose ucDAO position the specs convert, which decides
	// whether the ConvertToHaqq allowance is involved at all.
	type ucdaoPositionHolder int

	const (
		// holderSafe puts the position on the Safe itself. The Safe then converts its own
		// position: its own threshold is the authorization and no grant is consulted.
		holderSafe ucdaoPositionHolder = iota
		// holderOwnerOne puts the position on owner one. The Safe acts on someone else's
		// position, so the grant owner one issued to the Safe is what gates the call.
		//
		// NOTE: this configuration only works while owner one is the account that submits
		// execTransaction, because the precompile still requires origin == sender. If the
		// suite ever executes as owner two or through a relayer, these specs fail on that
		// guard rather than on the allowance.
		holderOwnerOne
	)

	type preparedSafeState struct {
		safeWalletAddr    common.Address
		safeWalletAccAddr sdk.AccAddress
		// positionAddr is the account whose ucDAO position the convert helpers move.
		positionAddr    common.Address
		positionAccAddr sdk.AccAddress
		liquidDenom     string
	}

	type safeBatchSnapshot struct {
		ucdaoBase      sdkmath.Int
		ucdaoLiquid    sdkmath.Int
		bankBase       sdkmath.Int
		bankLiquid     sdkmath.Int
		haqqBalance    sdkmath.Int
		ownerOneBank   sdkmath.Int
		moduleISLMBank sdkmath.Int
	}

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
		multiSend, err = multisendcontracts.LoadMultiSendContract()
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

		multiSendAddr, err = s.factory.DeployContract(
			deployer.Priv,
			evmtypes.EvmTxArgs{},
			factory.ContractDeploymentData{Contract: multiSend},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(multiSendAddr).NotTo(Equal(common.Address{}))
		Expect(s.network.NextBlock()).To(Succeed())
	})

	prepareSafeWithUcdaoPositions := func(holder ucdaoPositionHolder) preparedSafeState {
		oneAddr, onePriv := testutiltx.NewAddrKey()
		safeOwnerOne = keyring.Key{Addr: oneAddr, AccAddr: sdk.AccAddress(oneAddr.Bytes()), Priv: onePriv}
		twoAddr, twoPriv := testutiltx.NewAddrKey()
		safeOwnerTwo = keyring.Key{Addr: twoAddr, AccAddr: sdk.AccAddress(twoAddr.Bytes()), Priv: twoPriv}
		fund1500 := sdkmath.NewInt(1500).MulRaw(1e18)
		Expect(s.network.FundAccountWithBaseDenom(safeOwnerOne.AccAddr, fund1500)).To(Succeed())
		Expect(s.network.FundAccountWithBaseDenom(safeOwnerTwo.AccAddr, fund1500)).To(Succeed())
		Expect(s.network.NextBlock()).To(Succeed())

		ownerOneFunded, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(ownerOneFunded.Balance.Amount).To(Equal(fund1500), "owner1 should be fully funded before Safe setup")
		ownerTwoFunded, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(ownerTwoFunded.Balance.Amount).To(Equal(fund1500), "owner2 should be fully funded before Safe setup")

		transferToSafe := sdkmath.NewInt(500).MulRaw(1e18)
		expectedSafeBank := sdkmath.NewInt(1000).MulRaw(1e18)
		expectedOwnerTwoBank := sdkmath.NewInt(1000).MulRaw(1e18)
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
		Expect(s.network.NextBlock()).To(Succeed())

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
		Expect(amt.GT(n999)).To(BeTrue())
		Expect(amt.LT(n1000)).To(BeTrue())

		fAddr, fPriv := testutiltx.NewAddrKey()
		funder := keyring.Key{Addr: fAddr, AccAddr: sdk.AccAddress(fAddr.Bytes()), Priv: fPriv}
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

		ucdaoClient := s.network.GetUCDAOClient()
		safeUcdao, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: safeWalletAccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(safeUcdao.Balances.IsZero()).To(BeTrue())

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

		liquidMsg := liquidvestingtypes.NewMsgLiquidate(funder.AccAddr, funder.AccAddr, coin1000)
		resLiq, err := s.factory.CommitCosmosTx(funder.Priv, commonfactory.CosmosTxArgs{
			Msgs:     []sdk.Msg{liquidMsg},
			GasPrice: &gasPrice,
			Gas:      &gasLimit,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resLiq.IsOK()).To(BeTrue(), resLiq.Log)
		Expect(s.network.NextBlock()).To(Succeed())

		thousandIslm := sdkmath.NewInt(1000).MulRaw(1e18)
		funderAllBalances := s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), funder.AccAddr)
		liquidDenom := ""
		for _, coin := range funderAllBalances {
			if strings.HasPrefix(coin.Denom, "aLIQUID") && coin.Amount.Equal(thousandIslm) {
				liquidDenom = coin.Denom
				break
			}
		}
		Expect(liquidDenom).NotTo(BeEmpty(), "expected liquid denom with 1000 units after liquidation")
		fundLiquidMsg := ucdaotypes.NewMsgFund(sdk.NewCoins(sdk.NewCoin(liquidDenom, thousandIslm)), funder.AccAddr)
		resFundLiq, err := s.factory.CommitCosmosTx(funder.Priv, commonfactory.CosmosTxArgs{
			Msgs:     []sdk.Msg{fundLiquidMsg},
			GasPrice: &gasPrice,
			Gas:      &gasLimit,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resFundLiq.IsOK()).To(BeTrue(), resFundLiq.Log)
		Expect(s.network.NextBlock()).To(Succeed())

		positionAddr := safeWalletAddr
		positionAccAddr := safeWalletAccAddr
		if holder == holderOwnerOne {
			positionAddr = safeOwnerOne.Addr
			positionAccAddr = safeOwnerOne.AccAddr
		}

		transferPositionMsg := ucdaotypes.NewMsgTransferOwnershipWithAmount(
			funder.AccAddr,
			positionAccAddr,
			sdk.NewCoins(
				sdk.NewCoin(utils.BaseDenom, fundAmount),
				sdk.NewCoin(liquidDenom, thousandIslm),
			),
		)
		resXfer, err := s.factory.CommitCosmosTx(funder.Priv, commonfactory.CosmosTxArgs{
			Msgs:     []sdk.Msg{transferPositionMsg},
			GasPrice: &gasPrice,
			Gas:      &gasLimit,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resXfer.IsOK()).To(BeTrue(), resXfer.Log)
		Expect(s.network.NextBlock()).To(Succeed())

		positionUcdaoAfter, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: positionAccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(positionUcdaoAfter.Balances.AmountOf(utils.BaseDenom)).To(Equal(fundAmount))
		Expect(positionUcdaoAfter.Balances.AmountOf(liquidDenom)).To(Equal(thousandIslm))

		return preparedSafeState{
			safeWalletAddr:    safeWalletAddr,
			safeWalletAccAddr: safeWalletAccAddr,
			positionAddr:      positionAddr,
			positionAccAddr:   positionAccAddr,
			liquidDenom:       liquidDenom,
		}
	}

	execSafeConvertToHaqq := func(state preparedSafeState, amount sdkmath.Int, expectedNonce int64) (bool, *evmtypes.MsgEthereumTxResponse) {
		safeWalletAddr := state.safeWalletAddr
		_, nonceRes, err := s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &safeWalletAddr},
			factory.CallArgs{ContractABI: gnosisSafe.ABI, MethodName: "nonce"},
			testutil.LogCheckArgs{ExpPass: true},
		)
		Expect(err).NotTo(HaveOccurred())
		nonceOut, err := gnosisSafe.ABI.Methods["nonce"].Outputs.Unpack(nonceRes.Ret)
		Expect(err).NotTo(HaveOccurred())
		nonce, ok := nonceOut[0].(*big.Int)
		Expect(ok).To(BeTrue())
		Expect(nonce.Cmp(big.NewInt(expectedNonce))).To(Equal(0))

		ucdaoPc, err := ucdao.NewPrecompile(s.network.App.DaoKeeper, s.network.App.AuthzKeeper)
		Expect(err).NotTo(HaveOccurred())
		// The sender is the account whose ucDAO position moves. With holderSafe that is the
		// Safe acting on itself, which needs no grant; with holderOwnerOne the Safe acts on
		// the owner's position and the owner's grant to the Safe is the gate.
		convertCallData, err := ucdaoPc.ABI.Pack(ucdao.ConvertToHaqqMethod, state.positionAddr, state.positionAddr, amount.BigInt())
		Expect(err).NotTo(HaveOccurred())

		ucdaoPrecompileAddr := common.HexToAddress(evmtypes.UcdaoPrecompileAddress)
		getTxHashArgs := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "getTransactionHash",
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
				big.NewInt(expectedNonce),
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
		return execSuccess, execRes
	}

	approveForSafeConvert := func(safeWalletAddr common.Address, amount sdkmath.Int) {
		ucdaoPc, err := ucdao.NewPrecompile(s.network.App.DaoKeeper, s.network.App.AuthzKeeper)
		Expect(err).NotTo(HaveOccurred())
		ucdaoPrecompileAddr := common.HexToAddress(evmtypes.UcdaoPrecompileAddress)
		approveTxRes, err := s.factory.ExecuteContractCall(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &ucdaoPrecompileAddr},
			factory.CallArgs{
				ContractABI: ucdaoPc.ABI,
				MethodName:  authorization.ApproveMethod,
				Args: []interface{}{
					safeWalletAddr,
					amount.BigInt(),
					[]string{ucdao.ConvertToHaqqMsgURL},
				},
			},
		)
		Expect(err).NotTo(HaveOccurred())
		approveRes, err := s.factory.GetEvmTransactionResponseFromTxResult(approveTxRes)
		Expect(err).NotTo(HaveOccurred())
		approveOut, err := ucdaoPc.ABI.Methods[authorization.ApproveMethod].Outputs.Unpack(approveRes.Ret)
		Expect(err).NotTo(HaveOccurred())
		Expect(approveOut).To(HaveLen(1))
		approved, ok := approveOut[0].(bool)
		Expect(ok).To(BeTrue())
		Expect(approved).To(BeTrue())
	}

	revokeForSafeConvert := func(safeWalletAddr common.Address) {
		ucdaoPc, err := ucdao.NewPrecompile(s.network.App.DaoKeeper, s.network.App.AuthzKeeper)
		Expect(err).NotTo(HaveOccurred())
		ucdaoPrecompileAddr := common.HexToAddress(evmtypes.UcdaoPrecompileAddress)
		revokeTxRes, err := s.factory.ExecuteContractCall(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &ucdaoPrecompileAddr},
			factory.CallArgs{
				ContractABI: ucdaoPc.ABI,
				MethodName:  authorization.RevokeMethod,
				Args: []interface{}{
					safeWalletAddr,
					[]string{ucdao.ConvertToHaqqMsgURL},
				},
			},
		)
		Expect(err).NotTo(HaveOccurred())
		revokeRes, err := s.factory.GetEvmTransactionResponseFromTxResult(revokeTxRes)
		Expect(err).NotTo(HaveOccurred())
		revokeOut, err := ucdaoPc.ABI.Methods[authorization.RevokeMethod].Outputs.Unpack(revokeRes.Ret)
		Expect(err).NotTo(HaveOccurred())
		Expect(revokeOut).To(HaveLen(1))
		revoked, ok := revokeOut[0].(bool)
		Expect(ok).To(BeTrue())
		Expect(revoked).To(BeTrue())
	}

	liquidVestingModuleAddr := authtypes.NewModuleAddress(liquidvestingtypes.ModuleName)
	thousandIslmLiquid := sdkmath.NewInt(1000).MulRaw(1e18)

	getLiquidVestingModuleISLMBalance := func() sdkmath.Int {
		ctx := s.network.GetContext()
		return s.network.App.BankKeeper.GetBalance(ctx, liquidVestingModuleAddr, utils.BaseDenom).Amount
	}

	safeUcdaoLiquidBalance := func(acc sdk.AccAddress, liquidDenom string) sdkmath.Int {
		ucdaoClient := s.network.GetUCDAOClient()
		res, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: acc.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		return res.Balances.AmountOf(liquidDenom)
	}

	captureSafeBatchSnapshot := func(state preparedSafeState) safeBatchSnapshot {
		ucdaoClient := s.network.GetUCDAOClient()
		// The ucDAO position is the account the batch converts from, which is the Safe under
		// holderSafe and owner one under holderOwnerOne.
		positionUcdaoBalances, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: state.positionAccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())

		safeBankBase, err := s.grpcHandler.GetBalance(state.safeWalletAccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		safeBankLiquid, err := s.grpcHandler.GetBalance(state.safeWalletAccAddr, state.liquidDenom)
		Expect(err).NotTo(HaveOccurred())
		safeHaqq, err := s.grpcHandler.GetBalance(state.safeWalletAccAddr, ethiqtypes.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		ownerOneBank, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
		Expect(err).NotTo(HaveOccurred())

		return safeBatchSnapshot{
			ucdaoBase:      positionUcdaoBalances.Balances.AmountOf(utils.BaseDenom),
			ucdaoLiquid:    positionUcdaoBalances.Balances.AmountOf(state.liquidDenom),
			bankBase:       safeBankBase.Balance.Amount,
			bankLiquid:     safeBankLiquid.Balance.Amount,
			haqqBalance:    safeHaqq.Balance.Amount,
			ownerOneBank:   ownerOneBank.Balance.Amount,
			moduleISLMBank: getLiquidVestingModuleISLMBalance(),
		}
	}

	assertLiquidRedeemMatchesModuleISLM := func(
		safeAcc sdk.AccAddress,
		liquidDenom string,
		modBefore sdkmath.Int,
		liquidBefore sdkmath.Int,
	) {
		Expect(s.network.NextBlock()).To(Succeed())
		liquidAfter := safeUcdaoLiquidBalance(safeAcc, liquidDenom)
		modAfter := getLiquidVestingModuleISLMBalance()
		liquidRedeemed := liquidBefore.Sub(liquidAfter)
		modSpent := modBefore.Sub(modAfter)
		Expect(modSpent).To(Equal(liquidRedeemed),
			"liquid vesting module ISLM decrease should match Safe ucDAO liquid redeemed (ISLM-equivalent 1:1)")
	}

	assertSafeExecEvent := func(execRes *evmtypes.MsgEthereumTxResponse, eventName string, safeWalletAddr common.Address) {
		event := gnosisSafe.ABI.Events[eventName]
		found := false
		for i := range execRes.Logs {
			l := execRes.Logs[i]
			if len(l.Topics) == 0 {
				continue
			}
			if l.Topics[0] != event.ID.String() {
				continue
			}
			if common.HexToAddress(l.Address) != safeWalletAddr {
				continue
			}
			found = true
			break
		}
		Expect(found).To(BeTrue(), "expected %s event on Safe", eventName)
	}

	packMultiSendTx := func(operation uint8, to common.Address, value *big.Int, data []byte) []byte {
		txValue := big.NewInt(0)
		if value != nil {
			txValue = value
		}

		var entry bytes.Buffer
		entry.WriteByte(operation)
		entry.Write(to.Bytes())
		entry.Write(common.LeftPadBytes(txValue.Bytes(), 32))
		entry.Write(common.LeftPadBytes(big.NewInt(int64(len(data))).Bytes(), 32))
		entry.Write(data)
		return entry.Bytes()
	}

	readSafeNonce := func(safeWalletAddr common.Address) *big.Int {
		_, nonceRes, err := s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &safeWalletAddr},
			factory.CallArgs{ContractABI: gnosisSafe.ABI, MethodName: "nonce"},
			testutil.LogCheckArgs{ExpPass: true},
		)
		Expect(err).NotTo(HaveOccurred())
		nonceOut, err := gnosisSafe.ABI.Methods["nonce"].Outputs.Unpack(nonceRes.Ret)
		Expect(err).NotTo(HaveOccurred())
		Expect(nonceOut).To(HaveLen(1))
		nonce, ok := nonceOut[0].(*big.Int)
		Expect(ok).To(BeTrue())
		return nonce
	}

	safeExecMultiSendBatch := func(
		safeWalletAddr common.Address,
		batchTxs []byte,
		safeNonce *big.Int,
	) (bool, *evmtypes.MsgEthereumTxResponse) {
		multiSendCallData, err := multiSend.ABI.Pack("multiSend", batchTxs)
		Expect(err).NotTo(HaveOccurred())

		getTxHashArgs := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "getTransactionHash",
			Args: []interface{}{
				multiSendAddr,
				big.NewInt(0),
				multiSendCallData,
				uint8(1),
				big.NewInt(700_000),
				big.NewInt(0),
				big.NewInt(0),
				common.Address{},
				common.Address{},
				safeNonce,
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
		_, ok := txHashOutputs[0].([32]byte)
		Expect(ok).To(BeTrue())

		signature := make([]byte, 65)
		copy(signature[12:32], safeOwnerOne.Addr.Bytes())
		signature[64] = 1

		execTxRes, err := s.factory.ExecuteContractCall(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &safeWalletAddr},
			factory.CallArgs{
				ContractABI: gnosisSafe.ABI,
				MethodName:  "execTransaction",
				Args: []interface{}{
					multiSendAddr,
					big.NewInt(0),
					multiSendCallData,
					uint8(1),
					big.NewInt(700_000),
					big.NewInt(0),
					big.NewInt(0),
					common.Address{},
					common.Address{},
					signature,
				},
			},
		)
		Expect(err).NotTo(HaveOccurred())
		execRes, err := s.factory.GetEvmTransactionResponseFromTxResult(execTxRes)
		Expect(err).NotTo(HaveOccurred())
		execOutputs, err := gnosisSafe.ABI.Methods["execTransaction"].Outputs.Unpack(execRes.Ret)
		Expect(err).NotTo(HaveOccurred())
		Expect(execOutputs).To(HaveLen(1))
		execSuccess, ok := execOutputs[0].(bool)
		Expect(ok).To(BeTrue())
		return execSuccess, execRes
	}

	buildSafeConvertBatchTxData := func(state preparedSafeState, firstConvertAmount, secondConvertAmount sdkmath.Int) []byte {
		ucdaoPc, err := ucdao.NewPrecompile(s.network.App.DaoKeeper, s.network.App.AuthzKeeper)
		Expect(err).NotTo(HaveOccurred())
		precompileAddr := common.HexToAddress(evmtypes.UcdaoPrecompileAddress)
		firstCallData, err := ucdaoPc.ABI.Pack(ucdao.ConvertToHaqqMethod, state.positionAddr, state.positionAddr, firstConvertAmount.BigInt())
		Expect(err).NotTo(HaveOccurred())
		secondCallData, err := ucdaoPc.ABI.Pack(ucdao.ConvertToHaqqMethod, state.positionAddr, state.positionAddr, secondConvertAmount.BigInt())
		Expect(err).NotTo(HaveOccurred())
		return append(
			packMultiSendTx(0, precompileAddr, big.NewInt(0), firstCallData),
			packMultiSendTx(0, precompileAddr, big.NewInt(0), secondCallData)...,
		)
	}

	prepareSafeWithBatchMintBalances := func(holder ucdaoPositionHolder) preparedSafeState {
		oneAddr, onePriv := testutiltx.NewAddrKey()
		safeOwnerOne = keyring.Key{Addr: oneAddr, AccAddr: sdk.AccAddress(oneAddr.Bytes()), Priv: onePriv}
		twoAddr, twoPriv := testutiltx.NewAddrKey()
		safeOwnerTwo = keyring.Key{Addr: twoAddr, AccAddr: sdk.AccAddress(twoAddr.Bytes()), Priv: twoPriv}

		fund1500 := sdkmath.NewInt(1500).MulRaw(1e18)
		Expect(s.network.FundAccountWithBaseDenom(safeOwnerOne.AccAddr, fund1500)).To(Succeed())
		Expect(s.network.FundAccountWithBaseDenom(safeOwnerTwo.AccAddr, fund1500)).To(Succeed())
		Expect(s.network.NextBlock()).To(Succeed())

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
		safeWalletAccAddr := sdk.AccAddress(safeWalletAddr.Bytes())

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

		funderAddr, funderPriv := testutiltx.NewAddrKey()
		funder := keyring.Key{Addr: funderAddr, AccAddr: sdk.AccAddress(funderAddr.Bytes()), Priv: funderPriv}
		funderInitialIslm := sdkmath.NewInt(2000).MulRaw(1e18)
		Expect(s.network.FundAccountWithBaseDenom(funder.AccAddr, funderInitialIslm)).To(Succeed())
		Expect(s.network.NextBlock()).To(Succeed())

		liquidationCoin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(2000).MulRaw(1e18))
		startTime := s.network.GetContext().BlockTime()
		oneYearSec := int64(365 * 24 * 3600)
		lockupPeriods := sdkvesting.Periods{{Length: oneYearSec, Amount: sdk.NewCoins(liquidationCoin)}}
		vestingPeriods := sdkvesting.Periods{{Length: 1, Amount: sdk.NewCoins(liquidationCoin)}}
		var emptyValAddr sdk.ValAddress

		liquidSourceAddr, liquidSourcePriv := testutiltx.NewAddrKey()
		liquidSource := keyring.Key{
			Addr:    liquidSourceAddr,
			AccAddr: sdk.AccAddress(liquidSourceAddr.Bytes()),
			Priv:    liquidSourcePriv,
		}
		liquidSourceFund := sdkmath.NewInt(2500).MulRaw(1e18)
		Expect(s.network.FundAccountWithBaseDenom(liquidSource.AccAddr, liquidSourceFund)).To(Succeed())
		Expect(s.network.NextBlock()).To(Succeed())

		vestingGranterAddr, vestingGranterPriv := testutiltx.NewAddrKey()
		vestingGranter := keyring.Key{
			Addr:    vestingGranterAddr,
			AccAddr: sdk.AccAddress(vestingGranterAddr.Bytes()),
			Priv:    vestingGranterPriv,
		}
		vestingGranterFund := sdkmath.NewInt(2500).MulRaw(1e18)
		Expect(s.network.FundAccountWithBaseDenom(vestingGranter.AccAddr, vestingGranterFund)).To(Succeed())
		Expect(s.network.NextBlock()).To(Succeed())

		gasPrice := sdkmath.NewInt(1_000_000_000)
		gasLimit := uint64(400_000)
		convertMsg := vestingtypes.NewMsgConvertIntoVestingAccount(
			vestingGranter.AccAddr,
			liquidSource.AccAddr,
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

		liquidMsg := liquidvestingtypes.NewMsgLiquidate(liquidSource.AccAddr, liquidSource.AccAddr, liquidationCoin)
		resLiq, err := s.factory.CommitCosmosTx(liquidSource.Priv, commonfactory.CosmosTxArgs{
			Msgs:     []sdk.Msg{liquidMsg},
			GasPrice: &gasPrice,
			Gas:      &gasLimit,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resLiq.IsOK()).To(BeTrue(), resLiq.Log)
		Expect(s.network.NextBlock()).To(Succeed())

		liquidSourceBalances := s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), liquidSource.AccAddr)
		liquidDenom := ""
		for _, coin := range liquidSourceBalances {
			if strings.HasPrefix(coin.Denom, "aLIQUID") && coin.Amount.IsPositive() {
				liquidDenom = coin.Denom
				break
			}
		}
		Expect(liquidDenom).NotTo(BeEmpty(), "expected liquid denom after liquidation")

		ctx := s.network.GetContext()
		distributedLiquidAmount := sdkmath.NewInt(800).MulRaw(1e18)
		Expect(s.network.App.BankKeeper.SendCoins(
			ctx,
			liquidSource.AccAddr,
			funder.AccAddr,
			sdk.NewCoins(sdk.NewCoin(liquidDenom, distributedLiquidAmount)),
		)).To(Succeed())
		Expect(s.network.NextBlock()).To(Succeed())

		safeBankFundAmount := sdkmath.NewInt(500).MulRaw(1e18)
		Expect(s.network.App.BankKeeper.SendCoins(
			ctx,
			funder.AccAddr,
			safeWalletAccAddr,
			sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, safeBankFundAmount), sdk.NewCoin(liquidDenom, safeBankFundAmount)),
		)).To(Succeed())
		Expect(s.network.NextBlock()).To(Succeed())

		safeUcdaoBaseFund := sdkmath.NewInt(300).MulRaw(1e18)
		safeUcdaoLiquidFund := sdkmath.NewInt(300).MulRaw(1e18)
		resFund, err := s.factory.CommitCosmosTx(funder.Priv, commonfactory.CosmosTxArgs{
			Msgs: []sdk.Msg{ucdaotypes.NewMsgFund(
				sdk.NewCoins(
					sdk.NewCoin(utils.BaseDenom, safeUcdaoBaseFund),
					sdk.NewCoin(liquidDenom, safeUcdaoLiquidFund),
				),
				funder.AccAddr,
			)},
			GasPrice: &gasPrice,
			Gas:      &gasLimit,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resFund.IsOK()).To(BeTrue(), resFund.Log)
		Expect(s.network.NextBlock()).To(Succeed())

		positionAddr := safeWalletAddr
		positionAccAddr := safeWalletAccAddr
		if holder == holderOwnerOne {
			positionAddr = safeOwnerOne.Addr
			positionAccAddr = safeOwnerOne.AccAddr
		}

		resXfer, err := s.factory.CommitCosmosTx(funder.Priv, commonfactory.CosmosTxArgs{
			Msgs: []sdk.Msg{ucdaotypes.NewMsgTransferOwnershipWithAmount(
				funder.AccAddr,
				positionAccAddr,
				sdk.NewCoins(
					sdk.NewCoin(utils.BaseDenom, safeUcdaoBaseFund),
					sdk.NewCoin(liquidDenom, safeUcdaoLiquidFund),
				),
			)},
			GasPrice: &gasPrice,
			Gas:      &gasLimit,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resXfer.IsOK()).To(BeTrue(), resXfer.Log)
		Expect(s.network.NextBlock()).To(Succeed())

		ucdaoClient := s.network.GetUCDAOClient()
		positionUcdaoAfter, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: positionAccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(positionUcdaoAfter.Balances.AmountOf(utils.BaseDenom)).To(Equal(safeUcdaoBaseFund))
		Expect(positionUcdaoAfter.Balances.AmountOf(liquidDenom)).To(Equal(safeUcdaoLiquidFund))

		return preparedSafeState{
			safeWalletAddr:    safeWalletAddr,
			safeWalletAccAddr: safeWalletAccAddr,
			positionAddr:      positionAddr,
			positionAccAddr:   positionAccAddr,
			liquidDenom:       liquidDenom,
		}
	}

	It("runs two Safe convertToHaqq txs on its own position and spends all ucDAO", func() {
		// No approve: the Safe converts its own ucDAO position, so its own threshold is the
		// authorization and no grant is consulted.
		state := prepareSafeWithUcdaoPositions(holderSafe)

		Expect(s.network.NextBlock()).To(Succeed())
		liquidBefore := safeUcdaoLiquidBalance(state.safeWalletAccAddr, state.liquidDenom)
		Expect(liquidBefore).To(Equal(thousandIslmLiquid), "Safe ucDAO liquid before convert should be 1000 ISLM-equivalent")
		modBefore := getLiquidVestingModuleISLMBalance()

		firstAmount := sdkmath.NewInt(500).MulRaw(1e18)
		secondAmount := sdkmath.NewInt(1500).MulRaw(1e18)

		ctxEthiq := s.network.GetContext()
		firstMint, err := s.network.App.EthiqKeeper.CalculateHaqqCoinsToMint(ctxEthiq, firstAmount)
		Expect(err).NotTo(HaveOccurred())
		secondMint, err := s.network.App.EthiqKeeper.CalculateHaqqCoinsToMint(ctxEthiq, secondAmount)
		Expect(err).NotTo(HaveOccurred())

		firstSuccess, firstRes := execSafeConvertToHaqq(state, firstAmount, 0)
		Expect(firstSuccess).To(BeTrue())
		assertSafeExecEvent(firstRes, "ExecutionSuccess", state.safeWalletAddr)
		Expect(s.network.NextBlock()).To(Succeed())

		secondSuccess, secondRes := execSafeConvertToHaqq(state, secondAmount, 1)
		Expect(secondSuccess).To(BeTrue())
		assertSafeExecEvent(secondRes, "ExecutionSuccess", state.safeWalletAddr)
		Expect(s.network.NextBlock()).To(Succeed())

		ucdaoClient := s.network.GetUCDAOClient()
		safeUcdaoFinal, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: state.safeWalletAccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(safeUcdaoFinal.Balances.AmountOf(utils.BaseDenom)).To(Equal(sdkmath.ZeroInt()))
		Expect(safeUcdaoFinal.Balances.AmountOf(state.liquidDenom)).To(Equal(sdkmath.ZeroInt()))
		assertLiquidRedeemMatchesModuleISLM(state.safeWalletAccAddr, state.liquidDenom, modBefore, liquidBefore)

		safeHaqqAfter, err := s.grpcHandler.GetBalance(state.safeWalletAccAddr, ethiqtypes.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(safeHaqqAfter.Balance.Amount).To(Equal(firstMint.Add(secondMint)))
	})

	It("converts 1500 in one tx from mixed 1000 ISLM + 1000 liquid ucDAO", func() {
		// No approve: the Safe converts its own ucDAO position.
		state := prepareSafeWithUcdaoPositions(holderSafe)
		convertAmount := sdkmath.NewInt(1500).MulRaw(1e18)

		Expect(s.network.NextBlock()).To(Succeed())
		liquidBefore := safeUcdaoLiquidBalance(state.safeWalletAccAddr, state.liquidDenom)
		Expect(liquidBefore).To(Equal(thousandIslmLiquid), "Safe ucDAO liquid before convert should be 1000 ISLM-equivalent")
		modBefore := getLiquidVestingModuleISLMBalance()

		ctxEthiq := s.network.GetContext()
		expectedMint, err := s.network.App.EthiqKeeper.CalculateHaqqCoinsToMint(ctxEthiq, convertAmount)
		Expect(err).NotTo(HaveOccurred())

		execSuccess, execRes := execSafeConvertToHaqq(state, convertAmount, 0)
		Expect(execSuccess).To(BeTrue())
		assertSafeExecEvent(execRes, "ExecutionSuccess", state.safeWalletAddr)
		Expect(s.network.NextBlock()).To(Succeed())

		ucdaoClient := s.network.GetUCDAOClient()
		safeUcdaoFinal, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: state.safeWalletAccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(safeUcdaoFinal.Balances.AmountOf(utils.BaseDenom)).To(Equal(sdkmath.NewInt(500).MulRaw(1e18)))
		Expect(safeUcdaoFinal.Balances.AmountOf(state.liquidDenom)).To(Equal(sdkmath.ZeroInt()))
		assertLiquidRedeemMatchesModuleISLM(state.safeWalletAccAddr, state.liquidDenom, modBefore, liquidBefore)

		safeHaqqAfter, err := s.grpcHandler.GetBalance(state.safeWalletAccAddr, ethiqtypes.BaseDenom)
		Expect(err).NotTo(HaveOccurred())
		Expect(safeHaqqAfter.Balance.Amount).To(Equal(expectedMint))
	})

	It("converts its own position with no approve at all", func() {
		// The counterpart of the spec below: grant creation and consumption are bound to the
		// immediate EVM caller, so a Safe moving its own ucDAO position is authorized by its
		// own threshold. Before the identity-model fix this reverted unless some unrelated
		// EOA -- whoever happened to be tx.origin -- had granted the Safe.
		state := prepareSafeWithUcdaoPositions(holderSafe)

		execSuccess, execRes := execSafeConvertToHaqq(state, sdkmath.NewInt(500).MulRaw(1e18), 0)
		Expect(execSuccess).To(BeTrue(), "a Safe acting on its own position must need no grant")
		assertSafeExecEvent(execRes, "ExecutionSuccess", state.safeWalletAddr)
		Expect(s.network.NextBlock()).To(Succeed())

		// and nothing was granted to, or consumed from, the executing owner along the way
		auth, _ := s.network.App.AuthzKeeper.GetAuthorization(
			s.network.GetContext(),
			state.safeWalletAccAddr,
			safeOwnerOne.AccAddr,
			ucdao.ConvertToHaqqMsgURL,
		)
		Expect(auth).To(BeNil(), "the executing owner must not become the granter")
	})

	It("converts the owner's position when the owner approved enough", func() {
		// Positive control for the three specs below: without it they could pass for the
		// wrong reason -- an empty position, or the origin == sender guard -- instead of on
		// the allowance they mean to exercise.
		state := prepareSafeWithUcdaoPositions(holderOwnerOne)
		convertAmount := sdkmath.NewInt(1500).MulRaw(1e18)
		approveForSafeConvert(state.safeWalletAddr, convertAmount)
		Expect(s.network.NextBlock()).To(Succeed())

		execSuccess, execRes := execSafeConvertToHaqq(state, convertAmount, 0)
		Expect(execSuccess).To(BeTrue(), "the owner's grant must authorize the Safe")
		assertSafeExecEvent(execRes, "ExecutionSuccess", state.safeWalletAddr)
		Expect(s.network.NextBlock()).To(Succeed())

		ucdaoClient := s.network.GetUCDAOClient()
		positionFinal, err := ucdaoClient.AllBalances(context.Background(), &ucdaotypes.QueryAllBalancesRequest{
			Address: state.positionAccAddr.String(),
		})
		Expect(err).NotTo(HaveOccurred())
		// ucDAO spends the liquid position first, then the base one, so 1500 of 1000+1000
		// leaves 500 base behind. Same split as the holderSafe spec above.
		Expect(positionFinal.Balances.AmountOf(utils.BaseDenom)).To(Equal(sdkmath.NewInt(500).MulRaw(1e18)))
		Expect(positionFinal.Balances.AmountOf(state.liquidDenom)).To(Equal(sdkmath.ZeroInt()))

		// the allowance was spent in full, so the grant is gone
		auth, _ := s.network.App.AuthzKeeper.GetAuthorization(
			s.network.GetContext(),
			state.safeWalletAccAddr,
			safeOwnerOne.AccAddr,
			ucdao.ConvertToHaqqMsgURL,
		)
		Expect(auth).To(BeNil(), "a fully spent ConvertToHaqq allowance must be deleted")
	})

	It("fails without approve when the Safe acts on the owner's position", func() {
		state := prepareSafeWithUcdaoPositions(holderOwnerOne)
		execSuccess, execRes := execSafeConvertToHaqq(state, sdkmath.NewInt(500).MulRaw(1e18), 0)
		Expect(execSuccess).To(BeFalse())
		assertSafeExecEvent(execRes, "ExecutionFailure", state.safeWalletAddr)
	})

	It("fails when approve allowance is smaller than convertToHaqq amount", func() {
		// The allowance is the gate only when the Safe acts on someone else's position.
		state := prepareSafeWithUcdaoPositions(holderOwnerOne)
		approveForSafeConvert(state.safeWalletAddr, sdkmath.NewInt(1000).MulRaw(1e18))
		execSuccess, execRes := execSafeConvertToHaqq(state, sdkmath.NewInt(1500).MulRaw(1e18), 0)
		Expect(execSuccess).To(BeFalse())
		assertSafeExecEvent(execRes, "ExecutionFailure", state.safeWalletAddr)
	})

	It("fails after revoke when trying to convert again", func() {
		state := prepareSafeWithUcdaoPositions(holderOwnerOne)
		approveForSafeConvert(state.safeWalletAddr, sdkmath.NewInt(500).MulRaw(1e18))
		revokeForSafeConvert(state.safeWalletAddr)
		execSuccess, execRes := execSafeConvertToHaqq(state, sdkmath.NewInt(500).MulRaw(1e18), 0)
		Expect(execSuccess).To(BeFalse())
		assertSafeExecEvent(execRes, "ExecutionFailure", state.safeWalletAddr)
	})

	It("executes two free mints in one Safe batch from ucDAO with stable Safe balances", func() {
		// No approve: the Safe converts its own ucDAO position.
		state := prepareSafeWithBatchMintBalances(holderSafe)
		Expect(s.network.NextBlock()).To(Succeed())

		before := captureSafeBatchSnapshot(state)
		Expect(before.ucdaoBase).To(Equal(sdkmath.NewInt(300).MulRaw(1e18)))
		Expect(before.ucdaoLiquid).To(Equal(sdkmath.NewInt(300).MulRaw(1e18)))
		Expect(before.bankBase).To(Equal(sdkmath.NewInt(500).MulRaw(1e18)))
		Expect(before.bankLiquid).To(Equal(sdkmath.NewInt(500).MulRaw(1e18)))
		totalUcdaoIslmEquivBefore := before.ucdaoBase.Add(before.ucdaoLiquid)

		firstConvertAmount := sdkmath.NewInt(500).MulRaw(1e18)
		secondConvertAmount := sdkmath.NewInt(100).MulRaw(1e18)
		ctxEthiq := s.network.GetContext()
		firstMint, err := s.network.App.EthiqKeeper.CalculateHaqqCoinsToMint(ctxEthiq, firstConvertAmount)
		Expect(err).NotTo(HaveOccurred())
		secondMint, err := s.network.App.EthiqKeeper.CalculateHaqqCoinsToMint(ctxEthiq, secondConvertAmount)
		Expect(err).NotTo(HaveOccurred())

		batchTxData := buildSafeConvertBatchTxData(state, firstConvertAmount, secondConvertAmount)

		nonceBefore := readSafeNonce(state.safeWalletAddr)
		execOk, execRes := safeExecMultiSendBatch(state.safeWalletAddr, batchTxData, nonceBefore)
		Expect(execOk).To(BeTrue())
		assertSafeExecEvent(execRes, "ExecutionSuccess", state.safeWalletAddr)
		Expect(s.network.NextBlock()).To(Succeed())
		nonceAfter := readSafeNonce(state.safeWalletAddr)
		Expect(nonceAfter).To(Equal(new(big.Int).Add(nonceBefore, big.NewInt(1))))

		after := captureSafeBatchSnapshot(state)
		Expect(after.ucdaoBase).To(Equal(sdkmath.ZeroInt()))
		Expect(after.ucdaoLiquid).To(Equal(sdkmath.ZeroInt()))
		totalUcdaoIslmEquivAfter := after.ucdaoBase.Add(after.ucdaoLiquid)
		Expect(totalUcdaoIslmEquivBefore.Sub(totalUcdaoIslmEquivAfter)).To(Equal(firstConvertAmount.Add(secondConvertAmount)),
			"ucDAO should lose ISLM-equivalent value equal to both batched free mints")

		ucdaoLiquidRedeemed := before.ucdaoLiquid.Sub(after.ucdaoLiquid)
		Expect(before.moduleISLMBank.Sub(after.moduleISLMBank)).To(Equal(ucdaoLiquidRedeemed),
			"liquid vesting module ISLM decrease should match Safe ucDAO liquid redeemed during batched free mint (1:1)")

		Expect(after.bankBase).To(Equal(before.bankBase),
			"Safe bank ISLM should remain unchanged during ucDAO-only batched free mints")
		Expect(after.bankLiquid).To(Equal(before.bankLiquid),
			"Safe bank liquid balance should remain unchanged during ucDAO-only batched free mints")

		Expect(after.haqqBalance.Sub(before.haqqBalance)).To(Equal(firstMint.Add(secondMint)))
		ownerOneSpent := before.ownerOneBank.Sub(after.ownerOneBank)
		Expect(ownerOneSpent.IsNegative()).To(BeFalse())
		Expect(ownerOneSpent.LT(sdkmath.NewInt(2).MulRaw(1e18))).To(BeTrue(),
			"Safe owner bank ISLM spent on batched exec should be under 2 ISLM (gas only)")
	})

	It("executes a Safe batch on the owner's position within the allowance", func() {
		// Positive control for the batch spec below: it pins that the 500 + 100 batch does
		// go through on the owner's position when the allowance covers all 600, so the
		// revert below is attributable to the allowance and not to the configuration.
		state := prepareSafeWithBatchMintBalances(holderOwnerOne)
		approveForSafeConvert(state.safeWalletAddr, sdkmath.NewInt(600).MulRaw(1e18))
		Expect(s.network.NextBlock()).To(Succeed())

		before := captureSafeBatchSnapshot(state)
		batchTxData := buildSafeConvertBatchTxData(state, sdkmath.NewInt(500).MulRaw(1e18), sdkmath.NewInt(100).MulRaw(1e18))

		nonceBefore := readSafeNonce(state.safeWalletAddr)
		execOk, execRes := safeExecMultiSendBatch(state.safeWalletAddr, batchTxData, nonceBefore)
		Expect(execOk).To(BeTrue(), "a batch within the allowance must go through")
		assertSafeExecEvent(execRes, "ExecutionSuccess", state.safeWalletAddr)
		Expect(s.network.NextBlock()).To(Succeed())

		after := captureSafeBatchSnapshot(state)
		Expect(before.ucdaoBase.Add(before.ucdaoLiquid).Sub(after.ucdaoBase.Add(after.ucdaoLiquid))).
			To(Equal(sdkmath.NewInt(600).MulRaw(1e18)), "the batch should spend exactly the approved 600")
	})

	It("reverts whole Safe batch when second convertToHaqq exceeds allowance", func() {
		// The allowance is the gate only when the Safe acts on someone else's position.
		state := prepareSafeWithBatchMintBalances(holderOwnerOne)
		approveForSafeConvert(state.safeWalletAddr, sdkmath.NewInt(500).MulRaw(1e18))
		Expect(s.network.NextBlock()).To(Succeed())

		before := captureSafeBatchSnapshot(state)

		firstConvertAmount := sdkmath.NewInt(500).MulRaw(1e18)
		secondConvertAmount := sdkmath.NewInt(100).MulRaw(1e18)
		batchTxData := buildSafeConvertBatchTxData(state, firstConvertAmount, secondConvertAmount)

		nonceBefore := readSafeNonce(state.safeWalletAddr)
		execOk, execRes := safeExecMultiSendBatch(state.safeWalletAddr, batchTxData, nonceBefore)
		Expect(execOk).To(BeFalse())
		assertSafeExecEvent(execRes, "ExecutionFailure", state.safeWalletAddr)
		Expect(s.network.NextBlock()).To(Succeed())

		after := captureSafeBatchSnapshot(state)
		Expect(after.ucdaoBase).To(Equal(before.ucdaoBase))
		Expect(after.ucdaoLiquid).To(Equal(before.ucdaoLiquid))
		Expect(after.bankBase).To(Equal(before.bankBase),
			"Safe bank ISLM should remain unchanged on failed batch")
		Expect(after.bankLiquid).To(Equal(before.bankLiquid),
			"Safe bank liquid should remain unchanged on failed batch")

		Expect(after.haqqBalance).To(Equal(before.haqqBalance),
			"Safe HAQQ balance should remain unchanged on failed batch")

		Expect(after.moduleISLMBank).To(Equal(before.moduleISLMBank),
			"liquid vesting module ISLM should remain unchanged when batch fails atomically")

		ownerOneSpent := before.ownerOneBank.Sub(after.ownerOneBank)
		Expect(ownerOneSpent.IsNegative()).To(BeFalse())
		Expect(ownerOneSpent.LT(sdkmath.NewInt(2).MulRaw(1e18))).To(BeTrue(),
			"Safe owner bank ISLM spent on failed batched exec should be under 2 ISLM (gas only)")
	})
})
