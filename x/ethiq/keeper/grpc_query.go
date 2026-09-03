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

// MaxPageLimit caps how many applications a single GetApplications / GetSendersApplications
// page may return. Requests above it are rejected instead of silently truncated: these
// responses are offset-based and never carry a NextKey, so a truncated page would be
// indistinguishable from the last one and a client would quietly lose the tail of the list.
const MaxPageLimit = 100

// resolvePageRange turns an offset-based PageRequest into the [offset, offset+count) window
// over a list of the given total length. Key-based pagination is not supported by these
// queries, so a request carrying a key is rejected rather than answered with page one.
func resolvePageRange(pagination *query.PageRequest, total uint64) (offset, count uint64, err error) {
	limit := uint64(query.DefaultLimit)

	if pagination != nil {
		if len(pagination.Key) > 0 {
			return 0, 0, status.Error(codes.InvalidArgument, "key-based pagination is not supported; use offset and limit")
		}
		offset = pagination.Offset
		if pagination.Limit != 0 {
			limit = pagination.Limit
		}
	}

	if limit > MaxPageLimit {
		return 0, 0, status.Errorf(codes.InvalidArgument, "limit %d exceeds the maximum page size of %d", limit, MaxPageLimit)
	}

	if offset >= total {
		return offset, 0, nil
	}

	// offset < total, so the remainder cannot underflow and offset+count cannot overflow.
	count = limit
	if remaining := total - offset; remaining < count {
		count = remaining
	}

	return offset, count, nil
}

func (k Keeper) GetApplications(ctx context.Context, req *types.QueryGetApplicationsRequest) (*types.QueryGetApplicationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	total := types.TotalNumberOfApplications()

	offset, count, err := resolvePageRange(req.Pagination, total)
	if err != nil {
		return nil, err
	}

	applications := make([]types.BurnApplication, 0, count)
	paginationResponse := &query.PageResponse{}
	if req.Pagination != nil && req.Pagination.CountTotal {
		paginationResponse.Total = total
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for i := offset; i < offset+count; i++ {
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

	total := types.TotalNumberOfApplicationsBySender(req.SenderAddress)

	offset, count, err := resolvePageRange(req.Pagination, total)
	if err != nil {
		return nil, err
	}

	applications := make([]types.BurnApplication, 0, count)
	paginationResponse := &query.PageResponse{}
	if req.Pagination != nil && req.Pagination.CountTotal {
		paginationResponse.Total = total
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for i := offset; i < offset+count; i++ {
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
