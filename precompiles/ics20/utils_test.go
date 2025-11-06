// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package ics20_test

import (
	"math/big"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcgotesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	haqqapp "github.com/haqq-network/haqq/app"
	haqqcontracts "github.com/haqq-network/haqq/contracts"
	haqqibctesting "github.com/haqq-network/haqq/ibc/testing"
	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/erc20"
	"github.com/haqq-network/haqq/precompiles/ics20"
	"github.com/haqq-network/haqq/precompiles/testutil"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	haqqtestutil "github.com/haqq-network/haqq/testutil"
	testutiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

type erc20Meta struct {
	Name     string
	Symbol   string
	Decimals uint8
}

var (
	maxUint256Coins    = sdk.Coins{sdk.Coin{Denom: utils.BaseDenom, Amount: sdkmath.NewIntFromBigInt(abi.MaxUint256)}}
	maxUint256CmnCoins = []cmn.Coin{{Denom: utils.BaseDenom, Amount: abi.MaxUint256}}
	defaultCoins       = sdk.Coins{sdk.Coin{Denom: utils.BaseDenom, Amount: sdkmath.NewInt(1e18)}}
	baseDenomCmnCoin   = cmn.Coin{Denom: utils.BaseDenom, Amount: big.NewInt(1e18)}
	defaultCmnCoins    = []cmn.Coin{baseDenomCmnCoin}
	atomCoins          = sdk.Coins{sdk.Coin{Denom: "uatom", Amount: sdkmath.NewInt(1e18)}}
	atomCmnCoin        = cmn.Coin{Denom: "uatom", Amount: big.NewInt(1e18)}
	mutliSpendLimit    = sdk.Coins{sdk.Coin{Denom: utils.BaseDenom, Amount: sdkmath.NewInt(1e18)}, sdk.Coin{Denom: "uatom", Amount: sdkmath.NewInt(1e18)}}
	mutliCmnCoins      = []cmn.Coin{baseDenomCmnCoin, atomCmnCoin}
	testERC20          = erc20Meta{
		Name:     "TestCoin",
		Symbol:   "TC",
		Decimals: 18,
	}
)

// NewPrecompileContract creates a new precompile contract and sets the gas meter
func (s *PrecompileTestSuite) NewPrecompileContract(gas uint64) *vm.Contract {
	contract := vm.NewContract(vm.AccountRef(s.keyring.GetAddr(0)), s.precompile, big.NewInt(0), gas)

	ctx := s.network.GetContext().WithGasMeter(storetypes.NewInfiniteGasMeter())
	initialGas := ctx.GasMeter().GasConsumed()
	s.Require().Zero(initialGas)

	return contract
}

// NewTransferAuthorizationWithAllocations creates a new allocation for the given grantee and granter and the given coins
func (s *PrecompileTestSuite) NewTransferAuthorizationWithAllocations(ctx sdk.Context, app *haqqapp.Haqq, grantee, granter common.Address, allocations []transfertypes.Allocation) error {
	transferAuthz := &transfertypes.TransferAuthorization{Allocations: allocations}
	if err := transferAuthz.ValidateBasic(); err != nil {
		return err
	}

	// create the authorization
	expTime := s.network.GetContext().BlockTime().Add(cmn.DefaultExpirationDuration).UTC()
	return app.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), transferAuthz, &expTime)
}

// NewTransferAuthorization creates a new transfer authorization for the given grantee and granter and the given coins
func (s *PrecompileTestSuite) NewTransferAuthorization(ctx sdk.Context, app *haqqapp.Haqq, grantee, granter common.Address, path *haqqibctesting.Path, coins sdk.Coins, allowList []string, allowedPacketData []string) error {
	allocations := []transfertypes.Allocation{
		{
			SourcePort:        path.EndpointA.ChannelConfig.PortID,
			SourceChannel:     path.EndpointA.ChannelID,
			SpendLimit:        coins,
			AllowList:         allowList,
			AllowedPacketData: allowedPacketData,
		},
	}

	transferAuthz := &transfertypes.TransferAuthorization{Allocations: allocations}
	if err := transferAuthz.ValidateBasic(); err != nil {
		return err
	}

	// create the authorization
	expTime := s.network.GetContext().BlockTime().Add(cmn.DefaultExpirationDuration).UTC()
	return app.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), transferAuthz, &expTime)
}

