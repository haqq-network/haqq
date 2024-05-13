package types

import (
	"errors"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	vestingTypes "github.com/haqq-network/haqq/x/vesting/types"
)

// SubtractAmountFromPeriods subtracts coin amount from given periods proportionally,
// returns decreased periods and diff of initial periods and decreased periods
func SubtractAmountFromPeriods(
	minuendPeriods sdkvesting.Periods,
	subtrahend sdk.Coin,
) (decreasedPeriods, diffPeriods sdkvesting.Periods, err error) {
	minuendDenom := subtrahend.Denom
	minuendTotalAmount := minuendPeriods.TotalAmount().AmountOf(minuendDenom)
	subtrahendAmount := subtrahend.Amount

	if minuendTotalAmount.LT(subtrahendAmount) {
		return nil, nil, errors.New("insufficient locked up funds")
	}

	decreasedPeriods = make(sdkvesting.Periods, len(minuendPeriods))
	diffPeriods = make(sdkvesting.Periods, 0, len(minuendPeriods))
	copy(decreasedPeriods, minuendPeriods)

	totalSubtracted := math.NewInt(0)
	for i, minuendPeriod := range decreasedPeriods {
		minuendCoinAmount := minuendPeriod.Amount.AmountOf(minuendDenom)
		subtractedAmount := minuendCoinAmount.Mul(subtrahendAmount).Quo(minuendTotalAmount)
		subtrahendCoin := sdk.NewCoin(minuendDenom, subtractedAmount)
		decreasedPeriods[i].Amount = minuendPeriod.Amount.Sub(subtrahendCoin)
		totalSubtracted = totalSubtracted.Add(subtractedAmount)
		diffPeriods = append(diffPeriods, sdkvesting.Period{
			Length: minuendPeriod.Length,
			Amount: sdk.NewCoins(subtrahendCoin),
		})
	}

	// subtract residue from decreased periods tail
	// and add it to diff tail
	if len(decreasedPeriods) > 0 {
		residue := subtrahendAmount.Sub(totalSubtracted)
		for i := len(decreasedPeriods) - 1; i >= 0; i-- {
			periodMinuendDenomAmount := decreasedPeriods[i].Amount.AmountOf(minuendDenom)
			if periodMinuendDenomAmount.LT(residue) {
				decreasedPeriods[i].Amount = decreasedPeriods[i].Amount.Sub(sdk.NewCoin(minuendDenom, periodMinuendDenomAmount))
				diffPeriods[i].Amount = diffPeriods[i].Amount.Add(sdk.NewCoin(minuendDenom, periodMinuendDenomAmount))
				residue = residue.Sub(periodMinuendDenomAmount)
				continue
			}
			decreasedPeriods[i].Amount = decreasedPeriods[i].Amount.Sub(sdk.NewCoin(minuendDenom, residue))
			diffPeriods[i].Amount = diffPeriods[i].Amount.Add(sdk.NewCoin(minuendDenom, residue))
			break
		}
	}

	return decreasedPeriods, diffPeriods, nil
}

// ExtractUpcomingPeriods takes the list of periods with started time and
// returns list of periods which are currently upcoming
func ExtractUpcomingPeriods(startDate, endDate int64, periods sdkvesting.Periods, readTime int64) sdkvesting.Periods {
	pastPeriodsCount := vestingTypes.ReadPastPeriodCount(startDate, endDate, periods, readTime)
	upcomingPeriods := make(sdkvesting.Periods, len(periods)-pastPeriodsCount)
	copy(upcomingPeriods, periods[pastPeriodsCount:])

	return upcomingPeriods
}

// ExtractPastPeriods takes the list of periods with started time and
// returns list of periods which are already in the past
func ExtractPastPeriods(startDate, endDate int64, periods sdkvesting.Periods, readTime int64) sdkvesting.Periods {
	pastPeriodsCount := vestingTypes.ReadPastPeriodCount(startDate, endDate, periods, readTime)
	pastPeriods := make(sdkvesting.Periods, pastPeriodsCount)
	copy(pastPeriods, periods[:pastPeriodsCount])

	return pastPeriods
}

// ReplacePeriodsTail replaces the last N periods in original periods list with replacements period list
// where N is length of replacement list
func ReplacePeriodsTail(periods, replacement sdkvesting.Periods) sdkvesting.Periods {
	replacedPeriods := make(sdkvesting.Periods, 0, len(periods))
	if len(replacement) >= len(periods) {
		replacedPeriods = append(replacedPeriods, replacement...)
		return replacedPeriods
	}
	replacedPeriods = append(replacedPeriods, periods[:len(periods)-len(replacement)]...)
	replacedPeriods = append(replacedPeriods, replacement...)

	return replacedPeriods
}

// CurrentPeriodShift calculates how much time has passed since the beginning of the current period
func CurrentPeriodShift(startTime, currentTime int64, periods sdkvesting.Periods) int64 {
	if startTime >= currentTime {
		return 0
	}
	elapsedTime := startTime
	for _, period := range periods {
		if elapsedTime+period.Length > currentTime {
			return currentTime - elapsedTime
		}
		elapsedTime += period.Length
	}

	return 0
}
