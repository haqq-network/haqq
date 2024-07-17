package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/dao/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the distribution MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (k msgServer) Fund(goCtx context.Context, msg *types.MsgFund) (*types.MsgFundResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	depositer, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		return nil, err
	}
	if err := k.Keeper.Fund(ctx, msg.Amount, depositer); err != nil {
		return nil, err
	}

	return &types.MsgFundResponse{}, nil
}

func (k msgServer) TransferOwnership(goCtx context.Context, msg *types.MsgTransferOwnership) (*types.MsgTransferOwnershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	owner, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil, err
	}

	newOwner, err := sdk.AccAddressFromBech32(msg.NewOwner)
	if err != nil {
		return nil, err
	}

	if err := k.Keeper.TransferOwnership(ctx, owner, newOwner); err != nil {
		return nil, err
	}

	return &types.MsgTransferOwnershipResponse{}, nil
}