// GetTransferAuthorization returns the transfer authorization for the given grantee and granter
func (s *PrecompileTestSuite) GetTransferAuthorization(ctx sdk.Context, grantee, granter common.Address) *transfertypes.TransferAuthorization {
	grant, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, grantee.Bytes(), granter.Bytes(), ics20.TransferMsgURL)
	s.Require().NotNil(grant)
	transferAuthz, ok := grant.(*transfertypes.TransferAuthorization)
	s.Require().True(ok)
	s.Require().NotNil(transferAuthz)
	return transferAuthz
}

// CheckAllowanceChangeEvent is a helper function used to check the allowance change event arguments.
func (s *PrecompileTestSuite) CheckAllowanceChangeEvent(log *ethtypes.Log, amount *big.Int, isIncrease bool) {
	// Check event signature matches the one emitted
	event := s.precompile.ABI.Events[authorization.EventTypeIBCTransferAuthorization]
	s.Require().Equal(event.ID, common.HexToHash(log.Topics[0].Hex()))
	//nolint: gosec // G115 blockHeight is positive int64 and can't overflow uint64
	s.Require().Equal(log.BlockNumber, uint64(s.network.GetContext().BlockHeight()))

	var approvalEvent ics20.EventTransferAuthorization
	err := cmn.UnpackLog(s.precompile.ABI, &approvalEvent, authorization.EventTypeIBCTransferAuthorization, *log)
	s.Require().NoError(err)
	s.Require().Equal(s.keyring.GetAddr(0), approvalEvent.Grantee)
	s.Require().Equal(s.keyring.GetAddr(0), approvalEvent.Granter)
	s.Require().Equal("transfer", approvalEvent.Allocations[0].SourcePort)
	s.Require().Equal("channel-0", approvalEvent.Allocations[0].SourceChannel)

	allocationAmount := approvalEvent.Allocations[0].SpendLimit[0].Amount
	if isIncrease {
		newTotal := amount.Add(allocationAmount, amount)
		s.Require().Equal(amount, newTotal)
	} else {
		newTotal := amount.Sub(allocationAmount, amount)
		s.Require().Equal(amount, newTotal)
	}
}

// setupIBCTest makes the necessary setup of chains A & B
// for integration tests
func (s *PrecompileTestSuite) setupIBCTest() {
	s.coordinator.CommitNBlocks(s.chainA.ChainID, 2)
	s.coordinator.CommitNBlocks(s.chainB.ChainID, 2)

	ctx := s.chainA.GetContext()
	haqqApp := s.chainA.App.(*haqqapp.Haqq)
	evmParams := haqqApp.EvmKeeper.GetParams(ctx)
	evmParams.EvmDenom = utils.BaseDenom
	err := haqqApp.EvmKeeper.SetParams(ctx, evmParams)
	s.Require().NoError(err)

	// Set block proposer once, so its carried over on the ibc-go-testing suite
	validators, err := haqqApp.StakingKeeper.GetValidators(ctx, 3)
	s.Require().NoError(err)
	cons, err := validators[0].GetConsAddr()
	s.Require().NoError(err)
	
	// Update the proposed header with the proposer address
	header := s.chainA.ProposedHeader
	header.ProposerAddress = cons
	s.chainA.ProposedHeader = header

	err = haqqApp.StakingKeeper.SetValidatorByConsAddr(ctx, validators[0])
	s.Require().NoError(err)

	_, err = haqqApp.EvmKeeper.GetCoinbaseAddress(ctx, sdk.ConsAddress(cons))
	s.Require().NoError(err)

	// Mint coins locked on the haqq account generated with secp.
	amount, ok := sdkmath.NewIntFromString("1000000000000000000000")
	s.Require().True(ok)
	coinIslm := sdk.NewCoin(utils.BaseDenom, amount)
	coins := sdk.NewCoins(coinIslm)
	err = haqqApp.BankKeeper.MintCoins(ctx, coinomicstypes.ModuleName, coins)
	s.Require().NoError(err)
	err = haqqApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, coinomicstypes.ModuleName, s.chainA.SenderAccount.GetAddress(), coins)
	s.Require().NoError(err)

	s.chainA.NextBlock()

	s.transferPath = s.coordinator.Setup(s.chainA.ChainID, s.chainB.ChainID)
	s.Require().Equal("07-tendermint-0", s.transferPath.EndpointA.ClientID)
	s.Require().Equal("connection-0", s.transferPath.EndpointA.ConnectionID)
	s.Require().Equal("channel-0", s.transferPath.EndpointA.ChannelID)
}

