package liquid_test

import (
	"math/big"
	"strings"
	"testing"
	"time"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	sdkauthz "github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/liquid"
	"github.com/haqq-network/haqq/precompiles/testutil"
	safecontracts "github.com/haqq-network/haqq/precompiles/testutil/contracts/safe"
	haqqtestutil "github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	testutiltx "github.com/haqq-network/haqq/testutil/tx"
	haqqtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	liquidtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// TestLiquidIntegrationTestSuite runs the Ginkgo integration suite for the
// liquid vesting precompile. It coexists with the testify suite from
// setup_test.go - both are independent entry points.
func TestLiquidIntegrationTestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Liquid Vesting Precompile Integration Suite")
}

type liquidSafeSuite struct {
	network     *network.UnitTestNetwork
	factory     factory.TxFactory
	grpcHandler grpc.Handler
	precompile  *liquid.Precompile
	keyring     keyring.Keyring
}

func (s *liquidSafeSuite) setupNetwork(kr keyring.Keyring) {
	customGenesis := network.CustomGenesisState{}
	lvGenesis := liquidtypes.DefaultGenesisState()
	lvGenesis.Params.MinimumLiquidationAmount = sdkmath.NewInt(1_000_000)
	customGenesis[liquidtypes.ModuleName] = lvGenesis

	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(kr.GetAllAccAddrs()...),
		network.WithCustomGenesis(customGenesis),
	)
	gh := grpc.NewIntegrationHandler(nw)
	pc, err := liquid.NewPrecompile(nw.App.LiquidVestingKeeper, nw.App.AuthzKeeper)
	Expect(err).NotTo(HaveOccurred(), "failed to create liquid vesting precompile")

	s.network = nw
	s.factory = factory.New(nw, gh)
	s.grpcHandler = gh
	s.precompile = pc
	s.keyring = kr
}

var smartContractVestingTotalIntegration = sdk.NewCoins(
	sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(3000).MulRaw(1e18)),
)

func ptrTime(t time.Time) *time.Time {
	return &t
}

// createClawbackVestingAccountForSmartContract converts an existing smart
// contract account into a ClawbackVestingAccount for integration tests.
//
// IMPORTANT - test-only helper.
//
// In production this conversion is impossible: x/vesting MsgConvertIntoVestingAccount
// rejects contract accounts via IsContractAccount. The helper deliberately
// bypasses that guard by mutating the auth account directly. It MUST NOT be
// treated as a model for any on-chain flow.
func (s *liquidSafeSuite) createClawbackVestingAccountForSmartContract(ctx sdk.Context, addr sdk.AccAddress) {
	funder := sdk.AccAddress(liquidtypes.ModuleName)
	periodCoin := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1000).MulRaw(1e18)))
	lockupPeriods := sdkvesting.Periods{
		{Length: 100000, Amount: periodCoin},
		{Length: 100000, Amount: periodCoin},
		{Length: 100000, Amount: periodCoin},
	}
	vestingPeriods := sdkvesting.Periods{
		{Length: 0, Amount: smartContractVestingTotalIntegration},
	}

	existing := s.network.App.AccountKeeper.GetAccount(ctx, addr)
	Expect(existing).NotTo(BeNil(), "smart-contract account must exist before being wrapped into a vesting account")

	baseAccount := authtypes.NewBaseAccount(
		existing.GetAddress(),
		existing.GetPubKey(),
		existing.GetAccountNumber(),
		existing.GetSequence(),
	)

	var codeHashPtr *common.Hash
	if ethAcc, ok := existing.(*haqqtypes.EthAccount); ok {
		h := ethAcc.GetCodeHash()
		codeHashPtr = &h
	}

	startTime := ctx.BlockTime().Add(-10 * time.Second)
	clawbackAccount := vestingtypes.NewClawbackVestingAccount(
		baseAccount, funder, smartContractVestingTotalIntegration, startTime, lockupPeriods, vestingPeriods, codeHashPtr,
	)

	Expect(haqqtestutil.FundAccount(ctx, s.network.App.BankKeeper, addr, smartContractVestingTotalIntegration)).
		To(Succeed(), "failed to fund Safe vesting balance")
	s.network.App.AccountKeeper.SetAccount(ctx, clawbackAccount)
}

