package v180

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	evmkeeper "github.com/haqq-network/haqq/x/evm/keeper"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.8.0
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ek evmkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger().With("upgrade", UpgradeName)

		logger.Info("setting precompiles parameters")
		if err := setPrecompilesParams(ctx, ek); err != nil {
			logger.Error("error while setting precompiles parameters", "error", err)
		}

		// Leave modules are as-is to avoid running InitGenesis.
		logger.Debug("running module migrations ...")
		return mm.RunMigrations(ctx, configurator, vm)
	}
}

func setPrecompilesParams(ctx sdk.Context, ek evmkeeper.Keeper) error {
	params := ek.GetParams(ctx)
	params.ActivePrecompiles = evmtypes.AvailableEVMExtensions
	return ek.SetParams(ctx, params)
}
