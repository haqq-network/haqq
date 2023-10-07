package network

import (
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	pruningtypes "github.com/cosmos/cosmos-sdk/pruning/types"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	dbm "github.com/tendermint/tm-db"

	"github.com/haqq-network/haqq/app"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
)

type AppConstructor = func(val network.Validator) servertypes.Application

// NewAppConstructor returns a new simapp AppConstructor
func NewAppConstructor(encodingCfg params.EncodingConfig) network.AppConstructor {
	return func(val network.Validator) servertypes.Application {
		return app.NewHaqq(
			val.Ctx.Logger, dbm.NewMemDB(), nil, true, make(map[int64]bool), val.Ctx.Config.RootDir, 0,
			encodingCfg,
			simapp.EmptyAppOptions{},
			baseapp.SetPruning(pruningtypes.NewPruningOptionsFromString(val.AppConfig.Pruning)),
			baseapp.SetMinGasPrices(val.AppConfig.MinGasPrices),
		)
	}
}

func HaqqNetworkConfig() network.Config {
	defaultConfig := network.DefaultConfig()
	defaultConfig.BondDenom = "aISLM"
	defaultConfig.MinGasPrices = "0.000006aISLM"
	defaultConfig.TimeoutCommit = time.Nanosecond
	defaultConfig.ChainID = "haqq_11235-1"

	// some params for coinomics module
	defaultConfig.GenesisState[coinomicstypes.ModuleName] = []byte("{\"params\":{\"mint_denom\":\"aISLM\",\"blocks_per_era\":\"100\",\"enable_coinomics\":true},\"inflation\":\"0.000000000000000000\",\"era\":\"0\",\"era_started_at_block\":\"0\",\"era_target_mint\":{\"denom\":\"aISLM\",\"amount\":\"0\"},\"era_closing_supply\":{\"denom\":\"aISLM\",\"amount\":\"0\"},\"max_supply\":{\"denom\":\"aISLM\",\"amount\":\"100000000000000000000000000000\"}}")

	return defaultConfig
}

func HaqqNetworkConfigCoinomicsDisabled() network.Config {
	defaultConfig := network.DefaultConfig()
	defaultConfig.BondDenom = "aISLM"
	defaultConfig.MinGasPrices = "0.000006aISLM"
	defaultConfig.ChainID = "haqq_11235-1"

	// some params for coinomics module
	defaultConfig.GenesisState[coinomicstypes.ModuleName] = []byte("{\"params\":{\"mint_denom\":\"aISLM\",\"blocks_per_era\":\"100\",\"enable_coinomics\":false},\"inflation\":\"0.000000000000000000\",\"era\":\"0\",\"era_started_at_block\":\"0\",\"era_target_mint\":{\"denom\":\"aISLM\",\"amount\":\"0\"},\"era_closing_supply\":{\"denom\":\"aISLM\",\"amount\":\"0\"},\"max_supply\":{\"denom\":\"aISLM\",\"amount\":\"100000000000000000000000000000\"}}")

	return defaultConfig
}
