package v162

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func fixEvergreen(ctx sdk.Context, bk bankkeeper.Keeper, dk distrkeeper.Keeper) error {
	logger := ctx.Logger()
	logger.Info("Updating Evergreen balance...")

	feePool := dk.GetFeePool(ctx)

	macc := dk.GetDistributionAccount(ctx)
	balances := bk.GetAllBalances(ctx, macc.GetAddress())
	balancesDec := sdk.NewDecCoinsFromCoins(balances...)

	var outstandingRewards sdk.DecCoins
	dk.IterateValidatorOutstandingRewards(ctx, func(addr sdk.ValAddress, rewards types.ValidatorOutstandingRewards) (stop bool) {
		outstandingRewards = outstandingRewards.Add(rewards.Rewards...)
		return false
	})

	expectedCommunityPool, hasNeg := balancesDec.SafeSub(outstandingRewards)
	if hasNeg {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "expected community pool has negative amount: %s", expectedCommunityPool)
	}

	feePool.CommunityPool = expectedCommunityPool
	dk.SetFeePool(ctx, feePool)

	logger.Info("Evergreen balance successfully updated")
	return nil
}
