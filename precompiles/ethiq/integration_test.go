package ethiq_test

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/ethiq"
	"github.com/haqq-network/haqq/precompiles/ethiq/testdata"
	"github.com/haqq-network/haqq/precompiles/testutil"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
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
	var (
		// s is the precompile test suite to use for the tests
		s *PrecompileTestSuite
		// contractAddr is the address of the smart contract that will be deployed
		contractAddr    common.Address
		contractTwoAddr common.Address
		reverterAddr    common.Address

		// ethiqCallerContract is the contract instance calling into the ethiq precompile
		ethiqCallerContract    evmtypes.CompiledContract
		ethiqCallerTwoContract evmtypes.CompiledContract
		ethiqReverterContract  evmtypes.CompiledContract

		// approvalCheck is a configuration for the log checker to see if an approval event was emitted.
		approvalCheck testutil.LogCheckArgs
		// execRevertedCheck defines the default log checking arguments which include the
		// standard revert message
		execRevertedCheck testutil.LogCheckArgs
		// err is a basic error type
		err error

		// nonExistingAddr is an address that does not exist in the state of the test suite
		// nonExistingAddr = testutiltx.GenerateAddress()
		// nonExistingVal is a validator address that does not exist in the state of the test suite
		testContractInitialBalance = sdkmath.NewInt(1e18)
	)

	BeforeAll(func() {
		ethiqCallerContract, err = testdata.LoadEthiqCallerContract()
		Expect(err).To(BeNil(), "error while loading the EthiqCaller contract")
		ethiqCallerTwoContract, err = testdata.LoadEthiqCallerTwoContract()
		Expect(err).To(BeNil(), "error while loading the EthiqCallerTwo contract")
		ethiqReverterContract, err = contracts.LoadEthiqReverterContract()
		Expect(err).To(BeNil(), "error while loading the DEthiqReverter contract")
	})

	BeforeEach(func() {
		s = new(PrecompileTestSuite)
		s.SetupTest()
		burner := s.keyring.GetKey(0)

		contractAddr, err = s.factory.DeployContract(
			burner.Priv,
			evmtypes.EvmTxArgs{}, // NOTE: passing empty struct to use default values
			factory.ContractDeploymentData{
				Contract: ethiqCallerContract,
			},
		)
		Expect(err).To(BeNil(), "error while deploying the smart contract: %v", err)
		Expect(s.network.NextBlock()).To(BeNil())

		// Deploy EthiqCallerTwo contract
		contractTwoAddr, err = s.factory.DeployContract(
			burner.Priv,
			evmtypes.EvmTxArgs{}, // NOTE: passing empty struct to use default values
			factory.ContractDeploymentData{
				Contract: ethiqCallerTwoContract,
			},
		)
		Expect(err).To(BeNil(), "error while deploying the EthiqCallerTwo contract")
		Expect(s.network.NextBlock()).To(BeNil())

		// Deploy EthiqReverter contract
		reverterAddr, err = s.factory.DeployContract(
			burner.Priv,
			evmtypes.EvmTxArgs{}, // NOTE: passing empty struct to use default values
			factory.ContractDeploymentData{
				Contract: ethiqReverterContract,
			},
		)
		Expect(err).To(BeNil(), "error while deploying the EthiqReverter contract")
		Expect(s.network.NextBlock()).To(BeNil())

		// send some funds to the EthiqCallerTwo & EthiqReverter contracts to transfer to the
		// burner during the tx
		err := testutils.FundAccountWithBaseDenom(s.factory, s.network, burner, contractTwoAddr.Bytes(), testContractInitialBalance)
		Expect(err).To(BeNil(), "error while funding the smart contract: %v", err)
		Expect(s.network.NextBlock()).To(BeNil())
		err = testutils.FundAccountWithBaseDenom(s.factory, s.network, burner, reverterAddr.Bytes(), testContractInitialBalance)
		Expect(err).To(BeNil(), "error while funding the smart contract: %v", err)
		Expect(s.network.NextBlock()).To(BeNil())

		// check contract was correctly deployed
		cAcc := s.network.App.EvmKeeper.GetAccount(s.network.GetContext(), contractAddr)
		Expect(cAcc).ToNot(BeNil(), "contract account should exist")
		Expect(cAcc.IsContract()).To(BeTrue(), "account should be a contract")

		// populate default TxArgs
		txArgs.To = &contractAddr
		txArgs.GasLimit = 300000
		// populate default call args
		callArgs = factory.CallArgs{
			ContractABI: ethiqCallerContract.ABI,
		}
		// populate default approval args
		approveCallArgs = factory.CallArgs{
			ContractABI: ethiqCallerContract.ABI,
			MethodName:  "testApprove",
		}
		// populate default log check args
		defaultLogCheck = testutil.LogCheckArgs{
			ABIEvents: s.precompile.Events,
		}
		execRevertedCheck = defaultLogCheck.WithErrContains(vm.ErrExecutionReverted.Error())
		passCheck = defaultLogCheck.WithExpPass(true)
		approvalCheck = passCheck.WithExpEvents(authorization.EventTypeApproval)
	})

	Describe("when the precompile is not enabled in the EVM params", func() {
		It("should return an error", func() {
			sender := s.keyring.GetKey(0)

			// disable the precompile
			res, err := s.grpcHandler.GetEvmParams()
			Expect(err).To(BeNil(), "error while setting params")
			params := res.Params
			var activePrecompiles []string
			for _, precompile := range params.ActiveStaticPrecompiles {
				if precompile != s.precompile.Address().String() {
					activePrecompiles = append(activePrecompiles, precompile)
				}
			}
			params.ActiveStaticPrecompiles = activePrecompiles

			err = testutils.UpdateEvmParams(testutils.UpdateParamsInput{
				Tf:      s.factory,
				Network: s.network,
				Pk:      sender.Priv,
				Params:  params,
			})
			Expect(err).To(BeNil(), "error while setting params")

			// try to call the precompile
			callArgs.MethodName = "testMintHaqq"
			callArgs.Args = []interface{}{
				sender.Addr, sender.Addr, big.NewInt(2e18),
			}

			_, _, err = s.factory.CallContractAndCheckLogs(
				sender.Priv,
				txArgs, callArgs,
				execRevertedCheck,
			)
			Expect(err).To(BeNil(), "expected error while calling the precompile")
		})
	})

	Context("approving methods", func() {
		Context("with valid input", func() {
			It("should approve one method", func() {
				granter := s.keyring.GetKey(0)

				approveCallArgs.Args = []interface{}{
					contractAddr, []string{ethiq.MintHaqqMsgURL}, big.NewInt(1e18),
				}

				s.SetupApprovalWithContractCalls(granter, txArgs, approveCallArgs)
			})

			It("should update a previous approval", func() {
				granter := s.keyring.GetKey(0)

				approveCallArgs.Args = []interface{}{
					contractAddr, []string{ethiq.MintHaqqMsgURL}, big.NewInt(1e18),
				}

				s.SetupApprovalWithContractCalls(granter, txArgs, approveCallArgs)

				// update approval
				approveCallArgs.Args = []interface{}{
					contractAddr, []string{ethiq.MintHaqqMsgURL}, big.NewInt(2e18),
				}
				_, _, err = s.factory.CallContractAndCheckLogs(
					granter.Priv,
					txArgs, approveCallArgs,
					approvalCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)
				Expect(s.network.NextBlock()).To(BeNil())

				// check approvals
				authorization, expirationTime, err := CheckAuthorization(s.grpcHandler, s.network.GetEncodingConfig().InterfaceRegistry, ethiq.MintHaqqMsgURL, contractAddr, granter.Addr)
				Expect(err).To(BeNil())
				Expect(authorization).ToNot(BeNil(), "expected authorization to not be nil")
				Expect(expirationTime).ToNot(BeNil(), "expected expiration time to not be nil")
				authzMintHaqq, ok := authorization.(*ethiqtypes.MintHaqqAuthorization)
				Expect(ok).To(BeTrue())
				Expect(authzMintHaqq.MsgTypeURL()).To(Equal(ethiq.MintHaqqMsgURL), "expected authorization msg type url to be %s", ethiq.MintHaqqMsgURL)
				Expect(authzMintHaqq.SpendLimit.Amount).To(Equal(sdkmath.NewInt(2e18)), "expected different max tokens after updated approval")
			})

			It("should remove approval when setting amount to zero", func() {
				granter := s.keyring.GetKey(0)

				approveCallArgs.Args = []interface{}{
					contractAddr, []string{ethiq.MintHaqqMsgURL}, big.NewInt(1e18),
				}
				s.SetupApprovalWithContractCalls(granter, txArgs, approveCallArgs)
				Expect(s.network.NextBlock()).To(BeNil())

				// check approvals pre-removal
				allAuthz, err := s.grpcHandler.GetAuthorizations(sdk.AccAddress(contractAddr.Bytes()).String(), granter.AccAddr.String())
				Expect(err).To(BeNil(), "error while reading authorizations")
				Expect(allAuthz).To(HaveLen(1), "expected no authorizations")

				approveCallArgs.Args = []interface{}{
					contractAddr, []string{ethiq.MintHaqqMsgURL}, big.NewInt(0),
				}

				_, _, err = s.factory.CallContractAndCheckLogs(
					granter.Priv,
					txArgs, approveCallArgs,
					approvalCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract")
				Expect(s.network.NextBlock()).To(BeNil())

				// check approvals after approving with amount 0
				allAuthz, err = s.grpcHandler.GetAuthorizations(sdk.AccAddress(contractAddr.Bytes()).String(), granter.AccAddr.String())
				Expect(err).To(BeNil(), "error while reading authorizations")
				Expect(allAuthz).To(HaveLen(0), "expected no authorizations")
			})

			It("should not approve if the gas is not enough", func() {
				granter := s.keyring.GetKey(0)

				txArgs.GasLimit = 30000
				approveCallArgs.Args = []interface{}{
					contractAddr,
					[]string{
						ethiq.MintHaqqMsgURL,
					},
					big.NewInt(1e18),
				}

				_, _, err = s.factory.CallContractAndCheckLogs(
					granter.Priv,
					txArgs, approveCallArgs,
					execRevertedCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract")
			})
		})

		Context("with invalid input", func() {
			It("shouldn't approve for invalid methods", func() {
				granter := s.keyring.GetKey(0)

				approveCallArgs.Args = []interface{}{
					contractAddr, []string{"invalid method"}, big.NewInt(1e18),
				}

				_, _, err = s.factory.CallContractAndCheckLogs(
					granter.Priv,
					txArgs, approveCallArgs,
					execRevertedCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract")

				// check approvals
				allAuthz, err := s.grpcHandler.GetAuthorizations(sdk.AccAddress(contractAddr.Bytes()).String(), granter.AccAddr.String())
				Expect(err).To(BeNil(), "error while reading authorizations")
				Expect(allAuthz).To(HaveLen(0), "expected no authorizations")
			})
		})
	})

	Context("to revoke an approval", func() {
		BeforeEach(func() {
			callArgs.MethodName = "testRevoke"
		})

		It("should revoke when sending as the granter", func() {
			granter := s.keyring.GetKey(0)

			// set up an approval to be revoked
			approveCallArgs.Args = []interface{}{
				contractAddr, []string{ethiq.MintHaqqMsgURL}, big.NewInt(1e18),
			}

			s.SetupApprovalWithContractCalls(granter, txArgs, approveCallArgs)

			callArgs.Args = []interface{}{contractAddr, []string{ethiq.MintHaqqMsgURL}}

			revocationCheck := passCheck.WithExpEvents(authorization.EventTypeRevocation)

			_, _, err = s.factory.CallContractAndCheckLogs(
				granter.Priv,
				txArgs, callArgs,
				revocationCheck,
			)
			Expect(err).To(BeNil(), "error while calling the smart contract")
			Expect(s.network.NextBlock()).To(BeNil())

			// check approvals
			authz, _, err := CheckAuthorization(s.grpcHandler, s.network.GetEncodingConfig().InterfaceRegistry, ethiq.MintHaqqMsgURL, contractAddr, granter.Addr)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("no authorizations found for grantee %s and granter %s", contractAddr.Hex(), granter.Addr.Hex())))
			Expect(authz).To(BeNil(), "expected authorization to be revoked")
		})

		It("should not revoke when approval is issued by a different granter", func() {
			// Create a MintHaqq authorization where the granter is a different account from the default test suite one
			granter := s.keyring.GetKey(0)
			grantee := s.keyring.GetKey(1)
			differentGranterIdx := s.keyring.AddKey()
			differentGranter := s.keyring.GetKey(differentGranterIdx)

			mintHaqqAuthz, err := ethiqtypes.NewMintHaqqAuthorization(
				&sdk.Coin{Denom: utils.BaseDenom, Amount: sdkmath.NewInt(1e18)},
			)
			Expect(err).To(BeNil(), "failed to create authorization")

			expiration := s.network.GetContext().BlockTime().Add(time.Hour * 24 * 365).UTC()
			err = s.network.App.AuthzKeeper.SaveGrant(s.network.GetContext(), grantee.AccAddr, differentGranter.AccAddr, mintHaqqAuthz, &expiration)
			Expect(err).ToNot(HaveOccurred(), "failed to save authorization")
			authz, _, err := CheckAuthorization(s.grpcHandler, s.network.GetEncodingConfig().InterfaceRegistry, ethiq.MintHaqqMsgURL, grantee.Addr, differentGranter.Addr)
			Expect(err).To(BeNil())
			Expect(authz).ToNot(BeNil(), "expected authorization to be created")

			callArgs.Args = []interface{}{grantee.Addr, []string{ethiq.MintHaqqMsgURL}}

			_, _, err = s.factory.CallContractAndCheckLogs(
				granter.Priv,
				txArgs, callArgs,
				execRevertedCheck,
			)
			Expect(err).To(BeNil(), "error while calling the smart contract")

			// check approvals
			authz, _, err = CheckAuthorization(s.grpcHandler, s.network.GetEncodingConfig().InterfaceRegistry, ethiq.MintHaqqMsgURL, grantee.Addr, differentGranter.Addr)
			Expect(err).To(BeNil())
			Expect(authz).ToNot(BeNil(), "expected authorization not to be revoked")
		})

		It("should revert the execution when no approval is found", func() {
			granter := s.keyring.GetKey(0)
			callArgs.Args = []interface{}{contractAddr, []string{ethiq.MintHaqqMsgURL}}

			_, _, err = s.factory.CallContractAndCheckLogs(
				granter.Priv,
				txArgs, callArgs,
				execRevertedCheck,
			)
			Expect(err).To(BeNil(), "error while calling the smart contract")

			// check approvals
			authz, _, err := CheckAuthorization(s.grpcHandler, s.network.GetEncodingConfig().InterfaceRegistry, ethiq.MintHaqqMsgURL, contractAddr, granter.Addr)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("no authorizations found for grantee %s and granter %s", contractAddr.Hex(), granter.Addr.Hex())))
			Expect(authz).To(BeNil(), "expected no authorization to be found")
		})

		It("should not revoke if the approval is for a different message type", func() {
			granter := s.keyring.GetKey(0)

			// set up an approval
			approveCallArgs.Args = []interface{}{
				contractAddr, []string{ethiq.MintHaqqMsgURL}, big.NewInt(1e18),
			}

			s.SetupApprovalWithContractCalls(granter, txArgs, approveCallArgs)

			Expect(s.network.NextBlock()).To(BeNil(), "failed to advance block")

			callArgs.Args = []interface{}{contractAddr, []string{ethiq.MsgMintHaqqByApplicationMsgURL}}

			_, _, err = s.factory.CallContractAndCheckLogs(
				granter.Priv,
				txArgs, callArgs,
				execRevertedCheck,
			)
			Expect(err).To(BeNil(), "error while calling the smart contract")

			// check approval is still there
			s.ExpectAuthorization(
				ethiq.MintHaqqMsgURL,
				contractAddr,
				granter.Addr,
				&sdk.Coin{Denom: utils.BaseDenom, Amount: sdkmath.NewInt(1e18)},
				0,
			)
		})
	})

	Context("querying allowance", func() {
		BeforeEach(func() {
			callArgs.MethodName = "getAllowance"
		})
		It("without approval set it should show no allowance", func() {
			granter := s.keyring.GetKey(0)

			callArgs.Args = []interface{}{
				contractAddr, ethiq.MintHaqqMsgURL,
			}

			_, ethRes, err := s.factory.CallContractAndCheckLogs(
				granter.Priv,
				txArgs, callArgs,
				passCheck,
			)
			Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

			var allowanceInt *big.Int
			err = s.precompile.UnpackIntoInterface(&allowanceInt, "allowance", ethRes.Ret)
			Expect(err).To(BeNil(), "error while unmarshalling the allowance: %v", err)
			Expect(allowanceInt.Int64()).To(Equal(int64(0)), "expected empty allowance")
		})

		It("with approval set it should show the granted allowance", func() {
			granter := s.keyring.GetKey(0)

			// setup approval
			approveCallArgs.Args = []interface{}{
				contractAddr, []string{ethiq.MintHaqqMsgURL}, big.NewInt(1e18),
			}

			s.SetupApprovalWithContractCalls(granter, txArgs, approveCallArgs)

			// query allowance
			callArgs.Args = []interface{}{
				contractAddr, ethiq.MintHaqqMsgURL,
			}

			_, ethRes, err := s.factory.CallContractAndCheckLogs(
				granter.Priv,
				txArgs, callArgs,
				passCheck,
			)
			Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

			var allowanceInt *big.Int
			err = s.precompile.UnpackIntoInterface(&allowanceInt, "allowance", ethRes.Ret)
			Expect(err).To(BeNil(), "error while unmarshalling the allowance: %v", err)
			Expect(allowanceInt).To(Equal(big.NewInt(1e18)), "expected allowance to be 1e18")
		})
	})

	Context("burning and minting", func() {
		var balanceIslmBefore, balanceHaqqBefore *sdk.Coin

		BeforeEach(func() {
			burner := s.keyring.GetKey(0)

			// get the initial balances prior to the test
			res, err := s.grpcHandler.GetBalance(burner.AccAddr, ethiqtypes.BaseDenom)
			Expect(err).To(BeNil())
			Expect(res.Balance).NotTo(BeNil())
			balanceHaqqBefore = res.Balance

			res2, err := s.grpcHandler.GetBalance(burner.AccAddr, utils.BaseDenom)
			Expect(err).To(BeNil())
			Expect(res2.Balance).NotTo(BeNil())
			balanceIslmBefore = res2.Balance

			callArgs.MethodName = "testMintHaqq"
		})
		Context("without approval set", func() {
			BeforeEach(func() {
				granter := s.keyring.GetKey(0)

				authz, _, err := CheckAuthorization(s.grpcHandler, s.network.GetEncodingConfig().InterfaceRegistry, ethiq.MintHaqqMsgURL, contractAddr, granter.Addr)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("no authorizations found for grantee %s and granter %s", contractAddr.Hex(), granter.Addr.Hex())))
				Expect(authz).To(BeNil(), "expected authorization to be nil")
			})

			It("should not burn/mint", func() {
				Expect(s.network.App.EvmKeeper.GetAccount(s.network.GetContext(), contractAddr)).ToNot(BeNil(), "expected contract to exist")
				burner := s.keyring.GetKey(0)

				txArgs.GasPrice = big.NewInt(1e9)
				callArgs.Args = []interface{}{
					burner.Addr, burner.Addr, big.NewInt(1e18),
				}

				res, _, err := s.factory.CallContractAndCheckLogs(
					burner.Priv,
					txArgs, callArgs,
					execRevertedCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)
				Expect(s.network.NextBlock()).To(BeNil())

				feePaid := txArgs.GasPrice.Int64() * res.GasUsed

				// get the final balances after the test
				resBal1, err := s.grpcHandler.GetBalance(burner.AccAddr, ethiqtypes.BaseDenom)
				Expect(err).To(BeNil())
				Expect(resBal1.Balance).NotTo(BeNil())
				balanceHaqqAfter := resBal1.Balance

				resBal2, err := s.grpcHandler.GetBalance(burner.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil())
				Expect(resBal2.Balance).NotTo(BeNil())
				balanceIslmAfter := resBal2.Balance

				Expect(balanceHaqqAfter.Amount).To(Equal(balanceHaqqBefore.Amount), "balances of aHAQQ are not equal as expected")
				Expect(balanceIslmAfter.Amount).To(Equal(balanceIslmBefore.Amount.Sub(sdkmath.NewInt(feePaid))), "incorrect aISLM balance after")
			})
		})

		Context("with approval set", func() {
			BeforeEach(func() {
				granter := s.keyring.GetKey(0)

				approveCallArgs.Args = []interface{}{
					contractAddr, []string{ethiq.MintHaqqMsgURL}, big.NewInt(1e18),
				}

				s.SetupApprovalWithContractCalls(granter, txArgs, approveCallArgs)
				// add gas limit to avoid out of gas error
				txArgs.GasLimit = 500_000
				txArgs.GasPrice = big.NewInt(1e9)

				// get the initial balances prior to the test
				res, err := s.grpcHandler.GetBalance(granter.AccAddr, ethiqtypes.BaseDenom)
				Expect(err).To(BeNil())
				Expect(res.Balance).NotTo(BeNil())
				balanceHaqqBefore = res.Balance

				res2, err := s.grpcHandler.GetBalance(granter.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil())
				Expect(res2.Balance).NotTo(BeNil())
				balanceIslmBefore = res2.Balance
			})

			It("should burn/mint when not exceeding the allowance", func() {
				burner := s.keyring.GetKey(0)

				callArgs.Args = []interface{}{
					burner.Addr, burner.Addr, big.NewInt(1e18),
				}

				logCheckArgs := passCheck.WithExpEvents(ethiq.EventTypeMintHaqq)

				res, _, err := s.factory.CallContractAndCheckLogs(
					burner.Priv,
					txArgs, callArgs,
					logCheckArgs,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)
				Expect(s.network.NextBlock()).To(BeNil())

				feePaid := txArgs.GasPrice.Int64() * res.GasUsed

				// get the final balances after the test
				resBal1, err := s.grpcHandler.GetBalance(burner.AccAddr, ethiqtypes.BaseDenom)
				Expect(err).To(BeNil())
				Expect(resBal1.Balance).NotTo(BeNil())
				balanceHaqqAfter := resBal1.Balance

				resBal2, err := s.grpcHandler.GetBalance(burner.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil())
				Expect(resBal2.Balance).NotTo(BeNil())
				balanceIslmAfter := resBal2.Balance

				Expect(balanceHaqqAfter.Amount.GT(balanceHaqqBefore.Amount)).To(BeTrue(), "balances of aHAQQ should be increased by minted amount")
				Expect(balanceIslmAfter.Amount).To(Equal(balanceIslmBefore.Amount.Sub(sdkmath.NewInt(feePaid)).Sub(sdkmath.NewInt(1e18))), "aISLM balance should be reduced by burnt amount and fees")
			})

			It("should not burn/mint when exceeding the allowance", func() {
				burner := s.keyring.GetKey(0)

				callArgs.Args = []interface{}{
					burner.Addr, burner.Addr, big.NewInt(2e18),
				}
				res, _, err := s.factory.CallContractAndCheckLogs(
					burner.Priv,
					txArgs, callArgs,
					execRevertedCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)
				Expect(s.network.NextBlock()).To(BeNil())

				feePaid := txArgs.GasPrice.Int64() * res.GasUsed

				// get the final balances after the test
				resBal1, err := s.grpcHandler.GetBalance(burner.AccAddr, ethiqtypes.BaseDenom)
				Expect(err).To(BeNil())
				Expect(resBal1.Balance).NotTo(BeNil())
				balanceHaqqAfter := resBal1.Balance

				resBal2, err := s.grpcHandler.GetBalance(burner.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil())
				Expect(resBal2.Balance).NotTo(BeNil())
				balanceIslmAfter := resBal2.Balance

				Expect(balanceHaqqAfter.Amount).To(Equal(balanceHaqqBefore.Amount), "aHAQQ balance should stay untouched")
				Expect(balanceIslmAfter.Amount).To(Equal(balanceIslmBefore.Amount.Sub(sdkmath.NewInt(feePaid))), "aISLM balance should be reduced by fees amount")
			})

			It("should not burn/mint when sending from a different address", func() {
				burner := s.keyring.GetKey(0)
				differentBurner := s.keyring.GetKey(1)

				callArgs.Args = []interface{}{
					burner.Addr, burner.Addr, big.NewInt(2e18),
				}
				_, _, err = s.factory.CallContractAndCheckLogs(
					differentBurner.Priv,
					txArgs, callArgs,
					execRevertedCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)
				Expect(s.network.NextBlock()).To(BeNil())

				// get the final balances after the test
				resBal1, err := s.grpcHandler.GetBalance(burner.AccAddr, ethiqtypes.BaseDenom)
				Expect(err).To(BeNil())
				Expect(resBal1.Balance).NotTo(BeNil())
				balanceHaqqAfter := resBal1.Balance

				resBal2, err := s.grpcHandler.GetBalance(burner.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil())
				Expect(resBal2.Balance).NotTo(BeNil())
				balanceIslmAfter := resBal2.Balance

				Expect(balanceHaqqAfter.Amount).To(Equal(balanceHaqqBefore.Amount), "aHAQQ balance should stay untouched")
				Expect(balanceIslmAfter.Amount).To(Equal(balanceIslmBefore.Amount), "aISLM balance should stay untouched")
			})

			Context("Calling the precompile from the EthiqReverter contract", func() {
				var (
					txSenderInitialBalIslm, txSenderInitialBalHaqq *sdk.Coin
					contractInitialBalIslm, contractInitialBalHaqq *sdk.Coin
				)
				gasPrice := sdkmath.NewInt(1e9)
				burnAmt := sdkmath.NewInt(1e18)

				BeforeEach(func() {
					// set approval for the EthiqReverter contract
					s.SetupApproval(s.keyring.GetPrivKey(0), reverterAddr, burnAmt.BigInt(), []string{ethiq.MintHaqqMsgURL})

					balRes, err := s.grpcHandler.GetBalance(s.keyring.GetAccAddr(0), utils.BaseDenom)
					Expect(err).To(BeNil())
					txSenderInitialBalIslm = balRes.Balance

					balRes, err = s.grpcHandler.GetBalance(s.keyring.GetAccAddr(0), ethiqtypes.BaseDenom)
					Expect(err).To(BeNil())
					txSenderInitialBalHaqq = balRes.Balance

					balRes, err = s.grpcHandler.GetBalance(reverterAddr.Bytes(), utils.BaseDenom)
					Expect(err).To(BeNil())
					contractInitialBalIslm = balRes.Balance

					balRes, err = s.grpcHandler.GetBalance(reverterAddr.Bytes(), ethiqtypes.BaseDenom)
					Expect(err).To(BeNil())
					contractInitialBalHaqq = balRes.Balance
				})

				It("should revert the changes and NOT burn/mint - successful tx", func() {
					sender := s.keyring.GetKey(0)

					callArgs := factory.CallArgs{
						ContractABI: ethiqReverterContract.ABI,
						MethodName:  "run",
						Args: []interface{}{
							big.NewInt(5), sender.Addr,
						},
					}

					// Tx should be successful, but no state changes happened
					res, _, err := s.factory.CallContractAndCheckLogs(
						sender.Priv,
						evmtypes.EvmTxArgs{
							To:       &reverterAddr,
							GasPrice: gasPrice.BigInt(),
						},
						callArgs,
						passCheck,
					)
					Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)
					Expect(s.network.NextBlock()).To(BeNil())

					fees := gasPrice.MulRaw(res.GasUsed)

					// contract balance should remain unchanged
					balRes, err := s.grpcHandler.GetBalance(reverterAddr.Bytes(), utils.BaseDenom)
					Expect(err).To(BeNil())
					contractFinalBalIslm := balRes.Balance
					Expect(contractFinalBalIslm.Amount).To(Equal(contractInitialBalIslm.Amount))
					balRes, err = s.grpcHandler.GetBalance(reverterAddr.Bytes(), ethiqtypes.BaseDenom)
					Expect(err).To(BeNil())
					contractFinalBalHaqq := balRes.Balance
					Expect(contractFinalBalHaqq.Amount).To(Equal(contractInitialBalHaqq.Amount))

					// No burn/mint should be occurred
					balRes, err = s.grpcHandler.GetBalance(sender.AccAddr, ethiqtypes.BaseDenom)
					Expect(err).To(BeNil())
					txSenderFinalBalHaqq := balRes.Balance
					Expect(txSenderFinalBalHaqq.Amount).To(Equal(txSenderInitialBalHaqq.Amount))

					// Only fees deducted on tx sender
					balRes, err = s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
					Expect(err).To(BeNil())
					txSenderFinalBalIslm := balRes.Balance
					Expect(txSenderFinalBalIslm.Amount).To(Equal(txSenderInitialBalIslm.Amount.Sub(fees)))
				})

				It("should revert the changes and NOT burn/mint - failed tx - max precompile calls reached", func() {
					sender := s.keyring.GetKey(0)

					callArgs := factory.CallArgs{
						ContractABI: ethiqReverterContract.ABI,
						MethodName:  "multipleBurnMints",
						Args: []interface{}{
							big.NewInt(int64(evmtypes.MaxPrecompileCalls + 2)), sender.Addr,
						},
					}

					// Tx should fail due to MaxPrecompileCalls
					res, _, err := s.factory.CallContractAndCheckLogs(
						s.keyring.GetPrivKey(0),
						evmtypes.EvmTxArgs{
							To:       &reverterAddr,
							GasPrice: gasPrice.BigInt(),
						},
						callArgs,
						execRevertedCheck,
					)
					Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)
					Expect(s.network.NextBlock()).To(BeNil())

					fees := gasPrice.MulRaw(res.GasUsed)

					// contract balance should remain unchanged
					balRes, err := s.grpcHandler.GetBalance(reverterAddr.Bytes(), utils.BaseDenom)
					Expect(err).To(BeNil())
					contractFinalBalIslm := balRes.Balance
					Expect(contractFinalBalIslm.Amount).To(Equal(contractInitialBalIslm.Amount))
					balRes, err = s.grpcHandler.GetBalance(reverterAddr.Bytes(), ethiqtypes.BaseDenom)
					Expect(err).To(BeNil())
					contractFinalBalHaqq := balRes.Balance
					Expect(contractFinalBalHaqq.Amount).To(Equal(contractInitialBalHaqq.Amount))

					// No burn/mint should be occurred
					balRes, err = s.grpcHandler.GetBalance(sender.AccAddr, ethiqtypes.BaseDenom)
					Expect(err).To(BeNil())
					txSenderFinalBalHaqq := balRes.Balance
					Expect(txSenderFinalBalHaqq.Amount).To(Equal(txSenderInitialBalHaqq.Amount))

					// Only fees deducted on tx sender
					balRes, err = s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
					Expect(err).To(BeNil())
					txSenderFinalBalIslm := balRes.Balance
					Expect(txSenderFinalBalIslm.Amount).To(Equal(txSenderInitialBalIslm.Amount.Sub(fees)))
				})
			})
		})
	})

	Context("when using special call opcodes", func() {
		testcases := []struct {
			// calltype is the opcode to use
			calltype string
			// expTxPass defines if executing transactions should be possible with the given opcode.
			// Queries should work for all options.
			expTxPass bool
		}{
			{"call", true},
			// {"callcode", false}, // TODO Find out the reason of failing
			{"staticcall", false},
			{"delegatecall", false},
		}

		var (
			senderBalIslmBefore, senderBalIslmAfter *sdk.Coin
			senderBalHaqqBefore, senderBalHaqqAfter *sdk.Coin
		)

		BeforeEach(func() {
			granter := s.keyring.GetKey(0)

			// approve mintHaqq message
			approveCallArgs.Args = []interface{}{
				contractAddr, []string{ethiq.MintHaqqMsgURL}, big.NewInt(1e18),
			}

			s.SetupApprovalWithContractCalls(granter, txArgs, approveCallArgs)
			Expect(s.network.NextBlock()).To(BeNil(), "failed to advance block")

			// get the initial balances prior to the test
			res, err := s.grpcHandler.GetBalance(granter.AccAddr, ethiqtypes.BaseDenom)
			Expect(err).To(BeNil())
			Expect(res.Balance).NotTo(BeNil())
			senderBalHaqqBefore = res.Balance

			res2, err := s.grpcHandler.GetBalance(granter.AccAddr, utils.BaseDenom)
			Expect(err).To(BeNil())
			Expect(res2.Balance).NotTo(BeNil())
			senderBalIslmBefore = res2.Balance
		})

		for _, tc := range testcases {
			// NOTE: this is necessary because of Ginkgo behavior -- if not done, the value of tc
			// inside the It block will always be the last entry in the testcases slice
			testcase := tc

			It(fmt.Sprintf("should not execute transactions for calltype %q", testcase.calltype), func() {
				sender := s.keyring.GetKey(0)

				callArgs.MethodName = "testCallMintHaqq"
				callArgs.Args = []interface{}{
					sender.Addr, sender.Addr, big.NewInt(1e18), testcase.calltype,
				}

				checkArgs := execRevertedCheck
				if testcase.expTxPass {
					checkArgs = passCheck.WithExpEvents(ethiq.EventTypeMintHaqq)
				}

				txArgs.GasPrice = gasPrice.BigInt()
				resCall, _, err := s.factory.CallContractAndCheckLogs(
					sender.Priv,
					txArgs, callArgs,
					checkArgs,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract for calltype %s: %v", testcase.calltype, err)
				Expect(s.network.NextBlock()).To(BeNil())

				fees := gasPrice.MulRaw(resCall.GasUsed)

				// check final balances
				res, err := s.grpcHandler.GetBalance(sender.AccAddr, ethiqtypes.BaseDenom)
				Expect(err).To(BeNil())
				Expect(res.Balance).NotTo(BeNil())
				senderBalHaqqAfter = res.Balance

				res2, err := s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil())
				Expect(res2.Balance).NotTo(BeNil())
				senderBalIslmAfter = res2.Balance

				if testcase.expTxPass {
					Expect(senderBalHaqqAfter.Amount.GT(senderBalHaqqBefore.Amount)).To(BeTrue(), "aHAQQ should be minted")
					Expect(senderBalIslmAfter.Amount).To(Equal(senderBalIslmBefore.Amount.Sub(fees).Sub(sdkmath.NewInt(1e18))), "aISLM should be burnt")
				} else {
					Expect(senderBalHaqqAfter.Amount).To(Equal(senderBalHaqqBefore.Amount), "aHAQQ balance should remain untouched")
					Expect(senderBalIslmAfter.Amount).To(Equal(senderBalIslmBefore.Amount.Sub(fees)), "aISLM balance should be reduced by fees only")
				}
			})

			It(fmt.Sprintf("emulate safe tx for calltype %q", testcase.calltype), func() {
				sender := s.keyring.GetKey(0)

				// fund contract before test from another account
				err := s.factory.FundAccount(s.keyring.GetKey(1), contractAddr.Bytes(), sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(1e18))))
				Expect(err).To(BeNil())
				Expect(s.network.NextBlock()).To(BeNil())

				// get initial balances of contract before the test
				res, err := s.grpcHandler.GetBalance(contractAddr.Bytes(), ethiqtypes.BaseDenom)
				Expect(err).To(BeNil())
				Expect(res.Balance).NotTo(BeNil())
				contractBalHaqqBefore := res.Balance

				res, err = s.grpcHandler.GetBalance(contractAddr.Bytes(), utils.BaseDenom)
				Expect(err).To(BeNil())
				Expect(res.Balance).NotTo(BeNil())
				contractBalIslmBefore := res.Balance

				callArgs.MethodName = "testCallMintHaqq"
				callArgs.Args = []interface{}{
					contractAddr, contractAddr, big.NewInt(1e18), testcase.calltype,
				}

				checkArgs := execRevertedCheck
				if testcase.expTxPass {
					checkArgs = passCheck.WithExpEvents(ethiq.EventTypeMintHaqq)
				}

				txArgs.GasPrice = gasPrice.BigInt()
				resCall, _, err := s.factory.CallContractAndCheckLogs(
					sender.Priv,
					txArgs, callArgs,
					checkArgs,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract for calltype %s: %v", testcase.calltype, err)
				Expect(s.network.NextBlock()).To(BeNil())

				fees := gasPrice.MulRaw(resCall.GasUsed)

				// check final balances
				res, err = s.grpcHandler.GetBalance(sender.AccAddr, ethiqtypes.BaseDenom)
				Expect(err).To(BeNil())
				Expect(res.Balance).NotTo(BeNil())
				senderBalHaqqAfter = res.Balance

				res, err = s.grpcHandler.GetBalance(sender.AccAddr, utils.BaseDenom)
				Expect(err).To(BeNil())
				Expect(res.Balance).NotTo(BeNil())
				senderBalIslmAfter = res.Balance

				res, err = s.grpcHandler.GetBalance(contractAddr.Bytes(), ethiqtypes.BaseDenom)
				Expect(err).To(BeNil())
				Expect(res.Balance).NotTo(BeNil())
				contractBalHaqqAfter := res.Balance

				res, err = s.grpcHandler.GetBalance(contractAddr.Bytes(), utils.BaseDenom)
				Expect(err).To(BeNil())
				Expect(res.Balance).NotTo(BeNil())
				contractBalIslmAfter := res.Balance

				if testcase.expTxPass {
					Expect(senderBalHaqqAfter.Amount.String()).To(Equal(senderBalHaqqBefore.Amount.String()), "sender's aHAQQ balance should remain untouched")
					Expect(contractBalHaqqAfter.Amount.GT(contractBalHaqqBefore.Amount)).To(BeTrue(), "aHAQQ should be minted to contract account balance")
					Expect(senderBalIslmAfter.Amount.String()).To(Equal(senderBalIslmBefore.Amount.Sub(fees).String()), "sender's aISLM should be reduced by fees")
					Expect(contractBalIslmAfter.Amount.String()).To(Equal(contractBalIslmBefore.Amount.Sub(sdkmath.NewInt(1e18)).String()), "contract aISLM should be burnt")
				} else {
					Expect(senderBalHaqqAfter.Amount.String()).To(Equal(senderBalHaqqBefore.Amount.String()), "sender's aHAQQ balance should remain untouched")
					Expect(contractBalHaqqAfter.Amount.String()).To(Equal(contractBalHaqqBefore.Amount.String()), "contract aHAQQ balance should remain untouched")
					Expect(senderBalIslmAfter.Amount.String()).To(Equal(senderBalIslmBefore.Amount.Sub(fees).String()), "sender's aISLM balance should be reduced by fees only")
					Expect(contractBalIslmAfter.Amount.String()).To(Equal(contractBalIslmBefore.Amount.String()), "contract aISLM balance should remain untouched")
				}
			})
		}

		It("when emulating call via Safe contract", func() {
		})
	})
})

