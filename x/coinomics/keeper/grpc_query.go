package keeper

import (
	"context"

	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/coinomics/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) RewardCoefficient(
	c context.Context,
	_ *types.QueryRewardCoefficientRequest,
) (*types.QueryRewardCoefficientResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	return &types.QueryRewardCoefficientResponse{RewardCoefficient: params.RewardCoefficient}, nil
}

func (k Keeper) MaxSupply(
	c context.Context,
	_ *types.QueryMaxSupplyRequest,
) (*types.QueryMaxSupplyResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	maxSupply := k.GetMaxSupply(ctx)

	return &types.QueryMaxSupplyResponse{MaxSupply: maxSupply}, nil
}

// Params returns params of the mint module.
func (k Keeper) Params(
	c context.Context,
	_ *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
