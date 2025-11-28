package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/haqq-network/haqq/x/ethiq/types"
)

var _ types.QueryServer = Keeper{}

// TotalBurned implements the Query/TotalBurned gRPC method
func (k Keeper) TotalBurned(ctx context.Context, req *types.QueryTotalBurnedRequest) (*types.QueryTotalBurnedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	totalBurned := k.GetTotalBurnedAmount(sdkCtx)

	return &types.QueryTotalBurnedResponse{
		TotalBurned: totalBurned,
	}, nil
}

// Calculate implements the Query/Calculate gRPC method
func (k Keeper) Calculate(ctx context.Context, req *types.QueryCalculateRequest) (*types.QueryCalculateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ethiqAmount, ok := sdkmath.NewIntFromString(req.EthiqAmount)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid ethiq amount: %s", req.EthiqAmount))
	}

	if ethiqAmount.LTE(sdkmath.ZeroInt()) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("ethiq_amount must be positive: %x", req.EthiqAmount))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	supplyBefore := k.GetEthiqSupply(sdkCtx)
	supplyAfter := supplyBefore.Add(ethiqAmount)

	requiredISLM, pricePerUnit, err := k.CalculateRequiredISLM(sdkCtx, ethiqAmount)
	if err != nil {
		return nil, status.Error(codes.Internal, errorsmod.Wrap(err, "failed to calculate required ISLM").Error())
	}

	return &types.QueryCalculateResponse{
		RequiredIslm: requiredISLM,
		SupplyBefore: supplyBefore,
		SupplyAfter:  supplyAfter,
		PricePerUnit: pricePerUnit,
	}, nil
}

// Params implements the Query/Params gRPC method
func (k Keeper) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params := k.GetParams(sdkCtx)

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}
