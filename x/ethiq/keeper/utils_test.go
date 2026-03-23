package keeper_test

import (
	"errors"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/x/vesting/types"
)

func (suite *KeeperTestSuite) setupClawbackVestingAccount(ctx sdk.Context, vestingAcc, funderAcc sdk.AccAddress, balances sdk.Coins, withCliff bool) error { //nolint: unparam
	totalVestingCoins := testutil.TestVestingSchedule.TotalVestingCoins
	if totalVestingCoins.IsAllGT(balances) {
		return errors.New("should provide enough balance for the vesting schedule")
	}
	// fund the vesting account to set the account and then
	// send funds over to the funder account so free balance remains
	err := testutil.FundAccount(ctx, suite.network.App.BankKeeper, vestingAcc, balances)
	if err != nil {
		return err
	}
	err = suite.network.App.BankKeeper.SendCoins(ctx, vestingAcc, funderAcc, totalVestingCoins)
	if err != nil {
		return err
	}

	var vp []sdkvesting.Period
	if withCliff {
		vp = testutil.TestVestingSchedule.VestingPeriods
	}
	// create a clawback vesting account
	msgConv := types.NewMsgConvertIntoVestingAccount(
		funderAcc,
		vestingAcc,
		time.Unix(ctx.BlockTime().Unix()-5, 0),
		testutil.TestVestingSchedule.LockupPeriods,
		vp,
		true, false, nil,
	)
	if _, err = suite.network.App.VestingKeeper.ConvertIntoVestingAccount(ctx, msgConv); err != nil {
		return err
	}

	return nil
}
