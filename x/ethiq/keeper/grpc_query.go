package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
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

	if !islmAmount.GT(sdkmath.ZeroInt()) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("islm_amount must be positive and greater than zero: %s", req.IslmAmount))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	haqqToBeMinted, err := k.CalculateHaqqCoinsToMint(sdkCtx, islmAmount)
	if err != nil {
		return nil, status.Error(codes.Internal, errorsmod.Wrap(err, "failed to calculate aHAQQ amount").Error())
	}

	// Calculate average price per unit
	averagePrice := sdkmath.LegacyZeroDec()
	if !haqqToBeMinted.IsZero() {
		// both islmAmount and haqqToBeMinted are not zero
		averagePrice = sdkmath.LegacyNewDecFromInt(islmAmount).Quo(sdkmath.LegacyNewDecFromInt(haqqToBeMinted))
	}

	haqqSupplyBefore := k.bankKeeper.GetSupply(ctx, types.BaseDenom)
	haqqSupplyAfter := haqqSupplyBefore.Amount.Add(haqqToBeMinted)

	return &types.QueryCalculateResponse{
		EstimatedHaqqAmount: haqqToBeMinted,
		SupplyBefore:        haqqSupplyBefore.Amount,
		SupplyAfter:         haqqSupplyAfter,
		AveragePrice:        averagePrice,
	}, nil
}

func (k Keeper) CalculateForApplication(ctx context.Context, req *types.QueryCalculateForApplicationRequest) (*types.QueryCalculateForApplicationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	burnApplication, err := types.GetApplicationByID(req.ApplicationId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	haqqToBeMinted, err := CalculateHaqqAmount(burnApplication.BurnedBeforeAmount.Amount, burnApplication.BurnAmount.Amount)
	if err != nil {
		return nil, status.Error(codes.Internal, errorsmod.Wrap(err, "failed to calculate aHAQQ amount").Error())
	}

	// Calculate average price per unit
	averagePrice := sdkmath.LegacyZeroDec()
	if !haqqToBeMinted.IsZero() {
		// both islmAmount and haqqToBeMinted are not zero
		averagePrice = sdkmath.LegacyNewDecFromInt(burnApplication.BurnAmount.Amount).Quo(sdkmath.LegacyNewDecFromInt(haqqToBeMinted))
	}

	haqqSupplyBefore := k.bankKeeper.GetSupply(ctx, types.BaseDenom)
	haqqSupplyAfter := haqqSupplyBefore.Amount.Add(haqqToBeMinted)

	return &types.QueryCalculateForApplicationResponse{
		EstimatedHaqqAmount: haqqToBeMinted,
		SupplyBefore:        haqqSupplyBefore.Amount,
		SupplyAfter:         haqqSupplyAfter,
		AveragePrice:        averagePrice,
		ToAddress:           burnApplication.ToAddress,
	}, nil
}

func (k Keeper) GetApplications(ctx context.Context, req *types.QueryGetApplicationsRequest) (*types.QueryGetApplicationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	var offset, limit uint64
	offset = 0
	limit = query.DefaultLimit
	countTotal := false
	total := types.TotalNumberOfApplications()

	if req.Pagination != nil {
		offset = req.Pagination.Offset
		limit = req.Pagination.Limit
		countTotal = req.Pagination.CountTotal
	}
	if limit == 0 {
		limit = uint64(query.DefaultLimit)
	}

	lastOnThisPage := offset + limit - 1
	if lastOnThisPage >= total {
		lastOnThisPage = total - 1
	}

	applications := make([]types.BurnApplication, 0, limit)
	paginationResponse := &query.PageResponse{}
	if countTotal {
		paginationResponse.Total = total
	}

	if offset >= total {
		return &types.QueryGetApplicationsResponse{
			Applications: applications,
			Pagination:   paginationResponse,
		}, nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for i := offset; i < total && i <= lastOnThisPage; i++ {
		burnApplication, err := types.GetApplicationByID(i)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		burnApplication.IsExecuted = k.IsApplicationExecuted(sdkCtx, burnApplication.Id)
		applications = append(applications, *burnApplication)
	}

	return &types.QueryGetApplicationsResponse{
		Applications: applications,
		Pagination:   paginationResponse,
	}, nil
}

func (k Keeper) GetSendersApplications(ctx context.Context, req *types.QueryGetSendersApplicationsRequest) (*types.QueryGetSendersApplicationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	var offset, limit uint64
	offset = 0
	limit = query.DefaultLimit
	countTotal := false
	total := types.TotalNumberOfApplicationsBySender(req.SenderAddress)

	if req.Pagination != nil {
		offset = req.Pagination.Offset
		limit = req.Pagination.Limit
		countTotal = req.Pagination.CountTotal
	}
	if limit == 0 {
		limit = uint64(query.DefaultLimit)
	}

	lastOnThisPage := offset + limit - 1
	if lastOnThisPage >= total {
		lastOnThisPage = total - 1
	}

	applications := make([]types.BurnApplication, 0, limit)
	paginationResponse := &query.PageResponse{}
	if countTotal {
		paginationResponse.Total = total
	}

	if offset >= total {
		return &types.QueryGetSendersApplicationsResponse{
			Applications: applications,
			Pagination:   paginationResponse,
		}, nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for i := offset; i < total && i <= lastOnThisPage; i++ {
		burnApplication, err := types.GetSendersApplicationIDByIndex(req.SenderAddress, i)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		burnApplication.IsExecuted = k.IsApplicationExecuted(sdkCtx, burnApplication.Id)
		applications = append(applications, *burnApplication)
	}

	return &types.QueryGetSendersApplicationsResponse{
		Applications: applications,
		Pagination:   paginationResponse,
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
