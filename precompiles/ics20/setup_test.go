// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package ics20_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	haqqibctesting "github.com/haqq-network/haqq/ibc/testing"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/ics20"
	cmnnetwork "github.com/haqq-network/haqq/testutil/integration/common/network"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	"github.com/haqq-network/haqq/testutil/integration/ibc/coordinator"
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
	keyring := testkeyring.New(2)
	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	)
	grpcHandler := grpc.NewIntegrationHandler(nw)
	txFactory := factory.New(nw, grpcHandler)

	ctx := nw.GetContext()
	sk := nw.App.StakingKeeper
	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		panic(err)
	}

	s.bondDenom = bondDenom
	s.factory = txFactory
	s.grpcHandler = grpcHandler
	s.keyring = keyring
	s.network = nw

	s.network.NextBlock()

	if s.precompile, err = ics20.NewPrecompile(
		s.network.App.StakingKeeper,
		s.network.App.TransferKeeper,
		s.network.App.IBCKeeper.ChannelKeeper,
		s.network.App.AuthzKeeper,
	); err != nil {
		panic(err)
	}

	// Create a coordinator and 2 test chains that will be used in the testing suite
	s.coordinator = coordinator.NewIntegrationCoordinator(s.T(), []cmnnetwork.Network{s.network})
	dummyChainsIDs := s.coordinator.GetDummyChainsIDs()
	chainID2 := dummyChainsIDs[0]

	// Setup Chains in the testing suite
	s.chainA = s.coordinator.GetTestChain(cmn.DefaultChainID)
	s.chainB = s.coordinator.GetTestChain(chainID2)

	// set sender account on chainA
	s.coordinator.SetDefaultSignerForChain(
		cmn.DefaultChainID,
		s.keyring.GetPrivKey(0),
		s.network.App.AccountKeeper.GetAccount(ctx, s.keyring.GetAccAddr(0)),
	)

	if s.suiteIBCTesting {
		s.setupIBCTest()
	}
}
