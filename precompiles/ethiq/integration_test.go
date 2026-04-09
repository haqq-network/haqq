package ethiq_test

import (
	"fmt"
	"math/big"
	"testing"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/ethiq"
	"github.com/haqq-network/haqq/precompiles/testutil"
	safecontracts "github.com/haqq-network/haqq/precompiles/testutil/contracts/safe"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	testutils "github.com/haqq-network/haqq/testutil/integration/haqq/utils"
	testutiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func TestPrecompileIntegrationTestSuite(t *testing.T) {
	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ethiq Precompile Integration Suite")
}

// General variables used for integration tests
var (
	// differentAddr is an address generated for testing purposes that e.g. raises the different origin error
	// differentAddr, diffKey = testutiltx.NewAddrKey()

	// gasPrice is the gas price used for the transactions
	gasPrice = sdkmath.NewInt(1e9)
	// callArgs  are the default arguments for calling the smart contract
	//
	// NOTE: this has to be populated in a BeforeEach block because the contractAddr would otherwise be a nil address.
	callArgs, approveCallArgs factory.CallArgs

	// defaultLogCheck instantiates a log check arguments struct with the precompile ABI events populated.
	defaultLogCheck testutil.LogCheckArgs
	// passCheck defines the arguments to check if the precompile returns no error
	passCheck testutil.LogCheckArgs
	// outOfGasCheck defines the arguments to check if the precompile returns out of gas error
	outOfGasCheck testutil.LogCheckArgs
	// txArgs are the EVM transaction arguments to use in the transactions
	txArgs evmtypes.EvmTxArgs
	// islmCoin defines the 1 ISLM coin
	islmCoin = sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1e18))
)

