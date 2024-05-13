package v175

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	liquidvestingkeeper "github.com/haqq-network/haqq/x/liquidvesting/keeper"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.7.5
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	bk bankkeeper.Keeper,
	lk liquidvestingkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()

		logger.Info("##############################################")
		logger.Info("############  RUN UPGRADE v1.7.5  ############")
		logger.Info("##############################################")

		logger.Info("Start turning off liquid vesting module")
		if err := TurnOffLiquidVesting(ctx, bk, lk); err != nil {
			panic(err)
		}

		logger.Info("##############################################")
		logger.Info("#############  UPGRADE COMPLETE  #############")
		logger.Info("##############################################")

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
