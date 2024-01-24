package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

var _ types.MsgServer = Keeper{}

func (k Keeper) Liquidate(goCtx context.Context, msg *types.MsgLiquidate) (*types.MsgLiquidateResponse, error) {
	// get account
	ctx := sdk.UnwrapSDKContext(goCtx)
	address := sdk.MustAccAddressFromBech32(msg.Address)
	account := k.accountKeeper.GetAccount(ctx, address)
	if account == nil {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", msg.Address)
	}

	// check account is vesting account
	acc := k.accountKeeper.GetAccount(ctx, address)
	va, isClawback := acc.(*vestingtypes.ClawbackVestingAccount)
	if !isClawback {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s is regular nothing to liquidate", msg.Address)
	}

	// check there is not vesting periods on the schedule
	vestingPeriods := va.VestingPeriods
	if len(vestingPeriods) > 1 || vestingPeriods.TotalLength() > 0 || ctx.BlockTime().Before(va.StartTime) {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s has vesting periods, unable to liquidate locked coins", msg.Address)
	}

	// check account has only ISLM in locked in vesting
	hasTargetDenom, targetCoin := va.GetLockedOnly(ctx.BlockTime()).Find(msg.Amount.Denom)
	if !(hasTargetDenom) {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s doesn't contain coin specified as liquidation target", msg.Address)
	}

	// validate current locked periods have sufficient to be liquidated
	if targetCoin.IsLT(msg.Amount) {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s doesn't have sufficient amount of target coin for liquidation", msg.Address)
	}

	// calculate new schedule
	upcomingPeriods := types.ExtractUpcomingPeriods(va.GetStartTime(), va.GetEndTime(), va.LockupPeriods, ctx.BlockTime().Unix())
	decreasedPeriods, diffPeriods, err := types.SubtractAmountFromPeriods(upcomingPeriods, msg.Amount)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to calculate new schedule: %s", err.Error())
	}
	va.LockupPeriods = types.ReplacePeriodsTail(va.LockupPeriods, decreasedPeriods)
	k.accountKeeper.SetAccount(ctx, va)

	// transfer liquidated amount to liquid vesting module account
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, address, types.ModuleName, sdk.NewCoins(msg.Amount))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to transfer liquidated locked coins from account to module: %s", err.Error())
	}

	// create new sdk denom for liquidated locked coins
	diffPeriods[0].Length -= types.CurrentPeriodShift(va.StartTime.Unix(), ctx.BlockTime().Unix(), va.LockupPeriods)
	liquidDenom, err := k.CreateDenom(ctx, ctx.BlockTime().Unix(), diffPeriods)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to create denom for liquid token: %s", err.Error())
	}

	liquidTokenCoins := sdk.NewCoins(sdk.NewCoin(liquidDenom.GetDenom(), msg.Amount.Amount))
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, liquidTokenCoins)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to mint liquid token: %s", err.Error())
	}

	liquidTokenMetadata := banktypes.Metadata{
		Description: "Liquid vesting token",
		DenomUnits: []*banktypes.DenomUnit{{
			Denom:    liquidDenom.GetDenom(),
			Exponent: 0,
		}},
		Base: liquidDenom.GetDenom(),
	}

	k.bankKeeper.SetDenomMetaData(ctx, liquidTokenMetadata)

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, address, liquidTokenCoins)
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

// func (k Keeper) Redeem(ctx context.Context, redeem *types.MsgRedeem) (*types.MsgRedeemResponse, error) {
// query liquid token info
// check liquid token sufficient amount
// burn liquid token specified amount
// subtract burned amount from token schedule
// save modified token schedule
// convert to account into vesting account
// or
// just transfer tokens if there is no upcoming vesting  periods
//

// return &types.MsgRedeemResponse{}, nil
//}
