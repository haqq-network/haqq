package v1100

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.10.0
// This upgrade migrates from Cosmos SDK v0.50.9 to v0.53.4 and IBC v8 to v10
//
// Key changes in v0.53.x:
// - x/auth module now contains a PreBlocker (already configured in app.go SetOrderPreBlockers)
// - Support for unordered transactions (already enabled in app.go with WithUnorderedTransactions)
// - Module migrations are handled automatically by RunMigrations
//
// Reference: https://github.com/cosmos/cosmos-sdk/blob/v0.53.0/UPGRADING.md
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)
		logger := ctx.Logger().With("upgrade", UpgradeName)

		// Run module migrations for SDK v0.53.4 and IBC v10
		// This will automatically handle all module migrations including:
		// - Cosmos SDK v0.50.9 -> v0.53.4 migrations
		// - IBC v8 -> v10 migrations
		logger.Info("Running module migrations for SDK v0.53.4 and IBC v10...")
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			logger.Error("Migration failed", "error", err)
			return vm, err
		}

		logger.Info("Successfully completed v1.10.0 upgrade: Cosmos SDK v0.50.9 -> v0.53.4, IBC v8 -> v10")
		return vm, nil
	}
}
