package v164

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const ModuleName = "coinomics"

// CreateUpgradeHandler creates an SDK upgrade handler for v1.6.4
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	storeKey storetypes.StoreKey,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()

		logger.Info("start clean coinomics store")

		// Используйте переданный ключ хранилища
		store := ctx.KVStore(storeKey)

		iterator := sdk.KVStorePrefixIterator(store, nil)

		for ; iterator.Valid(); iterator.Next() {
			store.Delete(iterator.Key())
		}

		iterator.Close() // Не забудьте закрыть итератор
		logger.Info("cleared coinomics store")

		logger.Info("start default sdk migration for v1.6.4")

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
