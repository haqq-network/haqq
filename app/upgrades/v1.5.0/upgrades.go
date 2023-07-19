package v150

import (
	"strconv"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	dbm "github.com/tendermint/tm-db"

	vestingkeeper "github.com/haqq-network/haqq/x/vesting/keeper"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.4.0
func CreateUpgradeHandler(
	db dbm.DB,
	keys map[string]*storetypes.KVStoreKey,
	cdc codec.Codec,
	mm *module.Manager,
	configurator module.Configurator,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	sk stakingkeeper.Keeper,
	evm *evmkeeper.Keeper,
	vk vestingkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Info("### Run migration v1.5.0 ###")
		logger.Info("######### REVESTING ########")

		// TODO PUT HERE THE HEIGHT "BEFORE" THE UPGRADE AND BALANCE THRESHOLD
		ts := math.NewIntFromUint64(exp) // 1 ISLM
		ts = ts.MulRaw(threshold)        // 10 ISLM for tests

		logger.Info("# Upgrade height: " + strconv.FormatInt(ctx.BlockHeight(), 10))
		logger.Info("# History state height: " + strconv.FormatInt(historyStateHeight, 10))
		logger.Info("# Balance threshold: " + ts.String() + " aISLM")

		revesting := NewRevestingUpgradeHandler(ctx, ak, bk, sk, evm, vk, db, keys, cdc, historyStateHeight, ts)
		revesting.SetIgnoreList(getIgnoreList())

		if err := revesting.Run(); err != nil {
			panic(err)
		}

		// TODO Remove before release
		panic("test abort")

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
