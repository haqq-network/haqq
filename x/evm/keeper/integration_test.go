// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package keeper_test

import (
	"math/big"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	"cosmossdk.io/math"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/encoding"
	"github.com/haqq-network/haqq/precompiles/staking"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	integrationutils "github.com/haqq-network/haqq/testutil/integration/haqq/utils"
	"github.com/haqq-network/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

type IntegrationTestSuite struct {
	network     network.Network
	factory     factory.TxFactory
	grpcHandler grpc.Handler
	keyring     testkeyring.Keyring
}

// This test suite is meant to test the EVM module in the context of the EVMOS.
// It uses the integration test framework to spin up a local EVMOS network and
// perform transactions on it.
// The test suite focus on testing how the MsgEthereumTx message is handled under the
// different params configuration of the module while testing the different Tx types
// Ethereum supports (LegacyTx, AccessListTx, DynamicFeeTx) and the different types of
// transactions (transfer, contract deployment, contract call).
// Note that more in depth testing of the EVM and solidity execution is done through the
// hardhat and the nix setup.
var _ = Describe("Handling a MsgEthereumTx message", Label("EVM"), Ordered, func() {
	var s *IntegrationTestSuite

	BeforeAll(func() {
		keyring := testkeyring.New(4)
		integrationNetwork := network.New(
			network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
		)
		grpcHandler := grpc.NewIntegrationHandler(integrationNetwork)
		txFactory := factory.New(integrationNetwork, grpcHandler)
		s = &IntegrationTestSuite{
			network:     integrationNetwork,
			factory:     txFactory,
			grpcHandler: grpcHandler,
			keyring:     keyring,
		}
	})

	AfterEach(func() {
		// Start each test with a fresh block
		err := s.network.NextBlock()
		Expect(err).To(BeNil())
	})

	When("the params have default values", Ordered, func() {
		BeforeAll(func() {
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			err := s.network.UpdateEvmParams(defaultParams)
			Expect(err).To(BeNil())

			err = s.network.NextBlock()
			Expect(err).To(BeNil())
		})

		DescribeTable("Executes a transfer transaction", func(getTxArgs func() evmtypes.EvmTxArgs) {
			senderKey := s.keyring.GetKey(0)
			receiverKey := s.keyring.GetKey(1)
			denom := s.network.GetDenom()

			senderPrevBalanceResponse, err := s.grpcHandler.GetBalance(senderKey.AccAddr, denom)
			Expect(err).To(BeNil())
			senderPrevBalance := senderPrevBalanceResponse.GetBalance().Amount

			receiverPrevBalanceResponse, err := s.grpcHandler.GetBalance(receiverKey.AccAddr, denom)
			Expect(err).To(BeNil())
			receiverPrevBalance := receiverPrevBalanceResponse.GetBalance().Amount

			transferAmount := int64(1000)

			// Taking custom args from the table entry
			txArgs := getTxArgs()
			txArgs.Amount = big.NewInt(transferAmount)
			txArgs.To = &receiverKey.Addr

			res, err := s.factory.ExecuteEthTx(senderKey.Priv, txArgs)
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())

			err = s.network.NextBlock()
			Expect(err).To(BeNil())

			// Check sender balance after transaction
			senderBalanceResultBeforeFees := senderPrevBalance.Sub(math.NewInt(transferAmount))
			senderAfterBalance, err := s.grpcHandler.GetBalance(senderKey.AccAddr, denom)
			Expect(err).To(BeNil())
			Expect(senderAfterBalance.GetBalance().Amount.LTE(senderBalanceResultBeforeFees)).To(BeTrue())

			// Check receiver balance after transaction
			receiverBalanceResult := receiverPrevBalance.Add(math.NewInt(transferAmount))
			receverAfterBalanceResponse, err := s.grpcHandler.GetBalance(receiverKey.AccAddr, denom)
			Expect(err).To(BeNil())
			Expect(receverAfterBalanceResponse.GetBalance().Amount).To(Equal(receiverBalanceResult))
		},
			Entry("as a DynamicFeeTx", func() evmtypes.EvmTxArgs { return evmtypes.EvmTxArgs{} }),
			Entry("as an AccessListTx",
				func() evmtypes.EvmTxArgs {
					return evmtypes.EvmTxArgs{
						Accesses: &ethtypes.AccessList{{
							Address:     s.keyring.GetAddr(1),
							StorageKeys: []common.Hash{{0}},
						}},
					}
				},
			),
			Entry("as a LegacyTx", func() evmtypes.EvmTxArgs {
				return evmtypes.EvmTxArgs{
					GasPrice: big.NewInt(1e9),
				}
			}),
		)

		DescribeTable("Executes a contract deployment", func(getTxArgs func() evmtypes.EvmTxArgs) {
			// Deploy contract
			senderPriv := s.keyring.GetPrivKey(0)
			constructorArgs := []interface{}{"coin", "token", uint8(18)}
			compiledContract := contracts.ERC20MinterBurnerDecimalsContract

			txArgs := getTxArgs()
			contractAddr, err := s.factory.DeployContract(
				senderPriv,
				txArgs,
				factory.ContractDeploymentData{
					Contract:        compiledContract,
					ConstructorArgs: constructorArgs,
				},
			)
			Expect(err).To(BeNil())
			Expect(contractAddr).ToNot(Equal(common.Address{}))

			err = s.network.NextBlock()
			Expect(err).To(BeNil())

			// Check contract account got created correctly
			contractBechAddr := sdktypes.AccAddress(contractAddr.Bytes()).String()
			contractAccount, err := s.grpcHandler.GetAccount(contractBechAddr)
			Expect(err).To(BeNil())
			err = integrationutils.IsContractAccount(contractAccount)
			Expect(err).To(BeNil())
		},
			Entry("as a DynamicFeeTx", func() evmtypes.EvmTxArgs { return evmtypes.EvmTxArgs{} }),
			Entry("as an AccessListTx",
				func() evmtypes.EvmTxArgs {
					return evmtypes.EvmTxArgs{
						Accesses: &ethtypes.AccessList{{
							Address:     s.keyring.GetAddr(1),
							StorageKeys: []common.Hash{{0}},
						}},
					}
				},
			),
			Entry("as a LegacyTx", func() evmtypes.EvmTxArgs {
				return evmtypes.EvmTxArgs{
					GasPrice: big.NewInt(1e9),
				}
			}),
		)

		DescribeTable("Try MsgEthereumTx replay", func(getTxArgs func() evmtypes.EvmTxArgs) {
			// Deploy contract
			senderKey := s.keyring.GetKey(0)
			receiverKey := s.keyring.GetKey(1)
			denom := s.network.GetDenom()

			// Get balances before execution
			senderPrevBalanceResponse, err := s.grpcHandler.GetBalance(senderKey.AccAddr, denom)
			Expect(err).To(BeNil())
			senderPrevBalance := senderPrevBalanceResponse.GetBalance().Amount
			senderAccountRespBeforeAll, err := s.grpcHandler.GetEvmAccount(senderKey.Addr)
			Expect(err).To(BeNil())

			receiverPrevBalanceResponse, err := s.grpcHandler.GetBalance(receiverKey.AccAddr, denom)
			Expect(err).To(BeNil())
			receiverPrevBalance := receiverPrevBalanceResponse.GetBalance().Amount

			// Deploy contract msg
			txArgsDeploy := getTxArgs()
			constructorArgs := []interface{}{"coin", "token", uint8(18)}
			compiledContract := contracts.ERC20MinterBurnerDecimalsContract
			from := common.BytesToAddress(senderKey.AccAddr)
			deploymentData := factory.ContractDeploymentData{
				Contract:        compiledContract,
				ConstructorArgs: constructorArgs,
			}

			deployTxArgs, err := s.factory.GenerateDeployContractArgs(from, txArgsDeploy, deploymentData)
			Expect(err).To(BeNil())
			deployMsgEthereumTx, err := s.factory.GenerateMsgEthereumTx(senderKey.Priv, deployTxArgs)
			Expect(err).To(BeNil())
			deployMsgEthereumTxSigned, err := s.factory.SignMsgEthereumTx(senderKey.Priv, deployMsgEthereumTx)
			Expect(err).To(BeNil())

			// Transfer msg 1
			txArgsTransfer1 := getTxArgs()
			txArgsTransfer1.Amount = big.NewInt(int64(1000))
			txArgsTransfer1.To = &receiverKey.Addr
			txArgsTransfer1.Nonce = deployTxArgs.Nonce + 1 // increase nonce as it's next msg

			transferMsgEthereumTx1, err := s.factory.GenerateMsgEthereumTx(senderKey.Priv, txArgsTransfer1)
			Expect(err).To(BeNil())
			transferMsgEthereumTx1Signed, err := s.factory.SignMsgEthereumTx(senderKey.Priv, transferMsgEthereumTx1)
			Expect(err).To(BeNil())

			// Transfer msg 1
			txArgsTransfer2 := getTxArgs()
			txArgsTransfer2.Amount = big.NewInt(int64(500))
			txArgsTransfer2.To = &receiverKey.Addr
			txArgsTransfer2.Nonce = deployTxArgs.Nonce + 2 // increase nonce as it's next msg

			transferMsgEthereumTx2, err := s.factory.GenerateMsgEthereumTx(senderKey.Priv, txArgsTransfer2)
			Expect(err).To(BeNil())
			transferMsgEthereumTx2Signed, err := s.factory.SignMsgEthereumTx(senderKey.Priv, transferMsgEthereumTx2)
			Expect(err).To(BeNil())

			option, err := codectypes.NewAnyWithValue(&evmtypes.ExtensionOptionsEthereumTx{})
			Expect(err).To(BeNil())

			// Compose transaction with multiple msgs
			txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
			txBuilder1 := txConfig.NewTxBuilder()
			txFee1 := sdktypes.NewCoins().
				Add(sdktypes.Coin{Denom: utils.BaseDenom, Amount: math.NewIntFromBigInt(deployMsgEthereumTxSigned.GetFee())}).
				Add(sdktypes.Coin{Denom: utils.BaseDenom, Amount: math.NewIntFromBigInt(transferMsgEthereumTx1Signed.GetFee())}).
				Add(sdktypes.Coin{Denom: utils.BaseDenom, Amount: math.NewIntFromBigInt(transferMsgEthereumTx2Signed.GetFee())})
			txGasLimit1 := deployMsgEthereumTxSigned.GetGas() + transferMsgEthereumTx1Signed.GetGas() + transferMsgEthereumTx2Signed.GetGas()

			deployMsgEthereumTxSigned.From = ""
			transferMsgEthereumTx1Signed.From = ""
			transferMsgEthereumTx2Signed.From = ""

			err = txBuilder1.SetMsgs(&deployMsgEthereumTxSigned, &transferMsgEthereumTx1Signed, &transferMsgEthereumTx2Signed)
			Expect(err).To(BeNil())

			txBuilder1.(authtx.ExtensionOptionsTxBuilder).SetExtensionOptions(option)
			txBuilder1.SetGasLimit(txGasLimit1)
			txBuilder1.SetFeeAmount(txFee1)

			tx1 := txBuilder1.GetTx()
			txBytes, err := txConfig.TxEncoder()(tx1)
			Expect(err).To(BeNil())

			res, err := s.network.BroadcastTxSync(txBytes)
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())

			err = s.network.NextBlock()
			Expect(err).To(BeNil())

			// Check nonce
			senderAccountRespAfterTx1, err := s.grpcHandler.GetEvmAccount(senderKey.Addr)
			Expect(err).To(BeNil())
			// By default, expect invalid nonce! Increased by 1 instead of 3
			Expect(senderAccountRespAfterTx1.Nonce).To(Equal(senderAccountRespBeforeAll.Nonce + 3))

			// Check Balance
			senderBalanceAfterTx1Response, err := s.grpcHandler.GetBalance(senderKey.AccAddr, denom)
			Expect(err).To(BeNil())
			senderBalanceAfterTx1 := senderBalanceAfterTx1Response.GetBalance().Amount
			Expect(senderBalanceAfterTx1.LT(senderPrevBalance.Sub(math.NewInt(1500)))).To(BeTrue())

			receiverBalanceAfterTx1Response, err := s.grpcHandler.GetBalance(receiverKey.AccAddr, denom)
			Expect(err).To(BeNil())
			receiverBalanceAfterTx1 := receiverBalanceAfterTx1Response.GetBalance().Amount
			Expect(receiverBalanceAfterTx1).To(Equal(receiverPrevBalance.Add(math.NewInt(1500))))

			// Compose transaction with msg replay
			txBuilder2 := txConfig.NewTxBuilder()
			txFee2 := sdktypes.NewCoins().
				Add(sdktypes.Coin{Denom: utils.BaseDenom, Amount: math.NewIntFromBigInt(transferMsgEthereumTx1Signed.GetFee())}).
				Add(sdktypes.Coin{Denom: utils.BaseDenom, Amount: math.NewIntFromBigInt(transferMsgEthereumTx2Signed.GetFee())})
			txGasLimit2 := transferMsgEthereumTx1Signed.GetGas() + transferMsgEthereumTx2Signed.GetGas()

			err = txBuilder2.SetMsgs(&transferMsgEthereumTx1Signed, &transferMsgEthereumTx2Signed)
			Expect(err).To(BeNil())

			txBuilder2.(authtx.ExtensionOptionsTxBuilder).SetExtensionOptions(option)
			txBuilder2.SetGasLimit(txGasLimit2)
			txBuilder2.SetFeeAmount(txFee2)

			tx2 := txBuilder2.GetTx()
			txBytes2, err := txConfig.TxEncoder()(tx2)
			Expect(err).To(BeNil())

			res2, err := s.network.BroadcastTxSync(txBytes2)
			Expect(err).To(BeNil())
			Expect(res2.IsOK()).To(BeFalse())

			err = s.network.NextBlock()
			Expect(err).To(BeNil())

			// Check nonce
			senderAccountRespAfterTx2, err := s.grpcHandler.GetEvmAccount(senderKey.Addr)
			Expect(err).To(BeNil())
			// Expect correct nonce. Increased by 2 after second tx
			Expect(senderAccountRespAfterTx2.Nonce).To(Equal(txArgsTransfer2.Nonce + 1))

			// Check balances after replay
			senderBalanceAfterTx2Response, err := s.grpcHandler.GetBalance(senderKey.AccAddr, denom)
			Expect(err).To(BeNil())
			senderBalanceAfterTx2 := senderBalanceAfterTx2Response.GetBalance().Amount
			// By default, replay works and value will be 3000
			Expect(senderBalanceAfterTx2.LT(senderPrevBalance.Sub(math.NewInt(1500)))).To(BeTrue())

			receiverBalanceAfterTx2Response, err := s.grpcHandler.GetBalance(receiverKey.AccAddr, denom)
			Expect(err).To(BeNil())
			receiverBalanceAfterTx2 := receiverBalanceAfterTx2Response.GetBalance().Amount
			// By default, replay works and value will be 3000
			Expect(receiverBalanceAfterTx2).To(Equal(receiverPrevBalance.Add(math.NewInt(1500))))
		},
			Entry("as a DynamicFeeTx", func() evmtypes.EvmTxArgs { return evmtypes.EvmTxArgs{} }),
			Entry("as an AccessListTx",
				func() evmtypes.EvmTxArgs {
					return evmtypes.EvmTxArgs{
						Accesses: &ethtypes.AccessList{{
							Address:     s.keyring.GetAddr(1),
							StorageKeys: []common.Hash{{0}},
						}},
					}
				},
			),
			Entry("as a LegacyTx", func() evmtypes.EvmTxArgs {
				return evmtypes.EvmTxArgs{
					GasPrice: big.NewInt(1e9),
				}
			}),
		)

		DescribeTable("R&D Nonce state on failed transactions - Deploy contract", func(getTxArgs func() evmtypes.EvmTxArgs) {
			senderKey := s.keyring.GetKey(0)
			senderAccountRespBeforeAll, err := s.grpcHandler.GetEvmAccount(senderKey.Addr)
			Expect(err).To(BeNil())

			// Deploy contract msg
			txArgsDeploy := getTxArgs()
			constructorArgs := []interface{}{"coin", "token", uint8(18)}
			compiledContract := contracts.ERC20MinterBurnerDecimalsContract
			from := common.BytesToAddress(senderKey.AccAddr)
			deploymentData := factory.ContractDeploymentData{
				Contract:        compiledContract,
				ConstructorArgs: constructorArgs,
			}

			deployTxArgs, err := s.factory.GenerateDeployContractArgs(from, txArgsDeploy, deploymentData)
			Expect(err).To(BeNil())
			deployTxArgs.GasLimit = 1000 // Set low gas limit to force tx fail
			deployMsgEthereumTx, err := s.factory.GenerateMsgEthereumTx(senderKey.Priv, deployTxArgs)
			Expect(err).To(BeNil())
			deployMsgEthereumTxSigned, err := s.factory.SignMsgEthereumTx(senderKey.Priv, deployMsgEthereumTx)
			Expect(err).To(BeNil())

			// Compose transaction and execute
			txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
			txBuilder1 := txConfig.NewTxBuilder()
			txFee1 := sdktypes.NewCoins().Add(sdktypes.Coin{Denom: utils.BaseDenom, Amount: math.NewIntFromBigInt(deployMsgEthereumTxSigned.GetFee())})
			txGasLimit1 := deployMsgEthereumTxSigned.GetGas()

			deployMsgEthereumTxSigned.From = ""

			err = txBuilder1.SetMsgs(&deployMsgEthereumTxSigned)
			Expect(err).To(BeNil())

			option, err := codectypes.NewAnyWithValue(&evmtypes.ExtensionOptionsEthereumTx{})
			Expect(err).To(BeNil())

			txBuilder1.(authtx.ExtensionOptionsTxBuilder).SetExtensionOptions(option)
			txBuilder1.SetGasLimit(txGasLimit1)
			txBuilder1.SetFeeAmount(txFee1)

			tx1 := txBuilder1.GetTx()
			txBytes, err := txConfig.TxEncoder()(tx1)
			Expect(err).To(BeNil())

			res, err := s.network.BroadcastTxSync(txBytes)
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeFalse())
			println("contract deploy failed: " + res.Log)

			err = s.network.NextBlock()
			Expect(err).To(BeNil())

			// Check nonce
			senderAccountRespAfter, err := s.grpcHandler.GetEvmAccount(senderKey.Addr)
			Expect(err).To(BeNil())
			// Expect increased nonce even on failed tx
			Expect(senderAccountRespAfter.Nonce).To(Equal(senderAccountRespBeforeAll.Nonce + 1))
		},
			Entry("as a DynamicFeeTx", func() evmtypes.EvmTxArgs { return evmtypes.EvmTxArgs{} }),
			Entry("as an AccessListTx",
				func() evmtypes.EvmTxArgs {
					return evmtypes.EvmTxArgs{
						Accesses: &ethtypes.AccessList{{
							Address:     s.keyring.GetAddr(1),
							StorageKeys: []common.Hash{{0}},
						}},
					}
				},
			),
			Entry("as a LegacyTx", func() evmtypes.EvmTxArgs {
				return evmtypes.EvmTxArgs{
					GasPrice: big.NewInt(1e9),
				}
			}),
		)

		DescribeTable("R&D Nonce state on failed transactions - Transfer", func(getTxArgs func() evmtypes.EvmTxArgs) {
			senderKey := s.keyring.GetKey(0)
			receiverKey := s.keyring.GetKey(1)
			denom := s.network.GetDenom()

			senderPrevBalanceResponse, err := s.grpcHandler.GetBalance(senderKey.AccAddr, denom)
			Expect(err).To(BeNil())
			senderPrevBalance := senderPrevBalanceResponse.GetBalance().Amount
			senderAccountRespBeforeAll, err := s.grpcHandler.GetEvmAccount(senderKey.Addr)
			Expect(err).To(BeNil())

			receiverPrevBalanceResponse, err := s.grpcHandler.GetBalance(receiverKey.AccAddr, denom)
			Expect(err).To(BeNil())
			receiverPrevBalance := receiverPrevBalanceResponse.GetBalance().Amount

			// Transfer msg
			txArgsTransfer := getTxArgs()
			txArgsTransfer.Amount = big.NewInt(int64(1000))
			txArgsTransfer.To = &receiverKey.Addr
			txArgsTransfer.Nonce = senderAccountRespBeforeAll.Nonce
			txArgsTransfer.GasLimit = 10000 // Set low gas limit to force tx fail

			transferMsgEthereumTx, err := s.factory.GenerateMsgEthereumTx(senderKey.Priv, txArgsTransfer)
			Expect(err).To(BeNil())
			transferMsgEthereumTxSigned, err := s.factory.SignMsgEthereumTx(senderKey.Priv, transferMsgEthereumTx)
			Expect(err).To(BeNil())
			transferMsgEthereumTxSigned.From = ""

			// Compose and execute tx
			txConfig := encoding.MakeConfig(app.ModuleBasics).TxConfig
			txBuilder := txConfig.NewTxBuilder()
			txFee := sdktypes.NewCoins().Add(sdktypes.Coin{Denom: utils.BaseDenom, Amount: math.NewIntFromBigInt(transferMsgEthereumTxSigned.GetFee())})
			txGasLimit := transferMsgEthereumTxSigned.GetGas()

			err = txBuilder.SetMsgs(&transferMsgEthereumTxSigned)
			Expect(err).To(BeNil())

			option, err := codectypes.NewAnyWithValue(&evmtypes.ExtensionOptionsEthereumTx{})
			Expect(err).To(BeNil())
			txBuilder.(authtx.ExtensionOptionsTxBuilder).SetExtensionOptions(option)
			txBuilder.SetGasLimit(txGasLimit)
			txBuilder.SetFeeAmount(txFee)

			tx := txBuilder.GetTx()
			txBytes, err := txConfig.TxEncoder()(tx)
			Expect(err).To(BeNil())

			res, err := s.network.BroadcastTxSync(txBytes)
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeFalse())
			println("transfer failed: " + res.Log)

			err = s.network.NextBlock()
			Expect(err).To(BeNil())

			// Check nonce
			senderAccountRespAfter, err := s.grpcHandler.GetEvmAccount(senderKey.Addr)
			Expect(err).To(BeNil())
			// Expect increased nonce even on failed tx
			Expect(senderAccountRespAfter.Nonce).To(Equal(senderAccountRespBeforeAll.Nonce + 1))

			// Check balances
			senderBalanceAfterResponse, err := s.grpcHandler.GetBalance(senderKey.AccAddr, denom)
			Expect(err).To(BeNil())
			senderBalanceAfter := senderBalanceAfterResponse.GetBalance().Amount
			Expect(senderBalanceAfter.LT(senderPrevBalance)).To(BeTrue())

			receiverBalanceAfterResponse, err := s.grpcHandler.GetBalance(receiverKey.AccAddr, denom)
			Expect(err).To(BeNil())
			receiverBalanceAfter := receiverBalanceAfterResponse.GetBalance().Amount
			Expect(receiverBalanceAfter).To(Equal(receiverPrevBalance))
		},
			Entry("as a DynamicFeeTx", func() evmtypes.EvmTxArgs { return evmtypes.EvmTxArgs{} }),
			Entry("as an AccessListTx",
				func() evmtypes.EvmTxArgs {
					return evmtypes.EvmTxArgs{
						Accesses: &ethtypes.AccessList{{
							Address:     s.keyring.GetAddr(1),
							StorageKeys: []common.Hash{{0}},
						}},
					}
				},
			),
			Entry("as a LegacyTx", func() evmtypes.EvmTxArgs {
				return evmtypes.EvmTxArgs{
					GasPrice: big.NewInt(1e9),
				}
			}),
		)

		Context("With a predeployed ERC20MinterBurnerDecimalsContract", func() {
			var contractAddr common.Address

			BeforeEach(func() {
				// Deploy contract
				senderPriv := s.keyring.GetPrivKey(0)
				constructorArgs := []interface{}{"coin", "token", uint8(18)}
				compiledContract := contracts.ERC20MinterBurnerDecimalsContract

				var err error // Avoid shadowing
				contractAddr, err = s.factory.DeployContract(
					senderPriv,
					evmtypes.EvmTxArgs{}, // Default values
					factory.ContractDeploymentData{
						Contract:        compiledContract,
						ConstructorArgs: constructorArgs,
					},
				)
				Expect(err).To(BeNil())
				Expect(contractAddr).ToNot(Equal(common.Address{}))

				err = s.network.NextBlock()
				Expect(err).To(BeNil())
			})

			DescribeTable("Executes a contract call", func(getTxArgs func() evmtypes.EvmTxArgs) {
				senderPriv := s.keyring.GetPrivKey(0)
				compiledContract := contracts.ERC20MinterBurnerDecimalsContract
				recipientKey := s.keyring.GetKey(1)

				// Execute contract call
				mintTxArgs := getTxArgs()
				mintTxArgs.To = &contractAddr

				amountToMint := big.NewInt(1e18)
				mintArgs := factory.CallArgs{
					ContractABI: compiledContract.ABI,
					MethodName:  "mint",
					Args:        []interface{}{recipientKey.Addr, amountToMint},
				}
				mintResponse, err := s.factory.ExecuteContractCall(senderPriv, mintTxArgs, mintArgs)
				Expect(err).To(BeNil())
				Expect(mintResponse.IsOK()).To(Equal(true), "transaction should have succeeded", mintResponse.GetLog())

				err = checkMintTopics(mintResponse)
				Expect(err).To(BeNil())

				err = s.network.NextBlock()
				Expect(err).To(BeNil())

				totalSupplyTxArgs := evmtypes.EvmTxArgs{
					To: &contractAddr,
				}
				totalSupplyArgs := factory.CallArgs{
					ContractABI: compiledContract.ABI,
					MethodName:  "totalSupply",
					Args:        []interface{}{},
				}
				totalSupplyRes, err := s.factory.ExecuteContractCall(senderPriv, totalSupplyTxArgs, totalSupplyArgs)
				Expect(err).To(BeNil())
				Expect(totalSupplyRes.IsOK()).To(Equal(true), "transaction should have succeeded", totalSupplyRes.GetLog())

				var totalSupplyResponse *big.Int
				err = integrationutils.DecodeContractCallResponse(&totalSupplyResponse, totalSupplyArgs, totalSupplyRes)
				Expect(err).To(BeNil())
				Expect(totalSupplyResponse).To(Equal(amountToMint))
			},
				Entry("as a DynamicFeeTx", func() evmtypes.EvmTxArgs { return evmtypes.EvmTxArgs{} }),
				Entry("as an AccessListTx",
					func() evmtypes.EvmTxArgs {
						return evmtypes.EvmTxArgs{
							Accesses: &ethtypes.AccessList{{
								Address:     s.keyring.GetAddr(1),
								StorageKeys: []common.Hash{{0}},
							}},
						}
					},
				),
				Entry("as a LegacyTx", func() evmtypes.EvmTxArgs {
					return evmtypes.EvmTxArgs{
						GasPrice: big.NewInt(1e9),
					}
				}),
			)
		})

		It("should fail when ChainID is wrong", func() {
			senderPriv := s.keyring.GetPrivKey(0)
			receiver := s.keyring.GetKey(1)
			txArgs := evmtypes.EvmTxArgs{
				To:      &receiver.Addr,
				Amount:  big.NewInt(1000),
				ChainID: big.NewInt(1),
			}

			res, err := s.factory.ExecuteEthTx(senderPriv, txArgs)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid chain id"))
			// Transaction fails before being broadcasted
			Expect(res).To(Equal(abcitypes.ResponseDeliverTx{}))
		})
	})

	DescribeTable("Performs transfer and contract call", func(getTestParams func() evmtypes.Params, transferParams, contractCallParams PermissionsTableTest) {
		params := getTestParams()
		err := s.network.UpdateEvmParams(params)
		Expect(err).To(BeNil())

		err = s.network.NextBlock()
		Expect(err).To(BeNil())

		signer := s.keyring.GetKey(transferParams.SignerIndex)
		receiver := s.keyring.GetKey(1)
		txArgs := evmtypes.EvmTxArgs{
			To:     &receiver.Addr,
			Amount: big.NewInt(1000),
			// Hard coded gas limit to avoid failure on gas estimation because
			// of the param
			GasLimit: 100000,
		}
		res, err := s.factory.ExecuteEthTx(signer.Priv, txArgs)
		if transferParams.ExpFail {
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("does not have permission to perform a call"))
		} else {
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
		}

		senderKey := s.keyring.GetKey(contractCallParams.SignerIndex)
		contractAddress := common.HexToAddress(staking.PrecompileAddress)
		validatorAddress := s.network.GetValidators()[1].OperatorAddress
		contractABI, err := staking.LoadABI()
		Expect(err).To(BeNil())

		// If grpc query fails, that means there were no previous delegations
		prevDelegation := big.NewInt(0)
		prevDelegationRes, err := s.grpcHandler.GetDelegation(senderKey.AccAddr.String(), validatorAddress)
		if err == nil {
			prevDelegation = prevDelegationRes.DelegationResponse.Balance.Amount.BigInt()
		}

		amountToDelegate := big.NewInt(200)
		totalSupplyTxArgs := evmtypes.EvmTxArgs{
			To: &contractAddress,
		}

		// Perform a delegate transaction to the staking precompile
		delegateArgs := factory.CallArgs{
			ContractABI: contractABI,
			MethodName:  staking.DelegateMethod,
			Args:        []interface{}{senderKey.Addr, validatorAddress, amountToDelegate},
		}
		delegateResponse, err := s.factory.ExecuteContractCall(senderKey.Priv, totalSupplyTxArgs, delegateArgs)
		if contractCallParams.ExpFail {
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("does not have permission to perform a call"))
		} else {
			Expect(err).To(BeNil())
			Expect(delegateResponse.IsOK()).To(Equal(true), "transaction should have succeeded", delegateResponse.GetLog())

			err = s.network.NextBlock()
			Expect(err).To(BeNil())

			// Perform query to check the delegation was successful
			queryDelegationArgs := factory.CallArgs{
				ContractABI: contractABI,
				MethodName:  staking.DelegationMethod,
				Args:        []interface{}{senderKey.Addr, validatorAddress},
			}
			queryDelegationResponse, err := s.factory.ExecuteContractCall(senderKey.Priv, totalSupplyTxArgs, queryDelegationArgs)
			Expect(err).To(BeNil())
			Expect(queryDelegationResponse.IsOK()).To(Equal(true), "transaction should have succeeded", queryDelegationResponse.GetLog())

			// Make sure the delegation amount is correct
			var delegationOutput staking.DelegationOutput
			err = integrationutils.DecodeContractCallResponse(&delegationOutput, queryDelegationArgs, queryDelegationResponse)
			Expect(err).To(BeNil())

			expectedDelegationAmt := amountToDelegate.Add(amountToDelegate, prevDelegation)
			Expect(delegationOutput.Balance.Amount.String()).To(Equal(expectedDelegationAmt.String()))
		}
	},
		// Entry("transfer and call fail with CALL permission policy set to restricted", func() evmtypes.Params {
		// 	// Set params to default values
		// 	defaultParams := evmtypes.DefaultParams()
		// 	defaultParams.AccessControl.Call = evmtypes.AccessControlType{
		// 		AccessType:        evmtypes.AccessTypeRestricted,
		// 	}
		// 	return defaultParams
		// },
		// 	OpcodeTestTable{ExpFail: true, SignerIndex: 0},
		// 	OpcodeTestTable{ExpFail: true, SignerIndex: 0},
		// ),
		Entry("transfer and call succeed with CALL permission policy set to default and CREATE permission policy set to restricted", func() evmtypes.Params {
			blockedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Create = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypeRestricted,
				AccessControlList: []string{s.keyring.GetAddr(blockedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
		),
		Entry("transfer and call are successful with CALL permission policy set to permissionless and address not blocked", func() evmtypes.Params {
			blockedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Call = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissionless,
				AccessControlList: []string{s.keyring.GetAddr(blockedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
		),
		Entry("transfer fails with signer blocked and call succeeds with signer NOT blocked permission policy set to permissionless", func() evmtypes.Params {
			blockedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Call = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissionless,
				AccessControlList: []string{s.keyring.GetAddr(blockedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: true, SignerIndex: 1},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
		),
		Entry("transfer succeeds with signer NOT blocked and call fails with signer blocked permission policy set to permissionless", func() evmtypes.Params {
			blockedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Call = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissionless,
				AccessControlList: []string{s.keyring.GetAddr(blockedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
			PermissionsTableTest{ExpFail: true, SignerIndex: 1},
		),
		Entry("transfer and call succeeds with CALL permission policy set to permissioned and signer whitelisted on both", func() evmtypes.Params {
			blockedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Call = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissioned,
				AccessControlList: []string{s.keyring.GetAddr(blockedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: false, SignerIndex: 1},
			PermissionsTableTest{ExpFail: false, SignerIndex: 1},
		),
		Entry("transfer and call fails with CALL permission policy set to permissioned and signer not whitelisted on both", func() evmtypes.Params {
			blockedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Call = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissioned,
				AccessControlList: []string{s.keyring.GetAddr(blockedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: true, SignerIndex: 0},
			PermissionsTableTest{ExpFail: true, SignerIndex: 0},
		),
	)

	DescribeTable("Performs contract deployment and contract call with AccessControl", func(getTestParams func() evmtypes.Params, createParams, callParams PermissionsTableTest) {
		params := getTestParams()
		err := s.network.UpdateEvmParams(params)
		Expect(err).To(BeNil())

		err = s.network.NextBlock()
		Expect(err).To(BeNil())

		createSigner := s.keyring.GetPrivKey(createParams.SignerIndex)
		constructorArgs := []interface{}{"coin", "token", uint8(18)}
		compiledContract := contracts.ERC20MinterBurnerDecimalsContract

		contractAddr, err := s.factory.DeployContract(
			createSigner,
			evmtypes.EvmTxArgs{}, // Default values
			factory.ContractDeploymentData{
				Contract:        compiledContract,
				ConstructorArgs: constructorArgs,
			},
		)
		if createParams.ExpFail {
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("does not have permission to deploy contracts"))
			// If contract deployment is expected to fail, we can skip the rest of the test
			return
		}

		Expect(err).To(BeNil())
		Expect(contractAddr).ToNot(Equal(common.Address{}))

		err = s.network.NextBlock()
		Expect(err).To(BeNil())

		callSigner := s.keyring.GetPrivKey(callParams.SignerIndex)
		totalSupplyTxArgs := evmtypes.EvmTxArgs{
			To: &contractAddr,
		}
		totalSupplyArgs := factory.CallArgs{
			ContractABI: compiledContract.ABI,
			MethodName:  "totalSupply",
			Args:        []interface{}{},
		}
		res, err := s.factory.ExecuteContractCall(callSigner, totalSupplyTxArgs, totalSupplyArgs)
		if callParams.ExpFail {
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("does not have permission to perform a call"))
		} else {
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
		}
	},
		Entry("Create and call is successful with create permission policy set to permissionless and address not blocked ", func() evmtypes.Params {
			blockedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Create = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissionless,
				AccessControlList: []string{s.keyring.GetAddr(blockedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
		),
		Entry("Create fails with create permission policy set to permissionless and signer is blocked ", func() evmtypes.Params {
			blockedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Create = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissionless,
				AccessControlList: []string{s.keyring.GetAddr(blockedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: true, SignerIndex: 1},
			PermissionsTableTest{}, // Call should not be executed
		),
		Entry("Create and call is successful with call permission policy set to permissionless and address not blocked ", func() evmtypes.Params {
			blockedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Call = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissionless,
				AccessControlList: []string{s.keyring.GetAddr(blockedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
		),
		Entry("Create is successful and call fails with call permission policy set to permissionless and address blocked ", func() evmtypes.Params {
			blockedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Call = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissionless,
				AccessControlList: []string{s.keyring.GetAddr(blockedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
			PermissionsTableTest{ExpFail: true, SignerIndex: 1},
		),
		Entry("Create fails create permission policy set to restricted", func() evmtypes.Params {
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Create = evmtypes.AccessControlType{
				AccessType: evmtypes.AccessTypeRestricted,
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: true, SignerIndex: 0},
			PermissionsTableTest{}, // Call should not be executed
		),
		Entry("Create succeeds and call fails when call permission policy set to restricted", func() evmtypes.Params {
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Call = evmtypes.AccessControlType{
				AccessType: evmtypes.AccessTypeRestricted,
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
			PermissionsTableTest{ExpFail: true, SignerIndex: 0},
		),
		Entry("Create and call are successful with create permission policy set to permissioned and signer whitelisted", func() evmtypes.Params {
			whitelistedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Create = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissioned,
				AccessControlList: []string{s.keyring.GetAddr(whitelistedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: false, SignerIndex: 1},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
		),
		Entry("Create fails with create permission policy set to permissioned and signer NOT whitelisted", func() evmtypes.Params {
			whitelistedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Create = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissioned,
				AccessControlList: []string{s.keyring.GetAddr(whitelistedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: true, SignerIndex: 0},
			PermissionsTableTest{},
		),
		Entry("Create and call are successful with call permission policy set to permissioned and signer whitelisted", func() evmtypes.Params {
			whitelistedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Call = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissioned,
				AccessControlList: []string{s.keyring.GetAddr(whitelistedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
			PermissionsTableTest{ExpFail: false, SignerIndex: 1},
		),
		Entry("Create succeeds and call fails with call permission policy set to permissioned and signer NOT whitelisted", func() evmtypes.Params {
			whitelistedSignerIndex := 1
			// Set params to default values
			defaultParams := evmtypes.DefaultParams()
			defaultParams.AccessControl.Call = evmtypes.AccessControlType{
				AccessType:        evmtypes.AccessTypePermissioned,
				AccessControlList: []string{s.keyring.GetAddr(whitelistedSignerIndex).String()},
			}
			return defaultParams
		},
			PermissionsTableTest{ExpFail: false, SignerIndex: 0},
			PermissionsTableTest{ExpFail: true, SignerIndex: 0},
		),
	)
})

type PermissionsTableTest struct {
	ExpFail     bool
	SignerIndex int
}

func checkMintTopics(res abcitypes.ResponseDeliverTx) error {
	// Check contract call response has the expected topics for a mint
	// call within an ERC20 contract
	expectedTopics := []string{
		"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
		"0x0000000000000000000000000000000000000000000000000000000000000000",
	}
	return integrationutils.CheckTxTopics(res, expectedTopics)
}