// setTransferApproval sets the transfer approval for the given grantee and allocations
func (s *PrecompileTestSuite) setTransferApproval(
	args contracts.CallArgs,
	grantee common.Address,
	allocations []cmn.ICS20Allocation,
) {
	args.MethodName = authorization.ApproveMethod
	args.Args = []interface{}{
		grantee,
		allocations,
	}

	logCheckArgs := testutil.LogCheckArgs{
		ABIEvents: s.precompile.Events,
		ExpEvents: []string{authorization.EventTypeIBCTransferAuthorization},
		ExpPass:   true,
	}

	_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, args, logCheckArgs)
	Expect(err).To(BeNil(), "error while calling the contract to approve")

	s.chainA.NextBlock()

	// check auth created successfully
	authz, _ := s.network.App.AuthzKeeper.GetAuthorization(s.chainA.GetContext(), grantee.Bytes(), args.PrivKey.PubKey().Address().Bytes(), ics20.TransferMsgURL)
	Expect(authz).NotTo(BeNil())
	transferAuthz, ok := authz.(*transfertypes.TransferAuthorization)
	Expect(ok).To(BeTrue())
	Expect(len(transferAuthz.Allocations[0].SpendLimit)).To(Equal(len(allocations[0].SpendLimit)))
	for i, sl := range transferAuthz.Allocations[0].SpendLimit {
		// NOTE order may change if there're more than one coin
		Expect(sl.Denom).To(Equal(allocations[0].SpendLimit[i].Denom))
		Expect(sl.Amount.BigInt()).To(Equal(allocations[0].SpendLimit[i].Amount))
	}
}

// setTransferApprovalForContract sets the transfer approval for the given contract
func (s *PrecompileTestSuite) setTransferApprovalForContract(args contracts.CallArgs) {
	logCheckArgs := testutil.LogCheckArgs{
		ABIEvents: s.precompile.Events,
		ExpEvents: []string{authorization.EventTypeIBCTransferAuthorization},
		ExpPass:   true,
	}

	_, _, err := contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, args, logCheckArgs)
	Expect(err).To(BeNil(), "error while calling the contract to approve")

	s.chainA.NextBlock()

	// check auth created successfully
	authz, _ := s.network.App.AuthzKeeper.GetAuthorization(s.chainA.GetContext(), args.ContractAddr.Bytes(), args.PrivKey.PubKey().Address().Bytes(), ics20.TransferMsgURL)
	Expect(authz).NotTo(BeNil())
	transferAuthz, ok := authz.(*transfertypes.TransferAuthorization)
	Expect(ok).To(BeTrue())
	Expect(len(transferAuthz.Allocations) > 0).To(BeTrue())
}

// setupAllocationsForTesting sets the allocations for testing
func (s *PrecompileTestSuite) setupAllocationsForTesting() {
	defaultSingleAlloc = []cmn.ICS20Allocation{
		{
			SourcePort:        ibcgotesting.TransferPort,
			SourceChannel:     s.transferPath.EndpointA.ChannelID,
			SpendLimit:        defaultCmnCoins,
			AllowedPacketData: []string{"memo"},
		},
	}
}

