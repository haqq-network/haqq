package keeper_test

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	"github.com/haqq-network/haqq/x/vesting/types"
)

// setupClawbackVestingAccount sets up a clawback vesting account
// using the TestVestingSchedule. If exceeded balance is provided,
// will fund the vesting account with it.
func setupClawbackVestingAccount(ctx sdk.Context, nw *network.UnitTestNetwork, vestingAcc, funderAcc sdk.AccAddress, balances sdk.Coins) error {
	totalVestingCoins := testutil.TestVestingSchedule.TotalVestingCoins
	if totalVestingCoins.IsAllGT(balances) {
		return errors.New("should provide enough balance for the vesting schedule")
	}
	// fund the vesting account to set the account and then
	// send funds over to the funder account so free balance remains
	err := testutil.FundAccount(ctx, nw.App.BankKeeper, vestingAcc, balances)
	if err != nil {
		return err
	}
	err = nw.App.BankKeeper.SendCoins(ctx, vestingAcc, funderAcc, totalVestingCoins)
	if err != nil {
		return err
	}

	// create a clawback vesting account
	msgConv := types.NewMsgConvertIntoVestingAccount(
		funderAcc,
		vestingAcc,
		ctx.BlockTime(),
		testutil.TestVestingSchedule.LockupPeriods,
		testutil.TestVestingSchedule.VestingPeriods,
		true, false, nil,
	)
	if _, err = nw.App.VestingKeeper.ConvertIntoVestingAccount(ctx, msgConv); err != nil {
		return err
	}

	return nil
}
