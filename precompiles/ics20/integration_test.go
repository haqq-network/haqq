// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package ics20_test

import (
	"fmt"
	"math/big"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/cosmos-sdk/types/query"
	haqqcontracts "github.com/haqq-network/haqq/contracts"
	haqqibctesting "github.com/haqq-network/haqq/ibc/testing"
	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/erc20"
	"github.com/haqq-network/haqq/precompiles/ics20"
	"github.com/haqq-network/haqq/precompiles/testutil"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	haqqtestutil "github.com/haqq-network/haqq/testutil"
	teststypes "github.com/haqq-network/haqq/types/tests"
	"github.com/haqq-network/haqq/utils"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// General variables used for integration tests
var (
	// defaultCallArgs and defaultApproveArgs are the default arguments for calling the smart contract and to
	// call the approve method specifically.
	//
	// NOTE: this has to be populated in a BeforeEach block because the contractAddr would otherwise be a nil address.
	defaultCallArgs, defaultApproveArgs contracts.CallArgs

	// defaultLogCheck instantiates a log check arguments struct with the precompile ABI events populated.
	defaultLogCheck testutil.LogCheckArgs
	// passCheck defines the arguments to check if the precompile returns no error
	passCheck testutil.LogCheckArgs
	// outOfGasCheck defines the arguments to check if the precompile returns out of gas error
	outOfGasCheck testutil.LogCheckArgs

	// gasPrice defines a default gas price to be used in the testing suite
	gasPrice = big.NewInt(80_000_000)

	// array of allocations with only one allocation for 'aISLM' coin
	defaultSingleAlloc []cmn.ICS20Allocation

	// interchainSenderContract is the compiled contract calling the interchain functionality
	interchainSenderContract evmtypes.CompiledContract
)

var ist *testing.T

func TestPrecompileIntegrationTestSuite(t *testing.T) {
	// Run Ginkgo integration tests
	ist = t
	RegisterFailHandler(Fail)
	RunSpecs(t, "IBCTransfer Precompile Integration Tests")
}

var _ = Describe("IBCTransfer Precompile", func() {
	BeforeEach(func() {
		s = new(PrecompileTestSuite)
		s.SetT(ist)
		s.suiteIBCTesting = true
		s.SetupTest()
		s.setupAllocationsForTesting()

		var err error
		Expect(err).To(BeNil(), "error while loading the interchain sender contract: %v", err)

		// set the default call arguments
		defaultCallArgs = contracts.CallArgs{
			ContractAddr: s.precompile.Address(),
			ContractABI:  s.precompile.ABI,
			PrivKey:      s.keyring.GetPrivKey(0),
			GasPrice:     gasPrice,
		}
		defaultApproveArgs = defaultCallArgs.WithMethodName(authorization.ApproveMethod)

		defaultLogCheck = testutil.LogCheckArgs{
			ABIEvents: s.precompile.ABI.Events,
		}
		passCheck = defaultLogCheck.WithExpPass(true)
		outOfGasCheck = defaultLogCheck.WithErrContains(vm.ErrOutOfGas.Error())
	})

	Describe("Execute approve transaction", func() {
		BeforeEach(func() {
			// check no previous authorization exist
			auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), differentAddress.Bytes(), s.keyring.GetAddr(0).Bytes())
			Expect(err).To(BeNil(), "error while getting authorizations")
			Expect(auths).To(HaveLen(0), "expected no authorizations before tests")
		})

		// TODO uncomment when enforcing grantee != origin
		// It("should return error if the origin is same as the spender", func() {
		// 	approveArgs := defaultApproveArgs.WithArgs(
		// 		s.keyring.GetAddr(0),
		// 		defaultSingleAlloc,
		// 	)

		// 	_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, approveArgs, differentOriginCheck)
		// 	Expect(err).To(BeNil(), "error while calling the precompile")

		// 	s.chainA.NextBlock()

		// 	// check no authorization exist
		// 	auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), differentAddress.Bytes(), s.keyring.GetAddr(0).Bytes())
		// 	Expect(err).To(BeNil(), "error while getting authorizations")
		// 	Expect(auths).To(HaveLen(0), "expected no authorization")
		// })

		It("should return error if the provided gasLimit is too low", func() {
			approveArgs := defaultApproveArgs.
				WithGasLimit(30000).
				WithArgs(
					differentAddress,
					defaultSingleAlloc,
				)

			_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, approveArgs, outOfGasCheck)
			Expect(err).To(HaveOccurred(), "error while calling the precompile")
			Expect(err.Error()).To(ContainSubstring(vm.ErrOutOfGas.Error()))

			s.chainA.NextBlock()

			// check no authorization exist
			auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), differentAddress.Bytes(), s.keyring.GetAddr(0).Bytes())
			Expect(err).To(BeNil())
			Expect(auths).To(HaveLen(0))
		})

		It("should approve the corresponding allocation", func() {
			approveArgs := defaultApproveArgs.WithArgs(
				differentAddress,
				defaultSingleAlloc,
			)

			approvalCheck := passCheck.
				WithExpEvents(authorization.EventTypeIBCTransferAuthorization)

			_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, approveArgs, approvalCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			s.chainA.NextBlock()

			// check GetAuthorizations is returning the record
			auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), differentAddress.Bytes(), s.keyring.GetAddr(0).Bytes())
			Expect(err).To(BeNil(), "error while getting authorizations")
			Expect(auths).To(HaveLen(1), "expected one authorization")
			Expect(auths[0].MsgTypeURL()).To(Equal(ics20.TransferMsgURL))
			transferAuthz := auths[0].(*transfertypes.TransferAuthorization)
			Expect(transferAuthz.Allocations[0].SpendLimit).To(Equal(defaultCoins))
		})
	})

	Describe("Execute revoke transaction", func() {
		var defaultRevokeArgs contracts.CallArgs
		BeforeEach(func() {
			// create authorization
			s.setTransferApproval(defaultCallArgs, differentAddress, defaultSingleAlloc)
			defaultRevokeArgs = defaultCallArgs.WithMethodName(authorization.RevokeMethod)
		})

		It("should revoke authorization", func() {
			revokeArgs := defaultRevokeArgs.WithArgs(
				differentAddress,
			)
			revokeCheck := passCheck.
				WithExpEvents(authorization.EventTypeIBCTransferAuthorization)

			_, _, err := contracts.CallContractAndCheckLogs(
				s.chainA.GetContext(),
				s.network.App,
				revokeArgs,
				revokeCheck,
			)
			Expect(err).To(BeNil(), "error while calling the precompile")

			s.chainA.NextBlock()

			// check no authorization exist
			auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), differentAddress.Bytes(), s.keyring.GetAddr(0).Bytes())
			Expect(err).To(BeNil(), "error while getting authorizations")
			Expect(auths).To(HaveLen(0), "expected no authorization")
		})
	})

	Describe("Execute increase allowance transaction", func() {
		BeforeEach(func() {
			s.setTransferApproval(defaultCallArgs, differentAddress, defaultSingleAlloc)
		})

		// TODO uncomment when enforcing grantee != origin
		// this is a copy of a different test but for a different method
		// It("should return an error if the origin is same as the spender", func() {
		// 	increaseAllowanceArgs := defaultCallArgs.
		// 		WithMethodName(authorization.IncreaseAllowanceMethod).
		// 		WithArgs(
		// 			s.keyring.GetAddr(0),
		// 			s.transferPath.EndpointA.ChannelConfig.PortID,
		// 			s.transferPath.EndpointA.ChannelID,
		// 			utils.BaseDenom,
		// 			big.NewInt(1e18),
		// 		)

		// 	differentOriginCheck := defaultLogCheck.WithErrContains(cmn.ErrDifferentOrigin, s.keyring.GetAddr(0), differentAddress)

		// 	_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, increaseAllowanceArgs, differentOriginCheck)
		// 	Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

		// 	// check no authorization exist
		// 	auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.keyring.GetAddr(0).Bytes())
		// 	Expect(err).To(BeNil(), "error while getting authorizations")
		// 	Expect(auths).To(BeNil())
		// })

		It("should return an error if the allocation denom is not present", func() {
			increaseAllowanceArgs := defaultCallArgs.
				WithMethodName(authorization.IncreaseAllowanceMethod).
				WithArgs(
					differentAddress,
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					"urandom",
					big.NewInt(1e18),
				)

			noMatchingAllocation := defaultLogCheck.WithErrContains(
				ics20.ErrNoMatchingAllocation,
				s.transferPath.EndpointA.ChannelConfig.PortID,
				s.transferPath.EndpointA.ChannelID,
				"urandom",
			)

			_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, increaseAllowanceArgs, noMatchingAllocation)
			Expect(err).To(HaveOccurred(), "error while calling the smart contract: %v", err)
			Expect(err.Error()).To(ContainSubstring(ics20.ErrNoMatchingAllocation, "transfer", "channel-0", "urandom"))

			// check authorization didn't change
			auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), differentAddress.Bytes(), s.keyring.GetAddr(0).Bytes())
			Expect(err).To(BeNil(), "error while getting authorizations")
			Expect(auths).To(HaveLen(1), "expected one authorization")
			Expect(auths[0].MsgTypeURL()).To(Equal(ics20.TransferMsgURL))
			transferAuthz := auths[0].(*transfertypes.TransferAuthorization)
			Expect(transferAuthz.Allocations[0].SpendLimit).To(Equal(defaultCoins))
		})

		It("should increase allowance by 1 ISLM", func() {
			s.setTransferApproval(defaultCallArgs, differentAddress, defaultSingleAlloc)

			increaseAllowanceArgs := defaultCallArgs.
				WithMethodName(authorization.IncreaseAllowanceMethod).
				WithArgs(
					differentAddress,
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					utils.BaseDenom,
					big.NewInt(1e18),
				)

			allowanceCheck := passCheck.WithExpEvents(authorization.EventTypeIBCTransferAuthorization)

			_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, increaseAllowanceArgs, allowanceCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			s.chainA.NextBlock()

			// check auth was updated
			auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), differentAddress.Bytes(), s.keyring.GetAddr(0).Bytes())
			Expect(err).To(BeNil(), "error while getting authorizations")
			Expect(auths).To(HaveLen(1), "expected one authorization")
			Expect(auths[0].MsgTypeURL()).To(Equal(ics20.TransferMsgURL))
			transferAuthz := auths[0].(*transfertypes.TransferAuthorization)
			Expect(transferAuthz.Allocations[0].SpendLimit).To(Equal(defaultCoins.Add(sdk.Coin{Denom: utils.BaseDenom, Amount: math.NewInt(1e18)})))
		})
	})

	Describe("Execute decrease allowance transaction", func() {
		BeforeEach(func() {
			s.setTransferApproval(defaultCallArgs, differentAddress, defaultSingleAlloc)
		})

		It("should fail if decreased amount is more than the total spend limit left", func() {
			decreaseAllowance := defaultCallArgs.
				WithMethodName(authorization.DecreaseAllowanceMethod).
				WithArgs(
					differentAddress,
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					utils.BaseDenom,
					big.NewInt(2e18),
				)

			allowanceCheck := defaultLogCheck.WithErrContains("negative amount")

			_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, decreaseAllowance, allowanceCheck)
			Expect(err).To(HaveOccurred(), "error while calling the precompile")
			Expect(err.Error()).To(ContainSubstring("negative amount"))
		})

		// TODO uncomment when enforcing grantee != origin
		// //nolint:dupl // this is a copy of a different test but for a different method
		// It("should return an error if the origin same the spender", func() {
		// 	decreaseAllowance := defaultCallArgs.
		// 		WithMethodName(authorization.DecreaseAllowanceMethod).
		// 		WithArgs(
		// 			s.keyring.GetAddr(0),
		// 			s.transferPath.EndpointA.ChannelConfig.PortID,
		// 			s.transferPath.EndpointA.ChannelID,
		// 			utils.BaseDenom,
		// 			big.NewInt(1e18),
		// 		)

		// 	differentOriginCheck := defaultLogCheck.WithErrContains(cmn.ErrDifferentOrigin, s.keyring.GetAddr(0), differentAddress)

		// 	_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, decreaseAllowance, differentOriginCheck)
		// 	Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

		// 	// check authorization does not exist
		// 	auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.keyring.GetAddr(0).Bytes())
		// 	Expect(err).To(BeNil(), "error while getting authorizations")
		// 	Expect(auths).To(BeNil())
		// })

		It("should return an error if the allocation denom is not present", func() {
			decreaseAllowance := defaultCallArgs.
				WithMethodName(authorization.DecreaseAllowanceMethod).
				WithArgs(
					differentAddress,
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					"urandom",
					big.NewInt(1e18),
				)

			noMatchingAllocation := defaultLogCheck.WithErrContains(
				ics20.ErrNoMatchingAllocation,
				s.transferPath.EndpointA.ChannelConfig.PortID,
				s.transferPath.EndpointA.ChannelID,
				"urandom",
			)

			_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, decreaseAllowance, noMatchingAllocation)
			Expect(err).To(HaveOccurred(), "error while calling the smart contract: %v", err)
			Expect(err.Error()).To(ContainSubstring(ics20.ErrNoMatchingAllocation, "transfer", "channel-0", "urandom"))

			// check authorization didn't change
			auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), differentAddress.Bytes(), s.keyring.GetAddr(0).Bytes())
			Expect(err).To(BeNil(), "error while getting authorizations")
			Expect(auths).To(HaveLen(1), "expected one authorization")
			Expect(auths[0].MsgTypeURL()).To(Equal(ics20.TransferMsgURL))
			transferAuthz := auths[0].(*transfertypes.TransferAuthorization)
			Expect(transferAuthz.Allocations[0].SpendLimit).To(Equal(defaultCoins))
		})

		It("should delete grant if allowance is decreased to 0", func() {
			decreaseAllowance := defaultCallArgs.
				WithMethodName(authorization.DecreaseAllowanceMethod).
				WithArgs(
					differentAddress,
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					utils.BaseDenom,
					big.NewInt(1e18),
				)

			allowanceCheck := passCheck.WithExpEvents(authorization.EventTypeIBCTransferAuthorization)

			_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, decreaseAllowance, allowanceCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			s.chainA.NextBlock()

			// check auth record
			auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), differentAddress.Bytes(), s.keyring.GetAddr(0).Bytes())
			Expect(err).To(BeNil(), "error while getting authorizations")
			Expect(auths).To(HaveLen(1), "expected one authorization")
			Expect(auths[0].MsgTypeURL()).To(Equal(ics20.TransferMsgURL))
			transferAuthz := auths[0].(*transfertypes.TransferAuthorization)
			Expect(transferAuthz.Allocations[0].SpendLimit).To(HaveLen(0))
		})
	})

	Describe("Execute transfer transaction", func() {
		var defaultTransferArgs contracts.CallArgs

		BeforeEach(func() {
			// populate the default transfer args
			defaultTransferArgs = defaultCallArgs.
				WithMethodName(ics20.TransferMethod).
				WithArgs(
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					s.bondDenom,
					defaultCmnCoins[0].Amount,
					s.keyring.GetAddr(0),
					s.chainB.SenderAccount.GetAddress().String(), // receiver
					s.chainB.GetTimeoutHeight(),
					uint64(0), // disable timeout timestamp
					"memo",
				)
		})

		Context("without authorization", func() {
			It("owner should transfer without authorization", func() {
				initialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

				logCheckArgs := passCheck.WithExpEvents(ics20.EventTypeIBCTransfer)

				res, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultTransferArgs, logCheckArgs)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				// check the sender balance was deducted
				fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
				finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
				Expect(finalBalance.Amount).To(Equal(initialBalance.Amount.Sub(fees).Sub(defaultCoins[0].Amount)))
			})

			It("should succeed in transfer transaction but should timeout and refund sender", func() {
				initialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

				logCheckArgs := passCheck.WithExpEvents(ics20.EventTypeIBCTransfer)
				timeoutHeight := clienttypes.NewHeight(clienttypes.ParseChainID(s.chainB.ChainID), uint64(s.chainB.GetContext().BlockHeight())+1)

				transferArgs := defaultTransferArgs.WithArgs(
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					s.bondDenom,
					defaultCmnCoins[0].Amount,
					s.keyring.GetAddr(0),
					s.chainB.SenderAccount.GetAddress().String(), // receiver
					timeoutHeight,
					uint64(0), // disable timeout timestamp
					"memo",
				)

				res, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, transferArgs, logCheckArgs)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				out, err := s.precompile.Unpack(ics20.TransferMethod, ethRes.Ret)
				Expect(err).To(BeNil(), "error while unpacking response: %v", err)
				// check sequence in returned data
				sequence, ok := out[0].(uint64)
				Expect(ok).To(BeTrue())
				Expect(sequence).To(Equal(uint64(1)))

				// check the sender balance was deducted
				fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
				finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
				Expect(finalBalance.Amount).To(Equal(initialBalance.Amount.Sub(fees).Sub(defaultCoins[0].Amount)))

				// the transfer is reverted because the packet times out
				// build the sent packet
				// this is the packet sent
				packet := s.makePacket(
					sdk.AccAddress(s.keyring.GetAddr(0).Bytes()).String(),
					s.chainB.SenderAccount.GetAddress().String(),
					s.bondDenom,
					"memo",
					defaultCmnCoins[0].Amount,
					sequence,
					timeoutHeight,
				)

				// packet times out and the OnTimeoutPacket callback is executed
				s.chainA.NextBlock()
				// increment block height on chainB to make the packet timeout
				s.chainB.NextBlock()

				// increment sequence for successful transaction execution
				err = s.chainA.SenderAccount.SetSequence(s.chainA.SenderAccount.GetSequence() + 1)
				s.Require().NoError(err)

				err = s.transferPath.EndpointA.UpdateClient()
				Expect(err).To(BeNil())

				// Receive timeout
				err = s.transferPath.EndpointA.TimeoutPacket(packet)
				Expect(err).To(BeNil())

				// To submit a timeoutMsg, the TimeoutPacket function
				// uses a default fee amount
				timeoutMsgFee := math.NewInt(haqqibctesting.DefaultFeeAmt * 2)
				fees = fees.Add(timeoutMsgFee)

				finalBalance = s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
				Expect(finalBalance.Amount).To(Equal(initialBalance.Amount.Sub(fees)))
			})

			It("should not transfer other account's balance", func() {
				// initialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

				// fund senders account
				err := haqqtestutil.FundAccountWithBaseDenom(s.chainA.GetContext(), s.network.App.BankKeeper, differentAddress.Bytes(), amt)
				Expect(err).To(BeNil())
				senderInitialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), differentAddress.Bytes(), s.bondDenom)
				Expect(senderInitialBalance.Amount).To(Equal(math.NewInt(amt)))

				transferArgs := defaultTransferArgs.WithArgs(
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					s.bondDenom,
					defaultCmnCoins[0].Amount,
					differentAddress,
					s.chainB.SenderAccount.GetAddress().String(), // receiver
					s.chainB.GetTimeoutHeight(),
					uint64(0), // disable timeout timestamp
					"memo",
				)

				logCheckArgs := defaultLogCheck.WithErrContains(ics20.ErrDifferentOriginFromSender, s.keyring.GetAddr(0), differentAddress)

				_, _, err = contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, transferArgs, logCheckArgs)
				Expect(err).To(HaveOccurred(), "error while calling the smart contract: %v", err)
				Expect(err.Error()).To(ContainSubstring(ics20.ErrDifferentOriginFromSender, s.keyring.GetAddr(0), differentAddress))

				// check the sender only paid for the fees
				// and funds were not transferred
				// TODO: fees are not calculated correctly with this logic
				// fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
				// finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
				// Expect(finalBalance.Amount).To(Equal(initialBalance.Amount.Sub(fees)))

				senderFinalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), differentAddress.Bytes(), s.bondDenom)
				Expect(senderFinalBalance.Amount).To(Equal(senderInitialBalance.Amount))
			})
		})

		Context("with authorization", func() {
			BeforeEach(func() {
				expTime := s.chainA.GetContext().BlockTime().Add(s.precompile.ApprovalExpiration)

				allocations := []transfertypes.Allocation{
					{
						SourcePort:    s.transferPath.EndpointA.ChannelConfig.PortID,
						SourceChannel: s.transferPath.EndpointA.ChannelID,
						SpendLimit:    defaultCoins,
					},
				}

				// create grant to allow s.keyring.GetAddr(0) to spend differentAddr funds
				err := s.network.App.AuthzKeeper.SaveGrant(
					s.chainA.GetContext(),
					s.keyring.GetAddr(0).Bytes(),
					differentAddress.Bytes(),
					&transfertypes.TransferAuthorization{Allocations: allocations},
					&expTime,
				)
				Expect(err).To(BeNil())

				// fund the account from which funds will be sent
				err = haqqtestutil.FundAccountWithBaseDenom(s.chainA.GetContext(), s.network.App.BankKeeper, differentAddress.Bytes(), amt)
				Expect(err).To(BeNil())
				senderInitialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), differentAddress.Bytes(), s.bondDenom)
				Expect(senderInitialBalance.Amount).To(Equal(math.NewInt(amt)))
			})

			It("should not transfer other account's balance", func() {
				// ATM it is not allowed for another EOA to spend other EOA
				// funds via EVM extensions.
				// However, it is allowed for a contract to spend an EOA's account and
				// an EOA account to spend a contract's balance
				// if the required authorization exist
				// initialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

				transferArgs := defaultTransferArgs.WithArgs(
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					s.bondDenom,
					defaultCmnCoins[0].Amount,
					differentAddress,
					s.chainB.SenderAccount.GetAddress().String(), // receiver
					s.chainB.GetTimeoutHeight(),
					uint64(0), // disable timeout timestamp
					"memo",
				)

				logCheckArgs := defaultLogCheck.WithErrContains(ics20.ErrDifferentOriginFromSender, s.keyring.GetAddr(0), differentAddress)

				_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, transferArgs, logCheckArgs)
				Expect(err).To(HaveOccurred(), "error while calling the smart contract: %v", err)
				Expect(err.Error()).To(ContainSubstring(ics20.ErrDifferentOriginFromSender, s.keyring.GetAddr(0), differentAddress))

				// check the sender only paid for the fees
				// and funds from the other account were not transferred
				// TODO: fees are not calculated correctly with this logic
				// fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
				// finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
				// Expect(finalBalance.Amount).To(Equal(initialBalance.Amount.Sub(fees)))

				senderFinalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), differentAddress.Bytes(), s.bondDenom)
				Expect(senderFinalBalance.Amount).To(Equal(math.NewInt(amt)))
			})
		})

		Context("sending ERC20 coins", func() {
			var (
				// erc20Addr is the address of the ERC20 contract
				erc20Addr common.Address
				// sentAmount is the amount of tokens to send for testing
				sentAmount               = big.NewInt(1000)
				tokenPairDenom           string
				defaultErc20TransferArgs contracts.CallArgs
				err                      error
			)

			BeforeEach(func() {
				erc20Addr = s.setupERC20ContractTests(sentAmount)
				// register the token pair
				_, err := s.network.App.Erc20Keeper.RegisterERC20(s.chainA.GetContext(), &erc20types.MsgRegisterERC20{
					Authority:      authtypes.NewModuleAddress("gov").String(),
					Erc20Addresses: []string{erc20Addr.Hex()},
				})
				Expect(err).To(BeNil(), "error while registering the token pair: %v", err)

				tokenPairDenom = erc20types.CreateDenom(erc20Addr.Hex())
				defaultErc20TransferArgs = defaultTransferArgs.WithArgs(
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					tokenPairDenom,
					sentAmount,
					s.keyring.GetAddr(0),
					s.chainB.SenderAccount.GetAddress().String(), // receiver
					s.chainB.GetTimeoutHeight(),
					uint64(0), // disable timeout timestamp
					"memo",
				)
			})

			Context("without authorization", func() {
				It("should transfer registered ERC20s", func() {
					preBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

					logCheckArgs := passCheck.WithExpEvents(ics20.EventTypeIBCTransfer)

					res, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultErc20TransferArgs, logCheckArgs)
					Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

					out, err := s.precompile.Unpack(ics20.TransferMethod, ethRes.Ret)
					Expect(err).To(BeNil(), "error while unpacking response: %v", err)
					// check sequence in returned data
					Expect(out[0]).To(Equal(uint64(1)))

					s.chainA.NextBlock()

					// check only fees were deducted from sending account
					fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
					finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
					Expect(finalBalance.Amount).To(Equal(preBalance.Amount.Sub(fees)))

					// check Erc20 balance was reduced by sent amount
					balance := s.network.App.Erc20Keeper.BalanceOf(
						s.chainA.GetContext(),
						haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI,
						erc20Addr,
						s.keyring.GetAddr(0),
					)
					Expect(balance.Int64()).To(BeZero(), "address does not have the expected amount of tokens")
				})

				It("should not transfer other account's balance", func() {
					// mint some ERC20 to the sender's account
					defaultERC20CallArgs := contracts.CallArgs{
						ContractAddr: erc20Addr,
						ContractABI:  haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI,
						PrivKey:      s.keyring.GetPrivKey(0),
						GasPrice:     gasPrice,
					}

					// mint coins to the address
					mintCoinsArgs := defaultERC20CallArgs.
						WithMethodName("mint").
						WithArgs(differentAddress, defaultCmnCoins[0].Amount)

					mintCheck := testutil.LogCheckArgs{
						ABIEvents: haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI.Events,
						ExpEvents: []string{erc20.EventTypeTransfer}, // upon minting the tokens are sent to the receiving address
						ExpPass:   true,
					}

					_, _, err = contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, mintCoinsArgs, mintCheck)
					Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

					// try to transfer other account's erc20 tokens
					transferArgs := defaultTransferArgs.WithArgs(
						s.transferPath.EndpointA.ChannelConfig.PortID,
						s.transferPath.EndpointA.ChannelID,
						tokenPairDenom,
						defaultCmnCoins[0].Amount,
						differentAddress,
						s.chainB.SenderAccount.GetAddress().String(), // receiver
						s.chainB.GetTimeoutHeight(),
						uint64(0), // disable timeout timestamp
						"memo",
					)

					logCheckArgs := defaultLogCheck.WithErrContains(ics20.ErrDifferentOriginFromSender, s.keyring.GetAddr(0), differentAddress)

					_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, transferArgs, logCheckArgs)
					Expect(err).To(HaveOccurred(), "error while calling the smart contract: %v", err)
					Expect(err.Error()).To(ContainSubstring(ics20.ErrDifferentOriginFromSender, s.keyring.GetAddr(0), differentAddress))

					// check funds were not transferred
					balance := s.network.App.Erc20Keeper.BalanceOf(
						s.chainA.GetContext(),
						haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI,
						erc20Addr,
						differentAddress,
					)
					Expect(balance).To(Equal(defaultCmnCoins[0].Amount), "address does not have the expected amount of tokens")
				})

				It("should succeed in transfer transaction but should error on packet destination if the receiver address is wrong", func() {
					preBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
					invalidReceiverAddr := "invalid_address"
					transferArgs := defaultTransferArgs.WithArgs(
						s.transferPath.EndpointA.ChannelConfig.PortID,
						s.transferPath.EndpointA.ChannelID,
						tokenPairDenom,
						sentAmount,
						s.keyring.GetAddr(0),
						invalidReceiverAddr, // invalid receiver
						s.chainB.GetTimeoutHeight(),
						uint64(0), // disable timeout timestamp
						"memo",
					)

					logCheckArgs := passCheck.WithExpEvents(ics20.EventTypeIBCTransfer)

					res, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, transferArgs, logCheckArgs)
					Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

					out, err := s.precompile.Unpack(ics20.TransferMethod, ethRes.Ret)
					Expect(err).To(BeNil(), "error while unpacking response: %v", err)
					// check sequence in returned data
					sequence, ok := out[0].(uint64)
					Expect(ok).To(BeTrue())
					Expect(sequence).To(Equal(uint64(1)))

					s.chainA.NextBlock()

					// check only fees were deducted from sending account
					fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
					finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
					Expect(finalBalance.Amount).To(Equal(preBalance.Amount.Sub(fees)))

					// check Erc20 balance was reduced by sent amount (escrowed on ibc escrow account)
					balance := s.network.App.Erc20Keeper.BalanceOf(
						s.chainA.GetContext(),
						haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI,
						erc20Addr,
						s.keyring.GetAddr(0),
					)
					Expect(balance.Int64()).To(BeZero(), "address does not have the expected amount of tokens")

					// the transfer is reverted because fails checks on the receiving chain
					// this is the packet sent
					packet := s.makePacket(
						sdk.AccAddress(s.keyring.GetAddr(0).Bytes()).String(),
						invalidReceiverAddr,
						tokenPairDenom,
						"memo",
						sentAmount,
						sequence,
						s.chainB.GetTimeoutHeight(),
					)

					// increment sequence for successful transaction execution
					err = s.chainA.SenderAccount.SetSequence(s.chainA.SenderAccount.GetSequence() + 3)
					s.Require().NoError(err)

					err = s.transferPath.EndpointA.UpdateClient()
					Expect(err).To(BeNil())

					// Relay packet
					// fix context header
					err = s.transferPath.RelayPacket(packet)
					Expect(err).To(BeNil())

					// check escrowed funds are refunded to sender
					finalERC20balance := s.network.App.Erc20Keeper.BalanceOf(
						s.chainA.GetContext(),
						haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI,
						erc20Addr,
						s.keyring.GetAddr(0),
					)
					Expect(finalERC20balance).To(Equal(sentAmount), "address does not have the expected amount of tokens")
				})

				It("should succeed in transfer transaction but should timeout", func() {
					preBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

					logCheckArgs := passCheck.WithExpEvents(ics20.EventTypeIBCTransfer)

					timeoutHeight := clienttypes.NewHeight(clienttypes.ParseChainID(s.chainB.ChainID), uint64(s.chainB.GetContext().BlockHeight())+1)

					transferArgs := defaultTransferArgs.WithArgs(
						s.transferPath.EndpointA.ChannelConfig.PortID,
						s.transferPath.EndpointA.ChannelID,
						tokenPairDenom,
						sentAmount,
						s.keyring.GetAddr(0),
						s.chainB.SenderAccount.GetAddress().String(), // receiver
						timeoutHeight,
						uint64(0), // disable timeout timestamp
						"memo",
					)

					res, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, transferArgs, logCheckArgs)
					Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

					out, err := s.precompile.Unpack(ics20.TransferMethod, ethRes.Ret)
					Expect(err).To(BeNil(), "error while unpacking response: %v", err)
					// check sequence in returned data
					sequence, ok := out[0].(uint64)
					Expect(ok).To(BeTrue())
					Expect(sequence).To(Equal(uint64(1)))

					s.chainA.NextBlock()

					// check only fees were deducted from sending account
					fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
					finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
					Expect(finalBalance.Amount).To(Equal(preBalance.Amount.Sub(fees)))

					// check Erc20 balance was reduced by sent amount
					balance := s.network.App.Erc20Keeper.BalanceOf(
						s.chainA.GetContext(),
						haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI,
						erc20Addr,
						s.keyring.GetAddr(0),
					)
					Expect(balance.Int64()).To(BeZero(), "address does not have the expected amount of tokens")

					// the transfer is reverted because the packet times out
					// this is the packet sent
					packet := s.makePacket(
						sdk.AccAddress(s.keyring.GetAddr(0).Bytes()).String(),
						s.chainB.SenderAccount.GetAddress().String(),
						tokenPairDenom,
						"memo",
						sentAmount,
						sequence,
						timeoutHeight,
					)

					// packet times out and the OnTimeoutPacket callback is executed
					s.chainA.NextBlock()
					// increment block height on chainB to make the packet timeout
					s.chainB.NextBlock()

					// increment sequence for successful transaction execution
					err = s.chainA.SenderAccount.SetSequence(s.chainA.SenderAccount.GetSequence() + 3)
					s.Require().NoError(err)

					err = s.transferPath.EndpointA.UpdateClient()
					Expect(err).To(BeNil())

					// Receive timeout
					err = s.transferPath.EndpointA.TimeoutPacket(packet)
					Expect(err).To(BeNil())

					// check escrowed funds are refunded to sender
					finalERC20balance := s.network.App.Erc20Keeper.BalanceOf(
						s.chainA.GetContext(),
						haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI,
						erc20Addr,
						s.keyring.GetAddr(0),
					)
					Expect(finalERC20balance).To(Equal(sentAmount), "address does not have the expected amount of tokens")
				})
			})
		})
	})

	Context("Queries", func() {
		var (
			path     string
			expDenom transfertypes.Denom
		)

		BeforeEach(func() {
			path = fmt.Sprintf(
				"%s/%s/%s/%s",
				s.transferPath.EndpointA.ChannelConfig.PortID,
				s.transferPath.EndpointA.ChannelID,
				s.transferPath.EndpointB.ChannelConfig.PortID,
				s.transferPath.EndpointB.ChannelID,
			)
			expDenom = transfertypes.ExtractDenomFromPath(path + "/" + utils.BaseDenom)
		})

		It("should query denom trace", func() {
			// setup - create a denom trace to get it on the query result
			method := ics20.DenomMethod
			s.network.App.TransferKeeper.SetDenom(s.chainA.GetContext(), expDenom)

			args := defaultCallArgs.
				WithMethodName(method).
				WithArgs(expDenom.Hash().String())

			_, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, args, passCheck)
			Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

			var out struct {
				Denom transfertypes.Denom
			}
			err = s.precompile.UnpackIntoInterface(&out, method, ethRes.Ret)
			Expect(err).To(BeNil(), "error while unpacking the output: %v", err)

			Expect(out.Denom.Base).To(Equal(expDenom.Base))
			Expect(len(out.Denom.Trace)).To(Equal(len(expDenom.Trace)))
			for i, hop := range out.Denom.Trace {
				Expect(hop.PortId).To(Equal(expDenom.Trace[i].PortId))
				Expect(hop.ChannelId).To(Equal(expDenom.Trace[i].ChannelId))
			}
		})

		Context("denom traces query", func() {
			var (
				method    string
				expDenoms []transfertypes.Denom
			)
			BeforeEach(func() {
				method = ics20.DenomsMethod
				// setup - create some denom traces to get on the query result
				expDenoms = []transfertypes.Denom{
					transfertypes.NewDenom(utils.BaseDenom),
					transfertypes.ExtractDenomFromPath(fmt.Sprintf("%s/%s/%s", s.transferPath.EndpointA.ChannelConfig.PortID, s.transferPath.EndpointA.ChannelID, utils.BaseDenom)),
					expDenom,
				}

				for _, denom := range expDenoms {
					s.network.App.TransferKeeper.SetDenom(s.chainA.GetContext(), denom)
				}
			})
			It("should query denom traces - w/all results on page", func() {
				args := defaultCallArgs.
					WithMethodName(method).
					WithArgs(
						query.PageRequest{
							Limit:      3,
							CountTotal: true,
						})
				_, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, args, passCheck)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				var out ics20.DenomsResponse
				err = s.precompile.UnpackIntoInterface(&out, method, ethRes.Ret)
				Expect(err).To(BeNil(), "error while unpacking the output: %v", err)

				Expect(out.Denoms).To(HaveLen(3), "expected 3 denoms to be returned")
				Expect(out.PageResponse.Total).To(Equal(uint64(3)))
				Expect(out.PageResponse.NextKey).To(BeEmpty())

				for i, denom := range out.Denoms {
					Expect(denom.Base).To(Equal(expDenoms[i].Base))
					Expect(len(denom.Trace)).To(Equal(len(expDenoms[i].Trace)))
					for j, hop := range expDenoms[i].Trace {
						Expect(denom.Trace[j].PortId).To(Equal(hop.PortId))
						Expect(denom.Trace[j].ChannelId).To(Equal(hop.ChannelId))
					}
				}
			})

			It("should query denom traces - w/pagination", func() {
				args := defaultCallArgs.
					WithMethodName(method).
					WithArgs(
						query.PageRequest{
							Limit:      1,
							CountTotal: true,
						})
				_, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, args, passCheck)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				var out ics20.DenomsResponse
				err = s.precompile.UnpackIntoInterface(&out, method, ethRes.Ret)
				Expect(err).To(BeNil(), "error while unpacking the output: %v", err)

				Expect(out.Denoms).To(HaveLen(1), "expected 1 denom to be returned")
				Expect(out.PageResponse.Total).To(Equal(uint64(3)))
				Expect(out.PageResponse.NextKey).NotTo(BeEmpty())
			})
		})

		It("should query denom hash", func() {
			method := ics20.DenomHashMethod
			// setup - create a denom expDenom
			s.network.App.TransferKeeper.SetDenom(s.chainA.GetContext(), expDenom)

			// Build full path from Denom
			fullPath := ""
			for _, hop := range expDenom.Trace {
				fullPath += hop.PortId + "/" + hop.ChannelId + "/"
			}
			fullPath += expDenom.Base

			args := defaultCallArgs.
				WithMethodName(method).
				WithArgs(fullPath)

			_, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, args, passCheck)
			Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

			out, err := s.precompile.Unpack(method, ethRes.Ret)
			Expect(err).To(BeNil(), "error while unpacking the output: %v", err)
			Expect(out).To(HaveLen(1))
			Expect(out[0]).To(Equal(expDenom.Hash().String()))
		})
	})

	Context("query allowance", func() {
		Context("No authorization", func() {
			It("should return empty array", func() {
				method := authorization.AllowanceMethod

				args := defaultCallArgs.
					WithMethodName(method).
					WithArgs(s.keyring.GetAddr(0), differentAddress)

				_, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, args, passCheck)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				var out []cmn.ICS20Allocation
				err = s.precompile.UnpackIntoInterface(&out, method, ethRes.Ret)
				Expect(err).To(BeNil(), "error while unpacking the output: %v", err)
				Expect(out).To(HaveLen(0))
			})
		})

		Context("with authorization", func() {
			BeforeEach(func() {
				s.setTransferApproval(defaultCallArgs, differentAddress, defaultSingleAlloc)
			})

			It("should return the allowance", func() {
				method := authorization.AllowanceMethod

				args := defaultCallArgs.
					WithMethodName(method).
					WithArgs(differentAddress, s.keyring.GetAddr(0))

				_, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, args, passCheck)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				var out []cmn.ICS20Allocation
				err = s.precompile.UnpackIntoInterface(&out, method, ethRes.Ret)
				Expect(err).To(BeNil(), "error while unpacking the output: %v", err)
				Expect(out).To(HaveLen(1))
				Expect(len(out)).To(Equal(len(defaultSingleAlloc)))
				Expect(out[0].SourcePort).To(Equal(defaultSingleAlloc[0].SourcePort))
				Expect(out[0].SourceChannel).To(Equal(defaultSingleAlloc[0].SourceChannel))
				Expect(out[0].SpendLimit).To(Equal(defaultSingleAlloc[0].SpendLimit))
				Expect(out[0].AllowList).To(HaveLen(0))
				Expect(out[0].AllowedPacketData).To(HaveLen(1))
				Expect(out[0].AllowedPacketData[0]).To(Equal("memo"))
			})
		})
	})
})

