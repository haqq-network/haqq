package v150

import (
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"

	vestingkeeper "github.com/haqq-network/haqq/x/vesting/keeper"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.4.0
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	sk stakingkeeper.Keeper,
	evm *evmkeeper.Keeper,
	vk vestingkeeper.Keeper,
	cdc codec.Codec,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Info("##############################################")
		logger.Info("############ Run migration v1.5.0 ############")
		logger.Info("################## REVESTING #################")
		logger.Info("##############################################")

		ts := math.NewIntFromUint64(exp)
		ts = ts.MulRaw(threshold)
		logger.Info("## Balance threshold: " + ts.String() + " aISLM")

		revesting := NewRevestingUpgradeHandler(ctx, ak, bk, sk, evm, vk, ts, cdc)
		revesting.SetIgnoreList(getIgnoreList())
		if err := revesting.SetValidatorsList(getWhitelistedValidators()); err != nil {
			panic("failed to prepare validators list for upgrade" + err.Error())
		}

		if err := revesting.Run(); err != nil {
			panic(err)
		}

		logger.Info("##############################################")
		logger.Info("############# REVESTING COMPLETE #############")
		logger.Info("##############################################")
		// TODO Remove before release
		// panic("test abort")

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
