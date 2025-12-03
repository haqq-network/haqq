package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/x/ethiq/types"
)

var _ types.MsgServer = &msgServer{}

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the ethiq MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (k msgServer) MintEthiq(goCtx context.Context, msg *types.MsgMintEthiq) (*types.MsgMintEthiqResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Addresses are already validated above
	toAddress := sdk.MustAccAddressFromBech32(msg.ToAddress)
	fromAddress := sdk.MustAccAddressFromBech32(msg.FromAddress)

	// Call keeper Mint function
	if _, err := k.Keeper.ConvertToEthiq(ctx, msg.EthiqAmount, msg.MaxIslmAmount, fromAddress, toAddress); err != nil {
		return nil, err
	}

	return &types.MsgMintEthiqResponse{}, nil
}
