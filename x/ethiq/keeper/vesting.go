package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ethiq/types"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

func (k Keeper) redeemAllLiquidVestingCoins(ctx sdk.Context, fromAddress sdk.AccAddress, isUcdao bool) (sdkmath.Int, error) {
	balances := k.bankKeeper.GetAllBalances(ctx, fromAddress)

	// redeem all aLIQUID balances from liquid vesting module
	redeemedAmount := sdkmath.ZeroInt()
	for _, balance := range balances {
		if !utils.IsLiquidToken(balance.Denom) {
			continue
		}

		// redeem balance from liquid vesting module
		if err := k.liquidVestingKeeper.Redeem(ctx, fromAddress, fromAddress, balance); err != nil {
			return sdkmath.ZeroInt(), errorsmod.Wrap(err, types.ErrRedeemLiquidCoins.Error())
		}

		redeemedAmount = redeemedAmount.Add(balance.Amount)

		if isUcdao {
			// track total balance of UCDAO module
			// intended usage of methods that should be internal..
			k.ucdaoKeeper.TrackAddBalance(ctx, sdk.NewCoin(utils.BaseDenom, balance.Amount))
			k.ucdaoKeeper.TrackSubBalance(ctx, balance)
		}
	}

	return redeemedAmount, nil
}

func (k Keeper) unlockVestingCoins(ctx sdk.Context, fromAddress sdk.AccAddress, amt sdk.Coin) (sdk.Coin, error) {
	zeroCoin := sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt())

	fromAcc := k.accountKeeper.GetAccount(ctx, fromAddress)
	if fromAcc == nil {
		return zeroCoin, errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", fromAddress)
	}

	// check from account is vesting account
	va, isClawback := fromAcc.(*vestingtypes.ClawbackVestingAccount)
	if !isClawback {
		// If it's not a vesting account, there's no error, just return zero coins
		return zeroCoin, nil
	}

	// check there is not vesting periods on the schedule
	if !va.GetVestingCoins(ctx.BlockTime()).IsZero() {
		// we can't burn unvested coins
		return zeroCoin, nil
	}

	// check account has target denom locked in vesting
	hasTargetDenom, lockedBalance := va.GetLockedUpCoins(ctx.BlockTime()).Find(amt.Denom)
	if !hasTargetDenom {
		// if there's no aISLM coins locked, do nothing
		return zeroCoin, nil
	}

	// unlock only coins we have on account
	unlockCoins := amt
	if lockedBalance.IsLT(amt) {
		unlockCoins = lockedBalance
	}

	upcomingPeriods := liquidvestingtypes.ExtractUpcomingPeriods(va.GetStartTime(), va.GetEndTime(), va.LockupPeriods, ctx.BlockTime().Unix())
	decreasedPeriods, _, err := liquidvestingtypes.SubtractAmountFromPeriods(upcomingPeriods, unlockCoins)
	if err != nil {
		return zeroCoin, errorsmod.Wrapf(types.ErrUnlockCoins, "failed to calculate new schedule: %s", err.Error())
	}

	// all vesting periods are completed at this point, so we can reduce amounts without additional extracting logic
	decreasedVestingPeriods, _, err := liquidvestingtypes.SubtractAmountFromPeriods(va.VestingPeriods, unlockCoins)
	if err != nil {
		return zeroCoin, errorsmod.Wrapf(types.ErrUnlockCoins, "failed to calculate new schedule: %s", err.Error())
	}

	va.OriginalVesting = va.OriginalVesting.Sub(unlockCoins)
	va.LockupPeriods = liquidvestingtypes.ReplacePeriodsTail(va.LockupPeriods, decreasedPeriods)
	va.VestingPeriods = liquidvestingtypes.ReplacePeriodsTail(va.VestingPeriods, decreasedVestingPeriods)
	k.accountKeeper.SetAccount(ctx, va)

	return unlockCoins, nil
}
