package v174

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	liquidvestingkeeper "github.com/haqq-network/haqq/x/liquidvesting/keeper"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.7.3
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ak authkeeper.AccountKeeper,
	lk liquidvestingkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Info("##############################################")
		logger.Info("############ Run migration v1.7.4 ############")
		logger.Info("################ REVESTING V2 ################")
		logger.Info("##############################################")

		if err := StretchLockupScheduleForAccounts(ctx, ak, VestingStretchLength, time.Unix(LockupLengthThreshold, 0)); err != nil {
			panic(err)
		}

		if err := StretchLockupScheduleForLiquidVestingTokens(ctx, lk, VestingStretchLength, time.Unix(LockupLengthThreshold, 0)); err != nil {
			panic(err)
		}

		logger.Info("##############################################")
		logger.Info("############# REVESTING COMPLETE #############")
		logger.Info("##############################################")

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
