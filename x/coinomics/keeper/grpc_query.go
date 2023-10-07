package keeper

import (
	"context"

	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/coinomics/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Era(
	c context.Context,
	_ *types.QueryEraRequest,
) (*types.QueryEraResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	era := k.GetEra(ctx)

	return &types.QueryEraResponse{Era: era}, nil
}

func (k Keeper) InflationRate(
	c context.Context,
	_ *types.QueryInflationRateRequest,
) (*types.QueryInflationRateResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	inflation := k.GetInflation(ctx)

	return &types.QueryInflationRateResponse{InflationRate: inflation}, nil
}

func (k Keeper) MaxSupply(
	c context.Context,
	_ *types.QueryMaxSupplyRequest,
) (*types.QueryMaxSupplyResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	maxSupply := k.GetMaxSupply(ctx)

	return &types.QueryMaxSupplyResponse{MaxSupply: maxSupply}, nil
}

func (k Keeper) EraClosingSupply(
	c context.Context,
	_ *types.QueryEraClosingSupplyRequest,
) (*types.QueryEraClosingSupplyResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	eraClosingSupply := k.GetEraClosingSupply(ctx)

	return &types.QueryEraClosingSupplyResponse{EraClosingSupply: eraClosingSupply}, nil
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
