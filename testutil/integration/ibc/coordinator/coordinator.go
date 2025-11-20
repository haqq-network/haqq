package coordinator

import (
	"testing"
	"time"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"

	haqqibctesting "github.com/haqq-network/haqq/ibc/testing"
	"github.com/haqq-network/haqq/testutil/integration/common/network"
	ibcchain "github.com/haqq-network/haqq/testutil/integration/ibc/chain"
)

// Coordinator is the interface that defines the methods that are used to
// coordinate the execution of the IBC relayer.
type Coordinator interface {
	// IncrementTime iterates through all the TestChain's and increments their current header time
	// by 5 seconds.
	IncrementTime()
	// IncrementTimeBy iterates through all the TestChain's and increments their current header time
	// by specified time.
	IncrementTimeBy(increment time.Duration)
	// UpdateTime updates all clocks for the TestChains to the current global time.
	UpdateTime()
	// UpdateTimeForChain updates the clock for a specific chain.
	UpdateTimeForChain(chainID string)
	// GetChain returns the abstracted TestChain for a given chainID.
	GetChain(chainID string) ibcchain.Chain
	// GetTestChain returns the original TestChain for a given chainID.
	GetTestChain(chainID string) *ibctesting.TestChain
	// GetDummyChainsIDs returns the chainIDs for all dummy chains.
	GetDummyChainsIDs() []string
	// GetPath returns the transfer path for the chain ids 'a' and 'b'
	GetPath(a, b string) *haqqibctesting.Path
	// GetChainSenderAcc returns the sender account for the specified chain
	GetChainSenderAcc(chainID string) sdk.AccountI
	// SetDefaultSignerForChain sets the default signer for the chain with the given chainID.
	SetDefaultSignerForChain(chainID string, priv cryptotypes.PrivKey, acc sdk.AccountI)
	// Setup constructs a TM client, connection, and channel on both chains provided. It will
	// fail if any error occurs. The clientID's, TestConnections, and TestChannels are returned
	// for both chains. The channels created are connected to the ibc-transfer application.
	Setup(src, dst string) *haqqibctesting.Path
	// CommitBlock commits 1 block on the chain(s) with the given chainID(s).
	CommitBlock(chainIDs ...string)
	// CommitNBlocks commits n blocks on the chain with the given chainID.
	CommitNBlocks(chainID string, n uint64)
	// CommitAll commits 1 blocks on all chains within the coordinator.
	CommitAll()
}

var AmountOfDummyChains = 2

var _ Coordinator = (*IntegrationCoordinator)(nil)

// IntegrationCoordinator is a testing struct which contains N TestChain's. It handles keeping all chains
// in sync with regards to time.
// NOTE: When using the coordinator, it is important to commit blocks through the coordinator and not
// through the network interface directly. This is because the coordinator does not keep the context in
// sync with the network interface.
type IntegrationCoordinator struct {
	coord          *ibctesting.Coordinator
	dummyChainsIDs []string
}

// NewIntegrationCoordinator returns a new IntegrationCoordinator with N TestChain's.
func NewIntegrationCoordinator(t *testing.T, preConfiguredChains []network.Network) *IntegrationCoordinator {
	coord := &ibctesting.Coordinator{
		T:           t,
		CurrentTime: time.Now(),
	}
	ibcChains := getIBCChains(t, coord, preConfiguredChains)
	dummyChains, dummyChainsIDs := generateDummyChains(t, coord, AmountOfDummyChains)
	totalChains := mergeMaps(ibcChains, dummyChains)
	coord.Chains = totalChains
	return &IntegrationCoordinator{
		coord:          coord,
		dummyChainsIDs: dummyChainsIDs,
	}
}

// GetChain returns the TestChain for a given chainID but abstracted to our internal chain interface.
func (c *IntegrationCoordinator) GetChain(chainID string) ibcchain.Chain {
	return c.coord.GetChain(chainID)
}

// GetTestChain returns the TestChain for a given chainID.
func (c *IntegrationCoordinator) GetTestChain(chainID string) *ibctesting.TestChain {
	return c.coord.GetChain(chainID)
}

// GetDummyChainsIDs returns the chainIDs for all dummy chains.
func (c *IntegrationCoordinator) GetDummyChainsIDs() []string {
	return c.dummyChainsIDs
}

// GetPath returns the transfer path for the chain ids 'a' and 'b'
func (c *IntegrationCoordinator) GetPath(a, b string) *haqqibctesting.Path {
	chainA := c.coord.GetChain(a)
	chainB := c.coord.GetChain(b)

	return haqqibctesting.NewTransferPath(chainA, chainB)
}

// GetChainSenderAcc returns the TestChain's SenderAccount for a given chainID.
func (c *IntegrationCoordinator) GetChainSenderAcc(chainID string) sdk.AccountI {
	return c.coord.Chains[chainID].SenderAccount
}

// IncrementTime iterates through all the TestChain's and increments their current header time
// by 5 seconds.
//
// CONTRACT: this function must be called after every Commit on any TestChain.
func (c *IntegrationCoordinator) IncrementTime() {
	c.coord.IncrementTime()
}

// IncrementTimeBy iterates through all the TestChain's and increments their current header time
// by specified time.
func (c *IntegrationCoordinator) IncrementTimeBy(increment time.Duration) {
	c.coord.IncrementTimeBy(increment)
}

// UpdateTime updates all clocks for the TestChains to the current global time.
func (c *IntegrationCoordinator) UpdateTime() {
	c.coord.UpdateTime()
}

// UpdateTimeForChain updates the clock for a specific chain.
func (c *IntegrationCoordinator) UpdateTimeForChain(chainID string) {
	chain := c.coord.GetChain(chainID)
	c.coord.UpdateTimeForChain(chain)
}

// SetDefaultSignerForChain sets the default signer for the chain with the given chainID.
func (c *IntegrationCoordinator) SetDefaultSignerForChain(chainID string, priv cryptotypes.PrivKey, acc sdk.AccountI) {
	chain := c.coord.GetChain(chainID)
	chain.SenderPrivKey = priv
	chain.SenderAccount = acc
	chain.SenderAccounts = []ibctesting.SenderAccount{{SenderPrivKey: priv, SenderAccount: acc}}
}

// Setup constructs a TM client, connection, and channel on both chains provided. It will
// fail if any error occurs. The clientID's, TestConnections, and TestChannels are returned
// for both chains. The channels created are connected to the ibc-transfer application.
func (c *IntegrationCoordinator) Setup(a, b string) *haqqibctesting.Path {
	path := c.GetPath(a, b)
	haqqibctesting.SetupPath(c.coord, path)

	return path
}

// CommitBlock commits 1 block on the chain(s) with the given chainID(s).
func (c *IntegrationCoordinator) CommitBlock(chainIDs ...string) {
	for _, chainID := range chainIDs {
		chain := c.coord.GetChain(chainID)
		c.coord.CommitBlock(chain)
	}
}

// CommitNBlocks commits n blocks on the chain with the given chainID.
func (c *IntegrationCoordinator) CommitNBlocks(chainID string, n uint64) {
	chain := c.coord.GetChain(chainID)
	c.coord.CommitNBlocks(chain, n)
}

// CommitAll commits n blocks on the chain with the given chainID.
func (c *IntegrationCoordinator) CommitAll() {
	for _, chain := range c.coord.Chains {
		c.coord.CommitBlock(chain)
	}
}
