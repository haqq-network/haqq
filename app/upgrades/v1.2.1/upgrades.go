package v121

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.2.1
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Debug("run migration v1.2.1")

		// (evmos v8.2.0) feesplit module is deprecated since it is renamed to "revenue" module
		logger.Debug("deleting feesplit module from version map...")
		delete(vm, "feesplit")

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
