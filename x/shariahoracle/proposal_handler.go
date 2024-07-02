package shariahoracle

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
		case *types.GrantCACProposal:
			return handleGrantCACProposal(ctx, k, c)
		case *types.RevokeCACProposal:
			return handleRevokeCACProposal(ctx, k, c)
		case *types.UpdateCACContractProposal:
			return handleUpdateCACContractProposal(ctx, k, c)
		default:
			return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "unrecognized %s proposal content type: %T", types.ModuleName, c)
		}
	}
}

// handleGrantCACProposal
func handleGrantCACProposal(
	ctx sdk.Context,
	k *keeper.Keeper,
	p *types.GrantCACProposal,
) error {
	for _, grantee := range p.Grantees {
		cacMinted, err := k.DoesAddressHaveCAC(ctx, grantee)
		if err != nil {
			return err
		}

		if cacMinted {
			return errorsmod.Wrapf(types.ErrCACAlreadyGranted, "CAC already minted for address %s", grantee)
		}

		err = k.GrantCAC(ctx, grantee)
		if err != nil {
			return err
		}
	}

	return nil
}

// handleRevokeCACProposal
func handleRevokeCACProposal(
	ctx sdk.Context,
	k *keeper.Keeper,
	p *types.RevokeCACProposal,
) error {
	for _, grantee := range p.Grantees {
		cacMinted, err := k.DoesAddressHaveCAC(ctx, grantee)
		if err != nil {
			return err
		}

		if !cacMinted {
			return errorsmod.Wrapf(types.ErrCACNotGranted, "CAC not minted for address %s", grantee)
		}

		err = k.RevokeCAC(ctx, grantee)
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
