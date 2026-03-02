package v193

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/ethereum/go-ethereum/common"

	evmkeeper "github.com/haqq-network/haqq/x/evm/keeper"
	"github.com/haqq-network/haqq/x/evm/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for Haqq v1.9.3
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ek *evmkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)
		logger := ctx.Logger().With("upgrade", UpgradeName)

		// run the v1.9.3 migrations
		logger.Info("Running module migrations...")
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		// Enable Ethiq precompile after add the corresponding module
		params := ek.GetParams(ctx)
		if !ek.IsAvailableStaticPrecompile(&params, common.HexToAddress(types.EthiqPrecompileAddress)) {
			err = ek.EnableStaticPrecompiles(ctx, common.HexToAddress(types.EthiqPrecompileAddress))
		}

		return vm, err
	}
}
