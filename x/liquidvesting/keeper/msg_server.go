package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

var _ types.MsgServer = Keeper{}

func (k Keeper) Liquidate(goCtx context.Context, msg *types.MsgLiquidate) (*types.MsgLiquidateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

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
	vestingPeriods := va.VestingPeriods
	if len(vestingPeriods) > 1 || vestingPeriods.TotalLength() > 0 || ctx.BlockTime().Before(va.StartTime) {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s has vesting periods, unable to liquidate locked coins", msg.LiquidateFrom)
	}

	// check account has liquidation target denom locked in vesting
	hasTargetDenom, targetCoin := va.GetLockedOnly(ctx.BlockTime()).Find(msg.Amount.Denom)
	if !(hasTargetDenom) {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s doesn't contain coin specified as liquidation target", msg.LiquidateFrom)
	}

	// validate current locked periods have sufficient amount to be liquidated
	if targetCoin.IsLT(msg.Amount) {
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
	k.accountKeeper.SetAccount(ctx, va)

	// transfer liquidated amount to liquid vesting module account
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, liquidateFromAddress, types.ModuleName, sdk.NewCoins(msg.Amount))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to transfer liquidated locked coins from account to module: %s", err.Error())
	}

	// create new sdk denom for liquidated locked coins
	diffPeriods[0].Length -= types.CurrentPeriodShift(va.StartTime.Unix(), ctx.BlockTime().Unix(), va.LockupPeriods)
	liquidDenom, err := k.CreateDenom(ctx, msg.Amount.Denom, ctx.BlockTime().Unix(), diffPeriods)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to create denom for liquid token: %s", err.Error())
	}

	liquidTokenCoins := sdk.NewCoins(sdk.NewCoin(liquidDenom.GetLiquidDenom(), msg.Amount.Amount))
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, liquidTokenCoins)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to mint liquid token: %s", err.Error())
	}

	liquidTokenMetadata := banktypes.Metadata{
		Description: "Liquid vesting token",
		DenomUnits: []*banktypes.DenomUnit{{
			Denom:    liquidDenom.GetLiquidDenom(),
			Exponent: 0,
		}},
		Base: liquidDenom.GetLiquidDenom(),
	}

	k.bankKeeper.SetDenomMetaData(ctx, liquidTokenMetadata)

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, liquidateToAddress, liquidTokenCoins)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to transfer liquid tokens to vesting account %s", err.Error())
	}

	// bind newly created denom to erc20 token
	_, err = k.erc20Keeper.RegisterCoin(ctx, liquidTokenMetadata)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to create erc20 token pair: %s", err.Error())
	}

	return &types.MsgLiquidateResponse{}, nil
}

func (k Keeper) Redeem(goCtx context.Context, msg *types.MsgRedeem) (*types.MsgRedeemResponse, error) {
	// get accounts
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddress := sdk.MustAccAddressFromBech32(msg.RedeemFrom)
	fromAccount := k.accountKeeper.GetAccount(ctx, fromAddress)
	if fromAccount == nil {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", msg.RedeemFrom)
	}

	toAddress := sdk.MustAccAddressFromBech32(msg.RedeemTo)
	toAccount := k.accountKeeper.GetAccount(ctx, toAddress)
	if toAccount == nil {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", msg.RedeemTo)
	}

	// query liquid token info
	liquidDenom, found := k.GetDenom(ctx, msg.Amount.Denom)
	if !found {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "liquidDenom %s does not exist", msg.Amount.Denom)
	}

	// check fromAccount has enough liquid token in balance
	if hasBalance := k.bankKeeper.HasBalance(ctx, fromAddress, msg.Amount); !hasBalance {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "from account has insufficient balance")
	}

	// transfer liquid denom to liquidvesting module
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, fromAddress, types.ModuleName, sdk.NewCoins(msg.Amount))
	if err != nil {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to transfer liquid token to module: %s", err.Error())
	}

	// burn liquid token specified amount
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(msg.Amount))
	if err != nil {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to burn liquid tokens: %s", err.Error())
	}

	// subtract burned amount from token schedule
	decreasedPeriods, diffPeriods, err := types.SubtractAmountFromPeriods(liquidDenom.LockupPeriods, msg.Amount)
	if err != nil {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to calculate new liquid denom schedule: %s", err.Error())
	}
	// save modified token schedule
	err = k.UpdateDenomPeriods(ctx, liquidDenom.LiquidDenom, decreasedPeriods)
	if err != nil {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to update liquid denom schedule: %s", err.Error())
	}

	// transfer original token to account
	originalDenomCoins := sdk.NewCoins(sdk.NewCoin(liquidDenom.GetOriginalDenom(), msg.Amount.Amount))
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, toAddress, originalDenomCoins)
	if err != nil {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to transfer original denom to target account: %s", err.Error())
	}

	upcomingPeriods := types.ExtractUpcomingPeriods(
		liquidDenom.GetStartTime().Unix(),
		liquidDenom.GetEndTime().Unix(),
		diffPeriods,
		ctx.BlockTime().Unix(),
	)

	// if there is upcoming periods, apply vesting schedule on target account
	if len(upcomingPeriods) > 0 {
		_, _, _, err = k.vestingKeeper.ApplyVestingSchedule(
			ctx,
			k.accountKeeper.GetModuleAddress(types.ModuleName),
			toAddress,
			sdk.NewCoins(msg.Amount),
			liquidDenom.GetStartTime(),
			diffPeriods,
			sdkvesting.Periods{{Length: 0, Amount: sdk.NewCoins(msg.Amount)}},
			true,
		)
		if err != nil {
			return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to apply vesting schedule to account %s: %s", toAddress, err.Error())
		}
	}

	return &types.MsgRedeemResponse{}, nil
}
