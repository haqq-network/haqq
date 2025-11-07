// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package ics20_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"

	haqqibctesting "github.com/haqq-network/haqq/ibc/testing"
	"github.com/haqq-network/haqq/precompiles/ics20"
	cmnnetwork "github.com/haqq-network/haqq/testutil/integration/common/network"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	"github.com/haqq-network/haqq/testutil/integration/ibc/coordinator"

	sdkmath "cosmossdk.io/math"
	haqqapp "github.com/haqq-network/haqq/app"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
)

var s *PrecompileTestSuite

type PrecompileTestSuite struct {
	suite.Suite

	network     *network.UnitTestNetwork
	factory     factory.TxFactory
	grpcHandler grpc.Handler
	keyring     testkeyring.Keyring

	bondDenom  string
	precompile *ics20.Precompile

	coordinator  coordinator.Coordinator
	chainA       *ibctesting.TestChain
	chainB       *ibctesting.TestChain
	transferPath *haqqibctesting.Path

	suiteIBCTesting bool
}

func TestPrecompileTestSuite(t *testing.T) {
	suite.Run(t, new(PrecompileTestSuite))
}

func (s *PrecompileTestSuite) SetupTest() {
	if s.suiteIBCTesting {
		// dummy chains use the ibc testing chain setup
		// that uses the default sdk address prefix ('cosmos')
		// Update the prefix configs to use that prefix
		haqqibctesting.SetBech32Prefix("cosmos")
		// Also need to disable address cache to avoid using modules
		// accounts with 'evmos' addresses (because Evmos chain setup is first)
		sdk.SetAddrCacheEnabled(false)
	}

	keyring := testkeyring.New(2)
	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	)
	grpcHandler := grpc.NewIntegrationHandler(nw)
	txFactory := factory.New(nw, grpcHandler)

	ctx := nw.GetContext()
	bondDenom, err := nw.App.StakingKeeper.BondDenom(ctx)
	s.Require().NoError(err)

	s.bondDenom = bondDenom
	s.factory = txFactory
	s.grpcHandler = grpcHandler
	s.keyring = keyring
	s.network = nw

	// Ensure tests are not blocked by min gas price settings
	feemParams := s.network.App.FeeMarketKeeper.GetParams(ctx)
	feemParams.MinGasPrice = sdkmath.LegacyZeroDec()
	// Also ensure base fee does not enforce high minimums during tests
	feemParams.NoBaseFee = true
	feemParams.EnableHeight = 0
	s.Require().NoError(s.network.App.FeeMarketKeeper.SetParams(ctx, feemParams))

	if s.precompile, err = ics20.NewPrecompile(
		s.network.App.StakingKeeper,
		s.network.App.TransferKeeper,
		*s.network.App.IBCKeeper.ChannelKeeper,
		s.network.App.AuthzKeeper,
	); err != nil {
		panic(err)
	}

	// Create a coordinator and 2 test chains that will be used in the testing suite
	s.coordinator = coordinator.NewIntegrationCoordinator(s.T(), []cmnnetwork.Network{s.network})
	dummyChainsIDs := s.coordinator.GetDummyChainsIDs()
	chainID2 := dummyChainsIDs[0]

	// Setup Chains in the testing suite
	s.chainA = s.coordinator.GetTestChain(s.network.GetChainID())
	s.chainB = s.coordinator.GetTestChain(chainID2)

	// Lower gas requirements on both IBC test chains
	for _, c := range []*ibctesting.TestChain{s.chainA, s.chainB} {
		chainCtx := c.GetContext()
		if haqqApp, ok := c.App.(*haqqapp.Haqq); ok {
			params := haqqApp.FeeMarketKeeper.GetParams(chainCtx)
			params.MinGasPrice = sdkmath.LegacyZeroDec()
			params.NoBaseFee = true
			params.EnableHeight = 0
			_ = haqqApp.FeeMarketKeeper.SetParams(chainCtx, params)
		}
	}

	// Ensure chainB has our sender account with funds and set as default signer
	if haqqB, ok := s.chainB.App.(*haqqapp.Haqq); ok {
		chainBCtx := s.chainB.GetContext()
		addr := s.keyring.GetAccAddr(0)
		acc := haqqB.AccountKeeper.NewAccountWithAddress(chainBCtx, addr)
		haqqB.AccountKeeper.SetAccount(chainBCtx, acc)
		amount := sdkmath.NewInt(1_000_000_000_000_000_000)
		coins := sdk.NewCoins(sdk.NewCoin(s.bondDenom, amount))
		_ = haqqB.BankKeeper.MintCoins(chainBCtx, coinomicstypes.ModuleName, coins)
		_ = haqqB.BankKeeper.SendCoinsFromModuleToAccount(chainBCtx, coinomicstypes.ModuleName, addr, coins)
		// set default signer for chainB
		s.coordinator.SetDefaultSignerForChain(
			chainID2,
			s.keyring.GetPrivKey(0),
			acc,
		)
	}

	// set sender account on chainA
	s.coordinator.SetDefaultSignerForChain(
		s.network.GetChainID(),
		s.keyring.GetPrivKey(0),
		s.network.App.AccountKeeper.GetAccount(ctx, s.keyring.GetAccAddr(0)),
	)

	if s.suiteIBCTesting {
		s.setupIBCTest()
	}
}
