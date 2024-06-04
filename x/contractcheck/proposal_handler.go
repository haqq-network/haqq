package contractcheck

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/haqq-network/haqq/x/contractcheck/keeper"
	"github.com/haqq-network/haqq/x/contractcheck/types"
)

// NewErc20ProposalHandler creates a governance handler to manage new proposal types.
func NewContractCheckProposalHandler(k *keeper.Keeper) govv1beta1.Handler {
	return func(ctx sdk.Context, content govv1beta1.Content) error {
		switch c := content.(type) {
		case *types.MintNFTProposal:
			return handleMintNFTProposal(ctx, k, c)

		default:
			return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "unrecognized %s proposal content type: %T", types.ModuleName, c)
		}
	}
}

// handleMintNFTProposal handles the registration proposal for multiple
// native Cosmos coins
func handleMintNFTProposal(
	ctx sdk.Context,
	k *keeper.Keeper,
	p *types.MintNFTProposal,
) error {
	err := k.MintNFT(ctx, p.ContractAddress, p.DestinationAddress, p.NftUrl)
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"mint_nft",
			sdk.NewAttribute("contract_address", p.GetContractAddress()),
			sdk.NewAttribute("receiver_address", p.GetDestinationAddress()),
			sdk.NewAttribute("nft_uri", p.GetNftUrl()),
		),
	)

	return nil
}
