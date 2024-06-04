package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/x/erc20/types"
)

func (k Keeper) MintNFT(ctx sdk.Context, contractAddress, to, nftURL string) error {
	// mint erc721
	contract := common.HexToAddress(contractAddress)

	_, err := k.CallEVM(ctx,
		contracts.ERC721MinterBurnerStorageContract.ABI,
		types.ModuleAddress,
		contract,
		true,
		"safeMint",
		common.HexToAddress(to),
		nftURL,
	)
	if err != nil {
		return err
	}

	return nil
}
