package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

// Denom retrieves liquid token denom by its name
func (k Keeper) Denom(goCtx context.Context, req *types.QueryDenomRequest) (*types.QueryDenomResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	metadata, found := k.bankKeeper.GetDenomMetaData(ctx, req.Denom)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	unitIdentifier := req.Denom
	for _, unit := range metadata.DenomUnits {
		if unit.GetExponent() == 18 {
			unitIdentifier = unit.GetDenom()
		}
	}

	val, found := k.GetDenom(ctx, unitIdentifier)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryDenomResponse{Denom: val}, nil
}

// Denoms retrieves liquid tokens denoms
func (k Keeper) Denoms(goCtx context.Context, req *types.QueryDenomsRequest) (*types.QueryDenomsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var chains []types.Denom
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	chainStore := prefix.NewStore(store, types.KeyPrefix(types.DenomKeyPrefix))

	pageRes, err := query.Paginate(chainStore, req.Pagination, func(key []byte, value []byte) error {
		var chain types.Denom
		if err := k.cdc.Unmarshal(value, &chain); err != nil {
			return err
		}

		chains = append(chains, chain)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDenomsResponse{Denoms: chains, Pagination: pageRes}, nil
}
