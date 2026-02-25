package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/utils"

	"github.com/haqq-network/haqq/x/ethiq/types"
)

// CalculateHaqqCoinsToMint calculates the amount of aHAQQ to be minted in exchange for the given aISLM coins.
func (k Keeper) CalculateHaqqCoinsToMint(ctx sdk.Context, islmAmountToBurn sdkmath.Int) (sdkmath.Int, error) {
	// Short no-op circuit if module is disabled
	if !k.IsModuleEnabled(ctx) {
		return sdkmath.ZeroInt(), types.ErrModuleDisabled
	}

	// Validate islmAmountToBurn is positive and greater than zero
	if islmAmountToBurn.LTE(sdkmath.ZeroInt()) {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrInvalidAmount, "islm_amount must be positive and greater than zero, got %s", islmAmountToBurn.String())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	sumOfAllApplications := types.GetSumOfAllApplications()
	totalBurnedAmount := k.GetTotalBurnedAmount(sdkCtx)
	totalBurnedFromApplicationsAmount := k.GetTotalBurnedFromApplicationsAmount(sdkCtx)

	islmTotalBurnedBefore := totalBurnedAmount.Add(sdk.NewCoin(utils.BaseDenom, sumOfAllApplications)).Sub(totalBurnedFromApplicationsAmount)

	return CalculateHaqqAmount(islmTotalBurnedBefore.Amount, islmAmountToBurn)
}

// CalculateHaqqAmount calculates the amount of aHAQQ to be minted in exchange for the given aISLM coins.
// Final result depends on currentIslmTotalBurned amount.
func CalculateHaqqAmount(currentIslmTotalBurned, restAmountToBeBurned sdkmath.Int) (sdkmath.Int, error) {
	totalHaqqToBeMinted := sdkmath.ZeroInt()

	for _, pl := range types.Prices {
		if restAmountToBeBurned.IsZero() {
			// already burnt everything
			break
		}

		levelMaxAmount := pl.ToAmount()

		// exclude threshold amount
		if currentIslmTotalBurned.GTE(levelMaxAmount.Sub(sdkmath.OneInt())) {
			// go to next level if level is fulfilled
			continue
		}

		levelMinAmount := pl.FromAmount()

		// just in case... check that we are within the range
		if !(currentIslmTotalBurned.GTE(levelMinAmount) && currentIslmTotalBurned.LT(levelMaxAmount)) {
			// should never happen
			return sdkmath.Int{}, errorsmod.Wrap(types.ErrCalculationFailed, "failed to find price level")
		}

		unitPrice := pl.UnitPrice()

		// get the rest amount on this level,
		restAmountForThisLevel := levelMaxAmount.Sub(currentIslmTotalBurned).Sub(sdkmath.OneInt())

		// get amount to burn on this level
		burnOnThisLevel := restAmountForThisLevel
		if burnOnThisLevel.GT(restAmountToBeBurned) {
			burnOnThisLevel = restAmountToBeBurned
		}

		// track burning
		currentIslmTotalBurned = currentIslmTotalBurned.Add(burnOnThisLevel)
		restAmountToBeBurned = restAmountToBeBurned.Sub(burnOnThisLevel)

		// track minting
		haqqToBeMintedOnThisLevel := burnOnThisLevel.Quo(unitPrice).Mul(sdkmath.NewIntFromUint64(1e18))
		totalHaqqToBeMinted = totalHaqqToBeMinted.Add(haqqToBeMintedOnThisLevel)
	}

	return totalHaqqToBeMinted, nil
}
