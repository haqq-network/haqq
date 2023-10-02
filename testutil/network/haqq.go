package network

import (
	"github.com/haqq-network/haqq/utils"
	"time"

	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
)

func HaqqNetworkConfig() Config {
	defaultConfig := DefaultConfig()
	defaultConfig.BondDenom = utils.BaseDenom
	defaultConfig.MinGasPrices = "0.000006aISLM"
	defaultConfig.TimeoutCommit = time.Nanosecond
	defaultConfig.ChainID = "haqq_11235-1"

	// some params for coinomics module
	defaultConfig.GenesisState[coinomicstypes.ModuleName] = []byte("{\"params\":{\"mint_denom\":\"aISLM\",\"blocks_per_era\":\"100\",\"enable_coinomics\":true},\"inflation\":\"0.000000000000000000\",\"era\":\"0\",\"era_started_at_block\":\"0\",\"era_target_mint\":{\"denom\":\"aISLM\",\"amount\":\"0\"},\"era_closing_supply\":{\"denom\":\"aISLM\",\"amount\":\"0\"},\"max_supply\":{\"denom\":\"aISLM\",\"amount\":\"100000000000000000000000000000\"}}")

	return defaultConfig
}

func HaqqNetworkConfigCoinomicsDisabled() Config {
	defaultConfig := DefaultConfig()
	defaultConfig.BondDenom = utils.BaseDenom
	defaultConfig.MinGasPrices = "0.000006aISLM"
	defaultConfig.ChainID = "haqq_11235-1"

	// some params for coinomics module
	defaultConfig.GenesisState[coinomicstypes.ModuleName] = []byte("{\"params\":{\"mint_denom\":\"aISLM\",\"blocks_per_era\":\"100\",\"enable_coinomics\":false},\"inflation\":\"0.000000000000000000\",\"era\":\"0\",\"era_started_at_block\":\"0\",\"era_target_mint\":{\"denom\":\"aISLM\",\"amount\":\"0\"},\"era_closing_supply\":{\"denom\":\"aISLM\",\"amount\":\"0\"},\"max_supply\":{\"denom\":\"aISLM\",\"amount\":\"100000000000000000000000000000\"}}")

	return defaultConfig
}
