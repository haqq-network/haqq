package v1_2_0

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibc "github.com/cosmos/ibc-go/v3/modules/core"
	ibchost "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.0.2
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Debug("run migration v1.2.0")

		moduleVersion := ibc.AppModule{}.ConsensusVersion()
		ibcVersion, exists := vm[ibchost.ModuleName]
		if !exists || ibcVersion < moduleVersion {
			vm[ibchost.ModuleName] = moduleVersion
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
