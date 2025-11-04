package network

import (
	"testing"

	cmttypes "github.com/cometbft/cometbft/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// GetIBCChain returns a TestChain instance for the given network.
// Note: the sender accounts are not populated. Do not use this accounts to send transactions during tests.
// The keyring should be used instead.
func (n *IntegrationNetwork) GetIBCChain(t *testing.T, coord *ibctesting.Coordinator) *ibctesting.TestChain {
	// In ibc-go v10, TestChain structure changed - use NewTestChainWithValSet or similar constructor
	// For now, create a minimal chain structure compatible with v10
	chain := &ibctesting.TestChain{
		TB:                t,
		Coordinator:       coord,
		ChainID:           n.GetChainID(),
		App:               n.app,
		ProposedHeader:    n.ctx.BlockHeader(),
		TxConfig:          n.app.GetTxConfig(),
		Codec:             n.app.AppCodec(),
		Vals:              n.valSet,
		NextVals:          n.valSet,
		Signers:           n.valSigners,
		TrustedValidators: make(map[uint64]*cmttypes.ValidatorSet),
	}
	return chain
}
