package v176

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	erc20keeper "github.com/haqq-network/haqq/x/erc20/keeper"
	liquidvestingkeeper "github.com/haqq-network/haqq/x/liquidvesting/keeper"
	daokeeper "github.com/haqq-network/haqq/x/ucdao/keeper"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.7.6
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	sk stakingkeeper.Keeper,
	dk daokeeper.Keeper,
	lk liquidvestingkeeper.Keeper,
	erc20k erc20keeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()

		vm, vmErr := mm.RunMigrations(ctx, configurator, vm)

		logger.Info("##############################################")
		logger.Info("############  RUN UPGRADE v1.7.6  ############")
		logger.Info("##############################################")

		if err := FixLockupPeriods(ctx, ak); err != nil {
			logger.Error(fmt.Sprintf("Error fixing lockup periods: %s", err.Error()))
		}

		if err := TurnOnDAO(ctx, bk, lk, ak, sk, dk, erc20k); err != nil {
			logger.Error(fmt.Sprintf("Error turning on DAO: %s", err.Error()))
		}

		logger.Info("##############################################")
		logger.Info("#############  UPGRADE COMPLETE  #############")
		logger.Info("##############################################")

		return vm, vmErr
	}
}
