package network

import (
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func (n *IntegrationNetwork) UpdateEvmParams(params evmtypes.Params) error {
	return n.app.EvmKeeper.SetParams(n.ctx, params)
}

func (n *IntegrationNetwork) UpdateCoinomicsParams(params coinomicstypes.Params) error {
	n.app.CoinomicsKeeper.SetParams(n.ctx, params)
	return nil
}

func (n *IntegrationNetwork) UpdateGovParams(params govtypes.Params) error {
	return n.app.GovKeeper.SetParams(n.ctx, params)
}
