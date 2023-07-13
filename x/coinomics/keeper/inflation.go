package keeper

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/x/coinomics/types"
	"github.com/pkg/errors"
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

	// eraCoef can't be declared as constant because it's a sdk.Dec with a dynamic constructor
	eraCoef := sdk.NewDecWithPrec(95, 2) // 0.95

	switch {
	case eraNumber == 1:
		currentTotalSupply := k.bankKeeper.GetSupply(ctx, types.DefaultMintDenom)
		maxSupply := k.GetMaxSupply(ctx)

		totalMintNeeded := maxSupply.SubAmount(currentTotalSupply.Amount)
		// -----------  NUM ------------- / ---------- DEN -------------
		// era_period = 2 years
		// (1-era_coef)*total_mint_needed / (1-era_coef^(100/era_period))
		num := sdk.NewDecWithPrec(5, 2).Mul(sdk.NewDecFromInt(totalMintNeeded.Amount))
		den := sdk.OneDec().Sub(eraCoef.Power(50))
		target := num.Quo(den)

		return sdk.NewCoin(params.MintDenom, target.RoundInt())
	case eraNumber > 1 && eraNumber < 50:
		prevTargetMint := k.GetEraTargetMint(ctx)
		currTargetMint := sdk.NewDecFromInt(prevTargetMint.Amount).Mul(eraCoef)

		return sdk.NewCoin(types.DefaultMintDenom, currTargetMint.RoundInt())
	case eraNumber == 50:
		currentTotalSupply := k.bankKeeper.GetSupply(ctx, types.DefaultMintDenom)
		maxSupply := k.GetMaxSupply(ctx)

		return maxSupply.SubAmount(currentTotalSupply.Amount)
	default:
		return sdk.NewCoin(params.MintDenom, sdk.NewInt(0))
	}
}

func (k Keeper) CalcInflation(_ sdk.Context, era uint64, eraTargetSupply sdk.Coin, eraTargetMint sdk.Coin) sdk.Dec {
	if era > 50 {
		return sdk.NewDec(0)
	}

	if eraTargetSupply.IsZero() {
		return sdk.NewDec(0)
	}

	quoAmount := sdk.NewDecFromInt(eraTargetSupply.SubAmount(eraTargetMint.Amount).Amount)
	if quoAmount.IsZero() {
		return sdk.NewDec(0)
	}

	return sdk.NewDecFromInt(eraTargetMint.Amount).Quo(quoAmount).Mul(sdk.NewDec(100))
}

func (k Keeper) MintAndAllocateInflation(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	eraTargetMint := k.GetEraTargetMint(ctx)

	// BlocksPerEra is unsigned and can't be negative, so check only for zero value
	if params.BlocksPerEra == 0 {
		return errors.New("BlocksPerEra is zero")
	}

	// Check if BlocksPerEra is within the uint64 range
	if params.BlocksPerEra > math.MaxUint64 {
		return errors.New("BlocksPerEra is out of uint64 range")
	}

	totalMintOnBlockInt := eraTargetMint.Amount.Quo(sdk.NewIntFromUint64(params.BlocksPerEra))
	totalMintOnBlockCoin := sdk.NewCoin(params.MintDenom, totalMintOnBlockInt)

	// Mint coins to coinomics module
	if err := k.MintCoins(ctx, totalMintOnBlockCoin); err != nil {
		return errors.Wrap(err, "failed mint coins")
	}

	// Allocate remaining coinomics module balance to destribution
	err := k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.ModuleName,
		k.feeCollectorName,
		sdk.NewCoins(totalMintOnBlockCoin),
	)
	if err != nil {
		return errors.Wrap(err, "failed send coins from coinomics to distribution")
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
