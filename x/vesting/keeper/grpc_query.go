package keeper

import (
	"context"
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/spf13/cast"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/haqq-network/haqq/utils"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
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

	clawbackAccount, err := k.GetClawbackVestingAccount(ctx, addr)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"account at address '%s' either does not exist or is not a vesting account ", addr.String(),
		)
	}

	locked := clawbackAccount.GetLockedUpCoins(ctx.BlockTime())
	unvested := clawbackAccount.GetVestingCoins(ctx.BlockTime())
	vested := clawbackAccount.GetVestedCoins(ctx.BlockTime())

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
			locked := clawbackAccount.GetLockedUpCoins(ctx.BlockTime())
			unvested := clawbackAccount.GetVestingCoins(ctx.BlockTime())
			vested := clawbackAccount.GetVestedCoins(ctx.BlockTime())

			totalLocked = totalLocked.Add(locked...)
			totalUnvested = totalUnvested.Add(unvested...)
			totalVested = totalVested.Add(vested...)
		}
		return false
	})

	lvmAcc := k.accountKeeper.GetModuleAccount(ctx, liquidvestingtypes.ModuleName)
	if lvmAcc == nil {
		panic(sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", lvmAcc))
	}

	escrowedLiquidBalance := k.bankKeeper.GetBalance(ctx, lvmAcc.GetAddress(), utils.BaseDenom)
	totalLocked = totalLocked.Add(escrowedLiquidBalance)

	return &types.QueryTotalLockedResponse{
		Locked:   totalLocked,
		Unvested: totalUnvested,
		Vested:   totalVested,
	}, nil
}
