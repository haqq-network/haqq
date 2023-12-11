package v164

import (
	// storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	// paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	// coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
)

const ModuleName = "coinomics"

// CreateUpgradeHandler creates an SDK upgrade handler for v1.6.4
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	// storeKey storetypes.StoreKey,
	// paramsStoreKey storetypes.StoreKey,
	// paramsSubspace paramtypes.Subspace,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()

		// logger.Info("start clean coinomics store")

		// // Используйте переданный ключ хранилища
		// store := ctx.KVStore(storeKey)

		// iterator := sdk.KVStorePrefixIterator(store, nil)

		// for ; iterator.Valid(); iterator.Next() {
		// 	println("coinomics store key for delete")
		// 	println(string(iterator.Key()))

		// 	store.Delete(iterator.Key())
		// }

		// iterator.Close() // Не забудьте закрыть итератор
		// logger.Info("cleared coinomics store")

		// logger.Info("start cleaning params for module")

		// keyTable := coinomicstypes.ParamKeyTable()
		// paramsSubspace = paramsSubspace.WithKeyTable(keyTable)

		// paramsSubspace.SetParamSet(ctx, &coinomicstypes.Params{})

		// if paramsSubspace.HasKeyTable() {

		// 	paramsSubspace.WithKeyTable(paramtypes.NewKeyTable())
		// 	// paramsSubspace.IterateKeys(ctx, func(key []byte) bool {
		// 	// 	println(string(key))

		// 	// 	paramsSubspace.Update(ctx, key, nil)
		// 	// 	return false
		// 	// })
		// }

		// if !ps.HasKeyTable() {
		// 	ps = ps.WithKeyTable(types.ParamKeyTable())
		// }

		// paramsSubspace.Set(ctx, []byte(ModuleName), nil) // Получаем пространство параметров для модуля
		// iter := storeParams.Iterator(nil, nil)           // Создаем итератор для всех ключей
		// defer iter.Close()

		// for ; iter.Valid(); iter.Next() {
		// 	storeParams.Delete(iter.Key()) // Удаляем каждый ключ
		// }

		logger.Info("start default sdk migration for v1.6.4")

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
