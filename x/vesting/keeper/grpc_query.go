package keeper

import (
	"context"
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/spf13/cast"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/haqq-network/haqq/x/vesting/types"
)

var _ types.QueryServer = Keeper{}

// Balances returns the locked, unvested and vested amount of tokens for a
// clawback vesting account
func (k Keeper) Balances(
	goCtx context.Context,
	req *types.QueryBalancesRequest,
) (*types.QueryBalancesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get vesting account
	acc := k.accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return nil, status.Errorf(
			codes.NotFound,
			"account for address '%s'", req.Address,
		)
	}

	// Check if clawback vesting account
	clawbackAccount, isClawback := acc.(*types.ClawbackVestingAccount)
	if !isClawback {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"account at address '%s' is not a vesting account ", req.Address,
		)
	}

	locked := clawbackAccount.GetLockedOnly(ctx.BlockTime())
	unvested := clawbackAccount.GetUnvestedOnly(ctx.BlockTime())
	vested := clawbackAccount.GetVestedOnly(ctx.BlockTime())

	return &types.QueryBalancesResponse{
		Locked:   locked,
		Unvested: unvested,
		Vested:   vested,
	}, nil
}

func (k Keeper) TotalLocked(
	goCtx context.Context,
	req *types.QueryTotalLockedRequest,
) (*types.QueryTotalLockedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	isEnabled, isSet := os.LookupEnv("HAQQ_ENABLE_VESTING_STATS")
	if !isSet {
		isEnabled = "false"
	}
	if !cast.ToBool(isEnabled) {
		return nil, fmt.Errorf("vesting stats is disabled")
	}

	totalLocked := sdk.NewCoins()
	totalUnvested := sdk.NewCoins()
	totalVested := sdk.NewCoins()

	k.accountKeeper.IterateAccounts(ctx, func(acc authtypes.AccountI) bool {
		// Check if clawback vesting account
		clawbackAccount, isClawback := acc.(*types.ClawbackVestingAccount)
		if isClawback {
			locked := clawbackAccount.GetLockedOnly(ctx.BlockTime())
			unvested := clawbackAccount.GetUnvestedOnly(ctx.BlockTime())
			vested := clawbackAccount.GetVestedOnly(ctx.BlockTime())

			totalLocked = totalLocked.Add(locked...)
			totalUnvested = totalUnvested.Add(unvested...)
			totalVested = totalVested.Add(vested...)
		}
		return false
	})

	return &types.QueryTotalLockedResponse{
		Locked:   totalLocked,
		Unvested: totalUnvested,
		Vested:   totalVested,
	}, nil
}
