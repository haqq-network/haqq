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
		islmInitialOwnerFunding = sdkmath.NewInt(1500).MulRaw(1e18)
		islmTransferToSafe      = sdkmath.NewInt(500).MulRaw(1e18)
		islmExpectedSafeFree    = sdkmath.NewInt(1000).MulRaw(1e18)
		islmOwnerTwoBaseline    = sdkmath.NewInt(1000).MulRaw(1e18)
		islmOwnerOneFloor       = sdkmath.NewInt(999).MulRaw(1e18)
		islmOwnerOneCeil        = sdkmath.NewInt(1000).MulRaw(1e18)
		islmVestingTotal        = sdkmath.NewInt(3000).MulRaw(1e18)
		islmLiquidationAmount   = sdkmath.NewInt(500).MulRaw(1e18)
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

			// 7) Execute Safe -> LiquidVesting precompile liquidate(500 ISLM).
			//
			// The precompile requires authz whenever the EVM caller (Safe)
			// differs from tx origin (owner1). Grant it natively here so the
			// tested action remains the Safe EVM execution path, while setup
			// stays minimal for this step.
			liquidateGrant := sdkauthz.NewGenericAuthorization(sdk.MsgTypeURL(&liquidtypes.MsgLiquidate{}))
			Expect(s.network.App.AuthzKeeper.SaveGrant(
				s.network.GetContext(),
				safeWalletAccAddr,
				safeOwnerOne.AccAddr,
				liquidateGrant,
				ptrTime(s.network.GetContext().BlockTime().Add(time.Hour)),
			)).To(Succeed(), "failed to grant Safe authz to liquidate via precompile")

			liquidVestingModuleAddr := authtypes.NewModuleAddress(liquidtypes.ModuleName)
			moduleBaseBefore := s.network.App.BankKeeper.GetBalance(
				s.network.GetContext(), liquidVestingModuleAddr, utils.BaseDenom,
			).Amount
			safeBaseBeforeLiquidate := s.network.App.BankKeeper.GetBalance(
				s.network.GetContext(), safeWalletAccAddr, utils.BaseDenom,
			).Amount
			safeSpendableBeforeLiquidate := s.network.App.BankKeeper.SpendableCoin(
				s.network.GetContext(), safeWalletAccAddr, utils.BaseDenom,
			).Amount
			ownerOneBeforeLiquidate, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())

			liquidateCallData, err := s.precompile.ABI.Pack(
				liquid.LiquidateMethod,
				safeWalletAddr,
				safeWalletAddr,
				islmLiquidationAmount.BigInt(),
			)
			Expect(err).NotTo(HaveOccurred(), "failed to pack liquidate call data")

			_, nonceRes, err := s.factory.CallContractAndCheckLogs(
				safeOwnerOne.Priv,
				evmtypes.EvmTxArgs{To: &safeWalletAddr},
				factory.CallArgs{ContractABI: gnosisSafe.ABI, MethodName: "nonce"},
				testutil.LogCheckArgs{ExpPass: true},
			)
			Expect(err).NotTo(HaveOccurred(), "failed to read Safe nonce before liquidate")
			nonceOut, err := gnosisSafe.ABI.Methods["nonce"].Outputs.Unpack(nonceRes.Ret)
			Expect(err).NotTo(HaveOccurred(), "failed to decode Safe nonce")
			nonce, ok := nonceOut[0].(*big.Int)
			Expect(ok).To(BeTrue(), "Safe nonce output must be *big.Int")

			safeTxGas := big.NewInt(500_000)
			getTxHashArgs := factory.CallArgs{
				ContractABI: gnosisSafe.ABI,
				MethodName:  "getTransactionHash",
				Args: []interface{}{
					s.precompile.Address(),
					big.NewInt(0),
					liquidateCallData,
					uint8(0), // CALL
					safeTxGas,
					big.NewInt(0),
					big.NewInt(0),
					common.Address{},
					common.Address{},
					nonce,
				},
			}
			_, txHashRes, err := s.factory.CallContractAndCheckLogs(
				safeOwnerOne.Priv,
				evmtypes.EvmTxArgs{To: &safeWalletAddr},
				getTxHashArgs,
				testutil.LogCheckArgs{ExpPass: true},
			)
			Expect(err).NotTo(HaveOccurred(), "failed to calculate Safe tx hash")
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
						s.precompile.Address(),
						big.NewInt(0),
						liquidateCallData,
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
			Expect(err).NotTo(HaveOccurred(), "failed to execute Safe liquidate transaction")
			execRes, err := s.factory.GetEvmTransactionResponseFromTxResult(execTxRes)
			Expect(err).NotTo(HaveOccurred(), "failed to decode Safe liquidate tx response")
			execOut, err := gnosisSafe.ABI.Methods["execTransaction"].Outputs.Unpack(execRes.Ret)
			Expect(err).NotTo(HaveOccurred(), "failed to decode Safe execTransaction output")
			Expect(execOut).To(HaveLen(1))
			execSuccess, ok := execOut[0].(bool)
			Expect(ok).To(BeTrue(), "execTransaction output must be bool")
			Expect(execSuccess).To(BeTrue(), "Safe liquidate execTransaction must succeed")

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
			Expect(executionSuccessFound).To(BeTrue(), "Safe must emit ExecutionSuccess for liquidate")
			Expect(s.network.NextBlock()).To(Succeed(), "failed to advance block after Safe liquidate")

			// 8) Post-liquidation invariants.
			ownerOneAfterLiquidate, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
			Expect(err).NotTo(HaveOccurred())
			ownerOneLiquidateSpend := ownerOneBeforeLiquidate.Balance.Amount.Sub(ownerOneAfterLiquidate.Balance.Amount)
			Expect(ownerOneLiquidateSpend.IsNegative()).To(BeFalse(), "owner1 balance must not increase after Safe liquidate")
			Expect(ownerOneLiquidateSpend.LTE(sdkmath.NewInt(1).MulRaw(1e18))).To(BeTrue(),
				"owner1 must spend no more than 1 ISLM on Safe liquidate gas")

			safeBaseAfterLiquidate := s.network.App.BankKeeper.GetBalance(
				s.network.GetContext(), safeWalletAccAddr, utils.BaseDenom,
			).Amount
			safeSpendableAfterLiquidate := s.network.App.BankKeeper.SpendableCoin(
				s.network.GetContext(), safeWalletAccAddr, utils.BaseDenom,
			).Amount
			safeLockedAfterLiquidate := safeBaseAfterLiquidate.Sub(safeSpendableAfterLiquidate)
			moduleBaseAfter := s.network.App.BankKeeper.GetBalance(
				s.network.GetContext(), liquidVestingModuleAddr, utils.BaseDenom,
			).Amount

			liquidDenom := ""
			liquidAmount := sdkmath.ZeroInt()
			for _, coin := range s.network.App.BankKeeper.GetAllBalances(s.network.GetContext(), safeWalletAccAddr) {
				if strings.HasPrefix(coin.Denom, "aLIQUID") {
					liquidDenom = coin.Denom
					liquidAmount = coin.Amount
					break
				}
			}

			GinkgoWriter.Printf(
				"liquidate balances: safeBaseBefore=%s safeBaseAfter=%s spendableBefore=%s spendableAfter=%s lockedAfter=%s moduleBefore=%s moduleAfter=%s liquidDenom=%s liquidAmount=%s\n",
				safeBaseBeforeLiquidate.String(),
				safeBaseAfterLiquidate.String(),
				safeSpendableBeforeLiquidate.String(),
				safeSpendableAfterLiquidate.String(),
				safeLockedAfterLiquidate.String(),
				moduleBaseBefore.String(),
				moduleBaseAfter.String(),
				liquidDenom,
				liquidAmount.String(),
			)

			Expect(safeSpendableAfterLiquidate).To(Equal(islmExpectedSafeFree),
				"Safe spendable ISLM must remain exactly 1000 after liquidating locked vesting")
			Expect(safeLockedAfterLiquidate).To(Equal(islmVestingTotal.Sub(islmLiquidationAmount)),
				"Safe locked vesting ISLM must be 2500 after liquidating 500")
			Expect(safeBaseBeforeLiquidate.Sub(safeBaseAfterLiquidate)).To(Equal(islmLiquidationAmount),
				"Safe base ISLM bank balance must decrease exactly by the liquidated amount")
			Expect(moduleBaseAfter.Sub(moduleBaseBefore)).To(Equal(islmLiquidationAmount),
				"liquid vesting module ISLM balance must increase exactly by the liquidated amount")
			Expect(safeBaseBeforeLiquidate.Add(moduleBaseBefore)).To(Equal(safeBaseAfterLiquidate.Add(moduleBaseAfter)),
				"Safe ISLM decrease must be fully accounted for by the liquid vesting module increase")
			Expect(liquidDenom).NotTo(BeEmpty(), "Safe must receive a liquid vesting denom")
			Expect(liquidAmount).To(Equal(islmLiquidationAmount),
				"Safe must receive exactly 500 ISLM-equivalent liquid tokens")
			Expect(s.network.App.BankKeeper.GetSupply(s.network.GetContext(), liquidDenom).Amount).To(Equal(islmLiquidationAmount),
				"liquid token supply must equal the minted amount; no extra liquid tokens may appear")

			Expect(safeSpendableBeforeLiquidate).To(Equal(safeSpendableAfterLiquidate),
				"liquidating locked vesting must not change Safe spendable ISLM")
		})
	})
})
