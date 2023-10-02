package erc20

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/ethereum/go-ethereum/common"

	erc20types "github.com/evmos/evmos/v14/x/erc20/types"
	"github.com/haqq-network/haqq/x/erc20/keeper"
)

// NewErc20ProposalHandler creates a governance handler to manage new proposal types.
func NewErc20ProposalHandler(k *keeper.Keeper) govv1beta1.Handler {
	return func(ctx sdk.Context, content govv1beta1.Content) error {
		// Check if the conversion is globally enabled
		if !k.IsERC20Enabled(ctx) {
			return errorsmod.Wrap(
				erc20types.ErrERC20Disabled, "registration is currently disabled by governance",
			)
		}

		switch c := content.(type) {
		case *erc20types.RegisterCoinProposal:
			return handleRegisterCoinProposal(ctx, k, c)
		case *erc20types.RegisterERC20Proposal:
			return handleRegisterERC20Proposal(ctx, k, c)
		case *erc20types.ToggleTokenConversionProposal:
			return handleToggleConversionProposal(ctx, k, c)

		default:
			return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "unrecognized %s proposal content type: %T", erc20types.ModuleName, c)
		}
	}
}

// handleRegisterCoinProposal handles the registration proposal for multiple
// native Cosmos coins
func handleRegisterCoinProposal(
	ctx sdk.Context,
	k *keeper.Keeper,
	p *erc20types.RegisterCoinProposal,
) error {
	for _, metadata := range p.Metadata {
		pair, err := k.RegisterCoin(ctx, metadata)
		if err != nil {
			return err
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				erc20types.EventTypeRegisterCoin,
				sdk.NewAttribute(erc20types.AttributeKeyCosmosCoin, pair.Denom),
				sdk.NewAttribute(erc20types.AttributeKeyERC20Token, pair.Erc20Address),
			),
		)
	}

	return nil
}

// handleRegisterERC20Proposal handles the registration proposal for multiple
// ERC20 tokens
func handleRegisterERC20Proposal(
	ctx sdk.Context,
	k *keeper.Keeper,
	p *erc20types.RegisterERC20Proposal,
) error {
	for _, address := range p.Erc20Addresses {
		pair, err := k.RegisterERC20(ctx, common.HexToAddress(address))
		if err != nil {
			return err
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				erc20types.EventTypeRegisterERC20,
				sdk.NewAttribute(erc20types.AttributeKeyCosmosCoin, pair.Denom),
				sdk.NewAttribute(erc20types.AttributeKeyERC20Token, pair.Erc20Address),
			),
		)
	}

	return nil
}

// handleToggleConversionProposal handles the toggle proposal for a token pair
func handleToggleConversionProposal(
	ctx sdk.Context,
	k *keeper.Keeper,
	p *erc20types.ToggleTokenConversionProposal,
) error {
	pair, err := k.ToggleConversion(ctx, p.Token)
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			erc20types.EventTypeToggleTokenConversion,
			sdk.NewAttribute(erc20types.AttributeKeyCosmosCoin, pair.Denom),
			sdk.NewAttribute(erc20types.AttributeKeyERC20Token, pair.Erc20Address),
		),
	)

	return nil
}
