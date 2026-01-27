package ucdao_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/precompiles/ucdao"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
)

var s *PrecompileTestSuite

// PrecompileTestSuite is the implementation of the TestSuite interface for UCDAO precompile
// unit tests.
type PrecompileTestSuite struct {
	suite.Suite

	bondDenom string

	network     *network.UnitTestNetwork
	factory     factory.TxFactory
	grpcHandler grpc.Handler
	keyring     testkeyring.Keyring

	precompile *ucdao.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(PrecompileTestSuite)
	suite.Run(t, s)
}

func (s *PrecompileTestSuite) SetupTest() sdk.Context {
	keyring := testkeyring.New(2)
	unitNetwork := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	)
	grpcHandler := grpc.NewIntegrationHandler(unitNetwork)
	txFactory := factory.New(unitNetwork, grpcHandler)

	ctx := unitNetwork.GetContext()
	sk := unitNetwork.App.StakingKeeper
	bondDenom, err := sk.BondDenom(ctx)
	s.Require().NoError(err, "failed to get bond denom")
	s.Require().NotEmpty(bondDenom, "bond denom cannot be empty")

	s.bondDenom = bondDenom
	s.factory = txFactory
	s.grpcHandler = grpcHandler
	s.keyring = keyring
	s.network = unitNetwork

	s.precompile = s.setupUcdaoPrecompile()
	return ctx
}

// setupUcdaoPrecompile is a helper function to set up an instance of the UCDAO precompile.
func (s *PrecompileTestSuite) setupUcdaoPrecompile() *ucdao.Precompile {
	precompile, err := ucdao.NewPrecompile(
		s.network.App.DaoKeeper,
		s.network.App.BankKeeper,
		s.network.App.AuthzKeeper,
	)

	s.Require().NoError(err, "failed to create ucdao precompile")

	return precompile
}
