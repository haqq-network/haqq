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

		// Enable Ethiq and UCDAO precompiles
		var addPrecompiles []common.Address
		ethiqAddr := common.HexToAddress(types.EthiqPrecompileAddress)
		ucdaoAddr := common.HexToAddress(types.UcdaoPrecompileAddress)
		liquidAddr := common.HexToAddress(types.LiquidPrecompileAddress)
		params := ek.GetParams(ctx)
		if !ek.IsAvailableStaticPrecompile(&params, ethiqAddr) {
			addPrecompiles = append(addPrecompiles, ethiqAddr)
		}
		if !ek.IsAvailableStaticPrecompile(&params, ucdaoAddr) {
			addPrecompiles = append(addPrecompiles, ucdaoAddr)
		}
		if !ek.IsAvailableStaticPrecompile(&params, liquidAddr) {
			addPrecompiles = append(addPrecompiles, liquidAddr)
		}

		if len(addPrecompiles) > 0 {
			err = ek.EnableStaticPrecompiles(ctx, addPrecompiles...)
		}

		return vm, err
	}
}
