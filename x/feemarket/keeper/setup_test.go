package keeper_test

import (
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"

	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
)

type KeeperTestSuite struct {
	suite.Suite

	network     *network.UnitTestNetwork
	factory     factory.TxFactory
	grpcHandler grpc.Handler
	keyring     testkeyring.Keyring

	denom string
}

// SetupTest setup test environment
func (s *KeeperTestSuite) SetupTest() {
	keyring := testkeyring.New(2)
	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
		network.WithCustomBaseAppOpts(baseapp.SetMinGasPrices("10aevmos")),
	)
	grpcHandler := grpc.NewIntegrationHandler(nw)
	txFactory := factory.New(nw, grpcHandler)

	ctx := nw.GetContext()
	sk := nw.App.StakingKeeper
	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		panic(err)
	}

	s.denom = bondDenom
	s.factory = txFactory
	s.grpcHandler = grpcHandler
	s.keyring = keyring
	s.network = nw
}
