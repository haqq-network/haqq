package coordinator

import (
	"encoding/json"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"

	haqqapp "github.com/haqq-network/haqq/app"
	haqqibctesting "github.com/haqq-network/haqq/ibc/testing"
	"github.com/haqq-network/haqq/testutil/integration/common/network"
)

func init() {
	// Set the default testing app init function to use our Haqq app
	// ibctesting.DefaultTestingAppInit expects func() (TestingApp, map[string]json.RawMessage)
	// but our SetupTestingApp returns func(chainID string) func() (TestingApp, map[string]json.RawMessage)
	// So we create a wrapper that uses a default chainID
	ibctesting.DefaultTestingAppInit = func() (ibctesting.TestingApp, map[string]json.RawMessage) {
		appInit := haqqapp.SetupTestingApp("test-chain")
		app, genesis := appInit()
		return app, genesis
	}
}

// getIBCChains returns a map of TestChain's for the given network interface.
func getIBCChains(t *testing.T, coord *ibctesting.Coordinator, chains []network.Network) map[string]*ibctesting.TestChain {
	ibcChains := make(map[string]*ibctesting.TestChain)
	for _, chain := range chains {
		ibcChains[chain.GetChainID()] = chain.GetIBCChain(t, coord)
	}
	return ibcChains
}

// generateDummyChains returns a map of dummy chains to complement IBC connections for integration tests.
func generateDummyChains(t *testing.T, coord *ibctesting.Coordinator, numberOfChains int) (map[string]*ibctesting.TestChain, []string) {
	ibcChains := make(map[string]*ibctesting.TestChain)
	ids := make([]string, numberOfChains)
	// dummy chains use the ibc testing chain setup
	// that uses the default sdk address prefix ('cosmos')
	// Update the prefix configs to use that prefix
	haqqibctesting.SetBech32Prefix("cosmos")
	// Also need to disable address cache to avoid using modules
	// accounts with 'evmos' addresses (because Evmos chain setup is first)
	sdk.SetAddrCacheEnabled(false)
	for i := 1; i <= numberOfChains; i++ {
		// Use valid Haqq chain ID format: {prefix}_{EIP155}-{epoch}
		chainID := fmt.Sprintf("dummy_%d-1", i)
		ids[i-1] = chainID
		ibcChains[chainID] = ibctesting.NewTestChain(t, coord, chainID)
	}
	return ibcChains, ids
}

// mergeMaps merges two maps of TestChain's.
func mergeMaps(m1, m2 map[string]*ibctesting.TestChain) map[string]*ibctesting.TestChain {
	for k, v := range m2 {
		m1[k] = v
	}
	return m1
}
