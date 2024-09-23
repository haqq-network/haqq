package v181

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.8.1
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ak authkeeper.AccountKeeper,
	sk stakingkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger().With("upgrade", UpgradeName)

		logger.Info("migrate vesting accounts")
		migrateVestingAccounts(ctx, ak, sk)

		// Leave modules are as-is to avoid running InitGenesis.
		logger.Debug("running module migrations ...")
		return mm.RunMigrations(ctx, configurator, vm)
	}
}

func migrateVestingAccounts(ctx sdk.Context, ak authkeeper.AccountKeeper, sk stakingkeeper.Keeper) {
	logger := ctx.Logger().With("upgrade", UpgradeName)
	bondDenom := sk.BondDenom(ctx)
	var vaCount int64

	ak.IterateAccounts(ctx, func(acc types.AccountI) bool {
		va, ok := acc.(*vestingtypes.ClawbackVestingAccount)
		if !ok {
			return false
		}

		bondedAmt := sk.GetDelegatorBonded(ctx, va.GetAddress())
		unbondingAmt := sk.GetDelegatorUnbonding(ctx, va.GetAddress())
		delegatedAmt := bondedAmt.Add(unbondingAmt)
		delegated := sdk.NewCoins(sdk.NewCoin(bondDenom, delegatedAmt))
		zeroCoins := sdk.NewCoins()

		logger.Info(fmt.Sprintf("update vesting account: %s", va.GetAddress()))
		logger.Info(fmt.Sprintf(" - set DelegatedVesting: %s -> %s", va.DelegatedVesting.String(), zeroCoins.String()))
		logger.Info(fmt.Sprintf(" - set DelegatedFree: %s -> %s", va.DelegatedFree.String(), delegated.String()))

		va.DelegatedVesting = sdk.NewCoins()
		va.DelegatedFree = delegated

		logger.Info("---")
		ak.SetAccount(ctx, va)
		vaCount++
		return false
	})

	logger.Info(fmt.Sprintf("Updated Clawback Vesting Accounts: %d", vaCount))
}
