package contractcheck

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/haqq-network/haqq/x/shariahoracle/keeper"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
)

// NewShariahOracleProposalHandler creates a governance handler for shariah oracle module
func NewShariahOracleProposalHandler(k *keeper.Keeper) govv1beta1.Handler {
	return func(ctx sdk.Context, content govv1beta1.Content) error {
		switch c := content.(type) {
		case *types.MintCACProposal:
			return handleMintCACProposal(ctx, k, c)
		case *types.BurnCACProposal:
			return handleBurnCACProposal(ctx, k, c)
		case *types.UpdateCACContractProposal:
			return handleUpdateCACContractProposal(ctx, k, c)
		default:
			return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "unrecognized %s proposal content type: %T", types.ModuleName, c)
		}
	}
}

// handleMintCACProposal
func handleMintCACProposal(
	ctx sdk.Context,
	k *keeper.Keeper,
	p *types.MintCACProposal,
) error {
	for _, grantee := range p.Grantees {
		cacMinted, err := k.DoesAddressHaveCAC(ctx, grantee)
		if err != nil {
			return err
		}

		if cacMinted {
			return errorsmod.Wrapf(types.ErrCACAlreadyMinted, "CAC already minted for address %s", grantee)
		}

		err = k.MintCAC(ctx, grantee)
		if err != nil {
			return err
		}
	}

	return nil
}

// handleBurnCACProposal
func handleBurnCACProposal(
	ctx sdk.Context,
	k *keeper.Keeper,
	p *types.BurnCACProposal,
) error {
	for _, grantee := range p.Grantees {
		cacMinted, err := k.DoesAddressHaveCAC(ctx, grantee)
		if err != nil {
			return err
		}

		if !cacMinted {
			return errorsmod.Wrapf(types.ErrCACNotMinted, "CAC not minted for address %s", grantee)
		}

		err = k.BurnCAC(ctx, grantee)
		if err != nil {
			return err
		}
	}
	return nil
}

// handleUpdateCACContractProposal
func handleUpdateCACContractProposal(
	ctx sdk.Context,
	k *keeper.Keeper,
	p *types.UpdateCACContractProposal,
) error {
	return k.UpdateCACContract(ctx, p.NewImplementationAddress)
}