// Full Safe (Smart Contract Wallet) flow integration tests.
var _ = Describe("Full Safe (Smart Contract Wallet) flow", Ordered, func() {
	const (
		// safeIntegrationOwnerBalanceAfterTransfersIslm is the expected per-owner and aggregate Safe
		// aISLM balance after each owner transfers 500 ISLM to the Safe (see transferToSafeAmount in specs).
		safeIntegrationOwnerBalanceAfterTransfersIslm int64 = 1000
		safeIntegrationWaitlistBurnIslm               int64 = 250
		// safeIntegrationOwnerMaxGasSpendIslm caps how much aISLM an owner may spend vs a baseline (gas for approve/exec paths).
		safeIntegrationOwnerMaxGasSpendIslm int64 = 1
	)

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
		expectedSafeBalance := sdkmath.NewInt(safeIntegrationOwnerBalanceAfterTransfersIslm).MulRaw(1e18)
		expectedParticipantFinalBalance := sdkmath.NewInt(safeIntegrationOwnerBalanceAfterTransfersIslm).MulRaw(1e18)

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
		Expect(ownerOneSpentForMint.LTE(sdkmath.NewInt(safeIntegrationOwnerMaxGasSpendIslm).MulRaw(1e18))).To(BeTrue(), "first owner balance change should be at most 1 ISLM")

		ownerTwoSpentForMint := ownerTwoBalanceBeforeMintRes.Balance.Amount.Sub(ownerTwoBalanceAfterMintRes.Balance.Amount)
		Expect(ownerTwoSpentForMint.IsNegative()).To(BeFalse(), "second owner balance should not increase after mint tx")
		Expect(ownerTwoSpentForMint.LTE(sdkmath.NewInt(safeIntegrationOwnerMaxGasSpendIslm).MulRaw(1e18))).To(BeTrue(), "second owner balance change should be at most 1 ISLM")

		safeHaqqDelta := safeHaqqAfterMintRes.Balance.Amount.Sub(safeHaqqBeforeMintRes.Balance.Amount)
		Expect(safeHaqqDelta.IsPositive()).To(BeTrue(), "Safe wallet should receive HAQQ tokens")

		safeIslmDelta := safeIslmBeforeMintRes.Balance.Amount.Sub(safeIslmAfterMintRes.Balance.Amount)
		Expect(safeIslmDelta).To(Equal(mintAmount), "Safe ISLM balance should decrease by 500 ISLM")

		Expect(gnosisSafeAddr).NotTo(BeZero(), "GnosisSafe singleton must be deployed in BeforeEach")
		Expect(proxyFactoryAddr).NotTo(BeZero(), "GnosisSafeProxyFactory must be deployed in BeforeEach")
		Expect(safeWalletAddr).NotTo(BeZero(), "Safe wallet must be created")
		Expect(ownerTwoBalanceBeforeMintRes.Balance.Amount).To(Equal(expectedParticipantFinalBalance), "second owner baseline before mint should be 1000 ISLM")
	})

	It("should execute full Safe waitlist flow against ethiq precompile", func() {
		initialParticipantBalance := sdkmath.NewInt(1500).MulRaw(1e18)
		transferToSafeAmount := sdkmath.NewInt(500).MulRaw(1e18)
		expectedSafeBalance := sdkmath.NewInt(safeIntegrationOwnerBalanceAfterTransfersIslm).MulRaw(1e18)
		expectedParticipantFinalBalance := sdkmath.NewInt(safeIntegrationOwnerBalanceAfterTransfersIslm).MulRaw(1e18)

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

		ownerOneSpentBeforeMint := expectedParticipantFinalBalance.Sub(ownerOneBalanceBeforeMintRes.Balance.Amount)
		Expect(ownerOneSpentBeforeMint.IsNegative()).To(BeFalse(), "first owner balance should not exceed post-transfer baseline before mint")
		Expect(ownerOneSpentBeforeMint.LTE(sdkmath.NewInt(safeIntegrationOwnerMaxGasSpendIslm).MulRaw(1e18))).To(BeTrue(), "first owner should only pay bounded gas (createProxy) on top of 1000 ISLM")
		Expect(safeIslmBeforeMintRes.Balance.Amount).To(Equal(expectedSafeBalance), "Safe ISLM before mint should be 1000 ISLM")
		Expect(safeHaqqBeforeMintRes.Balance.Amount.IsZero()).To(BeTrue(), "Safe HAQQ balance should be zero before mint")

		// Waitlist entry: sender and receiver are the Safe account. SOURCE_OF_FUNDS_BANK means the
		// keeper burns aISLM from that account's bank balance (the Safe's own coins), not from UCDAO escrow.
		waitlistAppIslm := sdkmath.NewInt(safeIntegrationWaitlistBurnIslm).MulRaw(1e18)
		safeBech32 := safeWalletAccAddr.String()
		waitlistItem := ethiqtypes.ApplicationListItem{
			FromAddress:                safeBech32,
			ToAddress:                  safeBech32,
			FundSource:                 ethiqtypes.SourceOfFunds_SOURCE_OF_FUNDS_BANK,
			IslmAmount:                 waitlistAppIslm.String(),
			IslmAccumulatedBurntAmount: "0",
		}
		_, err = waitlistItem.AsBurnApplication()
		Expect(err).ToNot(HaveOccurred(), "waitlist application item must be valid before push")

		waitlistAppID, restoreWaitlist := ethiqtypes.PushRegisteredApplicationForIntegrationTest(waitlistItem)
		defer restoreWaitlist()

		waitlistApplication, err := ethiqtypes.GetApplicationByID(waitlistAppID)
		Expect(err).ToNot(HaveOccurred(), "registered application must be readable by ID")
		Expect(waitlistApplication.Id).To(Equal(waitlistAppID))
		Expect(waitlistApplication.FromAddress).To(Equal(safeBech32))
		Expect(waitlistApplication.ToAddress).To(Equal(safeBech32))
		Expect(waitlistApplication.Source).To(Equal(ethiqtypes.SourceOfFunds_SOURCE_OF_FUNDS_BANK))
		Expect(waitlistApplication.BurnAmount.Denom).To(Equal(utils.BaseDenom))
		Expect(waitlistApplication.BurnAmount.Amount).To(Equal(waitlistAppIslm))
		Expect(waitlistApplication.BurnedBeforeAmount.Denom).To(Equal(utils.BaseDenom))
		Expect(waitlistApplication.BurnedBeforeAmount.Amount.IsZero()).To(BeTrue())
		Expect(waitlistApplication.IsCanceled).To(BeFalse())
		Expect(waitlistApplication.IsExecuted).To(BeFalse())

		Expect(ethiqtypes.TotalNumberOfApplicationsBySender(safeBech32)).To(Equal(uint64(1)), "sender index must list one application for Safe")
		senderApp, err := ethiqtypes.GetSendersApplicationIDByIndex(safeBech32, 0)
		Expect(err).ToNot(HaveOccurred())
		Expect(senderApp.Id).To(Equal(waitlistAppID))
		Expect(senderApp.FromAddress).To(Equal(safeBech32))
		Expect(senderApp.ToAddress).To(Equal(safeBech32))
		Expect(senderApp.Source).To(Equal(ethiqtypes.SourceOfFunds_SOURCE_OF_FUNDS_BANK))
		Expect(senderApp.BurnAmount).To(Equal(waitlistApplication.BurnAmount))
		Expect(senderApp.BurnedBeforeAmount).To(Equal(waitlistApplication.BurnedBeforeAmount))

		chainCtx := s.network.GetContext()
		Expect(s.network.App.EthiqKeeper.IsApplicationExecuted(chainCtx, waitlistAppID)).To(BeFalse(), "application must not be executed before Safe mintHaqqByApplication")

		calcRes, err := s.network.App.EthiqKeeper.CalculateForApplication(chainCtx, &ethiqtypes.QueryCalculateForApplicationRequest{ApplicationId: waitlistAppID})
		Expect(err).ToNot(HaveOccurred(), "failed to calculate expected HAQQ for application")
		expectedHaqqMinted := calcRes.EstimatedHaqqAmount

		precompileAddr := s.precompile.Address()
		approveAppIDArgs := factory.CallArgs{
			ContractABI: s.precompile.ABI,
			MethodName:  ethiq.ApproveApplicationIDMethod,
			Args: []interface{}{
				safeWalletAddr,
				new(big.Int).SetUint64(waitlistAppID),
				[]string{ethiq.MsgMintHaqqByApplicationMsgURL},
			},
		}
		approveTxArgs := evmtypes.EvmTxArgs{
			To: &precompileAddr,
		}
		_, _, err = s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			approveTxArgs,
			approveAppIDArgs,
			testutil.LogCheckArgs{
				ABIEvents: s.precompile.Events,
				ExpEvents: []string{authorization.EventTypeApproval},
				ExpPass:   true,
			},
		)
		Expect(err).ToNot(HaveOccurred(), "failed to approve mintHaqqByApplication for Safe")
		Expect(s.network.NextBlock()).ToNot(HaveOccurred(), "failed to advance block after approveApplicationID")

		mintByAppCallData, err := s.precompile.ABI.Pack(
			ethiq.MintHaqqByApplication,
			safeWalletAddr,
			new(big.Int).SetUint64(waitlistAppID),
		)
		Expect(err).ToNot(HaveOccurred(), "failed to pack mintHaqqByApplication call data")

		getTxHashArgs := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "getTransactionHash",
			Args: []interface{}{
				s.precompile.Address(),
				big.NewInt(0),
				mintByAppCallData,
				uint8(0), // CALL
				big.NewInt(300000),
				big.NewInt(0),
				big.NewInt(0),
				common.Address{},
				common.Address{},
				big.NewInt(0), // first execTransaction after createProxy
			},
		}
		_, txHashRes, err := s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &safeWalletAddr},
			getTxHashArgs,
			testutil.LogCheckArgs{ExpPass: true},
		)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe transaction hash for mintHaqqByApplication")
		txHashOutputs, err := gnosisSafe.ABI.Methods["getTransactionHash"].Outputs.Unpack(txHashRes.Ret)
		Expect(err).ToNot(HaveOccurred(), "failed to decode Safe transaction hash output")
		Expect(txHashOutputs).To(HaveLen(1))
		txHash, ok := txHashOutputs[0].([32]byte)
		Expect(ok).To(BeTrue(), "unexpected tx hash type")
		Expect(txHash).ToNot(Equal([32]byte{}), "Safe tx hash should not be zero")

		signature := make([]byte, 65)
		copy(signature[12:32], safeOwnerOne.Addr.Bytes())
		signature[64] = 1

		execTxArgs := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "execTransaction",
			Args: []interface{}{
				s.precompile.Address(),
				big.NewInt(0),
				mintByAppCallData,
				uint8(0),
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
		Expect(err).ToNot(HaveOccurred(), "failed to execute Safe mintHaqqByApplication transaction")
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

		Expect(s.network.NextBlock()).ToNot(HaveOccurred(), "failed to advance block after Safe mintHaqqByApplication")

		ownerOneBalanceAfterExecRes, err := s.grpcHandler.GetBalance(safeOwnerOne.AccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get first owner balance after Safe exec")
		ownerTwoBalanceAfterExecRes, err := s.grpcHandler.GetBalance(safeOwnerTwo.AccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get second owner balance after Safe exec")
		safeIslmAfterExecRes, err := s.grpcHandler.GetBalance(safeWalletAccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe ISLM balance after exec")
		safeHaqqAfterExecRes, err := s.grpcHandler.GetBalance(safeWalletAccAddr, ethiqtypes.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe HAQQ balance after exec")

		ownerOneSpentForExec := ownerOneBalanceBeforeMintRes.Balance.Amount.Sub(ownerOneBalanceAfterExecRes.Balance.Amount)
		Expect(ownerOneSpentForExec.IsNegative()).To(BeFalse(), "first owner balance should not increase after Safe exec")
		Expect(ownerOneSpentForExec.LTE(sdkmath.NewInt(safeIntegrationOwnerMaxGasSpendIslm).MulRaw(1e18))).To(BeTrue(), "first owner should spend at most 1 ISLM (gas) after baseline before exec")

		Expect(ownerTwoBalanceAfterExecRes.Balance.Amount).To(Equal(ownerTwoBalanceBeforeMintRes.Balance.Amount), "second owner balance must be unchanged")

		safeHaqqDelta := safeHaqqAfterExecRes.Balance.Amount.Sub(safeHaqqBeforeMintRes.Balance.Amount)
		Expect(safeHaqqDelta).To(Equal(expectedHaqqMinted), "Safe should receive calculated HAQQ amount")

		safeIslmDelta := safeIslmBeforeMintRes.Balance.Amount.Sub(safeIslmAfterExecRes.Balance.Amount)
		Expect(safeIslmDelta).To(Equal(waitlistAppIslm), "Safe ISLM balance should decrease by application burn amount")

		Expect(s.network.App.EthiqKeeper.IsApplicationExecuted(s.network.GetContext(), waitlistAppID)).To(BeTrue(), "application must be marked executed after mintHaqqByApplication")

		// Re-grant application ID authz (the first grant is deleted after a successful mintHaqqByApplication).
		_, _, err = s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			approveTxArgs,
			approveAppIDArgs,
			testutil.LogCheckArgs{
				ABIEvents: s.precompile.Events,
				ExpEvents: []string{authorization.EventTypeApproval},
				ExpPass:   true,
			},
		)
		Expect(err).ToNot(HaveOccurred(), "failed to re-approve mintHaqqByApplication for second Safe attempt")
		Expect(s.network.NextBlock()).ToNot(HaveOccurred(), "failed to advance block after second approveApplicationID")

		getTxHashArgsSecond := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "getTransactionHash",
			Args: []interface{}{
				s.precompile.Address(),
				big.NewInt(0),
				mintByAppCallData,
				uint8(0), // CALL
				big.NewInt(300000),
				big.NewInt(0),
				big.NewInt(0),
				common.Address{},
				common.Address{},
				big.NewInt(1), // Safe nonce after first successful execTransaction
			},
		}
		_, txHashResSecond, err := s.factory.CallContractAndCheckLogs(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &safeWalletAddr},
			getTxHashArgsSecond,
			testutil.LogCheckArgs{ExpPass: true},
		)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe transaction hash for second exec")
		txHashOutputsSecond, err := gnosisSafe.ABI.Methods["getTransactionHash"].Outputs.Unpack(txHashResSecond.Ret)
		Expect(err).ToNot(HaveOccurred(), "failed to decode second Safe transaction hash output")
		Expect(txHashOutputsSecond).To(HaveLen(1))
		txHashSecond, ok := txHashOutputsSecond[0].([32]byte)
		Expect(ok).To(BeTrue(), "unexpected second tx hash type")
		Expect(txHashSecond).ToNot(Equal([32]byte{}), "second Safe tx hash should not be zero")

		signatureSecond := make([]byte, 65)
		copy(signatureSecond[12:32], safeOwnerOne.Addr.Bytes())
		signatureSecond[64] = 1

		execTxArgsSecond := factory.CallArgs{
			ContractABI: gnosisSafe.ABI,
			MethodName:  "execTransaction",
			Args: []interface{}{
				s.precompile.Address(),
				big.NewInt(0),
				mintByAppCallData,
				uint8(0),
				big.NewInt(300000),
				big.NewInt(0),
				big.NewInt(0),
				common.Address{},
				common.Address{},
				signatureSecond,
			},
		}
		execSecondRes, err := s.factory.ExecuteContractCall(
			safeOwnerOne.Priv,
			evmtypes.EvmTxArgs{To: &safeWalletAddr},
			execTxArgsSecond,
		)
		Expect(err).ToNot(HaveOccurred(), "Safe outer tx should succeed even when inner call fails")
		execSecondEvmRes, err := s.factory.GetEvmTransactionResponseFromTxResult(execSecondRes)
		Expect(err).ToNot(HaveOccurred(), "failed to decode second execTransaction response")
		execOutputsSecond, err := gnosisSafe.ABI.Methods["execTransaction"].Outputs.Unpack(execSecondEvmRes.Ret)
		Expect(err).ToNot(HaveOccurred(), "failed to decode second execTransaction output")
		Expect(execOutputsSecond).To(HaveLen(1))
		execSuccessSecond, ok := execOutputsSecond[0].(bool)
		Expect(ok).To(BeTrue(), "unexpected second execTransaction output type")
		Expect(execSuccessSecond).To(BeFalse(), "inner mintHaqqByApplication must fail for an already executed application")

		executionFailureEvent := gnosisSafe.ABI.Events["ExecutionFailure"]
		executionFailureFound := false
		for i := range execSecondEvmRes.Logs {
			l := execSecondEvmRes.Logs[i]
			if len(l.Topics) == 0 {
				continue
			}
			if l.Topics[0] != executionFailureEvent.ID.String() {
				continue
			}
			if common.HexToAddress(l.Address) != safeWalletAddr {
				continue
			}
			executionFailureFound = true
			break
		}
		Expect(executionFailureFound).To(BeTrue(), "ExecutionFailure event should be emitted when inner call fails")

		Expect(s.network.NextBlock()).ToNot(HaveOccurred(), "failed to advance block after second exec")

		safeIslmAfterSecondAttemptRes, err := s.grpcHandler.GetBalance(safeWalletAccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe ISLM after second attempt")
		safeHaqqAfterSecondAttemptRes, err := s.grpcHandler.GetBalance(safeWalletAccAddr, ethiqtypes.BaseDenom)
		Expect(err).ToNot(HaveOccurred(), "failed to get Safe HAQQ after second attempt")
		Expect(safeIslmAfterSecondAttemptRes.Balance.Amount).To(Equal(safeIslmAfterExecRes.Balance.Amount), "Safe ISLM must be unchanged when second execution reverts")
		Expect(safeHaqqAfterSecondAttemptRes.Balance.Amount).To(Equal(safeHaqqAfterExecRes.Balance.Amount), "Safe HAQQ must be unchanged when second execution reverts")
		Expect(s.network.App.EthiqKeeper.IsApplicationExecuted(s.network.GetContext(), waitlistAppID)).To(BeTrue(), "application must stay executed after failed replay")

		Expect(gnosisSafeAddr).NotTo(BeZero(), "GnosisSafe singleton must be deployed in BeforeEach")
		Expect(proxyFactoryAddr).NotTo(BeZero(), "GnosisSafeProxyFactory must be deployed in BeforeEach")
		Expect(safeWalletAddr).NotTo(BeZero(), "Safe wallet must be created")
		Expect(ownerTwoBalanceBeforeMintRes.Balance.Amount).To(Equal(expectedParticipantFinalBalance), "second owner baseline before mint should be 1000 ISLM")
	})
})