var _ = Describe("Calling ethiq precompile from EOA", func() {
	s := new(PrecompileTestSuite)

	BeforeEach(func() {
		s.SetupTest()

		// set the default call & check arguments
		callArgs = factory.CallArgs{ContractABI: s.precompile.ABI}
		approveCallArgs = factory.CallArgs{ContractABI: s.precompile.ABI}
		defaultLogCheck = testutil.LogCheckArgs{ABIEvents: s.precompile.Events}
		passCheck = defaultLogCheck.WithExpPass(true)
		outOfGasCheck = defaultLogCheck.WithErrContains(vm.ErrOutOfGas.Error())

		// reset tx args each test to avoid keeping custom
		// values of previous tests (e.g. gasLimit)
		precompileAddr := s.precompile.Address()
		txArgs = evmtypes.EvmTxArgs{To: &precompileAddr}
	})

	// =====================================
	// 				TRANSACTIONS
	// =====================================
	Describe("Execute MintHaqq transaction", func() {
		BeforeEach(func() {
			// set the default call arguments
			callArgs.MethodName = ethiq.MintHaqq
		})

		Describe("when the precompile is not enabled in the EVM params", func() {
			It("should succeed but not perform burn/mint", func() {
				sender := s.keyring.GetKey(0)

				// disable the precompile
				resParams, err := s.grpcHandler.GetEvmParams()
				Expect(err).To(BeNil())

				var activePrecompiles []string
				for _, precompile := range resParams.Params.ActiveStaticPrecompiles {
					if precompile != s.precompile.Address().String() {
						activePrecompiles = append(activePrecompiles, precompile)
					}
				}
				resParams.Params.ActiveStaticPrecompiles = activePrecompiles

				err = testutils.UpdateEvmParams(testutils.UpdateParamsInput{
					Tf:      s.factory,
					Network: s.network,
					Pk:      sender.Priv,
					Params:  resParams.Params,
				})
				Expect(err).To(BeNil(), "error while setting params")

				senderBalanceBeforeRes, err := s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil(), "error while retrieving sender's balance")

				txArgs.GasPrice = gasPrice.BigInt()
				txArgs.GasLimit = 30000
				callArgs.Args = []interface{}{
					sender.Addr,
					sender.Addr,
					islmCoin.Amount.BigInt(),
				}

				// Contract should not be called but the transaction should be successful
				// This is the expected behavior in Ethereum where there is a contract call
				// to a non existing contract
				expectedCheck := defaultLogCheck.
					WithExpEvents([]string{}...).
					WithExpPass(true)

				res, _, err := s.factory.CallContractAndCheckLogs(
					sender.Priv,
					txArgs,
					callArgs,
					expectedCheck,
				)
				Expect(err).To(BeNil(), "error while calling the precompile")
				Expect(s.network.NextBlock()).To(BeNil(), "error on NextBlock")

				fees := gasPrice.Mul(sdkmath.NewInt(res.GasUsed))
				expFinalBalanceAmt := senderBalanceBeforeRes.Balance.Amount.Sub(fees)

				// sender's balance should remain unchanged
				senderBalanceAfterRes, err := s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil(), "error while retrieving sender's balance")
				Expect(senderBalanceAfterRes.Balance.Amount).To(Equal(expFinalBalanceAmt), "sender balance is incorrect")
			})
		})

		Describe("Revert transaction", func() {
			It("should run out of gas if the gas limit is too low", func() {
				sender := s.keyring.GetKey(0)

				txArgs.GasPrice = gasPrice.BigInt()
				txArgs.GasLimit = 30000
				callArgs.Args = []interface{}{
					sender.Addr,
					sender.Addr,
					islmCoin.Amount.BigInt(),
				}

				senderBalanceBeforeRes, err := s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil(), "error while retrieving sender's balance")
				Expect(senderBalanceBeforeRes.Balance.Amount).To(Equal(network.PrefundedAccountInitialBalance), "sender balance is incorrect")

				res, _, err := s.factory.CallContractAndCheckLogs(
					sender.Priv,
					txArgs,
					callArgs,
					outOfGasCheck,
				)
				Expect(err).To(BeNil(), "error while calling the precompile")
				Expect(s.network.NextBlock()).To(BeNil(), "error on NextBlock")

				fees := gasPrice.Mul(sdkmath.NewInt(res.GasUsed))
				expFinalBalanceAmt := senderBalanceBeforeRes.Balance.Amount.Sub(fees)

				// sender's balance should remain unchanged
				senderBalanceAfterRes, err := s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil(), "error while retrieving sender's balance")
				Expect(senderBalanceAfterRes.Balance.Amount).To(Equal(expFinalBalanceAmt), "sender balance is incorrect")
			})
		})

		Describe("Execute approve transaction", func() {
			var granter, grantee keyring.Key

			BeforeEach(func() {
				granter = s.keyring.GetKey(0)
				grantee = s.keyring.GetKey(1)

				approveCallArgs.MethodName = authorization.ApproveMethod
			})

			It("should return error if the given method is not supported on the precompile", func() {
				approveCallArgs.Args = []interface{}{
					grantee.Addr,
					abi.MaxUint256,
					[]string{"unknownMethod"},
				}

				logCheckArgs := defaultLogCheck.WithErrContains(cmn.ErrInvalidMsgType, "ethiq", "unknownMethod")

				_, _, err := s.factory.CallContractAndCheckLogs(
					granter.Priv,
					txArgs,
					approveCallArgs,
					logCheckArgs,
				)
				Expect(err).To(BeNil(), "error while calling the contract and checking logs")
			})

			It("should approve the mintHaqq method with the max uint256 value", func() {
				s.SetupApproval(granter.Priv, grantee.Addr, abi.MaxUint256, []string{ethiq.MintHaqqMsgURL})
				s.ExpectAuthorization(ethiq.MintHaqqMsgURL, grantee.Addr, granter.Addr, nil, 0)
			})

			It("should approve the mintHaqq method with the limited value", func() {
				amt := big.NewInt(1e18)
				s.SetupApproval(granter.Priv, grantee.Addr, amt, []string{ethiq.MintHaqqMsgURL})
				expCoin := sdk.NewCoin(utils.BaseDenom, sdkmath.NewIntFromBigInt(amt))
				s.ExpectAuthorization(ethiq.MintHaqqMsgURL, grantee.Addr, granter.Addr, &expCoin, 0)
			})

			It("should refund leftover gas", func() {
				resBal, err := s.grpcHandler.GetBalance(granter.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil(), "error while getting balance")
				balancePre := resBal.Balance
				gasPrice := big.NewInt(1e9)

				// Call the precompile with a lot of gas
				approveCallArgs.Args = []interface{}{
					s.precompile.Address(),
					big.NewInt(1e18),
					[]string{ethiq.MintHaqqMsgURL},
				}
				txArgs.GasPrice = gasPrice

				approvalCheck := passCheck.WithExpEvents(authorization.EventTypeApproval)

				res, _, err := s.factory.CallContractAndCheckLogs(
					granter.Priv,
					txArgs, approveCallArgs,
					approvalCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)
				Expect(s.network.NextBlock()).To(BeNil())

				resBal, err = s.grpcHandler.GetBalance(granter.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil(), "error while getting balance")
				balancePost := resBal.Balance
				difference := balancePre.Sub(*balancePost)

				// NOTE: the expected difference is the gas price multiplied by the gas used, because the rest should be refunded
				expDifference := gasPrice.Int64() * res.GasUsed
				Expect(difference.Amount.Int64()).To(Equal(expDifference), "expected different total transaction cost")
			})
		})

		Describe("to burn/mint", func() {
			Context("as the token owner", func() {
				It("should burn/mint without need for authorization", func() {
					sender := s.keyring.GetKey(0)

					// get initial sender's balances
					senderIslmBalanceBeforeRes, err := s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
					Expect(err).To(BeNil(), "error while retrieving sender's balance of aISLM")
					Expect(senderIslmBalanceBeforeRes.Balance.Amount).To(Equal(network.PrefundedAccountInitialBalance), "sender balance of aISLM is incorrect")
					senderHaqqBalanceBeforeRes, err := s.grpcHandler.GetBalance(sender.AccAddr, ethiqtypes.BaseDenom)
					Expect(err).To(BeNil(), "error while retrieving sender's balance of aHAQQ")
					Expect(senderHaqqBalanceBeforeRes.Balance.IsZero()).To(BeTrue(), "sender balance of aHAQQ is incorrect")

					// prepare and execute tx
					txArgs.GasPrice = gasPrice.BigInt()
					txArgs.GasLimit = 300000
					callArgs.Args = []interface{}{
						sender.Addr,
						sender.Addr,
						islmCoin.Amount.BigInt(),
					}
					logCheckArgs := passCheck.WithExpEvents(ethiq.EventTypeMintHaqq)

					res, _, err := s.factory.CallContractAndCheckLogs(
						sender.Priv,
						txArgs,
						callArgs,
						logCheckArgs,
					)
					Expect(err).To(BeNil(), "error while calling the precompile")
					Expect(s.network.NextBlock()).To(BeNil(), "error on NextBlock")

					fees := gasPrice.Mul(sdkmath.NewInt(res.GasUsed))
					expFinalIslmBalanceAmt := senderIslmBalanceBeforeRes.Balance.Amount.Sub(fees).Sub(islmCoin.Amount)

					// sender's balances should change
					senderIslmBalanceAfterRes, err := s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
					Expect(err).To(BeNil(), "error while retrieving sender's balance of aISLM")
					Expect(senderIslmBalanceAfterRes.Balance.Amount).To(Equal(expFinalIslmBalanceAmt), "sender balance of aISLM after tx is incorrect")
					senderHaqqBalanceAfterRes, err := s.grpcHandler.GetBalance(sender.AccAddr, ethiqtypes.BaseDenom)
					Expect(err).To(BeNil(), "error while retrieving sender's balance of aHAQQ")
					Expect(senderHaqqBalanceAfterRes.Balance.IsZero()).To(BeFalse(), "sender balance of aHAQQ after tx is incorrect")
				})

				It("should not burn/mint if the account has no sufficient balance", func() {
					newAddr, newAddrPriv := testutiltx.NewAccAddressAndKey()
					err := testutils.FundAccountWithBaseDenom(s.factory, s.network, s.keyring.GetKey(0), newAddr, sdkmath.NewInt(1e17))
					Expect(err).To(BeNil(), "error while sending coins")
					Expect(s.network.NextBlock()).To(BeNil())

					// prepare and execute tx
					txArgs.GasPrice = gasPrice.BigInt()
					txArgs.GasLimit = 300000
					callArgs.Args = []interface{}{
						common.BytesToAddress(newAddr),
						common.BytesToAddress(newAddr),
						islmCoin.Amount.BigInt(),
					}
					logCheckArgs := defaultLogCheck.WithErrContains("insufficient funds")

					res, _, err := s.factory.CallContractAndCheckLogs(
						newAddrPriv,
						txArgs,
						callArgs,
						logCheckArgs,
					)
					Expect(err).To(BeNil(), "error while calling the precompile")
					Expect(s.network.NextBlock()).To(BeNil(), "error on NextBlock")

					fees := gasPrice.Mul(sdkmath.NewInt(res.GasUsed))
					expFinalIslmBalanceAmt := sdkmath.NewInt(1e17).Sub(fees)

					senderIslmBalanceAfterRes, err := s.grpcHandler.GetBalance(newAddr, utils.BaseDenom)
					Expect(err).To(BeNil(), "error while retrieving sender's balance of aISLM")
					Expect(senderIslmBalanceAfterRes.Balance.Amount).To(Equal(expFinalIslmBalanceAmt), "sender balance of aISLM after tx is incorrect")
					senderHaqqBalanceAfterRes, err := s.grpcHandler.GetBalance(newAddr, ethiqtypes.BaseDenom)
					Expect(err).To(BeNil(), "error while retrieving sender's balance of aHAQQ")
					Expect(senderHaqqBalanceAfterRes.Balance.IsZero()).To(BeTrue(), "sender balance of aHAQQ after tx is incorrect")
				})
			})

			Context("on behalf of another account", func() {
				It("should not burn/mint if sender address is not the origin", func() {
					sender := s.keyring.GetKey(0)
					differentAddr := testutiltx.GenerateAddress()

					// get initial sender's balances
					senderIslmBalanceBeforeRes, err := s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
					Expect(err).To(BeNil(), "error while retrieving sender's balance of aISLM")
					Expect(senderIslmBalanceBeforeRes.Balance.Amount).To(Equal(network.PrefundedAccountInitialBalance), "sender balance of aISLM is incorrect")
					senderHaqqBalanceBeforeRes, err := s.grpcHandler.GetBalance(sender.AccAddr, ethiqtypes.BaseDenom)
					Expect(err).To(BeNil(), "error while retrieving sender's balance of aHAQQ")
					Expect(senderHaqqBalanceBeforeRes.Balance.IsZero()).To(BeTrue(), "sender balance of aHAQQ is incorrect")

					// prepare and execute tx
					txArgs.GasPrice = gasPrice.BigInt()
					txArgs.GasLimit = 300000
					callArgs.Args = []interface{}{
						differentAddr,
						sender.Addr,
						islmCoin.Amount.BigInt(),
					}
					logCheckArgs := defaultLogCheck.WithErrContains(
						fmt.Sprintf(ethiq.ErrDifferentOriginFromSender, sender.Addr, differentAddr),
					)

					res, _, err := s.factory.CallContractAndCheckLogs(
						sender.Priv,
						txArgs,
						callArgs,
						logCheckArgs,
					)
					Expect(err).To(BeNil(), "error while calling the precompile")
					Expect(s.network.NextBlock()).To(BeNil(), "error on NextBlock")

					fees := gasPrice.Mul(sdkmath.NewInt(res.GasUsed))
					expFinalIslmBalanceAmt := senderIslmBalanceBeforeRes.Balance.Amount.Sub(fees)

					senderIslmBalanceAfterRes, err := s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
					Expect(err).To(BeNil(), "error while retrieving sender's balance of aISLM")
					Expect(senderIslmBalanceAfterRes.Balance.Amount).To(Equal(expFinalIslmBalanceAmt), "sender balance of aISLM after tx is incorrect")
					senderHaqqBalanceAfterRes, err := s.grpcHandler.GetBalance(sender.AccAddr, ethiqtypes.BaseDenom)
					Expect(err).To(BeNil(), "error while retrieving sender's balance of aHAQQ")
					Expect(senderHaqqBalanceAfterRes.Balance.IsZero()).To(BeTrue(), "sender balance of aHAQQ after tx is incorrect")
				})
			})
		})
	})
})

