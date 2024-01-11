package keeper

import (
	"context"

	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Denom(ctx context.Context, request *types.QueryDenomRequest) (*types.QueryDenomResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (k Keeper) Denoms(ctx context.Context, request *types.QueryDenomsRequest) (*types.QueryDenomsResponse, error) {
	// TODO implement me
	panic("implement me")
}
