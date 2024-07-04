package v177

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.7.7
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		vm, vmErr := mm.RunMigrations(ctx, configurator, vm)

		logger.Info("##############################################")
		logger.Info("############  RUN UPGRADE v1.7.7  ############")
		logger.Info("##############################################")
		logger.Info("##############################################")
		logger.Info("#############  UPGRADE COMPLETE  #############")
		logger.Info("##############################################")

		return vm, vmErr
	}
}