var _ = Describe("Liquid Vesting precompile with Gnosis Safe (smart contract wallet)", Ordered, func() {
	var (
		s                *liquidSafeSuite
		safeOwnerOne     keyring.Key
		safeOwnerTwo     keyring.Key
		gnosisSafe       evmtypes.CompiledContract
		gnosisSafeAddr   common.Address
		proxyFactory     evmtypes.CompiledContract
		proxyFactoryAddr common.Address
	)

	// Common balance constants used across the suite. All amounts are in
	// aISLM (1 ISLM = 1e18 aISLM) to match the on-chain unit.
	var (
		islmInitialOwnerFunding   = sdkmath.NewInt(1500).MulRaw(1e18)
		islmTransferToSafe        = sdkmath.NewInt(500).MulRaw(1e18)
		islmExpectedSafeFree      = sdkmath.NewInt(1000).MulRaw(1e18)
		islmOwnerTwoBaseline      = sdkmath.NewInt(1000).MulRaw(1e18)
		islmOwnerOneFloor         = sdkmath.NewInt(999).MulRaw(1e18)
		islmOwnerOneCeil          = sdkmath.NewInt(1000).MulRaw(1e18)
		islmVestingTotal          = sdkmath.NewInt(3000).MulRaw(1e18)
		islmFirstLiquidateAmount  = sdkmath.NewInt(500).MulRaw(1e18)
		islmSecondLiquidateAmount = sdkmath.NewInt(2000).MulRaw(1e18)
		islmThirdLiquidateAmount  = sdkmath.NewInt(500).MulRaw(1e18)
		islmFourthLiquidateAmount = sdkmath.NewInt(100).MulRaw(1e18)
		islmFirstRedeemAmount     = sdkmath.NewInt(250).MulRaw(1e18)
		islmSecondRedeemAmount    = sdkmath.NewInt(2000).MulRaw(1e18)
		islmThirdRedeemAmount     = sdkmath.NewInt(100).MulRaw(1e18)
		// 1 ISLM gas budget per Safe execTransaction (covers proxy + signature
		// verification + precompile call). Two execTransactions in this suite,
		// so owner1 must spend at most 2 ISLM in total across both.
		islmGasPerExecTransaction = sdkmath.NewInt(1).MulRaw(1e18)
	)

	BeforeEach(func() {
		kr := keyring.New(1)
		s = new(liquidSafeSuite)
		s.setupNetwork(kr)

		var err error
		gnosisSafe, err = safecontracts.LoadGnosisSafeContract()
		Expect(err).NotTo(HaveOccurred(), "failed to load GnosisSafe singleton contract")

		proxyFactory, err = safecontracts.LoadGnosisSafeProxyFactoryContract()
		Expect(err).NotTo(HaveOccurred(), "failed to load GnosisSafeProxyFactory contract")

		oneAddr, onePriv := testutiltx.NewAddrKey()
		safeOwnerOne = keyring.Key{
			Addr:    oneAddr,
			AccAddr: sdk.AccAddress(oneAddr.Bytes()),
			Priv:    onePriv,
		}
		twoAddr, twoPriv := testutiltx.NewAddrKey()
		safeOwnerTwo = keyring.Key{
			Addr:    twoAddr,
			AccAddr: sdk.AccAddress(twoAddr.Bytes()),
			Priv:    twoPriv,
		}

		Expect(s.network.FundAccountWithBaseDenom(safeOwnerOne.AccAddr, islmInitialOwnerFunding)).
			To(Succeed(), "failed to fund first Safe owner with 1500 ISLM")
		Expect(s.network.FundAccountWithBaseDenom(safeOwnerTwo.AccAddr, islmInitialOwnerFunding)).
			To(Succeed(), "failed to fund second Safe owner with 1500 ISLM")

		deployer := s.keyring.GetKey(0)
		gnosisSafeAddr, err = s.factory.DeployContract(
			deployer.Priv,
			evmtypes.EvmTxArgs{},
			factory.ContractDeploymentData{Contract: gnosisSafe},
		)
		Expect(err).NotTo(HaveOccurred(), "failed to deploy GnosisSafe singleton")
		Expect(gnosisSafeAddr).NotTo(Equal(common.Address{}), "GnosisSafe singleton address must be non-zero")

		proxyFactoryAddr, err = s.factory.DeployContract(
			deployer.Priv,
			evmtypes.EvmTxArgs{},
			factory.ContractDeploymentData{Contract: proxyFactory},
		)
		Expect(err).NotTo(HaveOccurred(), "failed to deploy GnosisSafeProxyFactory")
		Expect(proxyFactoryAddr).NotTo(Equal(common.Address{}), "GnosisSafeProxyFactory address must be non-zero")

		Expect(s.network.NextBlock()).To(Succeed(), "failed to advance block after Safe infrastructure deployment")
	})

	Describe("Safe deployment, funding and native conversion into a ClawbackVestingAccount", func() {
		It("should fund Safe with 2x500 ISLM and wrap it into a vesting account with 1000 free + 3000 vesting balances", func() {
			// 0) Owners must start with exactly 1500 ISLM each.
			ownerOneBefore, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			Expect(ownerOneBefore.Balance.Amount).To(Equal(islmInitialOwnerFunding),
				"owner1 must start with exactly 1500 ISLM before Safe deployment")
			ownerTwoBefore, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			Expect(ownerTwoBefore.Balance.Amount).To(Equal(islmInitialOwnerFunding),
				"owner2 must start with exactly 1500 ISLM before Safe deployment")

			// 1) Deploy a Safe wallet via GnosisSafeProxyFactory.createProxy.
			//    Threshold = 1 keeps signing simple for downstream test cases.
			safeSetupData, err := gnosisSafe.ABI.Pack(
				"setup",
				[]common.Address{safeOwnerOne.Addr, safeOwnerTwo.Addr},
				big.NewInt(1),
				common.Address{}, []byte{},
				common.Address{}, common.Address{},
				big.NewInt(0), common.Address{},
			)
			Expect(err).NotTo(HaveOccurred(), "failed to pack GnosisSafe setup calldata")

			createProxyRes, err := s.factory.ExecuteContractCall(
				safeOwnerOne.Priv,
				evmtypes.EvmTxArgs{To: &proxyFactoryAddr},
				factory.CallArgs{
					ContractABI: proxyFactory.ABI,
					MethodName:  "createProxy",
					Args:        []interface{}{gnosisSafeAddr, safeSetupData},
				},
			)
			Expect(err).NotTo(HaveOccurred(), "failed to broadcast createProxy tx")

			ethRes, err := s.factory.GetEvmTransactionResponseFromTxResult(createProxyRes)
			Expect(err).NotTo(HaveOccurred(), "failed to decode createProxy tx response")

			// Find the ProxyCreation log emitted by the factory and decode it
			// to recover the freshly-created Safe wallet address.
			proxyCreationEvent := proxyFactory.ABI.Events["ProxyCreation"]
			var proxyCreationLog *evmtypes.Log
			for i := range ethRes.Logs {
				l := ethRes.Logs[i]
				if len(l.Topics) == 0 || l.Topics[0] != proxyCreationEvent.ID.String() {
					continue
				}
				if common.HexToAddress(l.Address) != proxyFactoryAddr {
					continue
				}
				proxyCreationLog = l
				break
			}
			Expect(proxyCreationLog).NotTo(BeNil(), "ProxyCreation event must be emitted by the factory")

			eventInputs, err := proxyCreationEvent.Inputs.Unpack(proxyCreationLog.Data)
			Expect(err).NotTo(HaveOccurred(), "failed to decode ProxyCreation event payload")
			Expect(eventInputs).To(HaveLen(2), "unexpected ProxyCreation payload shape")
			safeWalletAddr, ok := eventInputs[0].(common.Address)
			Expect(ok).To(BeTrue(), "first ProxyCreation field must be the proxy address")
			Expect(safeWalletAddr).NotTo(Equal(common.Address{}), "Safe wallet address must be non-zero")
			Expect(eventInputs[1]).To(Equal(gnosisSafeAddr), "singleton in ProxyCreation must match the deployed GnosisSafe")

			Expect(s.network.NextBlock()).To(Succeed(), "failed to advance block after Safe wallet creation")

			// 2) Sanity-check Safe configuration via on-chain getters.
			_, thrRes, err := s.factory.CallContractAndCheckLogs(
				safeOwnerOne.Priv,
				evmtypes.EvmTxArgs{To: &safeWalletAddr},
				factory.CallArgs{ContractABI: gnosisSafe.ABI, MethodName: "getThreshold"},
				testutil.LogCheckArgs{ExpPass: true},
			)
			Expect(err).NotTo(HaveOccurred(), "getThreshold call must succeed")
			thrOut, err := gnosisSafe.ABI.Methods["getThreshold"].Outputs.Unpack(thrRes.Ret)
			Expect(err).NotTo(HaveOccurred())
			Expect(thrOut).To(HaveLen(1))
			Expect(thrOut[0]).To(Equal(big.NewInt(1)), "Safe threshold must be 1")

			_, ownersRes, err := s.factory.CallContractAndCheckLogs(
				safeOwnerOne.Priv,
				evmtypes.EvmTxArgs{To: &safeWalletAddr},
				factory.CallArgs{ContractABI: gnosisSafe.ABI, MethodName: "getOwners"},
				testutil.LogCheckArgs{ExpPass: true},
			)
			Expect(err).NotTo(HaveOccurred(), "getOwners call must succeed")
			ownersOut, err := gnosisSafe.ABI.Methods["getOwners"].Outputs.Unpack(ownersRes.Ret)
			Expect(err).NotTo(HaveOccurred())
			Expect(ownersOut).To(HaveLen(1))
			owners, ok := ownersOut[0].([]common.Address)
			Expect(ok).To(BeTrue())
			Expect(owners).To(HaveLen(2), "Safe must have exactly two owners")
			Expect(owners).To(ContainElements(safeOwnerOne.Addr, safeOwnerTwo.Addr),
				"Safe owners must match the configured pair")

			// 3) Each owner sends 500 ISLM to the Safe via direct bank transfer.
			//    We bypass the EVM here on purpose - the focus of this test is
			//    the vesting wrap, not the funding mechanics.
			safeWalletAccAddr := sdk.AccAddress(safeWalletAddr.Bytes())
			coins500 := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, islmTransferToSafe))
			ctx := s.network.GetContext()
			Expect(s.network.App.BankKeeper.SendCoins(ctx, safeOwnerOne.AccAddr, safeWalletAccAddr, coins500)).
				To(Succeed(), "owner1 must successfully transfer 500 ISLM to Safe")
			Expect(s.network.App.BankKeeper.SendCoins(ctx, safeOwnerTwo.AccAddr, safeWalletAccAddr, coins500)).
				To(Succeed(), "owner2 must successfully transfer 500 ISLM to Safe")
			Expect(s.network.NextBlock()).To(Succeed(), "failed to advance block after owner transfers")

			// 4) Pre-vesting balance assertions.
			safeBank, err := s.grpcHandler.GetBalance(safeWalletAccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			Expect(safeBank.Balance.Amount).To(Equal(islmExpectedSafeFree),
				"Safe must hold exactly 1000 ISLM after the two 500 ISLM transfers")

			ownerTwoAfterTransfer, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			Expect(ownerTwoAfterTransfer.Balance.Amount).To(Equal(islmOwnerTwoBaseline),
				"owner2 must have exactly 1000 ISLM (no gas spent - bank.SendCoins is keeper-level)")

			ownerOneAfterTransfer, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			ownerOneSpent := ownerOneAfterTransfer.Balance.Amount
			Expect(ownerOneSpent.GT(islmOwnerOneFloor)).To(BeTrue(),
				"owner1 must have > 999 ISLM (only gas spent on createProxy)")
			Expect(ownerOneSpent.LT(islmOwnerOneCeil)).To(BeTrue(),
				"owner1 must have < 1000 ISLM (some aISLM gas was spent on createProxy)")

			// 5) Wrap Safe into a ClawbackVestingAccount via the test-only
			//    native helper. In production this is impossible: the
			//    x/vesting MsgConvertIntoVestingAccount keeper rejects
			//    contract accounts via IsContractAccount. The helper
			//    deliberately bypasses that guard so we can build the
			//    "Safe is also a vesting account" fixture required by
			//    the liquid vesting precompile flow.
			s.createClawbackVestingAccountForSmartContract(s.network.GetContext(), safeWalletAccAddr)
			Expect(s.network.NextBlock()).To(Succeed(), "failed to advance block after vesting wrap")

			// 6) Post-vesting balance assertions.

			// 6a) Bank balance must be 1000 free + 3000 vesting = 4000 ISLM.
			expectedSafeBankAfterVesting := islmExpectedSafeFree.Add(islmVestingTotal)
			safeBankAfter, err := s.grpcHandler.GetBalance(safeWalletAccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			Expect(safeBankAfter.Balance.Amount).To(Equal(expectedSafeBankAfterVesting),
				"Safe bank balance must equal 1000 free + 3000 vesting = 4000 ISLM after wrap")

			// 6b) Spendable balance = bank - locked = 4000 - 3000 = 1000 ISLM.
			spendable := s.network.App.BankKeeper.SpendableCoin(s.network.GetContext(), safeWalletAccAddr, utils.BaseDenom)
			Expect(spendable.Amount).To(Equal(islmExpectedSafeFree),
				"Safe must have exactly 1000 spendable ISLM after vesting wrap")

			// 6c) Auth account is now a ClawbackVestingAccount with the
			//     expected schedule.
			acc := s.network.App.AccountKeeper.GetAccount(s.network.GetContext(), safeWalletAccAddr)
			Expect(acc).NotTo(BeNil(), "Safe auth account must exist after vesting wrap")
			cba, ok := acc.(*vestingtypes.ClawbackVestingAccount)
			Expect(ok).To(BeTrue(), "Safe auth account must be a ClawbackVestingAccount")
			Expect(cba.OriginalVesting.AmountOf(utils.BaseDenom)).To(Equal(islmVestingTotal),
				"OriginalVesting must equal the configured 3000 ISLM total")
			Expect(cba.LockupPeriods).To(HaveLen(3), "must have exactly 3 lockup periods")
			Expect(cba.LockedCoins(s.network.GetContext().BlockTime()).AmountOf(utils.BaseDenom)).
				To(Equal(islmVestingTotal),
					"all 3000 ISLM must still be locked (lockup started 10s ago, first period ends in ~100000s)")

			// 6d) The vesting wrap must not affect owner balances.
			ownerOneFinal, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			Expect(ownerOneFinal.Balance.Amount).To(Equal(ownerOneAfterTransfer.Balance.Amount),
				"owner1 balance must be unchanged by the native vesting wrap")
			ownerTwoFinal, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			Expect(ownerTwoFinal.Balance.Amount).To(Equal(islmOwnerTwoBaseline),
				"owner2 balance must be unchanged by the native vesting wrap")

			// 7) Authorize Safe to call MsgLiquidate and MsgRedeem on behalf of
			//    owner1.
			//
			// The precompile requires authz whenever the EVM caller (Safe)
			// differs from tx origin (owner1). Grant it natively here so the
			// tested action remains the Safe EVM execution path, while setup
			// stays minimal for this step.
			grantExpiry := ptrTime(s.network.GetContext().BlockTime().Add(time.Hour))
			Expect(s.network.App.AuthzKeeper.SaveGrant(
				s.network.GetContext(),
				safeWalletAccAddr,
				safeOwnerOne.AccAddr,
				sdkauthz.NewGenericAuthorization(sdk.MsgTypeURL(&liquidtypes.MsgLiquidate{})),
				grantExpiry,
			)).To(Succeed(), "failed to grant Safe authz to liquidate via precompile")
			Expect(s.network.App.AuthzKeeper.SaveGrant(
				s.network.GetContext(),
				safeWalletAccAddr,
				safeOwnerOne.AccAddr,
				sdkauthz.NewGenericAuthorization(sdk.MsgTypeURL(&liquidtypes.MsgRedeem{})),
				grantExpiry,
			)).To(Succeed(), "failed to grant Safe authz to redeem via precompile")

			liquidVestingModuleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)

			// execViaSafe drives the full Gnosis Safe execTransaction flow:
			//   read nonce -> compute tx hash -> approved-hash signature for
			//   owner1 -> execTransaction -> verify ExecutionSuccess.
			// It returns the EVM tx response so callers can introspect logs.
			//
			// Threshold = 1 lets us use the "approved hash" signature scheme
			// ({r=owner1, s=0, v=1}) instead of an off-chain ECDSA signature,
			// keeping the test free of low-level signing details.
			execViaSafe := func(targetAddr common.Address, callData []byte, opLabel string) *evmtypes.MsgEthereumTxResponse {
				GinkgoHelper()

				_, nonceRes, err := s.factory.CallContractAndCheckLogs(
					safeOwnerOne.Priv,
					evmtypes.EvmTxArgs{To: &safeWalletAddr},
					factory.CallArgs{ContractABI: gnosisSafe.ABI, MethodName: "nonce"},
					testutil.LogCheckArgs{ExpPass: true},
				)
				Expect(err).NotTo(HaveOccurred(), "failed to read Safe nonce before "+opLabel)
				nonceOut, err := gnosisSafe.ABI.Methods["nonce"].Outputs.Unpack(nonceRes.Ret)
				Expect(err).NotTo(HaveOccurred(), "failed to decode Safe nonce")
				nonce, ok := nonceOut[0].(*big.Int)
				Expect(ok).To(BeTrue(), "Safe nonce output must be *big.Int")

				safeTxGas := big.NewInt(500_000)
				_, txHashRes, err := s.factory.CallContractAndCheckLogs(
					safeOwnerOne.Priv,
					evmtypes.EvmTxArgs{To: &safeWalletAddr},
					factory.CallArgs{
						ContractABI: gnosisSafe.ABI,
						MethodName:  "getTransactionHash",
						Args: []interface{}{
							targetAddr,
							big.NewInt(0),
							callData,
							uint8(0), // CALL
							safeTxGas,
							big.NewInt(0),
							big.NewInt(0),
							common.Address{},
							common.Address{},
							nonce,
						},
					},
					testutil.LogCheckArgs{ExpPass: true},
				)
				Expect(err).NotTo(HaveOccurred(), "failed to calculate Safe tx hash for "+opLabel)
				txHashOut, err := gnosisSafe.ABI.Methods["getTransactionHash"].Outputs.Unpack(txHashRes.Ret)
				Expect(err).NotTo(HaveOccurred(), "failed to decode Safe tx hash")
				Expect(txHashOut).To(HaveLen(1))
				txHash, ok := txHashOut[0].([32]byte)
				Expect(ok).To(BeTrue(), "Safe tx hash output must be [32]byte")
				Expect(txHash).NotTo(Equal([32]byte{}), "Safe tx hash must be non-zero")

				// Safe signature format is {r}{s}{v}. For threshold=1 we use the
				// approved-hash signature form: v=1 and r=owner1 address.
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
							targetAddr,
							big.NewInt(0),
							callData,
							uint8(0), // CALL
							safeTxGas,
							big.NewInt(0),
							big.NewInt(0),
							common.Address{},
							common.Address{},
							signature,
						},
					},
				)
				Expect(err).NotTo(HaveOccurred(), "failed to execute Safe "+opLabel+" transaction")
				execRes, err := s.factory.GetEvmTransactionResponseFromTxResult(execTxRes)
				Expect(err).NotTo(HaveOccurred(), "failed to decode Safe "+opLabel+" tx response")
				execOut, err := gnosisSafe.ABI.Methods["execTransaction"].Outputs.Unpack(execRes.Ret)
				Expect(err).NotTo(HaveOccurred(), "failed to decode Safe execTransaction output")
				Expect(execOut).To(HaveLen(1))
				execSuccess, ok := execOut[0].(bool)
				Expect(ok).To(BeTrue(), "execTransaction output must be bool")
				Expect(execSuccess).To(BeTrue(), "Safe "+opLabel+" execTransaction must succeed")

				executionSuccessEvent := gnosisSafe.ABI.Events["ExecutionSuccess"]
				executionSuccessFound := false
				for i := range execRes.Logs {
					l := execRes.Logs[i]
					if len(l.Topics) == 0 || l.Topics[0] != executionSuccessEvent.ID.String() {
						continue
					}
					if common.HexToAddress(l.Address) != safeWalletAddr {
						continue
					}
					executionSuccessFound = true
					break
				}
				Expect(executionSuccessFound).To(BeTrue(), "Safe must emit ExecutionSuccess for "+opLabel)
				Expect(s.network.NextBlock()).To(Succeed(), "failed to advance block after Safe "+opLabel)
				return execRes
			}

			// liquidateViaSafe and redeemViaSafe pack the precompile call data
			// and dispatch through the shared Safe execTransaction helper. Both
			// take an explicit `to` so we can exercise the from == to (self)
			// path AND the from != to (cross-account) path against the same
			// Safe-as-vesting-account fixture.
			liquidateViaSafe := func(amount sdkmath.Int, to common.Address) *evmtypes.MsgEthereumTxResponse {
				GinkgoHelper()
				callData, err := s.precompile.ABI.Pack(
					liquid.LiquidateMethod,
					safeWalletAddr,
					to,
					amount.BigInt(),
				)
				Expect(err).NotTo(HaveOccurred(), "failed to pack liquidate call data")
				return execViaSafe(s.precompile.Address(), callData, "liquidate")
			}
			redeemViaSafe := func(denom string, amount sdkmath.Int, to common.Address) *evmtypes.MsgEthereumTxResponse {
				GinkgoHelper()
				callData, err := s.precompile.ABI.Pack(
					liquid.RedeemMethod,
					safeWalletAddr,
					to,
					denom,
					amount.BigInt(),
				)
				Expect(err).NotTo(HaveOccurred(), "failed to pack redeem call data")
				return execViaSafe(s.precompile.Address(), callData, "redeem")
			}

			// allLiquidBalances returns the bank balance of the Safe filtered to
			// liquid-vesting denoms ("aLIQUID*"), keyed by denom for easy
			// per-denom assertions.
			allLiquidBalances := func() map[string]sdkmath.Int {
				out := map[string]sdkmath.Int{}
				for _, coin := range s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), safeWalletAccAddr) {
					if strings.HasPrefix(coin.Denom, "aLIQUID") {
						out[coin.Denom] = coin.Amount
					}
				}
				return out
			}

			// safeAndModuleSnapshot captures the bank baselines we assert on
			// across each precompile call: Safe's base denom, Safe's spendable
			// base denom, the module's base denom, and the Safe's balance of
			// the given liquid denom (use empty string to skip).
			type bankSnapshot struct {
				safeBase      sdkmath.Int
				safeSpendable sdkmath.Int
				moduleBase    sdkmath.Int
				safeLiquid    sdkmath.Int
			}
			safeAndModuleSnapshot := func(liquidDenom string) bankSnapshot {
				ctx := s.network.GetContext()
				snap := bankSnapshot{
					safeBase:      s.network.App.BankKeeper.GetBalance(ctx, safeWalletAccAddr, utils.BaseDenom).Amount,
					safeSpendable: s.network.App.BankKeeper.SpendableCoin(ctx, safeWalletAccAddr, utils.BaseDenom).Amount,
					moduleBase:    s.network.App.BankKeeper.GetBalance(ctx, liquidVestingModuleAddr, utils.BaseDenom).Amount,
				}
				if liquidDenom != "" {
					snap.safeLiquid = s.network.App.BankKeeper.GetBalance(ctx, safeWalletAccAddr, liquidDenom).Amount
				}
				return snap
			}

			// 8) FIRST liquidation: 500 ISLM -> 500 aLIQUID0.
			moduleBaseBeforeFirst := s.network.App.BankKeeper.GetBalance(
				s.network.GetContext(), liquidVestingModuleAddr, utils.BaseDenom,
			).Amount
			safeBaseBeforeFirst := s.network.App.BankKeeper.GetBalance(
				s.network.GetContext(), safeWalletAccAddr, utils.BaseDenom,
			).Amount
			safeSpendableBeforeFirst := s.network.App.BankKeeper.SpendableCoin(
				s.network.GetContext(), safeWalletAccAddr, utils.BaseDenom,
			).Amount
			ownerOneBeforeFirst, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())

			resLiquidate := liquidateViaSafe(islmFirstLiquidateAmount, safeWalletAddr)
			Expect(resLiquidate).ToNot(BeNil())
			Expect(resLiquidate.Failed()).To(BeFalse(), "should not have failed")

			// 8a) Post-first-liquidation invariants.
			//
			// Expected state changes:
			//   - Safe locked vesting: 3000 -> 2500
			//   - Safe spendable: 1000 (unchanged)
			//   - Safe bank: 4000 -> 3500 (debited 500)
			//   - Module bank: 0 -> 500 (credited 500)
			//   - Safe receives 500 of a brand-new aLIQUID0 denom
			ownerOneAfterFirst, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			ownerOneFirstSpend := ownerOneBeforeFirst.Balance.Amount.Sub(ownerOneAfterFirst.Balance.Amount)
			Expect(ownerOneFirstSpend.IsNegative()).To(BeFalse(),
				"owner1 balance must not increase after the first Safe liquidate")
			Expect(ownerOneFirstSpend.LTE(islmGasPerExecTransaction)).To(BeTrue(),
				"owner1 must spend no more than 1 ISLM on the first Safe liquidate gas")

			safeBaseAfterFirst := s.network.App.BankKeeper.GetBalance(
				s.network.GetContext(), safeWalletAccAddr, utils.BaseDenom,
			).Amount
			safeSpendableAfterFirst := s.network.App.BankKeeper.SpendableCoin(
				s.network.GetContext(), safeWalletAccAddr, utils.BaseDenom,
			).Amount
			safeLockedAfterFirst := safeBaseAfterFirst.Sub(safeSpendableAfterFirst)
			moduleBaseAfterFirst := s.network.App.BankKeeper.GetBalance(
				s.network.GetContext(), liquidVestingModuleAddr, utils.BaseDenom,
			).Amount

			Expect(safeSpendableAfterFirst).To(Equal(islmExpectedSafeFree),
				"Safe spendable ISLM must remain exactly 1000 after liquidating locked vesting")
			Expect(safeSpendableBeforeFirst).To(Equal(safeSpendableAfterFirst),
				"liquidating locked vesting must not change Safe spendable ISLM")
			Expect(safeLockedAfterFirst).To(Equal(islmVestingTotal.Sub(islmFirstLiquidateAmount)),
				"Safe locked vesting ISLM must be 2500 after liquidating 500")
			Expect(safeBaseBeforeFirst.Sub(safeBaseAfterFirst)).To(Equal(islmFirstLiquidateAmount),
				"Safe base ISLM bank balance must decrease exactly by the first liquidated amount")
			Expect(moduleBaseAfterFirst.Sub(moduleBaseBeforeFirst)).To(Equal(islmFirstLiquidateAmount),
				"liquid vesting module ISLM balance must increase exactly by the first liquidated amount")
			Expect(safeBaseBeforeFirst.Add(moduleBaseBeforeFirst)).To(Equal(safeBaseAfterFirst.Add(moduleBaseAfterFirst)),
				"Safe ISLM decrease must be fully accounted for by the liquid vesting module increase (first call)")

			liquidAfterFirst := allLiquidBalances()
			Expect(liquidAfterFirst).To(HaveLen(1),
				"Safe must hold exactly one liquid denom after the first liquidation")
			Expect(liquidAfterFirst).To(HaveKeyWithValue("aLIQUID0", islmFirstLiquidateAmount),
				"first liquidation must mint exactly 500 of aLIQUID0 into Safe")
			Expect(s.network.App.BankKeeper.GetSupply(s.network.GetContext(), "aLIQUID0").Amount).
				To(Equal(islmFirstLiquidateAmount),
					"aLIQUID0 supply must equal the first mint; no extra liquid tokens may appear")

			// 9) SECOND liquidation: 2000 ISLM -> 2000 aLIQUID1.
			//
			// Each Liquidate call increments the keeper's denom counter and
			// produces a NEW liquid denom; aLIQUID0 from the previous step
			// stays untouched in Safe's balance.
			moduleBaseBeforeSecond := moduleBaseAfterFirst
			safeBaseBeforeSecond := safeBaseAfterFirst
			safeSpendableBeforeSecond := safeSpendableAfterFirst
			ownerOneBeforeSecond, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())

			resLiquidate = liquidateViaSafe(islmSecondLiquidateAmount, safeWalletAddr)
			Expect(resLiquidate).ToNot(BeNil())
			Expect(resLiquidate.Failed()).To(BeFalse(), "should not have failed")
			// 9a) Post-second-liquidation invariants.
			//
			// Expected state changes (cumulative):
			//   - Safe locked vesting: 2500 -> 500
			//   - Safe spendable: 1000 (unchanged)
			//   - Safe bank: 3500 -> 1500 (debited 2000 more)
			//   - Module bank: 500 -> 2500 (credited 2000 more)
			//   - Safe gains a SECOND liquid denom aLIQUID1 with 2000 minted
			//   - aLIQUID0 stays at exactly 500 (untouched by the second call)
			ownerOneAfterSecond, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			ownerOneSecondSpend := ownerOneBeforeSecond.Balance.Amount.Sub(ownerOneAfterSecond.Balance.Amount)
			Expect(ownerOneSecondSpend.IsNegative()).To(BeFalse(),
				"owner1 balance must not increase after the second Safe liquidate")
			Expect(ownerOneSecondSpend.LTE(islmGasPerExecTransaction)).To(BeTrue(),
				"owner1 must spend no more than 1 ISLM on the second Safe liquidate gas")

			safeBaseAfterSecond := s.network.App.BankKeeper.GetBalance(
				s.network.GetContext(), safeWalletAccAddr, utils.BaseDenom,
			).Amount
			safeSpendableAfterSecond := s.network.App.BankKeeper.SpendableCoin(
				s.network.GetContext(), safeWalletAccAddr, utils.BaseDenom,
			).Amount
			safeLockedAfterSecond := safeBaseAfterSecond.Sub(safeSpendableAfterSecond)
			moduleBaseAfterSecond := s.network.App.BankKeeper.GetBalance(
				s.network.GetContext(), liquidVestingModuleAddr, utils.BaseDenom,
			).Amount

			expectedFinalLocked := islmVestingTotal.Sub(islmFirstLiquidateAmount).Sub(islmSecondLiquidateAmount)
			expectedFinalModule := islmFirstLiquidateAmount.Add(islmSecondLiquidateAmount)

			Expect(safeSpendableAfterSecond).To(Equal(islmExpectedSafeFree),
				"Safe spendable ISLM must still be exactly 1000 after the second liquidation")
			Expect(safeSpendableBeforeSecond).To(Equal(safeSpendableAfterSecond),
				"second liquidation must not change Safe spendable ISLM")
			Expect(safeLockedAfterSecond).To(Equal(expectedFinalLocked),
				"Safe locked vesting ISLM must be 500 after liquidating 500 + 2000 of 3000")
			Expect(safeBaseBeforeSecond.Sub(safeBaseAfterSecond)).To(Equal(islmSecondLiquidateAmount),
				"Safe base ISLM bank balance must decrease exactly by the second liquidated amount")
			Expect(moduleBaseAfterSecond.Sub(moduleBaseBeforeSecond)).To(Equal(islmSecondLiquidateAmount),
				"module ISLM balance must increase exactly by the second liquidated amount")
			Expect(moduleBaseAfterSecond).To(Equal(expectedFinalModule),
				"module ISLM balance must equal the sum of all liquidations (500 + 2000 = 2500)")
			Expect(safeBaseBeforeSecond.Add(moduleBaseBeforeSecond)).To(Equal(safeBaseAfterSecond.Add(moduleBaseAfterSecond)),
				"Safe ISLM decrease must be fully accounted for by the module increase (second call)")

			// 9b) Liquid token bookkeeping after both liquidations.
			liquidAfterSecond := allLiquidBalances()
			Expect(liquidAfterSecond).To(HaveLen(2),
				"Safe must hold exactly two distinct liquid denoms after two liquidations")
			Expect(liquidAfterSecond).To(HaveKeyWithValue("aLIQUID0", islmFirstLiquidateAmount),
				"aLIQUID0 amount in Safe must remain 500 (untouched by the second liquidation)")
			Expect(liquidAfterSecond).To(HaveKeyWithValue("aLIQUID1", islmSecondLiquidateAmount),
				"second liquidation must mint exactly 2000 of aLIQUID1 into Safe")
			Expect(s.network.App.BankKeeper.GetSupply(s.network.GetContext(), "aLIQUID0").Amount).
				To(Equal(islmFirstLiquidateAmount),
					"aLIQUID0 total supply must remain 500 after the second liquidation")
			Expect(s.network.App.BankKeeper.GetSupply(s.network.GetContext(), "aLIQUID1").Amount).
				To(Equal(islmSecondLiquidateAmount),
					"aLIQUID1 total supply must equal the second mint; no extra liquid tokens may appear")

			// 10) FIRST redeem: 250 of aLIQUID0 back into base ISLM.
			//
			// Redeem flow inside the keeper:
			//   - Safe (redeemFrom) -> module: 250 aLIQUID0
			//   - module burns      :         250 aLIQUID0   (supply -= 250)
			//   - module -> Safe (redeemTo):  250 aISLM
			//
			// Because aLIQUID0's lockup window has not elapsed yet (test runs
			// in milliseconds, the first lockup period is ~100000s long), the
			// keeper applies the corresponding diffPeriods as a NEW vesting
			// schedule on the Safe (redeemTo). That means the 250 aISLM that
			// just landed in the Safe are immediately re-locked under that
			// schedule, so Safe's spendable ISLM must NOT change.
			snapBeforeRedeem1 := safeAndModuleSnapshot("aLIQUID0")
			ownerOneBeforeRedeem1, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			supplyLiquid0BeforeRedeem1 := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), "aLIQUID0").Amount

			resRedeem := redeemViaSafe("aLIQUID0", islmFirstRedeemAmount, safeWalletAddr)
			Expect(resRedeem).ToNot(BeNil())
			Expect(resRedeem.Failed()).To(BeFalse(), "should not have failed")

			snapAfterRedeem1 := safeAndModuleSnapshot("aLIQUID0")
			ownerOneAfterRedeem1, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			ownerOneRedeem1Spend := ownerOneBeforeRedeem1.Balance.Amount.Sub(ownerOneAfterRedeem1.Balance.Amount)
			Expect(ownerOneRedeem1Spend.IsNegative()).To(BeFalse(),
				"owner1 balance must not increase after the first Safe redeem")
			Expect(ownerOneRedeem1Spend.LTE(islmGasPerExecTransaction)).To(BeTrue(),
				"owner1 must spend no more than 1 ISLM on the first Safe redeem gas")

			Expect(snapAfterRedeem1.safeBase.Sub(snapBeforeRedeem1.safeBase)).To(Equal(islmFirstRedeemAmount),
				"Safe base ISLM bank balance must increase exactly by the first redeemed amount (250)")
			Expect(snapBeforeRedeem1.safeLiquid.Sub(snapAfterRedeem1.safeLiquid)).To(Equal(islmFirstRedeemAmount),
				"Safe aLIQUID0 balance must decrease exactly by the first redeemed amount (250)")
			Expect(snapBeforeRedeem1.moduleBase.Sub(snapAfterRedeem1.moduleBase)).To(Equal(islmFirstRedeemAmount),
				"module ISLM balance must decrease exactly by the first redeemed amount (250)")
			Expect(snapBeforeRedeem1.safeBase.Add(snapBeforeRedeem1.moduleBase)).
				To(Equal(snapAfterRedeem1.safeBase.Add(snapAfterRedeem1.moduleBase)),
					"module ISLM decrease must be fully accounted for by the Safe ISLM increase (first redeem)")
			Expect(supplyLiquid0BeforeRedeem1.Sub(s.network.App.BankKeeper.GetSupply(s.network.GetContext(), "aLIQUID0").Amount)).
				To(Equal(islmFirstRedeemAmount),
					"aLIQUID0 supply must decrease by exactly the first redeemed amount via burn")
			Expect(snapAfterRedeem1.safeLiquid).To(Equal(islmFirstLiquidateAmount.Sub(islmFirstRedeemAmount)),
				"Safe must keep exactly 500 - 250 = 250 of aLIQUID0 after the first redeem")
			Expect(snapAfterRedeem1.safeSpendable).To(Equal(snapBeforeRedeem1.safeSpendable),
				"first redeem must not change Safe spendable ISLM (the redeemed amount lands re-locked)")

			// 11) SECOND redeem: 2000 of aLIQUID1 (full supply) back into base ISLM.
			//
			// This fully redeems aLIQUID1: decreasedPeriods.TotalAmount() == 0,
			// so the keeper deletes the aLIQUID1 denom from x/liquidvesting and
			// disables the corresponding ERC20 conversion. After this step the
			// Safe must hold zero aLIQUID1.
			snapBeforeRedeem2 := safeAndModuleSnapshot("aLIQUID1")
			ownerOneBeforeRedeem2, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			supplyLiquid1BeforeRedeem2 := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), "aLIQUID1").Amount
			Expect(supplyLiquid1BeforeRedeem2).To(Equal(islmSecondRedeemAmount),
				"sanity: aLIQUID1 total supply must equal the amount we are about to redeem")

			resRedeem = redeemViaSafe("aLIQUID1", islmSecondRedeemAmount, safeWalletAddr)
			Expect(resRedeem).ToNot(BeNil())
			Expect(resRedeem.Failed()).To(BeFalse(), "should not have failed")

			snapAfterRedeem2 := safeAndModuleSnapshot("aLIQUID1")
			ownerOneAfterRedeem2, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			ownerOneRedeem2Spend := ownerOneBeforeRedeem2.Balance.Amount.Sub(ownerOneAfterRedeem2.Balance.Amount)
			Expect(ownerOneRedeem2Spend.IsNegative()).To(BeFalse(),
				"owner1 balance must not increase after the second Safe redeem")
			Expect(ownerOneRedeem2Spend.LTE(islmGasPerExecTransaction)).To(BeTrue(),
				"owner1 must spend no more than 1 ISLM on the second Safe redeem gas")

			Expect(snapAfterRedeem2.safeBase.Sub(snapBeforeRedeem2.safeBase)).To(Equal(islmSecondRedeemAmount),
				"Safe base ISLM bank balance must increase exactly by the second redeemed amount (2000)")
			Expect(snapBeforeRedeem2.safeLiquid.Sub(snapAfterRedeem2.safeLiquid)).To(Equal(islmSecondRedeemAmount),
				"Safe aLIQUID1 balance must decrease exactly by the second redeemed amount (2000)")
			Expect(snapBeforeRedeem2.moduleBase.Sub(snapAfterRedeem2.moduleBase)).To(Equal(islmSecondRedeemAmount),
				"module ISLM balance must decrease exactly by the second redeemed amount (2000)")
			Expect(snapBeforeRedeem2.safeBase.Add(snapBeforeRedeem2.moduleBase)).
				To(Equal(snapAfterRedeem2.safeBase.Add(snapAfterRedeem2.moduleBase)),
					"module ISLM decrease must be fully accounted for by the Safe ISLM increase (second redeem)")
			Expect(snapAfterRedeem2.safeLiquid.IsZero()).To(BeTrue(),
				"Safe must hold zero aLIQUID1 after redeeming the entire supply")
			Expect(s.network.App.BankKeeper.GetSupply(s.network.GetContext(), "aLIQUID1").Amount.IsZero()).
				To(BeTrue(), "aLIQUID1 total supply must be zero after a full redeem")
			Expect(snapAfterRedeem2.safeSpendable).To(Equal(snapBeforeRedeem2.safeSpendable),
				"second redeem must not change Safe spendable ISLM (the redeemed amount lands re-locked)")

			// 11a) Cumulative liquid-token bookkeeping after both redeems.
			//   - aLIQUID0: 500 minted - 250 redeemed = 250 remaining in Safe
			//   - aLIQUID1: 2000 minted - 2000 redeemed = 0 (denom deleted)
			liquidAfterRedeems := allLiquidBalances()
			Expect(liquidAfterRedeems).To(HaveKeyWithValue("aLIQUID0", islmFirstLiquidateAmount.Sub(islmFirstRedeemAmount)),
				"aLIQUID0 in Safe must equal the unredeemed remainder after the first partial redeem")
			Expect(liquidAfterRedeems).NotTo(HaveKey("aLIQUID1"),
				"aLIQUID1 must be fully drained from Safe after the second (full) redeem")

			// 12) THIRD liquidation: another 500 ISLM -> brand-new aLIQUID2.
			//
			// The keeper's denom counter is monotonic; even though aLIQUID1 was
			// just deleted, the counter is at 2 by now and the new denom is
			// aLIQUID2 (not a recycled aLIQUID1).
			snapBeforeLiquidate3 := safeAndModuleSnapshot("")
			ownerOneBeforeLiquidate3, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())

			resLiquidate = liquidateViaSafe(islmThirdLiquidateAmount, safeWalletAddr)
			Expect(resLiquidate).ToNot(BeNil())
			Expect(resLiquidate.Failed()).To(BeFalse(), "should not have failed")

			snapAfterLiquidate3 := safeAndModuleSnapshot("")
			ownerOneAfterLiquidate3, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			ownerOneLiquidate3Spend := ownerOneBeforeLiquidate3.Balance.Amount.Sub(ownerOneAfterLiquidate3.Balance.Amount)
			Expect(ownerOneLiquidate3Spend.IsNegative()).To(BeFalse(),
				"owner1 balance must not increase after the third Safe liquidate")
			Expect(ownerOneLiquidate3Spend.LTE(islmGasPerExecTransaction)).To(BeTrue(),
				"owner1 must spend no more than 1 ISLM on the third Safe liquidate gas")

			Expect(snapBeforeLiquidate3.safeBase.Sub(snapAfterLiquidate3.safeBase)).To(Equal(islmThirdLiquidateAmount),
				"Safe base ISLM bank balance must decrease exactly by the third liquidated amount (500)")
			Expect(snapAfterLiquidate3.moduleBase.Sub(snapBeforeLiquidate3.moduleBase)).To(Equal(islmThirdLiquidateAmount),
				"module ISLM balance must increase exactly by the third liquidated amount (500)")
			Expect(snapBeforeLiquidate3.safeBase.Add(snapBeforeLiquidate3.moduleBase)).
				To(Equal(snapAfterLiquidate3.safeBase.Add(snapAfterLiquidate3.moduleBase)),
					"Safe ISLM decrease must be fully accounted for by the module increase (third liquidation)")
			Expect(snapAfterLiquidate3.safeSpendable).To(Equal(snapBeforeLiquidate3.safeSpendable),
				"third liquidation must not change Safe spendable ISLM")

			liquidAfterLiquidate3 := allLiquidBalances()
			Expect(liquidAfterLiquidate3).To(HaveKeyWithValue("aLIQUID2", islmThirdLiquidateAmount),
				"third liquidation must mint exactly 500 of aLIQUID2 into Safe")
			Expect(liquidAfterLiquidate3).To(HaveKeyWithValue("aLIQUID0", islmFirstLiquidateAmount.Sub(islmFirstRedeemAmount)),
				"aLIQUID0 in Safe must remain at 250 (untouched by the third liquidation)")
			Expect(liquidAfterLiquidate3).NotTo(HaveKey("aLIQUID1"),
				"aLIQUID1 must remain absent (counter is monotonic; new denom is aLIQUID2)")
			Expect(s.network.App.BankKeeper.GetSupply(s.network.GetContext(), "aLIQUID2").Amount).
				To(Equal(islmThirdLiquidateAmount),
					"aLIQUID2 total supply must equal the third mint; no extra liquid tokens may appear")

			// 13) THIRD redeem with redeemFrom != redeemTo: 100 aLIQUID0
			//     redeemed from Safe, principal credited to owner2.
			//
			// This explicitly exercises the cross-account redeem branch of
			// the precompile mirror (where redeemFrom and redeemTo are two
			// distinct addresses, both passed to mirrorBankBaseDeltasIntoStateDB):
			//   - Safe (redeemFrom): only aLIQUID0 moves; base aISLM is unchanged,
			//     so the mirror snapshot resolves to a zero delta and is a no-op.
			//   - owner2 (redeemTo): receives 100 aISLM from the module account,
			//     so the mirror must emit a single Add(owner2, 100) entry.
			//
			// This is the regression case for the dedup logic: previous
			// versions of the mirror called twice on the same address would
			// double-credit when from == to; here we additionally pin down the
			// "two distinct addresses, distinct mirror entries" path so that a
			// future regression in either direction is caught.
			snapBeforeRedeem3 := safeAndModuleSnapshot("aLIQUID0")
			ownerTwoBeforeRedeem3, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			ownerOneBeforeRedeem3, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			supplyLiquid0BeforeRedeem3 := s.network.App.BankKeeper.GetSupply(s.network.GetContext(), "aLIQUID0").Amount

			resRedeem = redeemViaSafe("aLIQUID0", islmThirdRedeemAmount, safeOwnerTwo.Addr)
			Expect(resRedeem).ToNot(BeNil())
			Expect(resRedeem.Failed()).To(BeFalse(), "should not have failed")

			snapAfterRedeem3 := safeAndModuleSnapshot("aLIQUID0")
			ownerTwoAfterRedeem3, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			ownerOneAfterRedeem3, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())

			ownerOneRedeem3Spend := ownerOneBeforeRedeem3.Balance.Amount.Sub(ownerOneAfterRedeem3.Balance.Amount)
			Expect(ownerOneRedeem3Spend.IsNegative()).To(BeFalse(),
				"owner1 balance must not increase after the cross-account Safe redeem")
			Expect(ownerOneRedeem3Spend.LTE(islmGasPerExecTransaction)).To(BeTrue(),
				"owner1 must spend no more than 1 ISLM on the cross-account Safe redeem gas")

			Expect(snapAfterRedeem3.safeBase).To(Equal(snapBeforeRedeem3.safeBase),
				"Safe base ISLM must NOT change on cross-account redeem (Safe is redeemFrom only)")
			Expect(snapBeforeRedeem3.safeLiquid.Sub(snapAfterRedeem3.safeLiquid)).To(Equal(islmThirdRedeemAmount),
				"Safe aLIQUID0 balance must decrease exactly by the cross-account redeemed amount (100)")
			Expect(snapBeforeRedeem3.moduleBase.Sub(snapAfterRedeem3.moduleBase)).To(Equal(islmThirdRedeemAmount),
				"module ISLM balance must decrease exactly by the cross-account redeemed amount (100)")
			Expect(ownerTwoAfterRedeem3.Balance.Amount.Sub(ownerTwoBeforeRedeem3.Balance.Amount)).
				To(Equal(islmThirdRedeemAmount),
					"owner2 base ISLM must increase exactly by the cross-account redeemed amount (100)")
			Expect(supplyLiquid0BeforeRedeem3.Sub(s.network.App.BankKeeper.GetSupply(s.network.GetContext(), "aLIQUID0").Amount)).
				To(Equal(islmThirdRedeemAmount),
					"aLIQUID0 supply must decrease by exactly the cross-account redeemed amount via burn")
			Expect(snapBeforeRedeem3.moduleBase.Add(ownerTwoBeforeRedeem3.Balance.Amount)).
				To(Equal(snapAfterRedeem3.moduleBase.Add(ownerTwoAfterRedeem3.Balance.Amount)),
					"module ISLM decrease must be fully accounted for by owner2 ISLM increase (cross-account redeem)")

			// 14) FOURTH liquidate with liquidateFrom != liquidateTo: 100 aISLM
			//     of Safe's locked vesting -> owner2 receives the freshly minted
			//     liquid denom (aLIQUID3).
			//
			// Mirror invariant: the only base-denom mover here is liquidateFrom
			// (Safe), debited 100 aISLM via SendCoinsFromAccountToModule. The
			// liquidateTo side (owner2) receives ONLY aLIQUID3, which is NOT
			// the EVM gas denom, so even though we sample its base balance the
			// mirror must produce a zero delta for owner2.
			snapBeforeLiquidate4 := safeAndModuleSnapshot("")
			ownerTwoBeforeLiquidate4, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			ownerOneBeforeLiquidate4, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())

			resLiquidate = liquidateViaSafe(islmFourthLiquidateAmount, safeOwnerTwo.Addr)
			Expect(resLiquidate).ToNot(BeNil())
			Expect(resLiquidate.Failed()).To(BeFalse(), "should not have failed")

			snapAfterLiquidate4 := safeAndModuleSnapshot("")
			ownerTwoAfterLiquidate4, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			ownerOneAfterLiquidate4, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())

			ownerOneLiquidate4Spend := ownerOneBeforeLiquidate4.Balance.Amount.Sub(ownerOneAfterLiquidate4.Balance.Amount)
			Expect(ownerOneLiquidate4Spend.IsNegative()).To(BeFalse(),
				"owner1 balance must not increase after the cross-account Safe liquidate")
			Expect(ownerOneLiquidate4Spend.LTE(islmGasPerExecTransaction)).To(BeTrue(),
				"owner1 must spend no more than 1 ISLM on the cross-account Safe liquidate gas")

			Expect(snapBeforeLiquidate4.safeBase.Sub(snapAfterLiquidate4.safeBase)).
				To(Equal(islmFourthLiquidateAmount),
					"Safe base ISLM must decrease exactly by the cross-account liquidated amount (100)")
			Expect(snapAfterLiquidate4.moduleBase.Sub(snapBeforeLiquidate4.moduleBase)).
				To(Equal(islmFourthLiquidateAmount),
					"module ISLM must increase exactly by the cross-account liquidated amount (100)")
			Expect(snapBeforeLiquidate4.safeBase.Add(snapBeforeLiquidate4.moduleBase)).
				To(Equal(snapAfterLiquidate4.safeBase.Add(snapAfterLiquidate4.moduleBase)),
					"Safe ISLM decrease must be fully accounted for by the module increase (cross-account liquidate)")
			Expect(ownerTwoAfterLiquidate4.Balance.Amount).To(Equal(ownerTwoBeforeLiquidate4.Balance.Amount),
				"owner2 base ISLM must NOT change on cross-account liquidate (owner2 only receives aLIQUID3)")

			ownerTwoAccBalances := s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), safeOwnerTwo.AccAddr)
			Expect(ownerTwoAccBalances.AmountOf("aLIQUID3")).To(Equal(islmFourthLiquidateAmount),
				"owner2 must receive exactly 100 of the freshly minted aLIQUID3 (counter is monotonic; previously deleted aLIQUID1 is NOT recycled)")
			Expect(s.network.App.BankKeeper.GetSupply(s.network.GetContext(), "aLIQUID3").Amount).
				To(Equal(islmFourthLiquidateAmount),
					"aLIQUID3 total supply must equal the fourth mint; no extra liquid tokens may appear")
			liquidAfterLiquidate4 := allLiquidBalances()
			Expect(liquidAfterLiquidate4).NotTo(HaveKey("aLIQUID3"),
				"Safe must NOT receive aLIQUID3 - it was minted to owner2 (cross-account liquidate)")
		})
	})
})
