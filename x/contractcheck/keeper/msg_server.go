package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/haqq-network/haqq/x/contractcheck/types"
)

var _ types.MsgServer = Keeper{}

// Mint liquidates specified amount of token locked in vesting into liquid token
func (k Keeper) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	// mint erc721
	ctx := sdk.UnwrapSDKContext(goCtx)
	err := k.MintNFT(ctx, msg.Address, msg.To, msg.Uri)
	if err != nil {
		return nil, err
	}

	return &types.MsgMintResponse{}, nil
}
