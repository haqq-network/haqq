package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/x/contractcheck/types"
)

var _ types.MsgServer = Keeper{}

// Mint liquidates specified amount of token locked in vesting into liquid token
func (k Keeper) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	// mint erc721
	ctx := sdk.UnwrapSDKContext(goCtx)

	contract := common.HexToAddress(msg.Address)

	_, err := k.CallEVM(ctx,
		contracts.ERC721MinterBurnerStorageContract.ABI,
		types.ModuleAddress,
		contract,
		true,
		"safeMint",
		common.HexToAddress(msg.To),
		msg.Uri,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgMintResponse{}, nil
}
