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
