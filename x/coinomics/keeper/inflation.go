package keeper

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/x/coinomics/types"
)

// NextPhase calculus
func (k Keeper) CountEraForBlock(ctx sdk.Context, params types.Params, currentEra uint64, currentBlock uint64) uint64 {
	if currentEra == 0 {
		return 1
	}

	params = k.GetParams(ctx)

	startedBlock := k.GetEraStartedAtBlock(ctx)
	nextEraBlock := params.BlocksPerEra + startedBlock

	if currentBlock < nextEraBlock {
		return currentEra
	}

	return currentEra + 1
}

func (k Keeper) CalcTargetMintForEra(ctx sdk.Context, eraNumber uint64) sdk.Coin {
	params := k.GetParams(ctx)

	eraCoef := sdk.NewDecWithPrec(95, 2) // 0.95

	if eraNumber == 1 {
		eraPeriod := uint64(2) // 2 years
		currentTotalSupply := k.bankKeeper.GetSupply(ctx, types.DefaultMintDenom)
		maxSupply := k.GetMaxSupply(ctx)

		totalMintNeeded := maxSupply.SubAmount(currentTotalSupply.Amount)

		// -----------  NUM ------------- / ---------- DEN -------------
		// (1-era_coef)*total_mint_needed / (1-era_coef^(100/era_period))
		num := (sdk.OneDec().Sub(eraCoef)).Mul(sdk.NewDecFromInt(totalMintNeeded.Amount))
		den := sdk.OneDec().Sub(eraCoef.Power(100 / eraPeriod))

		target := num.Quo(den)

		return sdk.NewCoin(params.MintDenom, target.RoundInt())
	} else if eraNumber > 1 && eraNumber < 50 {
		prevTargetMint := k.GetEraTargetMint(ctx)
		currTargetMint := sdk.NewDecFromInt(prevTargetMint.Amount).Mul(eraCoef)

		return sdk.NewCoin(types.DefaultMintDenom, currTargetMint.RoundInt())
	} else if eraNumber == 50 {
		currentTotalSupply := k.bankKeeper.GetSupply(ctx, types.DefaultMintDenom)
		maxSupply := k.GetMaxSupply(ctx)

		return maxSupply.SubAmount(currentTotalSupply.Amount)
	} else {
		return sdk.NewCoin(params.MintDenom, sdk.NewInt(0))
	}
}

func (k Keeper) CalcInflation(ctx sdk.Context, era uint64, eraTargetSupply sdk.Coin, eraTargetMint sdk.Coin) sdk.Dec {
	if era > 50 {
		return sdk.NewDec(0)
	}

	return sdk.NewDecFromInt(eraTargetMint.Amount).
		Quo(sdk.NewDecFromInt(eraTargetSupply.SubAmount(eraTargetMint.Amount).Amount)).
		Mul(sdk.NewDec(100))
}

func (k Keeper) MintAndAllocateInflation(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	eraTargetMint := k.GetEraTargetMint(ctx)

	// BlocksPerEra is unsigned and can't be negative, so check only for zero value
	if params.BlocksPerEra == 0 {
		return errors.New("BlocksPerEra is zero")
	}

	totalMintOnBlockInt := eraTargetMint.Amount.Quo(sdk.NewIntFromUint64(params.BlocksPerEra))
	totalMintOnBlockCoin := sdk.NewCoin(params.MintDenom, totalMintOnBlockInt)

	// Mint coins to coinomics module
	if err := k.MintCoins(ctx, totalMintOnBlockCoin); err != nil {
		ctx.Logger().Error("FAILED MintCoins: ", err.Error())
	}

	// Allocate remaining coinomics module balance to destribution
	err := k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.ModuleName,
		k.feeCollectorName,
		sdk.NewCoins(totalMintOnBlockCoin),
	)
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) MintCoins(ctx sdk.Context, coin sdk.Coin) error {
	coins := sdk.NewCoins(coin)

	// skip as no coins need to be minted
	if coins.Empty() {
		return nil
	}

	return k.bankKeeper.MintCoins(ctx, types.ModuleName, coins)
}
