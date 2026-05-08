package v194

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/ethereum/go-ethereum/common"

	evmkeeper "github.com/haqq-network/haqq/x/evm/keeper"
	"github.com/haqq-network/haqq/x/evm/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for Haqq v1.9.4
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ek *evmkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)
		logger := ctx.Logger().With("upgrade", UpgradeName)

		// run the v1.9.4 migrations
		logger.Info("Running module migrations...")
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		// Enable LiquidVesting precompile
		var addPrecompiles []common.Address
		liquidAddr := common.HexToAddress(types.LiquidPrecompileAddress)
		params := ek.GetParams(ctx)
		if !ek.IsAvailableStaticPrecompile(&params, liquidAddr) {
			addPrecompiles = append(addPrecompiles, liquidAddr)
		}

		if len(addPrecompiles) > 0 {
			err = ek.EnableStaticPrecompiles(ctx, addPrecompiles...)
		}

		return vm, err
	}
}
