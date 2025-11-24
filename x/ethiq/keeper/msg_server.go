package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/ethiq/types"
)

var _ types.MsgServer = msgServer{}

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

	params := k.GetParams(ctx)

	// Short no-op circuit if module is disabled
	if !params.Enabled {
		return &types.MsgMintEthiqResponse{}, nil
	}

	// Validate toAddress
	toAddress, err := sdk.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid to_address: %v", err)
	}

	if toAddress.Empty() {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, "to_address cannot be empty")
	}

	// Validate fromAddress
	fromAddress, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid from_address: %v", err)
	}

	ethiqAmount := msg.EthiqAmount
	maxIslmAmount := msg.MaxIslmAmount

	// Return error if ethiqAmount is outbound the limits
	if ethiqAmount.LT(params.MinMintPerTx) {
		return nil, errorsmod.Wrapf(types.ErrInvalidAmount, "ethiq_amount %s is less than min_mint_per_tx %s", ethiqAmount, params.MinMintPerTx)
	}

	if ethiqAmount.GT(params.MaxMintPerTx) {
		return nil, errorsmod.Wrapf(types.ErrInvalidAmount, "ethiq_amount %s is greater than max_mint_per_tx %s", ethiqAmount, params.MaxMintPerTx)
	}

	// Return error if ethiqAmount less than 1
	if ethiqAmount.LT(sdkmath.OneInt()) {
		return nil, errorsmod.Wrap(types.ErrInvalidAmount, "ethiq_amount must be at least 1")
	}

	// Calculate required ISLM amount
	requiredISLM, _, err := k.CalculateRequiredISLM(ctx, ethiqAmount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to calculate required ISLM")
	}

	// Return error if RequiredISLM less than 1
	if requiredISLM.LT(sdkmath.OneInt()) {
		return nil, errorsmod.Wrap(types.ErrInvalidAmount, "calculated required ISLM is less than 1")
	}

	// Return error if RequiredISLM greater than maxIslmAmount
	if requiredISLM.GT(maxIslmAmount) {
		return nil, errorsmod.Wrapf(types.ErrInsufficientFunds, "required ISLM %s is greater than max_islm_amount %s", requiredISLM, maxIslmAmount)
	}

	moduleAddr := k.accountKeeper.GetModuleAddress(types.ModuleName)

	// Send RequiredISLM coins to module account
	islmCoin := sdk.NewCoin(types.ISLMBaseDenom, requiredISLM)
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, fromAddress, types.ModuleName, sdk.NewCoins(islmCoin))
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to send ISLM to module account")
	}

	// Burn RequiredISLM coins from module account
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(islmCoin))
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to burn ISLM coins")
	}

	// Update TotalBurnedAmount
	k.AddToTotalBurnedAmount(ctx, requiredISLM)

	// Mint ethiqAmount coins to module account
	ethiqCoin := sdk.NewCoin(types.BaseDenom, ethiqAmount)
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(ethiqCoin))
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to mint ethiq coins")
	}

	// Send minted ethiqAmount from module account to toAddress
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, toAddress, sdk.NewCoins(ethiqCoin))
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to send ethiq to recipient")
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeMintExecuted,
			sdk.NewAttribute(types.AttributeKeyEthiqAmount, ethiqAmount.String()),
			sdk.NewAttribute(types.AttributeKeyISLMAmount, requiredISLM.String()),
			sdk.NewAttribute(types.AttributeKeyToAddress, toAddress.String()),
			sdk.NewAttribute(types.AttributeKeyFromAddress, fromAddress.String()),
		),
	)

	_ = moduleAddr // silence unused variable warning

	return &types.MsgMintEthiqResponse{}, nil
}
