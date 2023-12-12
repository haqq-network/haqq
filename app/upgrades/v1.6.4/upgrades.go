package v164

import (
	"strings"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/haqq-network/haqq/utils"
	coinomicskeeper "github.com/haqq-network/haqq/x/coinomics/keeper"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.6.4
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	storeKey storetypes.StoreKey,
	paramsStoreKey storetypes.StoreKey,
	dk distrkeeper.Keeper,
	ck coinomicskeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		MigrateStore(ctx, storeKey, paramsStoreKey, dk, ck)

		return mm.RunMigrations(ctx, configurator, vm)
	}
}

// MigrateStore migrates the x/coinomics module state
func MigrateStore(ctx sdk.Context, storeKey storetypes.StoreKey,
	paramsStoreKey storetypes.StoreKey, dk distrkeeper.Keeper,
	ck coinomicskeeper.Keeper) {
	// clean module state
	store := ctx.KVStore(storeKey)
	iterator := sdk.KVStorePrefixIterator(store, nil)

	for ; iterator.Valid(); iterator.Next() {
		println("coinomics store key for delete")
		println(string(iterator.Key()))

		store.Delete(iterator.Key())
	}

	// clean params
	paramsStore := ctx.KVStore(paramsStoreKey)
	paramsIterator := sdk.KVStorePrefixIterator(paramsStore, nil)

	for ; paramsIterator.Valid(); paramsIterator.Next() {
		if strings.Contains(string(paramsIterator.Key()), "coinomics/") {
			println("coinomics store key/value")

			println(string(paramsIterator.Key()))
			println(string(paramsIterator.Value()))

			paramsStore.Delete(paramsIterator.Key())
		}
	}

	// reset coinomics params
	defaultParams := coinomicstypes.DefaultParams()

	if utils.IsMainNetwork(ctx.ChainID()) {
		defaultParams.EnableCoinomics = false
	}

	maxSupply := sdk.Coin{Denom: "aISLM", Amount: math.NewIntWithDecimal(100_000_000_000, 18)} // 100bn ISLM

	ck.SetMaxSupply(ctx, maxSupply)
	ck.SetParams(ctx, defaultParams)
}
