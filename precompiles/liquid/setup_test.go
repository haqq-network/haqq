package liquid_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	"github.com/haqq-network/haqq/precompiles/liquid"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	liquidtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
)

type PrecompileTestSuite struct {
	suite.Suite

	network     *network.UnitTestNetwork
	factory     factory.TxFactory
	grpcHandler grpc.Handler
	keyring     testkeyring.Keyring
	precompile  *liquid.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	suite.Run(t, new(PrecompileTestSuite))
}

func (s *PrecompileTestSuite) SetupTest() {
	keyring := testkeyring.New(2)

	customGenesis := network.CustomGenesisState{}
	lvGenesis := liquidtypes.DefaultGenesisState()
	lvGenesis.Params.MinimumLiquidationAmount = sdkmath.NewInt(1_000_000)
	customGenesis[liquidtypes.ModuleName] = lvGenesis

	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
		network.WithCustomGenesis(customGenesis),
	)
	grpcHandler := grpc.NewIntegrationHandler(nw)
	txFactory := factory.New(nw, grpcHandler)

	s.keyring = keyring
	s.network = nw
	s.factory = txFactory
	s.grpcHandler = grpcHandler

	var err error
	s.precompile, err = liquid.NewPrecompile(
		nw.App.LiquidVestingKeeper,
		nw.App.AuthzKeeper,
	)
	s.Require().NoError(err)
}
