package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/contracts"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

var _ types.MsgServer = Keeper{}

// Liquidate liquidates specified amount of token locked in vesting into liquid token
func (k Keeper) Liquidate(goCtx context.Context, msg *types.MsgLiquidate) (*types.MsgLiquidateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsLiquidVestingEnabled(ctx) {
		return nil, errorsmod.Wrapf(types.ErrModuleIsDisabled, "liquid vesting module is disabled")
	}

	// check amount denom
	if msg.Amount.Denom != "aISLM" {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "unable to liquidate any other coin except aISLM")
	}

	// check amount
	minLiquidation := k.GetParams(ctx).MinimumLiquidationAmount
	if msg.Amount.IsLT(sdk.NewCoin("aISLM", minLiquidation)) {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "unable to liquidate amount lesser than %d", minLiquidation)
	}

	// get account
	liquidateFromAddress := sdk.MustAccAddressFromBech32(msg.LiquidateFrom)
	liquidateFromAccount := k.accountKeeper.GetAccount(ctx, liquidateFromAddress)
	if liquidateFromAccount == nil {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", msg.LiquidateFrom)
	}

	// set to address
	liquidateToAddress := liquidateFromAddress
	if msg.LiquidateTo != msg.LiquidateFrom {
		liquidateToAddress = sdk.MustAccAddressFromBech32(msg.LiquidateTo)
	}

	// check from account is vesting account
	va, isClawback := liquidateFromAccount.(*vestingtypes.ClawbackVestingAccount)
	if !isClawback {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s is regular nothing to liquidate", msg.LiquidateFrom)
	}

	// check there is not vesting periods on the schedule
	if !va.GetUnvestedOnly(ctx.BlockTime()).IsZero() {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s has vesting ongoing periods, unable to liquidate unvested coins", msg.LiquidateFrom)
	}

	// check account has liquidation target denom locked in vesting
	hasTargetDenom, lockedBalance := va.GetLockedOnly(ctx.BlockTime()).Find(msg.Amount.Denom)
	if !(hasTargetDenom) {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s doesn't contain coin specified as liquidation target", msg.LiquidateFrom)
	}

	// validate current locked periods have sufficient amount to be liquidated
	if lockedBalance.IsLT(msg.Amount) {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s doesn't have sufficient amount of target coin for liquidation", msg.LiquidateFrom)
	}

	// calculate new schedule
	upcomingPeriods := types.ExtractUpcomingPeriods(va.GetStartTime(), va.GetEndTime(), va.LockupPeriods, ctx.BlockTime().Unix())
	decreasedPeriods, diffPeriods, err := types.SubtractAmountFromPeriods(upcomingPeriods, msg.Amount)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to calculate new schedule: %s", err.Error())
	}
	va.LockupPeriods = types.ReplacePeriodsTail(va.LockupPeriods, decreasedPeriods)
	va.OriginalVesting = va.OriginalVesting.Sub(msg.Amount)

	// all vesting periods are completed at this point, so we can reduce amounts without additional extracting logic
	decreasedVestingPeriods, _, err := types.SubtractAmountFromPeriods(va.VestingPeriods, msg.Amount)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to calculate new schedule: %s", err.Error())
	}

	va.VestingPeriods = types.ReplacePeriodsTail(va.VestingPeriods, decreasedVestingPeriods)

	k.accountKeeper.SetAccount(ctx, va)

	// transfer liquidated amount to liquid vesting module account
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, liquidateFromAddress, types.ModuleName, sdk.NewCoins(msg.Amount))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to transfer liquidated locked coins from account to module: %s", err.Error())
	}

	diffPeriods[0].Length -= types.CurrentPeriodShift(va.StartTime.Unix(), ctx.BlockTime().Unix(), va.LockupPeriods)
	liquidDenom, err := k.CreateDenom(ctx, msg.Amount.Denom, ctx.BlockTime().Unix(), diffPeriods)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to create denom for liquid token: %s", err.Error())
	}

	// create new sdk denom for liquidated locked coins
	liquidTokenMetadata := banktypes.Metadata{
		Description: "Liquid vesting token",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    liquidDenom.GetBaseDenom(),
				Exponent: 0,
			},
			{
				Denom:    liquidDenom.GetDisplayDenom(),
				Exponent: 18,
			},
		},
		Base:    liquidDenom.GetBaseDenom(),
		Display: liquidDenom.GetDisplayDenom(),
		Name:    liquidDenom.GetDisplayDenom(),
		Symbol:  liquidDenom.GetDisplayDenom(),
	}

	liquidTokenCoin := sdk.NewCoin(liquidDenom.GetBaseDenom(), msg.Amount.Amount)
	liquidTokenCoins := sdk.NewCoins(liquidTokenCoin)
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, liquidTokenCoins)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to mint liquid token: %s", err.Error())
	}

	k.bankKeeper.SetDenomMetaData(ctx, liquidTokenMetadata)

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, liquidateToAddress, liquidTokenCoins)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to transfer liquid tokens to account %s", err.Error())
	}

	// bind newly created denom to erc20 token
	_, err = k.erc20Keeper.RegisterCoin(ctx, liquidTokenMetadata)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to create erc20 token pair: %s", err.Error())
	}

	// convert new liquid token to erc20 token
	// Build MsgConvertCoin, from recipient to recipient since Liquidation already occurred
	evmLiquidateToAddress := common.BytesToAddress(liquidateToAddress.Bytes())
	msgConvert := erc20types.NewMsgConvertCoin(liquidTokenCoin, evmLiquidateToAddress, liquidateToAddress)
	if _, err := k.erc20Keeper.ConvertCoin(sdk.WrapSDKContext(ctx), msgConvert); err != nil {
		return nil, errorsmod.Wrap(err, "failed to convert liquid tokens into erc20 tokens")
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeLiquidate,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.LiquidateFrom),
				sdk.NewAttribute(types.AttributeKeyDestination, msg.LiquidateTo),
				sdk.NewAttribute(types.AttributeKeyAmount, liquidTokenCoin.String()),
			),
		},
	)

	return &types.MsgLiquidateResponse{}, nil
}

