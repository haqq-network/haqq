package v175

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	liquidvestingkeeper "github.com/haqq-network/haqq/x/liquidvesting/keeper"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

func TurnOffLiquidVesting(ctx sdk.Context, bk bankkeeper.Keeper, lk liquidvestingkeeper.Keeper) error {
	logger := ctx.Logger()
	logger.Info("Start turning off liquid vesting module")

	// turn on liquid vesting module
	if !lk.IsLiquidVestingEnabled(ctx) {
		lk.SetLiquidVestingEnabled(ctx, true)
		return nil
	}

	bk.IterateAllBalances(ctx, func(addr sdk.AccAddress, balance sdk.Coin) (stop bool) {
		// get all liquid vesting denoms
		if strings.HasPrefix(balance.Denom, "aLIQUID") {
			// Your code logic here
			redeemMsg := liquidvestingtypes.MsgRedeem{
				RedeemFrom: addr.String(),
				RedeemTo:   addr.String(),
				Amount:     balance,
			}

			if _, err := lk.Redeem(ctx, &redeemMsg); err != nil {
				panic(err)
			}
		}

		return true
	})

	// turn off liquid vesting module
	lk.SetLiquidVestingEnabled(ctx, false)

	return nil
}
