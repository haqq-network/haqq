package v190

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	vestingkeeper "github.com/haqq-network/haqq/x/vesting/keeper"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.9.0
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ak authkeeper.AccountKeeper,
	vk vestingkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Info("##############################################")
		logger.Info("############ Run migration v1.9.0 ############")
		logger.Info("########## CONVERT VESTING ACCOUNTS ##########")
		logger.Info("########### BACK INTO ETH ACCOUNTS ###########")
		logger.Info("##############################################")

		if err := ConvertClawbackVestingAccountsIntoEth(ctx, ak, vk); err != nil {
			panic(err)
		}

		logger.Info("##############################################")
		logger.Info("############ CONVERSION COMPLETE! ############")
		logger.Info("##############################################")

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
