package keeper

import (
	"crypto/sha256"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ethiq/types"
)

// BurnIslmForHaqq burns aISLM coins and mints aHAQQ coins.
// It validates the burn request, calculates amount of aHAQQ to be minted, burns aISLM, and mints aHAQQ
// Returns the actual aHAQQ amount minted and any error
func (k Keeper) BurnIslmForHaqq(ctx sdk.Context, islmAmount sdkmath.Int, fromAddress, toAddress sdk.AccAddress) (sdkmath.Int, error) {
	// Short no-op circuit if module is disabled
	if !k.IsModuleEnabled(ctx) {
		return sdkmath.ZeroInt(), types.ErrModuleDisabled
	}

	// Validate fromAddress
	if fromAddress.Empty() {
		return sdkmath.ZeroInt(), errorsmod.Wrap(types.ErrInvalidAddress, "from_address cannot be empty")
	}

	// Validate toAddress
	if toAddress.Empty() {
		return sdkmath.ZeroInt(), errorsmod.Wrap(types.ErrInvalidAddress, "to_address cannot be empty")
	}

	// Return error if islmAmount is less than one (negative or zero)
	if islmAmount.LT(sdkmath.OneInt()) {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrInvalidAmount, "islm_amount must be greater than zero, got %s", islmAmount)
	}

	// Calculate aHAQQ amount to be minted
	haqqAmount, err := k.CalculateHaqqCoinsToMint(ctx, islmAmount)
	if err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrap(err, "failed to calculate aHAQQ amount to be minted")
	}

	if err := k.validateAmountToBeMinted(ctx, haqqAmount); err != nil {
		return sdkmath.ZeroInt(), err
	}

	if err := k.redeemAllLiquidVestingCoins(ctx, fromAddress, false); err != nil {
		return sdkmath.ZeroInt(), err
	}
	haqqSupplyBefore := k.GetHaqqSupply(ctx)
	haqqSupplyAfter := haqqSupplyBefore.Add(haqqAmount)
	if haqqSupplyAfter.GT(k.GetMaxSupply(ctx)) {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrExceedsMaxSupply, "aHAQQ supply after tx: %s", haqqSupplyAfter.String())
	}

	vestingIslmUsed, err := k.unlockVestingCoins(ctx, fromAddress, sdk.NewCoin(utils.BaseDenom, islmAmount))
	if err != nil {
		return sdkmath.ZeroInt(), err
	}
	freeIslmUsed := islmAmount.Sub(vestingIslmUsed.Amount)

	// Send aISLM coins to module account and burn
	if err := k.burnCoins(ctx, fromAddress, islmAmount, false); err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrap(err, "failed to burn aISLM coins")
	}

	// Mint aHAQQ coins to module account and send to recipient
	if err := k.mintCoins(ctx, toAddress, haqqAmount); err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrap(err, "failed to mint aHAQQ coins")
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeMintExecuted,
			sdk.NewAttribute(types.AttributeKeySender, fromAddress.String()),
			sdk.NewAttribute(types.AttributeKeyReceiver, toAddress.String()),
			sdk.NewAttribute(types.AttributeKeyHaqqMinted, haqqAmount.String()),
			sdk.NewAttribute(types.AttributeKeyIslmSpent, islmAmount.String()),
			sdk.NewAttribute(types.AttributeKeyHaqqSupplyBefore, haqqSupplyBefore.String()),
			sdk.NewAttribute(types.AttributeKeyHaqqSupplyAfter, haqqSupplyAfter.String()),
			sdk.NewAttribute(types.AttributeKeyIslmVestingUsed, vestingIslmUsed.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyIslmFreeUsed, freeIslmUsed.String()),
		),
	)

	return haqqAmount, nil
}

