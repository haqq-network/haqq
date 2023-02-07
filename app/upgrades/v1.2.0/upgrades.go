package v1_2_0

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibc "github.com/cosmos/ibc-go/v5/modules/core"
	ibchost "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.2.0
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Debug("run migration v1.2.0")

		// Skip module migrations if existing version is greater than new one (should not happen).
		moduleVersion := ibc.AppModule{}.ConsensusVersion()
		if ibcVersion, exists := vm[ibchost.ModuleName]; exists && ibcVersion > moduleVersion {
			vm[ibchost.ModuleName] = moduleVersion
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
