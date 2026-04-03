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
	if !islmAmountToBurn.GT(sdkmath.ZeroInt()) {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrInvalidAmount, "islm_amount must be positive and greater than zero, got %s", islmAmountToBurn.String())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	sumOfAllApplications, err := types.GetSumOfAllApplications()
	if err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrap(types.ErrCalculationFailed, err.Error())
	}
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
		levelMinAmount := pl.FromAmount()

		// Price levels use inclusive upper bounds: the valid burn range is [from, to-1].
		// The value at "to" itself is the "from" of the next level, so it is excluded here.
		// This means each level's last valid position is to-1, and the level is considered
		// fully consumed when currentIslmTotalBurned >= to-1.
		if currentIslmTotalBurned.GTE(levelMaxAmount.Sub(sdkmath.OneInt())) {
			// go to next level if level is fulfilled
			continue
		}

		// Verify we are within the current level's inclusive range [from-1, to).
		// The from-1 accounts for the case when we enter this level at position to[N-1]-1
		// from the previous level (since from[N] == to[N-1]).
		if !(currentIslmTotalBurned.GTE(levelMinAmount.Sub(sdkmath.OneInt())) && currentIslmTotalBurned.LT(levelMaxAmount)) {
			// should never happen
			return sdkmath.Int{}, errorsmod.Wrap(types.ErrCalculationFailed, "failed to find price level")
		}

		unitPrice := pl.UnitPrice()

		// Remaining capacity at this level: (to - 1) - currentBurned, because the
		// last valid position is to-1 (the "to" boundary is exclusive).
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
		haqqToBeMintedOnThisLevel := sdkmath.LegacyNewDecFromInt(burnOnThisLevel).Quo(unitPrice).TruncateInt()
		totalHaqqToBeMinted = totalHaqqToBeMinted.Add(haqqToBeMintedOnThisLevel)
	}

	if restAmountToBeBurned.IsPositive() {
		return sdkmath.Int{}, errorsmod.Wrapf(types.ErrExceedsPricingCurve, "remaining unaccounted burn amount: %s", restAmountToBeBurned.String())
	}

	return totalHaqqToBeMinted, nil
}