var _ = Describe("Calling ethiq precompile via Solidity", Ordered, func() {
	Describe("when the precompile is not enabled in the EVM params", func() {
		It("should return an error", func() {
			Skip("TODO Implement solidity tests")
		})
	})
})

// Full Safe (Smart Contract Wallet) flow integration tests.
var _ = Describe("Full Safe (Smart Contract Wallet) flow", Ordered, func() {
	var (
		s                *PrecompileTestSuite
		safeOwnerOne     keyring.Key
		safeOwnerTwo     keyring.Key
		safeWalletAddr   common.Address
		gnosisSafe       evmtypes.CompiledContract
		gnosisSafeAddr   common.Address
		proxyFactory     evmtypes.CompiledContract
		proxyFactoryAddr common.Address
		safeSetupData    []byte
		deployErr        error
	)

	BeforeEach(func() {
		s = new(PrecompileTestSuite)
		s.SetupTest()

		gnosisSafe, deployErr = safecontracts.LoadGnosisSafeContract()
		Expect(deployErr).ToNot(HaveOccurred(), "failed to load GnosisSafe singleton contract")

		proxyFactory, deployErr = safecontracts.LoadGnosisSafeProxyFactoryContract()
		Expect(deployErr).ToNot(HaveOccurred(), "failed to load GnosisSafeProxyFactory contract")

		sender := s.keyring.GetKey(0)
		safeOwnerOneAddr, safeOwnerOnePriv := testutiltx.NewAddrKey()
		safeOwnerOne = keyring.Key{
			Addr:    safeOwnerOneAddr,
			AccAddr: sdk.AccAddress(safeOwnerOneAddr.Bytes()),
			Priv:    safeOwnerOnePriv,
		}
		safeOwnerTwoAddr, safeOwnerTwoPriv := testutiltx.NewAddrKey()
		safeOwnerTwo = keyring.Key{
			Addr:    safeOwnerTwoAddr,
			AccAddr: sdk.AccAddress(safeOwnerTwoAddr.Bytes()),
			Priv:    safeOwnerTwoPriv,
		}

		fundAmount := sdkmath.NewInt(1500).MulRaw(1e18)
		deployErr = s.network.FundAccountWithBaseDenom(safeOwnerOne.AccAddr, fundAmount)
		Expect(deployErr).ToNot(HaveOccurred(), "failed to fund first Safe owner")
		deployErr = s.network.FundAccountWithBaseDenom(safeOwnerTwo.AccAddr, fundAmount)
		Expect(deployErr).ToNot(HaveOccurred(), "failed to fund second Safe owner")

		gnosisSafeAddr, deployErr = s.factory.DeployContract(
			sender.Priv,
			evmtypes.EvmTxArgs{},
			factory.ContractDeploymentData{
				Contract: gnosisSafe,
			},
		)
		Expect(deployErr).ToNot(HaveOccurred(), "failed to deploy GnosisSafe singleton")
		Expect(gnosisSafeAddr).ToNot(Equal(common.Address{}), "singleton address should be non-zero")

		proxyFactoryAddr, deployErr = s.factory.DeployContract(
			sender.Priv,
			evmtypes.EvmTxArgs{},
			factory.ContractDeploymentData{
				Contract: proxyFactory,
			},
		)
		Expect(deployErr).ToNot(HaveOccurred(), "failed to deploy GnosisSafeProxyFactory")
		Expect(proxyFactoryAddr).ToNot(Equal(common.Address{}), "factory address should be non-zero")

		Expect(s.network.NextBlock()).ToNot(HaveOccurred(), "failed to advance block")
	})

	It("should execute full Safe flow against ethiq precompile", func() {
		initialParticipantBalance := sdkmath.NewInt(1500).MulRaw(1e18)
		transferToSafeAmount := sdkmath.NewInt(500).MulRaw(1e18)
		expectedSafeBalance := sdkmath.NewInt(1000).MulRaw(1e18)
		expectedParticipantFinalBalance := sdkmath.NewInt(1000).MulRaw(1e18)

		ownerOneBalanceBeforeRes, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get first owner balance before Safe creation")
		ownerTwoBalanceBeforeRes, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get second owner balance before Safe creation")
		Expect(ownerOneBalanceBeforeRes.Balance.Amount).To(Equal(initialParticipantBalance), "unexpected first owner initial balance")
		Expect(ownerTwoBalanceBeforeRes.Balance.Amount).To(Equal(initialParticipantBalance), "unexpected second owner initial balance")

		safeSetupData, deployErr = gnosisSafe.ABI.Pack(
			"setup",
			[]common.Address{safeOwnerOne.Addr, safeOwnerTwo.Addr},
			big.NewInt(1), // threshold = 1 out of 2 owners
			common.Address{},
			[]byte{},
			common.Address{},
			common.Address{},
			big.NewInt(0),
			common.Address{},
		)
		Expect(deployErr).ToNot(HaveOccurred(), "failed to pack GnosisSafe setup calldata")

		createProxyArgs := factory.CallArgs{
			ContractABI: proxyFactory.ABI,
			MethodName:  "createProxy",
			Args: []interface{}{
				gnosisSafeAddr,
				safeSetupData,
			},
		}

		createProxyTxArgs := evmtypes.EvmTxArgs{
			To: &proxyFactoryAddr,
		}

		createProxyRes, err := s.factory.ExecuteContractCall(
			safeOwnerOne.Priv,
			createProxyTxArgs,
			createProxyArgs,
		)
		Expect(err).ToNot(HaveOccurred(), "failed to broadcast createProxy tx")

		ethRes, err := s.factory.GetEvmTransactionResponseFromTxResult(createProxyRes)
		Expect(err).ToNot(HaveOccurred(), "failed to decode createProxy tx response")

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
		Expect(proxyCreationLog).ToNot(BeNil(), "ProxyCreation event not found in createProxy logs")

		eventInputs, err := proxyFactory.ABI.Events["ProxyCreation"].Inputs.Unpack(proxyCreationLog.Data)
		Expect(err).ToNot(HaveOccurred(), "failed to decode ProxyCreation event data")
		Expect(eventInputs).To(HaveLen(2), "unexpected ProxyCreation event payload")

		var ok bool
		safeWalletAddr, ok = eventInputs[0].(common.Address)
		Expect(ok).To(BeTrue(), "invalid proxy address type in ProxyCreation event")
		Expect(safeWalletAddr).ToNot(Equal(common.Address{}), "created Safe wallet address should be non-zero")
		Expect(eventInputs[1]).To(Equal(gnosisSafeAddr), "singleton in event should match deployed GnosisSafe")

		Expect(s.network.NextBlock()).ToNot(HaveOccurred(), "failed to advance block after Safe wallet creation")

		thresholdQueryTxArgs := evmtypes.EvmTxArgs{
			To: &safeWalletAddr,
		}
		thresholdQueryArgs := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "getThreshold",
		}
		_, thresholdRes, err := s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			thresholdQueryTxArgs,
			thresholdQueryArgs,
			testutil.LogCheckArgs{ExpPass: true},
		)
		Expect(err).ToNot(HaveOccurred(), "failed to query Safe threshold")

		thresholdOutputs, err := gnosisSafe.ABI.Methods["getThreshold"].Outputs.Unpack(thresholdRes.Ret)
		Expect(err).ToNot(HaveOccurred(), "failed to decode getThreshold output")
		Expect(thresholdOutputs).To(HaveLen(1))
		Expect(thresholdOutputs[0]).To(Equal(big.NewInt(1)), "expected threshold to be 1")

		ownersQueryArgs := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "getOwners",
		}
		_, ownersRes, err := s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			thresholdQueryTxArgs,
			ownersQueryArgs,
			testutil.LogCheckArgs{ExpPass: true},
		)
		Expect(err).ToNot(HaveOccurred(), "failed to query Safe owners")

		ownersOutputs, err := gnosisSafe.ABI.Methods["getOwners"].Outputs.Unpack(ownersRes.Ret)
		Expect(err).ToNot(HaveOccurred(), "failed to decode getOwners output")
		Expect(ownersOutputs).To(HaveLen(1))
		owners, ok := ownersOutputs[0].([]common.Address)
		Expect(ok).To(BeTrue(), "invalid owners output type")
		Expect(owners).To(ContainElements(safeOwnerOne.Addr, safeOwnerTwo.Addr), "Safe owners mismatch")
		Expect(owners).To(HaveLen(2), "expected exactly two Safe owners")

		safeWalletAccAddr := sdk.AccAddress(safeWalletAddr.Bytes())
		transferCoins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, transferToSafeAmount))
		ctx := s.network.GetContext()

		err = s.network.App.BankKeeper.SendCoins(ctx, safeOwnerOne.AccAddr, safeWalletAccAddr, transferCoins)
		Expect(err).ToNot(HaveOccurred(), "failed to transfer 500 ISLM from first owner to Safe")
		err = s.network.App.BankKeeper.SendCoins(ctx, safeOwnerTwo.AccAddr, safeWalletAccAddr, transferCoins)
		Expect(err).ToNot(HaveOccurred(), "failed to transfer 500 ISLM from second owner to Safe")

		safeBalanceRes, err := s.grpcHandler.GetBalance(safeWalletAccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe wallet balance")
		Expect(safeBalanceRes.Balance.Amount).To(Equal(expectedSafeBalance), "Safe wallet balance should be 1000 ISLM")

		ownerOneBalanceBeforeMintRes, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get first owner balance before Safe mint tx")
		ownerTwoBalanceBeforeMintRes, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get second owner balance before Safe mint tx")
		safeIslmBeforeMintRes, err := s.grpcHandler.GetBalance(safeWalletAccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe ISLM balance before mint tx")
		safeHaqqBeforeMintRes, err := s.grpcHandler.GetBalance(safeWalletAccAddr, ethiqtypes.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe HAQQ balance before mint tx")

		mintAmount := sdkmath.NewInt(500).MulRaw(1e18)
		// Grant Safe wallet authorization to mint on behalf of owner1 for the exact mint amount.
		s.SetupApproval(
			safeOwnerOne.Priv,
			safeWalletAddr,
			mintAmount.BigInt(),
			[]string{ethiq.MintHaqqMsgURL},
		)

		mintCallData, err := s.precompile.ABI.Pack(
			ethiq.MintHaqq,
			safeWalletAddr,
			safeWalletAddr, // receiver must be Safe wallet
			mintAmount.BigInt(),
		)
		Expect(err).ToNot(HaveOccurred(), "failed to pack ethiq mintHaqq call data")

		getTxHashArgs := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "getTransactionHash",
			Args: []interface{}{
				s.precompile.Address(),
				big.NewInt(0),
				mintCallData,
				uint8(0), // CALL
				big.NewInt(300000),
				big.NewInt(0),
				big.NewInt(0),
				common.Address{},
				common.Address{},
				big.NewInt(0), // nonce after createProxy
			},
		}
		_, txHashRes, err := s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &safeWalletAddr},
			getTxHashArgs,
			testutil.LogCheckArgs{ExpPass: true},
		)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe transaction hash")
		txHashOutputs, err := gnosisSafe.ABI.Methods["getTransactionHash"].Outputs.Unpack(txHashRes.Ret)
		Expect(err).ToNot(HaveOccurred(), "failed to decode Safe transaction hash output")
		Expect(txHashOutputs).To(HaveLen(1))
		txHash, ok := txHashOutputs[0].([32]byte)
		Expect(ok).To(BeTrue(), "unexpected tx hash type")
		Expect(txHash).ToNot(Equal([32]byte{}), "Safe tx hash should not be zero")

		// Safe signature format is {r}{s}{v}. For threshold=1 we can use "approved hash" style signature:
		// v=1 and r=owner address (msg.sender is owner1, so hash is considered approved by sender).
		signature := make([]byte, 65)
		copy(signature[12:32], safeOwnerOne.Addr.Bytes()) // r contains owner address in last 20 bytes
		signature[64] = 1                                 // v = 1 approved-hash signature

		execTxArgs := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "execTransaction",
			Args: []interface{}{
				s.precompile.Address(),
				big.NewInt(0),
				mintCallData,
				uint8(0), // CALL
				big.NewInt(300000),
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
		Expect(err).ToNot(HaveOccurred(), "failed to execute Safe mintHaqq transaction")
		execRes, err := s.factory.GetEvmTransactionResponseFromTxResult(execTxRes)
		Expect(err).ToNot(HaveOccurred(), "failed to decode execTransaction response")
		execOutputs, err := gnosisSafe.ABI.Methods["execTransaction"].Outputs.Unpack(execRes.Ret)
		Expect(err).ToNot(HaveOccurred(), "failed to decode execTransaction output")
		Expect(execOutputs).To(HaveLen(1))
		execSuccess, ok := execOutputs[0].(bool)
		Expect(ok).To(BeTrue(), "unexpected execTransaction output type")
		Expect(execSuccess).To(BeTrue(), "Safe execTransaction should succeed")

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
		Expect(executionSuccessFound).To(BeTrue(), "ExecutionSuccess event should be emitted by Safe")

		Expect(s.network.NextBlock()).ToNot(HaveOccurred(), "failed to advance block after Safe mint tx")

		ownerOneBalanceAfterMintRes, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get first owner balance after Safe mint tx")
		ownerTwoBalanceAfterMintRes, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get second owner balance after Safe mint tx")
		safeIslmAfterMintRes, err := s.grpcHandler.GetBalance(safeWalletAccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe ISLM balance after mint tx")
		safeHaqqAfterMintRes, err := s.grpcHandler.GetBalance(safeWalletAccAddr, ethiqtypes.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe HAQQ balance after mint tx")

		ownerOneSpentForMint := ownerOneBalanceBeforeMintRes.Balance.Amount.Sub(ownerOneBalanceAfterMintRes.Balance.Amount)
		Expect(ownerOneSpentForMint.IsNegative()).To(BeFalse(), "first owner balance should not increase after mint tx")
		Expect(ownerOneSpentForMint.LTE(sdkmath.NewInt(1).MulRaw(1e18))).To(BeTrue(), "first owner balance change should be at most 1 ISLM")

		ownerTwoSpentForMint := ownerTwoBalanceBeforeMintRes.Balance.Amount.Sub(ownerTwoBalanceAfterMintRes.Balance.Amount)
		Expect(ownerTwoSpentForMint.IsNegative()).To(BeFalse(), "second owner balance should not increase after mint tx")
		Expect(ownerTwoSpentForMint.LTE(sdkmath.NewInt(1).MulRaw(1e18))).To(BeTrue(), "second owner balance change should be at most 1 ISLM")

		safeHaqqDelta := safeHaqqAfterMintRes.Balance.Amount.Sub(safeHaqqBeforeMintRes.Balance.Amount)
		Expect(safeHaqqDelta.IsPositive()).To(BeTrue(), "Safe wallet should receive HAQQ tokens")

		safeIslmDelta := safeIslmBeforeMintRes.Balance.Amount.Sub(safeIslmAfterMintRes.Balance.Amount)
		Expect(safeIslmDelta).To(Equal(mintAmount), "Safe ISLM balance should decrease by 500 ISLM")

		Expect(gnosisSafeAddr).NotTo(BeZero(), "GnosisSafe singleton must be deployed in BeforeEach")
		Expect(proxyFactoryAddr).NotTo(BeZero(), "GnosisSafeProxyFactory must be deployed in BeforeEach")
		Expect(safeWalletAddr).NotTo(BeZero(), "Safe wallet must be created")
		Expect(ownerTwoBalanceBeforeMintRes.Balance.Amount).To(Equal(expectedParticipantFinalBalance), "second owner baseline before mint should be 1000 ISLM")
	})
})