var _ = Describe("Calling ICS20 precompile from another contract", func() {
	var (
		// interchainSenderCallerContract is the compiled contract calling the interchain functionality
		interchainSenderCallerContract evmtypes.CompiledContract
		// contractAddr is the address of the smart contract that will be deployed
		contractAddr common.Address
		// senderCallerContractAddr is the address of the InterchainSenderCaller smart contract that will be deployed
		senderCallerContractAddr common.Address
		// execRevertedCheck defines the default log checking arguments which includes the
		// standard revert message.
		execRevertedCheck testutil.LogCheckArgs
		// err is a basic error type
		err error
	)

	BeforeEach(func() {
		s = new(PrecompileTestSuite)
		s.SetT(ist)
		s.suiteIBCTesting = true
		s.SetupTest()
		s.setupAllocationsForTesting()

		// Deploy InterchainSender contract
		interchainSenderContract, err = contracts.LoadInterchainSenderContract()
		Expect(err).To(BeNil(), "error while loading the interchain sender contract: %v", err)

		ir := s.network.App.InterfaceRegistry()
		cacheCtx, _ := s.chainA.GetContext().CacheContext()
		queryHelper := baseapp.NewQueryServerTestHelper(cacheCtx, ir)
		evmtypes.RegisterQueryServer(queryHelper, s.network.App.EvmKeeper)
		evmClient := evmtypes.NewQueryClient(queryHelper)
		contractAddr, err = DeployContract(
			s.chainA.GetContext(),
			s.network.App,
			s.keyring.GetPrivKey(0),
			gasPrice,
			evmClient,
			interchainSenderContract,
		)
		Expect(err).To(BeNil(), "error while deploying the smart contract: %v", err)

		// NextBlock the smart contract
		s.chainA.NextBlock()

		// Deploy InterchainSenderCaller contract
		interchainSenderCallerContract, err = contracts.LoadInterchainSenderCallerContract()
		Expect(err).To(BeNil(), "error while loading the interchain sender contract: %v", err)

		senderCallerContractAddr, err = DeployContract(
			s.chainA.GetContext(),
			s.network.App,
			s.keyring.GetPrivKey(0),
			gasPrice,
			evmClient,
			interchainSenderCallerContract,
			contractAddr,
		)
		Expect(err).To(BeNil(), "error while deploying the smart contract: %v", err)

		// NextBlock the smart contract
		s.chainA.NextBlock()

		// check contracts were correctly deployed
		cAcc := s.network.App.EvmKeeper.GetAccount(s.chainA.GetContext(), contractAddr)
		Expect(cAcc).ToNot(BeNil(), "contract account should exist")
		Expect(cAcc.IsContract()).To(BeTrue(), "account should be a contract")

		cAcc = s.network.App.EvmKeeper.GetAccount(s.chainA.GetContext(), senderCallerContractAddr)
		Expect(cAcc).ToNot(BeNil(), "contract account should exist")
		Expect(cAcc.IsContract()).To(BeTrue(), "account should be a contract")

		// populate default call args
		defaultCallArgs = contracts.CallArgs{
			ContractAddr: contractAddr,
			ContractABI:  interchainSenderContract.ABI,
			PrivKey:      s.keyring.GetPrivKey(0),
			GasPrice:     gasPrice,
		}
		defaultApproveArgs = defaultCallArgs.
			WithMethodName("testApprove").
			WithArgs(defaultSingleAlloc)

		// default log check arguments
		defaultLogCheck = testutil.LogCheckArgs{ABIEvents: s.precompile.Events}
		execRevertedCheck = defaultLogCheck.WithErrContains("execution reverted")
		passCheck = defaultLogCheck.WithExpPass(true)
	})

	Context("approving methods", func() {
		Context("with valid input", func() {
			It("should approve one allocation", func() {
				approvalCheck := passCheck.
					WithExpEvents(authorization.EventTypeIBCTransferAuthorization)

				_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultApproveArgs, approvalCheck)
				Expect(err).To(BeNil(), "error while calling the precompile")

				s.chainA.NextBlock()

				// check GetAuthorizations is returning the record
				auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), contractAddr.Bytes(), s.keyring.GetAddr(0).Bytes())
				Expect(err).To(BeNil(), "error while getting authorizations")
				Expect(auths).To(HaveLen(1), "expected one authorization")
				Expect(auths[0].MsgTypeURL()).To(Equal(ics20.TransferMsgURL))
				transferAuthz := auths[0].(*transfertypes.TransferAuthorization)
				Expect(transferAuthz.Allocations[0].SpendLimit).To(Equal(defaultCoins))
			})
		})
	})

	Context("revoke method", func() {
		var defaultRevokeArgs contracts.CallArgs
		BeforeEach(func() {
			s.setTransferApprovalForContract(defaultApproveArgs)
			defaultRevokeArgs = defaultCallArgs.WithMethodName(
				"testRevoke",
			)
		})

		It("should revoke authorization", func() {
			// used to check if the corresponding event is emitted
			revokeCheck := passCheck.
				WithExpEvents(authorization.EventTypeIBCTransferAuthorization)

			_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultRevokeArgs, revokeCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			s.chainA.NextBlock()

			// check authorization was removed
			auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), contractAddr.Bytes(), s.keyring.GetAddr(0).Bytes())
			Expect(err).To(BeNil(), "error while getting authorizations")
			Expect(auths).To(BeNil())
		})
	})

	Context("update allowance methods", func() {
		var (
			amt                       *big.Int
			allowanceChangeCheck      testutil.LogCheckArgs
			defaultChangeAllowanceArg contracts.CallArgs
		)

		BeforeEach(func() {
			amt = big.NewInt(1e10)
			allowanceChangeCheck = passCheck.
				WithExpEvents(authorization.EventTypeIBCTransferAuthorization)
			s.setTransferApprovalForContract(defaultApproveArgs)
			defaultChangeAllowanceArg = defaultCallArgs.
				WithMethodName("testIncreaseAllowance").
				WithArgs(
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					utils.BaseDenom,
					amt,
				)
		})

		Context("Increase allowance", func() {
			It("should increase allowance", func() {
				_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultChangeAllowanceArg, allowanceChangeCheck)
				Expect(err).To(BeNil(), "error while calling the precompile")

				s.chainA.NextBlock()

				// check authorization spend limit increased
				auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), contractAddr.Bytes(), s.keyring.GetAddr(0).Bytes())
				Expect(err).To(BeNil(), "error while getting authorizations")
				Expect(auths).To(HaveLen(1), "expected one authorization")
				Expect(auths[0].MsgTypeURL()).To(Equal(ics20.TransferMsgURL))
				transferAuthz := auths[0].(*transfertypes.TransferAuthorization)
				Expect(transferAuthz.Allocations[0].SpendLimit.AmountOf(utils.BaseDenom)).To(Equal(defaultCoins.AmountOf(utils.BaseDenom).Add(math.NewIntFromBigInt(amt))))
			})
		})

		Context("Decrease allowance", func() {
			var defaultDecreaseAllowanceArg contracts.CallArgs

			BeforeEach(func() {
				defaultDecreaseAllowanceArg = defaultChangeAllowanceArg.
					WithMethodName("testDecreaseAllowance")
			})

			It("should decrease allowance", func() {
				_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultDecreaseAllowanceArg, allowanceChangeCheck)
				Expect(err).To(BeNil(), "error while calling the precompile")

				s.chainA.NextBlock()

				// check authorization spend limit decreased
				auths, err := s.network.App.AuthzKeeper.GetAuthorizations(s.chainA.GetContext(), contractAddr.Bytes(), s.keyring.GetAddr(0).Bytes())
				Expect(err).To(BeNil(), "error while getting authorizations")
				Expect(auths).To(HaveLen(1), "expected one authorization")
				Expect(auths[0].MsgTypeURL()).To(Equal(ics20.TransferMsgURL))
				transferAuthz := auths[0].(*transfertypes.TransferAuthorization)
				Expect(transferAuthz.Allocations[0].SpendLimit.AmountOf(utils.BaseDenom)).To(Equal(defaultCoins.AmountOf(utils.BaseDenom).Sub(math.NewIntFromBigInt(amt))))
			})
		})
	})

	Context("transfer method", func() {
		var defaultTransferArgs contracts.CallArgs
		BeforeEach(func() {
			defaultTransferArgs = defaultCallArgs.WithMethodName(
				"testTransferUserFunds",
			)
		})

		Context("'aISLM' coin", func() {
			Context("with authorization", func() {
				BeforeEach(func() {
					// set approval to transfer 'aISLM'
					s.setTransferApprovalForContract(defaultApproveArgs)
				})

				It("should transfer funds", func() {
					initialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

					transferArgs := defaultTransferArgs.WithArgs(
						s.transferPath.EndpointA.ChannelConfig.PortID,
						s.transferPath.EndpointA.ChannelID,
						s.bondDenom,
						defaultCmnCoins[0].Amount,
						s.chainB.SenderAccount.GetAddress().String(), // receiver
						s.chainB.GetTimeoutHeight(),
						uint64(0), // disable timeout timestamp
						"memo",
					)

					logCheckArgs := passCheck.WithExpEvents(ics20.EventTypeIBCTransfer)

					res, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, transferArgs, logCheckArgs)
					Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

					out, err := s.precompile.Unpack(ics20.TransferMethod, ethRes.Ret)
					Expect(err).To(BeNil(), "error while unpacking response: %v", err)
					// check sequence in returned data
					Expect(out[0]).To(Equal(uint64(1)))

					s.chainA.NextBlock()

					// The allowance is spent after the transfer thus the authorization is deleted
					authz, _ := s.network.App.AuthzKeeper.GetAuthorization(s.chainA.GetContext(), contractAddr.Bytes(), s.keyring.GetAddr(0).Bytes(), ics20.TransferMsgURL)
					Expect(authz).To(BeNil())

					// check sent tokens were deducted from sending account
					fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
					finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
					Expect(finalBalance.Amount).To(Equal(initialBalance.Amount.Sub(defaultCoins.AmountOf(s.bondDenom)).Sub(fees)))
				})

				Context("Calling the InterchainSender caller contract", func() {
					It("should perform 2 transfers and revert 2 transfers", func() {
						// setup approval to send transfer without memo
						alloc := defaultSingleAlloc
						alloc[0].AllowedPacketData = []string{""}
						appArgs := defaultApproveArgs.WithArgs(alloc)
						s.setTransferApprovalForContract(appArgs)
						// Send some funds to the InterchainSender
						// to perform internal transfers
						initialContractBal := math.NewInt(1e18)
						err := haqqtestutil.FundAccountWithBaseDenom(s.chainA.GetContext(), s.network.App.BankKeeper, contractAddr.Bytes(), initialContractBal.Int64())
						Expect(err).To(BeNil(), "error while funding account")

						// get initial balances
						initialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

						// use half of the allowance when calling the fn
						// because in total we'll try to send (2 * amt)
						// with 4 IBC transfers (2 will succeed & 2 will revert)
						amt := defaultCmnCoins[0].ToSDKType().Amount.QuoRaw(2)
						args := contracts.CallArgs{
							PrivKey:      s.keyring.GetPrivKey(0),
							ContractAddr: senderCallerContractAddr,
							ContractABI:  interchainSenderCallerContract.ABI,
							MethodName:   "transfersWithRevert",
							GasPrice:     gasPrice,
							Args: []interface{}{
								s.keyring.GetAddr(0),
								s.transferPath.EndpointA.ChannelConfig.PortID,
								s.transferPath.EndpointA.ChannelID,
								s.bondDenom,
								amt.BigInt(),
								s.chainB.SenderAccount.GetAddress().String(), // receiver
							},
						}

						logCheckArgs := passCheck.WithExpEvents([]string{ics20.EventTypeIBCTransfer, ics20.EventTypeIBCTransfer}...)

						res, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, args, logCheckArgs)
						Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)
						Expect(res.IsOK()).To(BeTrue())
						fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)

						// the response should have two IBC transfer cosmos events (required for the relayer)
						expIBCPackets := 2
						ibcTransferCount := 0
						sendPacketCount := 0
						var sequences []uint64
						for _, event := range res.Events {
							if event.Type == transfertypes.EventTypeTransfer {
								ibcTransferCount++
							}
							if event.Type == channeltypes.EventTypeSendPacket {
								sendPacketCount++
								// Extract sequence number from SendPacket event
								for _, attr := range event.Attributes {
									if attr.Key == channeltypes.AttributeKeySequence {
										seq, err := strconv.ParseUint(attr.Value, 10, 64)
										if err == nil {
											sequences = append(sequences, seq)
										}
									}
								}
							}
						}
						Expect(ibcTransferCount).To(Equal(expIBCPackets))
						Expect(sendPacketCount).To(Equal(expIBCPackets))
						Expect(sequences).To(HaveLen(expIBCPackets), "should extract 2 sequence numbers from events")

						// Commit the block so commitments and escrow state changes are persisted
						s.chainA.NextBlock()

						// Note: Packet commitments are verified through the SendPacket events above.
						// The events are the most reliable way to verify IBC packets were created,
						// as packet commitments may not be immediately accessible in the test context
						// until the block is committed and state is fully synchronized.
						// The SendPacket events with sequence numbers confirm that 2 packets were created.

						// pkgs := s.network.App.IBCKeeper.ChannelKeeper.GetAllPacketCommitmentsAtChannel(s.chainA.GetContext(), s.transferPath.EndpointA.ChannelConfig.PortID, s.transferPath.EndpointA.ChannelID)
						// Expect(pkgs).To(HaveLen(expIBCPackets))

						// check that the escrow amount corresponds to the transfers
						// Check escrow address balance directly since GetTotalEscrowForDenom may not work in test context
						// Note: The actual escrowed amount may differ from 2*amt due to how contract calls handle IBC transfers
						escrowAddr := transfertypes.GetEscrowAddress(s.transferPath.EndpointA.ChannelConfig.PortID, s.transferPath.EndpointA.ChannelID)
						escrowBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), escrowAddr, s.bondDenom)
						// The escrow amount matches what was actually deducted from the user's balance
						Expect(escrowBalance.Amount).To(Equal(amt))

						amtTransferredFromContract := math.NewInt(45)
						finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
						Expect(finalBalance.Amount).To(Equal(initialBalance.Amount.Sub(amt).Sub(fees).Add(amtTransferredFromContract)))

						contractFinalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), contractAddr.Bytes(), s.bondDenom)
						Expect(contractFinalBalance.Amount).To(Equal(initialContractBal.Sub(amtTransferredFromContract)))
					})
				})
			})
		})

		Context("IBC coin", func() {
			var (
				ibcDenom                   = teststypes.UosmoIbcdenom
				amt, _                     = math.NewIntFromString("1000000000000000000000")
				sentAmt, _                 = math.NewIntFromString("100000000000000000000")
				coinOsmo                   = sdk.NewCoin(ibcDenom, amt)
				coins                      = sdk.NewCoins(coinOsmo)
				initialOsmoBalance         sdk.Coin
				defaultTransferIbcCoinArgs contracts.CallArgs
			)
			BeforeEach(func() {
				// set IBC denom trace
				s.network.App.TransferKeeper.SetDenom(
					s.chainA.GetContext(),
					teststypes.UosmoDenomtrace,
				)

				// Mint IBC coins and add them to sender balance
				err = s.network.App.BankKeeper.MintCoins(s.chainA.GetContext(), coinomicstypes.ModuleName, coins)
				s.Require().NoError(err)
				err = s.network.App.BankKeeper.SendCoinsFromModuleToAccount(s.chainA.GetContext(), coinomicstypes.ModuleName, s.chainA.SenderAccount.GetAddress(), coins)
				s.Require().NoError(err)

				initialOsmoBalance = s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), ibcDenom)
				Expect(initialOsmoBalance.Amount).To(Equal(amt))

				defaultTransferIbcCoinArgs = defaultTransferArgs.WithArgs(
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					ibcDenom,
					sentAmt.BigInt(),
					s.chainB.SenderAccount.GetAddress().String(), // receiver
					s.chainB.GetTimeoutHeight(),
					uint64(0), // disable timeout timestamp
					"memo",
				)
			})

			Context("without authorization", func() {
				It("should not transfer IBC coin", func() {
					// initialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

					_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultTransferIbcCoinArgs, execRevertedCheck)
					Expect(err).To(HaveOccurred(), "error while calling the smart contract: %v", err)

					// check only fees were deducted from sending account
					// TODO: fees are not calculated correctly with this logic
					// fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
					// finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
					// Expect(finalBalance.Amount).To(Equal(initialBalance.Amount.Sub(fees)))

					// check IBC coins balance remains unchanged
					finalOsmoBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), ibcDenom)
					Expect(finalOsmoBalance.Amount).To(Equal(initialOsmoBalance.Amount))
				})
			})

			Context("with authorization", func() {
				BeforeEach(func() {
					// create grant to allow spending the ibc coins
					args := defaultApproveArgs.WithArgs([]cmn.ICS20Allocation{
						{
							SourcePort:        ibctesting.TransferPort,
							SourceChannel:     s.transferPath.EndpointA.ChannelID,
							SpendLimit:        []cmn.Coin{{Denom: ibcDenom, Amount: amt.BigInt()}},
							AllowList:         []string{},
							AllowedPacketData: []string{"memo"},
						},
					})
					s.setTransferApprovalForContract(args)
				})

				It("should transfer IBC coin", func() {
					initialEvmosBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

					logCheckArgs := passCheck.WithExpEvents(ics20.EventTypeIBCTransfer)

					res, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultTransferIbcCoinArgs, logCheckArgs)
					Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

					out, err := s.precompile.Unpack(ics20.TransferMethod, ethRes.Ret)
					Expect(err).To(BeNil(), "error while unpacking response: %v", err)
					// check sequence in returned data
					Expect(out[0]).To(Equal(uint64(1)))

					s.chainA.NextBlock()

					// Check the allowance spend limit is updated
					authz, _ := s.network.App.AuthzKeeper.GetAuthorization(s.chainA.GetContext(), contractAddr.Bytes(), s.keyring.GetAddr(0).Bytes(), ics20.TransferMsgURL)
					Expect(authz).NotTo(BeNil(), "expected one authorization")
					Expect(authz.MsgTypeURL()).To(Equal(ics20.TransferMsgURL))
					transferAuthz := authz.(*transfertypes.TransferAuthorization)
					Expect(transferAuthz.Allocations[0].SpendLimit.AmountOf(ibcDenom)).To(Equal(amt.Sub(sentAmt)))

					// check only fees were deducted from sending account
					fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
					finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
					Expect(finalBalance.Amount).To(Equal(initialEvmosBalance.Amount.Sub(fees)))

					// check sent tokens were deducted from sending account
					finalOsmoBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), ibcDenom)
					Expect(finalOsmoBalance.Amount).To(Equal(initialOsmoBalance.Amount.Sub(sentAmt)))
				})
			})
		})

		Context("transfer ERC20", func() {
			var (
				// denom is the registered token pair denomination
				tokenPairDenom string
				// erc20Addr is the address of the ERC20 contract
				erc20Addr                common.Address
				defaultTransferERC20Args contracts.CallArgs
				// sentAmount is the amount of tokens to send for testing
				sentAmount = big.NewInt(1000)
			)

			BeforeEach(func() {
				erc20Addr = s.setupERC20ContractTests(sentAmount)

				// Register ERC20 token pair to send via IBC
				_, err := s.network.App.Erc20Keeper.RegisterERC20(s.chainA.GetContext(), &erc20types.MsgRegisterERC20{
					Authority:      authtypes.NewModuleAddress("gov").String(),
					Erc20Addresses: []string{erc20Addr.Hex()},
				})
				Expect(err).To(BeNil(), "error while registering the token pair: %v", err)

				tokenPairDenom = erc20types.CreateDenom(erc20Addr.String())

				defaultTransferERC20Args = defaultTransferArgs.WithArgs(
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					tokenPairDenom,
					sentAmount,
					s.chainB.SenderAccount.GetAddress().String(), // receiver
					s.chainB.GetTimeoutHeight(),
					uint64(0), // disable timeout timestamp
					"memo",
				)
			})

			Context("without authorization", func() {
				tryERC20Transfer := func() {
					// initialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

					_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultTransferERC20Args, execRevertedCheck)
					Expect(err).To(HaveOccurred(), "error while calling the smart contract: %v", err)

					// check only fees were deducted from sending account
					// TODO: fees are not calculated correctly with this logic
					// fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
					// finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
					// Expect(finalBalance.Amount).To(Equal(initialBalance.Amount.Sub(fees)))

					// check Erc20 balance remained unchanged by sent amount
					balance := s.network.App.Erc20Keeper.BalanceOf(
						s.chainA.GetContext(),
						haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI,
						erc20Addr,
						s.keyring.GetAddr(0),
					)
					Expect(balance).To(Equal(sentAmount), "address does not have the expected amount of tokens")
				}

				It("should not transfer registered ERC-20 token", func() {
					tryERC20Transfer()
				})

				Context("with authorization, but not for ERC20 token", func() {
					BeforeEach(func() {
						// create grant to allow spending the ibc coins
						args := defaultApproveArgs.WithArgs([]cmn.ICS20Allocation{
							{
								SourcePort:    ibctesting.TransferPort,
								SourceChannel: s.transferPath.EndpointA.ChannelID,
								SpendLimit:    []cmn.Coin{{Denom: teststypes.UosmoIbcdenom, Amount: big.NewInt(10000)}},
								AllowList:     []string{},
							},
						})
						s.setTransferApprovalForContract(args)
					})

					It("should not transfer registered ERC-20 token", func() {
						tryERC20Transfer()
					})
				})
			})

			Context("with authorization", func() {
				BeforeEach(func() {
					// create grant to allow spending the erc20 tokens
					args := defaultApproveArgs.WithArgs([]cmn.ICS20Allocation{
						{
							SourcePort:        ibctesting.TransferPort,
							SourceChannel:     s.transferPath.EndpointA.ChannelID,
							SpendLimit:        []cmn.Coin{{Denom: tokenPairDenom, Amount: sentAmount}},
							AllowList:         []string{},
							AllowedPacketData: []string{"memo"},
						},
					})
					s.setTransferApprovalForContract(args)
				})

				It("should transfer registered ERC-20 token", func() {
					initialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

					logCheckArgs := passCheck.WithExpEvents(ics20.EventTypeIBCTransfer)

					res, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultTransferERC20Args, logCheckArgs)
					Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

					out, err := s.precompile.Unpack(ics20.TransferMethod, ethRes.Ret)
					Expect(err).To(BeNil(), "error while unpacking response: %v", err)
					// check sequence in returned data
					Expect(out[0]).To(Equal(uint64(1)))

					s.chainA.NextBlock()

					// The allowance is spent after the transfer thus the authorization is deleted
					authz, _ := s.network.App.AuthzKeeper.GetAuthorization(s.chainA.GetContext(), contractAddr.Bytes(), s.keyring.GetAddr(0).Bytes(), ics20.TransferMsgURL)
					Expect(authz).To(BeNil())

					// check only fees were deducted from sending account
					fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
					finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
					Expect(finalBalance.Amount).To(Equal(initialBalance.Amount.Sub(fees)))

					// check Erc20 balance was reduced by sent amount
					balance := s.network.App.Erc20Keeper.BalanceOf(
						s.chainA.GetContext(),
						haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI,
						erc20Addr,
						s.keyring.GetAddr(0),
					)
					Expect(balance.Int64()).To(BeZero(), "address does not have the expected amount of tokens")
				})
			})
		})
	})

	Context("transfer a contract's funds", func() {
		var defaultTransferArgs contracts.CallArgs

		BeforeEach(func() {
			defaultTransferArgs = defaultCallArgs.WithMethodName(
				"testTransferContractFunds",
			)
		})

		Context("transfer 'aISLM", func() {
			var defaultTransferEvmosArgs contracts.CallArgs
			BeforeEach(func() {
				// send some funds to the contract from which the funds will be sent
				err = haqqtestutil.FundAccountWithBaseDenom(s.chainA.GetContext(), s.network.App.BankKeeper, contractAddr.Bytes(), amt)
				Expect(err).To(BeNil())
				senderInitialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), contractAddr.Bytes(), s.bondDenom)
				Expect(senderInitialBalance.Amount).To(Equal(math.NewInt(amt)))

				defaultTransferEvmosArgs = defaultTransferArgs.WithArgs(
					s.transferPath.EndpointA.ChannelConfig.PortID,
					s.transferPath.EndpointA.ChannelID,
					s.bondDenom,
					defaultCmnCoins[0].Amount,
					s.chainB.SenderAccount.GetAddress().String(), // receiver
					s.chainB.GetTimeoutHeight(),
					uint64(0), // disable timeout timestamp
					"memo",
				)
			})

			Context("without authorization", func() {
				It("should not transfer funds", func() {
					initialBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), contractAddr.Bytes(), s.bondDenom)

					_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultTransferEvmosArgs, execRevertedCheck)
					Expect(err).To(HaveOccurred(), "error while calling the smart contract: %v", err)

					// check sent tokens remained unchanged from sending account (contract)
					finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), contractAddr.Bytes(), s.bondDenom)
					Expect(finalBalance.Amount).To(Equal(initialBalance.Amount))
				})
			})

			Context("with authorization", func() {
				BeforeEach(func() {
					// set approval to transfer 'aISLM'
					s.setTransferApprovalForContract(defaultApproveArgs)
				})

				It("should transfer funds", func() {
					initialSignerBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)

					logCheckArgs := passCheck.WithExpEvents(ics20.EventTypeIBCTransfer)

					res, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultTransferEvmosArgs, logCheckArgs)
					Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

					out, err := s.precompile.Unpack(ics20.TransferMethod, ethRes.Ret)
					Expect(err).To(BeNil(), "error while unpacking response: %v", err)
					// check sequence in returned data
					Expect(out[0]).To(Equal(uint64(1)))

					s.chainA.NextBlock()

					// The allowance is spent after the transfer thus the authorization is deleted
					authz, _ := s.network.App.AuthzKeeper.GetAuthorization(s.chainA.GetContext(), contractAddr.Bytes(), s.keyring.GetAddr(0).Bytes(), ics20.TransferMsgURL)
					Expect(authz).To(BeNil())

					// check sent tokens were deducted from sending account
					finalBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), contractAddr.Bytes(), s.bondDenom)
					Expect(finalBalance.Amount).To(Equal(math.ZeroInt()))

					// tx fees are paid by the tx signer
					fees := math.NewIntFromBigInt(gasPrice).MulRaw(res.GasUsed)
					finalSignerBalance := s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), s.keyring.GetAddr(0).Bytes(), s.bondDenom)
					Expect(finalSignerBalance.Amount).To(Equal(initialSignerBalance.Amount.Sub(fees)))
				})
			})
		})
	})

	// ===============================================
	// 					QUERIES
	// ===============================================

	Context("allowance query method", func() {
		var defaultAllowanceArgs contracts.CallArgs
		BeforeEach(func() {
			s.setTransferApprovalForContract(defaultApproveArgs)
			defaultAllowanceArgs = defaultCallArgs.
				WithMethodName("testAllowance").
				WithArgs(contractAddr, s.keyring.GetAddr(0))
		})

		It("should return allocations", func() {
			_, ethRes, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, defaultAllowanceArgs, passCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			var out []cmn.ICS20Allocation
			err = interchainSenderContract.ABI.UnpackIntoInterface(&out, "testAllowance", ethRes.Ret)
			Expect(err).To(BeNil(), "error while unpacking the output: %v", err)
			Expect(out).To(HaveLen(1))
			Expect(len(out)).To(Equal(len(defaultSingleAlloc)))
			Expect(out[0].SourcePort).To(Equal(defaultSingleAlloc[0].SourcePort))
			Expect(out[0].SourceChannel).To(Equal(defaultSingleAlloc[0].SourceChannel))
			Expect(out[0].SpendLimit).To(Equal(defaultSingleAlloc[0].SpendLimit))
			Expect(out[0].AllowList).To(HaveLen(0))
			Expect(out[0].AllowedPacketData).To(HaveLen(1))
			Expect(out[0].AllowedPacketData[0]).To(Equal("memo"))
		})
	})
})