// Redeem redeems specified amount of liquid token into original locked token and adds them to account
func (k Keeper) Redeem(goCtx context.Context, msg *types.MsgRedeem) (*types.MsgRedeemResponse, error) {
	// get accounts
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsLiquidVestingEnabled(ctx) {
		return nil, errorsmod.Wrapf(types.ErrModuleIsDisabled, "liquid vesting module is disabled")
	}

	fromAddress := sdk.MustAccAddressFromBech32(msg.RedeemFrom)
	fromAccount := k.accountKeeper.GetAccount(ctx, fromAddress)
	if fromAccount == nil {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", msg.RedeemFrom)
	}

	toAddress := sdk.MustAccAddressFromBech32(msg.RedeemTo)

	// query liquid token info
	liquidDenom, found := k.GetDenom(ctx, msg.Amount.Denom)
	if !found {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "liquidDenom %s does not exist", msg.Amount.Denom)
	}

	// get token pair
	tokenPairID := k.erc20Keeper.GetTokenPairID(ctx, msg.Amount.Denom)
	if len(tokenPairID) == 0 {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "token pair for denom %s not found", msg.Amount.Denom)
	}
	tokenPair, found := k.erc20Keeper.GetTokenPair(ctx, tokenPairID)
	if !found || !tokenPair.Enabled {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "token pair for denom %s not found", msg.Amount.Denom)
	}

	// check fromAccount has enough liquid token in balance
	if balance := k.bankKeeper.GetBalance(ctx, fromAddress, msg.Amount.Denom); balance.IsLT(msg.Amount) {
		// get erc20 liquid token balance
		contract := tokenPair.GetERC20Contract()
		erc20LiquidTokenBalance := math.NewIntFromBigInt(k.erc20Keeper.BalanceOf(
			ctx,
			contracts.ERC20MinterBurnerDecimalsContract.ABI,
			contract,
			common.BytesToAddress(fromAddress.Bytes()),
		))

		// check if erc20 + cosmos tokens are sufficient for redeem
		if erc20LiquidTokenBalance.Add(balance.Amount).LT(msg.Amount.Amount) {
			return nil, errorsmod.Wrapf(types.ErrRedeemFailed, "from account has insufficient balance")
		}

		// transfer token from erc20 layer to get sufficient amount
		amountToConvert := msg.Amount.Amount.Sub(balance.Amount)
		msgConvert := erc20types.NewMsgConvertERC20(
			amountToConvert,
			fromAddress,
			contract,
			common.BytesToAddress(fromAddress.Bytes()),
		)
		_, err := k.erc20Keeper.ConvertERC20(sdk.WrapSDKContext(ctx), msgConvert)
		if err != nil {
			return nil, errorsmod.Wrapf(types.ErrRedeemFailed, "failed to convert erc20 token: %s", err.Error())
		}
	}

	// transfer liquid denom to liquidvesting module
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, fromAddress, types.ModuleName, sdk.NewCoins(msg.Amount))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrRedeemFailed, "failed to transfer liquid token to module: %s", err.Error())
	}

	// burn liquid token specified amount
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(msg.Amount))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrRedeemFailed, "failed to burn liquid tokens: %s", err.Error())
	}

	// subtract burned amount from token schedule
	originalDenomCoin := sdk.NewCoin(liquidDenom.GetOriginalDenom(), msg.Amount.Amount)
	decreasedPeriods, diffPeriods, err := types.SubtractAmountFromPeriods(liquidDenom.LockupPeriods, originalDenomCoin)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrRedeemFailed, "failed to calculate new liquid denom schedule: %s", err.Error())
	}
	// save modified token schedule
	if decreasedPeriods.TotalAmount().IsZero() {
		k.DeleteDenom(ctx, liquidDenom.GetBaseDenom())
		if tokenPair.Enabled {
			_, err := k.erc20Keeper.ToggleConversion(ctx, msg.Amount.Denom)
			if err != nil {
				return nil, errorsmod.Wrapf(types.ErrRedeemFailed, "failed to disable conversion: %s", err.Error())
			}
		}
	} else {
		err = k.UpdateDenomPeriods(ctx, liquidDenom.GetBaseDenom(), decreasedPeriods)
		if err != nil {
			return nil, errorsmod.Wrapf(types.ErrRedeemFailed, "failed to update liquid denom schedule: %s", err.Error())
		}
	}

	// transfer original token to account
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, toAddress, sdk.NewCoins(originalDenomCoin))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrRedeemFailed, "failed to transfer original denom to target account: %s", err.Error())
	}

	upcomingPeriods := types.ExtractUpcomingPeriods(
		liquidDenom.GetStartTime().Unix(),
		liquidDenom.GetEndTime().Unix(),
		diffPeriods,
		ctx.BlockTime().Unix(),
	)

	// if there are upcoming periods, apply vesting schedule on target account
	if len(upcomingPeriods) > 0 {
		funder := k.accountKeeper.GetModuleAddress(types.ModuleName)
		// check if toAddress already a vesting account to apply current funder
		toAccount := k.accountKeeper.GetAccount(ctx, toAddress)
		if toAccount == nil {
			return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", toAddress)
		}
		toVestingAcc, isClawback := toAccount.(*vestingtypes.ClawbackVestingAccount)
		if isClawback {
			funder = sdk.MustAccAddressFromBech32(toVestingAcc.FunderAddress)
		}

		_, _, _, err = k.vestingKeeper.ApplyVestingSchedule(
			ctx,
			funder,
			toAddress,
			sdk.NewCoins(originalDenomCoin),
			liquidDenom.GetStartTime(),
			diffPeriods,
			sdkvesting.Periods{{Length: 0, Amount: sdk.NewCoins(originalDenomCoin)}},
			true,
		)
		if err != nil {
			return nil, errorsmod.Wrapf(types.ErrRedeemFailed, "failed to apply vesting schedule to account %s: %s", toAddress, err.Error())
		}
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeRedeem,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.RedeemFrom),
				sdk.NewAttribute(types.AttributeKeyDestination, msg.RedeemTo),
				sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
			),
		},
	)

	return &types.MsgRedeemResponse{}, nil
}
