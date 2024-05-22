package keeper

import (
	"context"

	"github.com/haqq-network/haqq/x/contractcheck/types"
)

var _ types.MsgServer = Keeper{}

// Mint liquidates specified amount of token locked in vesting into liquid token
func (k Keeper) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	// mint erc721
	return nil, nil
}
