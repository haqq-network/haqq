package v190

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	vestingkeeper "github.com/haqq-network/haqq/x/vesting/keeper"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

func ConvertClawbackVestingAccountsIntoEth(ctx sdk.Context, ak authkeeper.AccountKeeper, vk vestingkeeper.Keeper) error {
	logger := ctx.Logger()

	// Iterate all accounts
	ak.IterateAccounts(ctx, func(acc authtypes.AccountI) (stop bool) {
		// Check if acc is vesting account
		vacc, ok := acc.(*vestingtypes.ClawbackVestingAccount)
		if !ok {
			// proceed with the next account
			return false
		}

		// check if account has any vesting coins left
		if !vacc.GetVestingCoins(ctx.BlockTime()).IsZero() {
			// proceed with the next account
			return false
		}

		// check if account has any locked up coins left
		if vacc.HasLockedCoins(ctx.BlockTime()) {
			// proceed with the next account
			return false
		}

		msg := vestingtypes.NewMsgConvertVestingAccount(vacc.GetAddress())
		_, err := vk.ConvertVestingAccount(ctx, msg)
		if err != nil {
			// should never happen
			logger.Error(fmt.Sprintf("failed to convert vesting account (%s): %e", vacc.GetAddress().String(), err))
			return false
		} else {
			logger.Info(fmt.Sprintf("Converted account: %s", vacc.GetAddress().String()))
		}

		return false
	})

	return nil
}
