package keeper

import (
	"context"

	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

var _ types.MsgServer = Keeper{}

func (k Keeper) Liquidate(ctx context.Context, liquidate *types.MsgLiquidate) (*types.MsgLiquidateResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (k Keeper) Redeem(ctx context.Context, redeem *types.MsgRedeem) (*types.MsgRedeemResponse, error) {
	// TODO implement me
	panic("implement me")
}