// TODO upstream this change to haqq (adding gasPrice)
// DeployContract deploys a contract with the provided private key,
// compiled contract data and constructor arguments
func DeployContract(
	ctx sdk.Context,
	app *haqqapp.Haqq,
	priv cryptotypes.PrivKey,
	gasPrice *big.Int,
	queryClientEvm evmtypes.QueryClient,
	contract evmtypes.CompiledContract,
	constructorArgs ...interface{},
) (common.Address, error) {
	chainID := app.EvmKeeper.ChainID()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	nonce := app.EvmKeeper.GetNonce(ctx, from)

	ctorArgs, err := contract.ABI.Pack("", constructorArgs...)
	if err != nil {
		return common.Address{}, err
	}

	data := append(contract.Bin, ctorArgs...) //nolint:gocritic
	gas, err := testutiltx.GasLimit(ctx, from, data, queryClientEvm)
	if err != nil {
		return common.Address{}, err
	}

	msgEthereumTx := evmtypes.NewTx(&evmtypes.EvmTxArgs{
		ChainID:   chainID,
		Nonce:     nonce,
		GasLimit:  gas,
		GasFeeCap: app.FeeMarketKeeper.GetBaseFee(ctx),
		GasTipCap: big.NewInt(1),
		GasPrice:  gasPrice,
		Input:     data,
		Accesses:  &ethtypes.AccessList{},
	})
	msgEthereumTx.From = from.String()

	res, err := haqqtestutil.DeliverEthTx(ctx, app, priv, msgEthereumTx)
	if err != nil {
		return common.Address{}, err
	}

	if _, err := haqqtestutil.CheckEthTxResponse(res, app.AppCodec()); err != nil {
		return common.Address{}, err
	}

	return crypto.CreateAddress(from, nonce), nil
}

// DeployERC20Contract deploys a ERC20 token with the provided name, symbol and decimals
func (s *PrecompileTestSuite) DeployERC20Contract(chain *ibcgotesting.TestChain, name, symbol string, decimals uint8) (common.Address, error) {
	addr, err := DeployContract(
		chain.GetContext(),
		s.network.App,
		s.keyring.GetPrivKey(0),
		gasPrice,
		s.network.GetEvmClient(),
		haqqcontracts.ERC20MinterBurnerDecimalsContract,
		name,
		symbol,
		decimals,
	)
	chain.NextBlock()
	return addr, err
}

// setupERC20ContractTests deploys a ERC20 token
// and mint some tokens to the deployer address (s.address).
// The amount of tokens sent to the deployer address is defined in
// the 'amount' input argument
func (s *PrecompileTestSuite) setupERC20ContractTests(amount *big.Int) common.Address {
	erc20Addr, err := s.DeployERC20Contract(s.chainA, testERC20.Name, testERC20.Symbol, testERC20.Decimals)
	Expect(err).To(BeNil(), "error while deploying ERC20 contract: %v", err)

	defaultERC20CallArgs := contracts.CallArgs{
		ContractAddr: erc20Addr,
		ContractABI:  haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI,
		PrivKey:      s.keyring.GetPrivKey(0),
		GasPrice:     gasPrice,
	}

	// mint coins to the address
	mintCoinsArgs := defaultERC20CallArgs.
		WithMethodName("mint").
		WithArgs(s.keyring.GetAddr(0), amount)

	mintCheck := testutil.LogCheckArgs{
		ABIEvents: haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI.Events,
		ExpEvents: []string{erc20.EventTypeTransfer}, // upon minting the tokens are sent to the receiving address
		ExpPass:   true,
	}

	_, _, err = contracts.CallContractAndCheckLogs(s.chainA.GetContext(), s.network.App, mintCoinsArgs, mintCheck)
	Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

	s.chainA.NextBlock()

	// check that the address has the tokens -- this has to be done using the stateDB because
	// unregistered token pairs do not show up in the bank keeper
	balance := s.network.App.Erc20Keeper.BalanceOf(
		s.chainA.GetContext(),
		haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI,
		erc20Addr,
		s.keyring.GetAddr(0),
	)
	Expect(balance).To(Equal(amount), "address does not have the expected amount of tokens")

	return erc20Addr
}

// makePacket is a helper function to build the sent IBC packet
// to perform an ICS20 tranfer.
// This packet is then used to test the IBC callbacks (Timeout, Ack)
func (s *PrecompileTestSuite) makePacket(
	senderAddr,
	receiverAddr,
	denom,
	memo string,
	amt *big.Int,
	seq uint64,
	timeoutHeight clienttypes.Height,
) channeltypes.Packet {
	packetData := transfertypes.NewFungibleTokenPacketData(
		denom,
		amt.String(),
		senderAddr,
		receiverAddr,
		memo,
	)

	return channeltypes.NewPacket(
		packetData.GetBytes(),
		seq,
		s.transferPath.EndpointA.ChannelConfig.PortID,
		s.transferPath.EndpointA.ChannelID,
		s.transferPath.EndpointB.ChannelConfig.PortID,
		s.transferPath.EndpointB.ChannelID,
		timeoutHeight,
		0,
	)
}
