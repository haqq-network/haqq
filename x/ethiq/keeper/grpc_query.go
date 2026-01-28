package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/haqq-network/haqq/utils"
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
	totalBurnedFromApplications := k.GetTotalBurnedFromApplicationsAmount(sdkCtx)

	return &types.QueryTotalBurnedResponse{
		TotalBurned:                 totalBurned,
		TotalBurnedFromApplications: totalBurnedFromApplications,
	}, nil
}

// Calculate implements the Query/Calculate gRPC method
func (k Keeper) Calculate(ctx context.Context, req *types.QueryCalculateRequest) (*types.QueryCalculateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	islmAmount, ok := sdkmath.NewIntFromString(req.IslmAmount)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid islm amount: %s", req.IslmAmount))
	}

	if islmAmount.LTE(sdkmath.ZeroInt()) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("islm_amount must be positive and greater than zero: %x", req.IslmAmount))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	haqqSupplyBefore := k.bankKeeper.GetSupply(ctx, types.BaseDenom)

	// Get burnt islm amount
	sumOfAllApplications, err := SumOfAllApplications()
	if err != nil {
		return nil, err
	}
	totalBurnedAmount := k.GetTotalBurnedAmount(sdkCtx)
	totalBurnedFromApplicationsAmount := k.GetTotalBurnedFromApplicationsAmount(sdkCtx)
	alreadyBurntIslmAmount := totalBurnedAmount.Add(sdk.NewCoin(utils.BaseDenom, sumOfAllApplications)).Sub(totalBurnedFromApplicationsAmount)

	haqqToBeMinted, pricePerUnit, err := k.CalculateHaqqCoinsToMint(sdkCtx, alreadyBurntIslmAmount.Amount, islmAmount)
	if err != nil {
		return nil, status.Error(codes.Internal, errorsmod.Wrap(err, "failed to calculate required ISLM").Error())
	}

	haqqSupplyAfter := haqqSupplyBefore.Amount.Add(haqqToBeMinted)

	return &types.QueryCalculateResponse{
		EstimatedHaqqAmount: haqqToBeMinted,
		SupplyBefore:        haqqSupplyBefore.Amount,
		SupplyAfter:         haqqSupplyAfter,
		PricePerUnit:        pricePerUnit,
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