func (k Keeper) BurnIslmForHaqqByApplicationID(ctx sdk.Context, appID uint64) (sdkmath.Int, error) {
	// Short no-op circuit if module is disabled
	if !k.IsModuleEnabled(ctx) {
		return sdkmath.ZeroInt(), types.ErrModuleDisabled
	}

	application, err := types.GetApplicationByID(appID)
	if err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrInvalidApplicationID, "application %d not found", appID)
	}

	if k.IsApplicationExecuted(ctx, application.Id) {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrInvalidApplicationID, "application ID %d is already executed", application.Id)
	}

	isUcdao := application.Source == types.SourceOfFunds_SOURCE_OF_FUNDS_UCDAO
	fromAddress := sdk.MustAccAddressFromBech32(application.FromAddress)
	toAddress := sdk.MustAccAddressFromBech32(application.ToAddress)
	islmCoinsToBurn := application.BurnAmount
	islmPreBurntByApplications := application.BurnedBeforeAmount

	// use UCDAO escrow address if needed
	if isUcdao {
		fromAddress = GetUCDAOEscrowAddress(fromAddress)
	}

	// redeem liquid tokens on UCDAO escrow account
	if err := k.redeemAllLiquidVestingCoins(ctx, fromAddress, isUcdao); err != nil {
		return sdkmath.ZeroInt(), err
	}
	// Calculate aHAQQ amount to be minted
	haqqAmount, err := CalculateHaqqAmount(islmPreBurntByApplications.Amount, islmCoinsToBurn.Amount)
	if err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrap(err, "failed to calculate aHAQQ amount to be minted")
	}

	if err := k.validateAmountToBeMinted(ctx, haqqAmount); err != nil {
		return sdkmath.ZeroInt(), err
	}

	haqqSupplyBefore := k.GetHaqqSupply(ctx)
	haqqSupplyAfter := haqqSupplyBefore.Add(haqqAmount)
	if haqqSupplyAfter.GT(k.GetMaxSupply(ctx)) {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrExceedsMaxSupply, "aHAQQ supply after tx: %s", haqqSupplyAfter.String())
	}

	vestingIslmUsed, err := k.unlockVestingCoins(ctx, fromAddress, islmCoinsToBurn)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}
	freeIslmUsed := islmCoinsToBurn.Sub(vestingIslmUsed)

	if err := k.burnCoins(ctx, fromAddress, islmCoinsToBurn.Amount, true); err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrBurnCoins, "%v", err)
	}

	if err := k.mintCoins(ctx, toAddress, haqqAmount); err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrMintCoins, "%v", err)
	}

	k.SetApplicationAsExecuted(ctx, appID)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeMintByApplicationIDExecuted,
			sdk.NewAttribute(types.AttributeKeySender, fromAddress.String()),
			sdk.NewAttribute(types.AttributeKeyReceiver, toAddress.String()),
			sdk.NewAttribute(types.AttributeKeyHaqqMinted, haqqAmount.String()),
			sdk.NewAttribute(types.AttributeKeyIslmSpent, islmCoinsToBurn.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyHaqqSupplyBefore, haqqSupplyBefore.String()),
			sdk.NewAttribute(types.AttributeKeyHaqqSupplyAfter, haqqSupplyAfter.String()),
			sdk.NewAttribute(types.AttributeKeyIslmVestingUsed, vestingIslmUsed.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyIslmFreeUsed, freeIslmUsed.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyApplicationID, fmt.Sprintf("%d", appID)),
			sdk.NewAttribute(types.AttributeKeyApplicationFundsSource, types.FundSources[application.Source]),
		),
	)

	return haqqAmount, nil
}

// validateAmountToBeMinted checks whether the specified amount meets the set criteria according to the module parameters.
func (k Keeper) validateAmountToBeMinted(ctx sdk.Context, amt sdkmath.Int) error {
	params := k.GetParams(ctx)

	// Return error if haqq amount is less than MinMintPerTx
	if amt.LT(params.MinMintPerTx) {
		return errorsmod.Wrapf(types.ErrInvalidAmount, "haqq_amount is less than min_mint_per_tx: %s < %s", amt.String(), params.MinMintPerTx)
	}

	// Return error if haqq amount is greater than MaxMintPerTx
	if amt.GT(params.MaxMintPerTx) {
		return errorsmod.Wrapf(types.ErrInvalidAmount, "haqq_amount is greater than max_mint_per_tx: %s > %s", amt.String(), params.MaxMintPerTx)
	}

	// Return error if haqq amount is less than 1, bank can't mint zero coins
	if amt.LT(sdkmath.OneInt()) {
		return errorsmod.Wrapf(types.ErrInvalidAmount, "haqq_amount must be at least 1, got %s", amt.String())
	}

	return nil
}

// burnCoins burns aISLM coins from the given address using standard bank BurnCoins method
func (k Keeper) burnCoins(ctx sdk.Context, from sdk.AccAddress, amt sdkmath.Int, isApplication bool) error {
	islmCoin := sdk.NewCoin(utils.BaseDenom, amt)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, from, types.ModuleName, sdk.NewCoins(islmCoin)); err != nil {
		return errorsmod.Wrap(err, "failed to send aISLM to module account")
	}

	// Burn aISLM coins from module account
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(islmCoin)); err != nil {
		return errorsmod.Wrap(err, "failed to burn aISLM coins")
	}

	// Update TotalBurnedAmount
	k.AddToTotalBurnedAmount(ctx, amt)
	if isApplication {
		k.AddToTotalBurnedFromApplicationsAmount(ctx, amt)
	}

	return nil
}

// mintCoins mints aHAQQ coins to the given address using standard bank MintCoins method
func (k Keeper) mintCoins(ctx sdk.Context, to sdk.AccAddress, amt sdkmath.Int) error {
	haqqCoin := sdk.NewCoin(types.BaseDenom, amt)
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(haqqCoin)); err != nil {
		return errorsmod.Wrap(err, "failed to mint aHAQQ coins")
	}

	// Send minted aHAQQ from module account to toAddress
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, to, sdk.NewCoins(haqqCoin)); err != nil {
		return errorsmod.Wrap(err, "failed to send aHAQQ to recipient")
	}

	return nil
}

// GetUCDAOEscrowAddress returns the escrow address for the specified share owner.
// This function is based on native GetEscrowAddress of IBC Transfer module and
// follows the format as outlined in ADR 028 with minimal changes:
// https://github.com/cosmos/cosmos-sdk/blob/master/docs/architecture/adr-028-public-key-addresses.md
func GetUCDAOEscrowAddress(owner sdk.AccAddress) sdk.AccAddress {
	preImage := []byte("ucdao")
	preImage = append(preImage, 0)
	preImage = append(preImage, owner.Bytes()...)
	hash := sha256.Sum256(preImage)
	return hash[:20]
}
