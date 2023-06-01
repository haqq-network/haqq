package v1_4_0

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/haqq-network/haqq/types"
	coinomicskeeper "github.com/haqq-network/haqq/x/coinomics/keeper"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v1.4.0
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	sk stakingkeeper.Keeper,
	ck coinomicskeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger()
		logger.Info("run migration v1.4.0")

		// Reset coinomics state for TestEdge2
		if types.IsTestEdge2Network(ctx.ChainID()) {
			if err := ResetCoinomicsState(ctx, sk, ck); err != nil {
				logger.Error("FAILED: reset coinomics params error: ", err.Error())
			}
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}

func ResetCoinomicsState(ctx sdk.Context, sk stakingkeeper.Keeper, ck coinomicskeeper.Keeper) error {
	newCoinomicsParams := coinomicstypes.DefaultParams()
	newCoinomicsParams.MintDenom = sk.BondDenom(ctx)
	maxSupply := ck.GetMaxSupply(ctx)
	doubledMaxSupply := maxSupply.Amount.Mul(sdk.NewInt(2))

	ck.SetParams(ctx, newCoinomicsParams)
	ck.SetEra(ctx, 0)
	ck.SetEraStartedAtBlock(ctx, uint64(ctx.BlockHeight()))
	ck.SetMaxSupply(ctx, sdk.NewCoin(newCoinomicsParams.MintDenom, doubledMaxSupply))

	return nil
}
