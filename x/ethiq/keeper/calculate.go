package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/ethiq/types"
)

// CalculateHaqqCoinsToMint calculates the amount of aHAQQ to be minted in exchange for the given aISLM coins.
func (k Keeper) CalculateHaqqCoinsToMint(ctx sdk.Context, islmTotalBurnedBefore, islmAmountToBurn sdkmath.Int) (sdkmath.Int, sdkmath.LegacyDec, error) {
	// Short no-op circuit if module is disabled
	if !k.IsModuleEnabled(ctx) {
		return sdkmath.Int{}, sdkmath.LegacyDec{}, types.ErrModuleDisabled
	}

	// Validate islmAmountToBurn is positive and greater than zero
	if islmAmountToBurn.LTE(sdkmath.ZeroInt()) {
		return sdkmath.Int{}, sdkmath.LegacyDec{}, errorsmod.Wrapf(types.ErrInvalidAmount, "islm_amount must be positive and greater than zero, got %s", islmAmountToBurn)
	}

	totalHaqqToBeMinted, err := CalculateHaqqAmount(islmTotalBurnedBefore, islmAmountToBurn)
	if err != nil {
		return sdkmath.Int{}, sdkmath.LegacyDec{}, err
	}

	// Calculate average price per unit
	// islmAmountToBurn is guaranteed to be positive due to validation above
	pricePerUnit := sdkmath.LegacyNewDecFromInt(islmAmountToBurn).Quo(sdkmath.LegacyNewDecFromInt(totalHaqqToBeMinted))

	return totalHaqqToBeMinted, pricePerUnit, nil
}

func CalculateHaqqAmount(currentIslmTotalBurned, restAmountToBeBurned sdkmath.Int) (sdkmath.Int, error) {
	totalHaqqToBeMinted := sdkmath.ZeroInt()
	for power, level := range priceLevels {
		if restAmountToBeBurned.IsZero() {
			// already burnt everything
			break
		}

		tAmt, err := level.Amount()
		if err != nil {
			return sdkmath.Int{}, errorsmod.Wrapf(types.ErrCalculationFailed, "failed to parse price level %d: %e", power, err)
		}

		// exclude threshold amount
		if currentIslmTotalBurned.GTE(tAmt.Sub(sdkmath.OneInt())) {
			// go to next level if level is fulfilled
			continue
		}

		// get the rest amount on this level,
		restAmountForThisLevel := tAmt.Sub(currentIslmTotalBurned).Sub(sdkmath.OneInt())

		// get amount to burn on this level
		burnOnThisLevel := restAmountForThisLevel
		if burnOnThisLevel.GT(restAmountToBeBurned) {
			burnOnThisLevel = restAmountToBeBurned
		}

		// track burning
		currentIslmTotalBurned = currentIslmTotalBurned.Add(burnOnThisLevel)
		restAmountToBeBurned = restAmountToBeBurned.Sub(burnOnThisLevel)
		// amount to mint on this level, price is the 2 powered by level key
		//  - 2^0 = 1
		//  - 2^1 = 2
		//  - 2^2 = 4
		//  - etc...
		haqqToBeMintedOnThisLevel := burnOnThisLevel.QuoRaw(int64(2 ^ power))
		// track minting
		totalHaqqToBeMinted = totalHaqqToBeMinted.Add(haqqToBeMintedOnThisLevel)
	}

	return totalHaqqToBeMinted, nil
}