// EOA waitlist application flow (no Safe): fund account, register BANK application, mint via precompile, reject replay.
var _ = Describe("EOA waitlist application flow", Ordered, func() {
	const (
		eoaWaitlistInitialFundIslm int64 = 1000
		eoaWaitlistBurnIslm        int64 = 250
		// eoaWaitlistRemainingIslmUpperExclusive is initialFund - burn in whole ISLM (balance stays below this once gas is paid).
		eoaWaitlistRemainingIslmUpperExclusive int64 = 750
		// eoaWaitlistMaxGasSpendIslm upper-bounds owner aISLM spent on tx gas vs the post-burn remainder (whole ISLM).
		eoaWaitlistMaxGasSpendIslm int64 = 1
	)

	var (
		s          *PrecompileTestSuite
		eoaAddr    common.Address
		eoaPriv    *ethsecp256k1.PrivKey
		eoaAccAddr sdk.AccAddress
	)

	BeforeEach(func() {
		s = new(PrecompileTestSuite)
		s.SetupTest()

		eoaAddr, eoaPriv = testutiltx.NewAddrKey()
		eoaAccAddr = sdk.AccAddress(eoaAddr.Bytes())

		fundAmount := sdkmath.NewInt(eoaWaitlistInitialFundIslm).MulRaw(1e18)
		Expect(s.network.FundAccountWithBaseDenom(eoaAccAddr, fundAmount)).ToNot(HaveOccurred())
		Expect(s.network.NextBlock()).ToNot(HaveOccurred())
	})

	It("should execute application from EOA and fail on replay", func() {
		initialIslm := sdkmath.NewInt(eoaWaitlistInitialFundIslm).MulRaw(1e18)
		bal0, err := s.grpcHandler.GetBalance(eoaAccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred())
		Expect(bal0.Balance.Amount).To(Equal(initialIslm))

		haqq0, err := s.grpcHandler.GetBalance(eoaAccAddr, ethiqtypes.BaseDenom)
		Expect(err).ToNot(HaveOccurred())
		Expect(haqq0.Balance.Amount.IsZero()).To(BeTrue())

		waitlistAppIslm := sdkmath.NewInt(eoaWaitlistBurnIslm).MulRaw(1e18)
		eoaBech32 := eoaAccAddr.String()
		waitlistItem := ethiqtypes.ApplicationListItem{
			FromAddress:                eoaBech32,
			ToAddress:                  eoaBech32,
			FundSource:                 ethiqtypes.SourceOfFunds_SOURCE_OF_FUNDS_BANK,
			IslmAmount:                 waitlistAppIslm.String(),
			IslmAccumulatedBurntAmount: "0",
		}
		_, err = waitlistItem.AsBurnApplication()
		Expect(err).ToNot(HaveOccurred())

		waitlistAppID, restoreWaitlist := ethiqtypes.PushRegisteredApplicationForIntegrationTest(waitlistItem)
		defer restoreWaitlist()

		precompileAddr := s.precompile.Address()
		mintByAppArgs := factory.CallArgs{
			ContractABI: s.precompile.ABI,
			MethodName:  ethiq.MintHaqqByApplication,
			Args: []interface{}{
				eoaAddr,
				new(big.Int).SetUint64(waitlistAppID),
			},
		}
		txArgsToPrecompile := evmtypes.EvmTxArgs{
			To: &precompileAddr,
		}

		_, err = s.factory.ExecuteContractCall(eoaPriv, txArgsToPrecompile, mintByAppArgs)
		Expect(err).ToNot(HaveOccurred(), "first mintHaqqByApplication from EOA should succeed")
		Expect(s.network.NextBlock()).ToNot(HaveOccurred())

		afterIslm, err := s.grpcHandler.GetBalance(eoaAccAddr, utils.BaseDenom)
		Expect(err).ToNot(HaveOccurred())
		afterHaqq, err := s.grpcHandler.GetBalance(eoaAccAddr, ethiqtypes.BaseDenom)
		Expect(err).ToNot(HaveOccurred())

		// After burn, remainder is at most eoaWaitlistRemainingIslmUpperExclusive ISLM; gas pulls it below that but above (upper - max gas).
		Expect(afterIslm.Balance.Amount.GT(
			sdkmath.NewInt(eoaWaitlistRemainingIslmUpperExclusive - eoaWaitlistMaxGasSpendIslm).MulRaw(1e18),
		)).To(BeTrue())
		Expect(afterIslm.Balance.Amount.LT(sdkmath.NewInt(eoaWaitlistRemainingIslmUpperExclusive).MulRaw(1e18))).To(BeTrue())
		Expect(afterHaqq.Balance.Amount.IsPositive()).To(BeTrue(), "EOA should receive minted aHAQQ")

		Expect(s.network.App.EthiqKeeper.IsApplicationExecuted(s.network.GetContext(), waitlistAppID)).To(BeTrue())

		_, err = s.factory.ExecuteContractCall(eoaPriv, txArgsToPrecompile, mintByAppArgs)
		Expect(err).To(HaveOccurred(), "second mintHaqqByApplication for the same application must fail")
		Expect(err.Error()).To(ContainSubstring("already executed"))

		Expect(s.network.NextBlock()).ToNot(HaveOccurred())
		Expect(s.network.App.EthiqKeeper.IsApplicationExecuted(s.network.GetContext(), waitlistAppID)).To(BeTrue(),
			"application must remain executed after failed EOA replay")
	})
})
