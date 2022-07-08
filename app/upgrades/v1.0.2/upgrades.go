package v1_0_2

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/haqq-network/haqq/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	erc20types "github.com/tharsis/evmos/v4/x/erc20/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.0.2
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	bk bankkeeper.Keeper,
	cdc codec.BinaryCodec,
	distrStoreKey sdk.StoreKey,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		// Refs:
		// - https://docs.cosmos.network/master/building-modules/upgrade.html#registering-migrations
		// - https://docs.cosmos.network/master/migrations/chain-upgrade-guide-044.html#chain-upgrade

		if types.IsMainNetwork(ctx.ChainID()) ||
			types.IsTestEdgeNetwork(ctx.ChainID()) ||
			types.IsLocalNetwork(ctx.ChainID()) {
			logger.Debug("run migration v1.0.2")

			FixTotalSupply(ctx, bk, cdc, distrStoreKey)
		}

		// Leave modules are as-is to avoid running InitGenesis.
		return mm.RunMigrations(ctx, configurator, vm)
	}
}

// ref: https://github.com/haqq-network/haqq/issues/4
func FixTotalSupply(ctx sdk.Context, bk bankkeeper.Keeper, cdc codec.BinaryCodec, distrStoreKey sdk.StoreKey) {
	// amount
	mintAmount := sdk.NewCoins(sdk.NewCoin("aISLM", sdk.NewInt(70000000)))

	// mint to bank module
	bk.MintCoins(ctx, erc20types.ModuleName, mintAmount)

	// send coins to distribution module
	bk.SendCoinsFromModuleToModule(ctx, erc20types.ModuleName, distrtypes.ModuleName, mintAmount)

	// update community pool amount
	kvstore := ctx.MultiStore().GetKVStore(distrStoreKey)
	feePoolBin := kvstore.Get(distrtypes.FeePoolKey)

	var feePool distrtypes.FeePool
	cdc.MustUnmarshal(feePoolBin, &feePool)

	coins := sdk.NewDecCoinsFromCoins(mintAmount...)
	feePool.CommunityPool = feePool.CommunityPool.Add(coins...)

	b := cdc.MustMarshal(&feePool)
	kvstore.Set(distrtypes.FeePoolKey, b)
}
