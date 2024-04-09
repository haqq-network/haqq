package v174

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	liquidvestingkeeper "github.com/haqq-network/haqq/x/liquidvesting/keeper"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
	"time"
)

// 1767225600 is timestamp for 2026-01-01
var lockupThreshold = time.Unix(1767225600, 0)

const threeYearsAsDays = 1095

func StretchLockupScheduleForAccounts(ctx sdk.Context, ak authkeeper.AccountKeeper) error {
	// Iterate all accounts
	ak.IterateAccounts(ctx, func(acc authtypes.AccountI) (stop bool) {
		// Check if acc is vesting account
		vacc, ok := acc.(*vestingtypes.ClawbackVestingAccount)
		if !ok {
			return false
		}

		// if end time for unlock account is after 2026-01-01 modify schedule
		if time.Unix(vacc.GetEndTime(), 0).After(lockupThreshold) {
			upcomingPeriods := liquidvestingtypes.ExtractUpcomingPeriods(vacc.GetStartTime(), vacc.GetEndTime(), vacc.LockupPeriods, ctx.BlockTime().Unix())
			stretchedUpcomingPeriods := stretchPeriods(upcomingPeriods, threeYearsAsDays)
			pastPeriods := liquidvestingtypes.ExtractPastPeriods(vacc.GetStartTime(), vacc.GetEndTime(), vacc.LockupPeriods, ctx.BlockTime().Unix())

			// add 1095 days (three years to the end time)
			vacc.EndTime = vacc.EndTime + 86_400*1095
			// set stretched lockup periods
			vacc.LockupPeriods = append(pastPeriods, stretchedUpcomingPeriods...)
			ak.SetAccount(ctx, vacc)
		}

		return false
	})

	return nil
}

func StretchLockupScheduleForLiquidVestingTokens(ctx sdk.Context, lk liquidvestingkeeper.Keeper) error {
	// Iterate all denoms
	lk.IterateDenoms(ctx, func(denom liquidvestingtypes.Denom) (stop bool) {
		// if end time for liquid denom is after 2026-01-01 modify schedule
		if denom.EndTime.After(lockupThreshold) {
			upcomingPeriods := liquidvestingtypes.ExtractUpcomingPeriods(denom.StartTime.Unix(), denom.EndTime.Unix(), denom.LockupPeriods, ctx.BlockTime().Unix())
			stretchedUpcomingPeriods := stretchPeriods(upcomingPeriods, threeYearsAsDays)
			pastPeriods := liquidvestingtypes.ExtractPastPeriods(denom.StartTime.Unix(), denom.EndTime.Unix(), denom.LockupPeriods, ctx.BlockTime().Unix())

			// add 1095 days (three years to the end time)
			denom.EndTime = time.Unix(denom.EndTime.Unix()+86_400*1095, 0)
			// set stretched lockup periods
			denom.LockupPeriods = append(pastPeriods, stretchedUpcomingPeriods...)
			lk.SetDenom(ctx, denom)
		}

		return false
	})

	return nil
}

func stretchPeriods(periods sdkvesting.Periods, stretchDays int64) sdkvesting.Periods {
	const Denom = "aISLM"

	periodsLengthInDays := periods.TotalLength() / 86_400
	stretchedPerDayLockupAmount := periods.TotalAmount().AmountOf(Denom).Quo(sdkmath.NewInt(periodsLengthInDays + stretchDays))
	totalAmount := periods.TotalAmount().AmountOf(Denom)
	extraLengthAmount := stretchedPerDayLockupAmount.Mul(sdkmath.NewInt(stretchDays))

	// update amount of existing periods
	updatedPeriods := make(sdkvesting.Periods, len(periods))
	copy(periods, updatedPeriods)
	for _, period := range updatedPeriods {
		newAmount := period.Amount.AmountOf(Denom).Mul(totalAmount.Sub(extraLengthAmount)).Quo(totalAmount)
		for i, coin := range period.Amount {
			if coin.Denom == Denom {
				period.Amount[i] = sdk.NewCoin(Denom, newAmount)
			}
		}
	}

	// allocate extra periods
	extraPeriods := make(sdkvesting.Periods, stretchDays)
	for i := range extraPeriods {
		extraPeriods[i] = sdkvesting.Period{
			Length: 86_400,
			Amount: sdk.NewCoins(sdk.NewCoin(Denom, stretchedPerDayLockupAmount)),
		}
	}

	// calculate total remainder and add it to the last period of the stretched periods
	updatedPeriodsAmount := updatedPeriods.TotalAmount().AmountOf(Denom)
	extraPeriodsAmount := extraPeriods.TotalAmount().AmountOf(Denom)
	calculationDiff := totalAmount.Sub(updatedPeriodsAmount.Add(extraPeriodsAmount))
	extraPeriods[stretchDays-1].Amount.Add(sdk.NewCoin(Denom, calculationDiff))

	return append(updatedPeriods, extraPeriods...)
}
