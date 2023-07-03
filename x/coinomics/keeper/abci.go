package keeper

import (
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/coinomics/types"
)

func (k Keeper) EndBlocker(ctx sdk.Context) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	params := k.GetParams(ctx)

	// NOTE: ignore end of block if coinomics is disabled
	if !params.EnableCoinomics {
		return
	}

	currentBlock := uint64(ctx.BlockHeight())
	currentEra := k.GetEra(ctx)

	// Short-circuit as all inflation calculations will be 0 after 50 eras (100 years) have passed
	if currentEra > 50 {
		return
	}

	eraForBlock := k.CountEraForBlock(ctx, params, currentEra, currentBlock)

	if currentEra != eraForBlock {
		k.SetEra(ctx, eraForBlock)
		k.SetEraStartedAtBlock(ctx, currentBlock)

		nextEraTargetMint := k.CalcTargetMintForEra(ctx, eraForBlock)

		currentTotalSupply := k.bankKeeper.GetSupply(ctx, types.DefaultMintDenom)
		nextEraClosingSupply := currentTotalSupply.AddAmount(nextEraTargetMint.Amount)
		nextEraInflation := k.CalcInflation(ctx, eraForBlock, nextEraClosingSupply, nextEraTargetMint)

		k.SetEraTargetMint(ctx, nextEraTargetMint)
		k.SetEraClosingSupply(ctx, nextEraClosingSupply)
		k.SetInflation(ctx, nextEraInflation)
	}

	if err := k.MintAndAllocateInflation(ctx); err != nil {
		ctx.Logger().Error("Failed MintAndAllocateInflation: ", err.Error())
	}
}
