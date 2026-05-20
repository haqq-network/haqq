package keeper_test

import (
	"time"

	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	"github.com/haqq-network/haqq/x/epochs/types"
)

const (
	day             = time.Hour * 24
	week            = time.Hour * 24 * 7
	month           = time.Hour * 24 * 31
	monthIdentifier = "month"
)

// KeeperTestSuite is the implementation of the test suite for the
// Epochs module.
type KeeperTestSuite struct {
	network *network.UnitTestNetwork
	keyring keyring.Keyring
	handler grpc.Handler
}

// SetupTest is the setup function for epoch module tests. If epochsInfo is provided empty
// the default genesis for the epoch module is used. Extra network options can be passed
// to customize the underlying UnitTestNetwork (e.g. network.WithStartTime).
func SetupTest(epochsInfo []types.EpochInfo, opts ...network.ConfigOption) *KeeperTestSuite {
	keys := keyring.New(1)

	customGenesis := network.CustomGenesisState{}
	epochsGenesis := types.DefaultGenesisState()

	if len(epochsInfo) > 0 {
		epochsGenesis = types.NewGenesisState(epochsInfo)
	}

	customGenesis[types.ModuleName] = epochsGenesis

	nwOpts := append([]network.ConfigOption{
		network.WithPreFundedAccounts(keys.GetAllAccAddrs()...),
		network.WithCustomGenesis(customGenesis),
	}, opts...)
	nw := network.NewUnitTestNetwork(nwOpts...)

	gh := grpc.NewIntegrationHandler(nw)

	return &KeeperTestSuite{
		network: nw,
		keyring: keys,
		handler: gh,
	}
}
