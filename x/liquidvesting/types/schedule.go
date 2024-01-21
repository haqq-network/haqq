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
		return nil, nil, errors.New("total amount of minuend periods is less than subtrahend amount")
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

	if len(decreasedPeriods) > 0 {
		residue := subtrahendAmount.Sub(totalSubtracted)
		residueCoin := sdk.NewCoin(minuendDenom, residue)
		decreasedPeriods[len(decreasedPeriods)-1].Amount = decreasedPeriods[len(decreasedPeriods)-1].Amount.Sub(residueCoin)
		diffPeriods[len(diffPeriods)-1].Amount = diffPeriods[len(diffPeriods)-1].Amount.Add(residueCoin)
	}

	return decreasedPeriods, diffPeriods, nil
}

func ExtractUpcomingPeriods(startDate, endDate int64, periods sdkvesting.Periods, readTime int64) sdkvesting.Periods {
	pastPeriods := vestingTypes.ReadPastPeriodCount(startDate, endDate, periods, readTime)
	upcomingPeriods := make(sdkvesting.Periods, len(periods)-pastPeriods)
	copy(upcomingPeriods, periods[pastPeriods:])

	return upcomingPeriods
}

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

func CurrentPeriodShift(startTime, currentTime int64, periods sdkvesting.Periods) int64 {
	if startTime >= currentTime {
		return 0
	}
	elapsedTime := startTime
	for _, period := range periods {
		if elapsedTime+period.Length >= currentTime {
			return currentTime - elapsedTime
		}
		elapsedTime += period.Length
	}

	return 0
}
