package v176

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

func FixLockupPeriods(ctx sdk.Context, ak authkeeper.AccountKeeper) error {
	logger := ctx.Logger()
	logger.Info("Start turning on DAO module")

	ak.IterateAccounts(ctx, func(acc authtypes.AccountI) (stop bool) {
		// Check if acc is vesting account
		vacc, ok := acc.(*vestingtypes.ClawbackVestingAccount)
		if !ok {
			return false
		}

		firstPeriod := vacc.LockupPeriods[0]
		firstPeriodTime := time.Unix(vacc.StartTime.Unix()+firstPeriod.Length, 0)

		if firstPeriodTime.After(time.Unix(LockupLengthThreshold, 0)) && vacc.GetEndTime() > EndTimeForCheck {
			unixBlockTime := ctx.BlockTime().Unix()

			sumOfLockupPeriods := math.NewInt(0)
			elapsedTime := vacc.GetStartTime()
			countPeriods := 0

			newLookupPeriods := make(sdkvestingtypes.Periods, 0, len(vacc.LockupPeriods))

			for _, period := range vacc.LockupPeriods {
				elapsedTime += period.Length

				if elapsedTime >= unixBlockTime {
					countPeriods += 1
					sumOfLockupPeriods = sumOfLockupPeriods.Add(period.Amount[0].Amount)
				}
			}

			sumPerPeriod := sumOfLockupPeriods.QuoRaw(int64(countPeriods))
			diff := sumOfLockupPeriods.Sub(sumPerPeriod.Mul(math.NewIntFromUint64(uint64(countPeriods))))

			for index, period := range vacc.LockupPeriods {
				period.Amount[0].Amount = sumPerPeriod

				if index == len(vacc.LockupPeriods)-1 {
					period.Amount[0].Amount = period.Amount[0].Amount.Add(diff)
				}

				newLookupPeriods = append(newLookupPeriods, period)
			}

			vacc.LockupPeriods = newLookupPeriods
			ak.SetAccount(ctx, vacc)
		}

		return false
	})

	return nil
}
