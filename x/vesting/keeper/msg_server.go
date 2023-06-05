// Copyright 2022 Evmos Foundation
// This file is part of the Evmos Network packages.
//
// Evmos is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Evmos packages are distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Evmos packages. If not, see https://github.com/evmos/evmos/blob/main/LICENSE

package keeper

import (
	"context"
	"strconv"
	"time"

	ethtypes "github.com/evmos/ethermint/types"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/ethereum/go-ethereum/common"
	evmosvestingtypes "github.com/evmos/evmos/v10/x/vesting/types"
	"github.com/haqq-network/haqq/x/vesting/types"
)

var _ types.MsgServer = &Keeper{}

// CreateClawbackVestingAccount creates a new ClawbackVestingAccount, or merges
// a grant into an existing one.
func (k Keeper) CreateClawbackVestingAccount(
	goCtx context.Context,
	msg *types.MsgCreateClawbackVestingAccount,
) (*types.MsgCreateClawbackVestingAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ak := k.accountKeeper
	bk := k.bankKeeper

	// Error checked during msg validation
	from := sdk.MustAccAddressFromBech32(msg.FromAddress)
	to := sdk.MustAccAddressFromBech32(msg.ToAddress)

	if bk.BlockedAddr(to) {
		return nil, errorsmod.Wrapf(errortypes.ErrUnauthorized,
			"%s is not allowed to receive funds", msg.ToAddress,
		)
	}

	vestingCoins := msg.VestingPeriods.TotalAmount()
	lockupCoins := msg.LockupPeriods.TotalAmount()

	// If lockup absent, default to an instant unlock schedule
	if !vestingCoins.IsZero() && len(msg.LockupPeriods) == 0 {
		msg.LockupPeriods = sdkvesting.Periods{
			{Length: 0, Amount: vestingCoins},
		}
		lockupCoins = vestingCoins
	}

	// If vesting absent, default to an instant vesting schedule
	if !lockupCoins.IsZero() && len(msg.VestingPeriods) == 0 {
		msg.VestingPeriods = sdkvesting.Periods{
			{Length: 0, Amount: lockupCoins},
		}
		vestingCoins = lockupCoins
	}

	// The vesting and lockup schedules must describe the same total amount.
	// IsEqual can panic, so use (a == b) <=> (a <= b && b <= a).
	if !(vestingCoins.IsAllLTE(lockupCoins) && lockupCoins.IsAllLTE(vestingCoins)) {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest,
			"lockup and vesting amounts must be equal",
		)
	}

	// Add Grant if vesting account exists, "merge" is true and funder is correct.
	// Otherwise create a new Clawback Vesting Account
	madeNewAcc := false
	acc := ak.GetAccount(ctx, to)
	var vestingAcc *types.ClawbackVestingAccount

	if acc != nil {
		var isClawback bool
		vestingAcc, isClawback = acc.(*types.ClawbackVestingAccount)

		switch {
		case !msg.Merge && isClawback:
			return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s already exists; consider using --merge", msg.ToAddress)
		case !msg.Merge && !isClawback:
			return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s already exists", msg.ToAddress)
		case msg.Merge && !isClawback:
			return nil, errorsmod.Wrapf(errortypes.ErrNotSupported, "account %s must be a clawback vesting account", msg.ToAddress)
		case msg.FromAddress != vestingAcc.FunderAddress:
			return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s can only accept grants from account %s", msg.ToAddress, vestingAcc.FunderAddress)
		}

		err := k.addGrant(ctx, vestingAcc, msg.GetStartTime().Unix(), msg.GetLockupPeriods(), msg.GetVestingPeriods(), vestingCoins)
		if err != nil {
			return nil, err
		}
		ak.SetAccount(ctx, vestingAcc)
	} else {
		baseAcc := authtypes.NewBaseAccountWithAddress(to)
		vestingAcc = types.NewClawbackVestingAccount(
			baseAcc,
			from,
			vestingCoins,
			msg.StartTime,
			msg.LockupPeriods,
			msg.VestingPeriods,
		)
		acc := ak.NewAccount(ctx, vestingAcc)
		ak.SetAccount(ctx, acc)
		madeNewAcc = true
	}

	if madeNewAcc {
		defer func() {
			telemetry.IncrCounter(1, "new", "account")

			//for _, a := range vestingCoins {
			//	if a.Amount.IsInt64() {
			//		telemetry.SetGaugeWithLabels(
			//			[]string{"tx", "msg", "create_clawback_vesting_account"},
			//			float32(a.Amount.Int64()),
			//			[]metrics.Label{telemetry.NewLabel("denom", a.Denom)},
			//		)
			//	}
			//}
		}()
	}

	// Send coins from the funder to vesting account
	if err := bk.SendCoins(ctx, from, to, vestingCoins); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeCreateClawbackVestingAccount,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.FromAddress),
				sdk.NewAttribute(types.AttributeKeyCoins, vestingCoins.String()),
				sdk.NewAttribute(types.AttributeKeyStartTime, msg.StartTime.String()),
				sdk.NewAttribute(types.AttributeKeyMerge, strconv.FormatBool(msg.Merge)),
				sdk.NewAttribute(types.AttributeKeyAccount, msg.ToAddress),
			),
		},
	)

	return &types.MsgCreateClawbackVestingAccountResponse{}, nil
}

// Clawback removes the unvested amount from a ClawbackVestingAccount.
// The destination defaults to the funder address, but can be overridden.
func (k Keeper) Clawback(
	goCtx context.Context,
	msg *types.MsgClawback,
) (*types.MsgClawbackResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ak := k.accountKeeper
	bk := k.bankKeeper

	// NOTE: ignore error in case dest address is not defined
	dest, _ := sdk.AccAddressFromBech32(msg.DestAddress)

	// NOTE: error checked during msg validation
	addr := sdk.MustAccAddressFromBech32(msg.AccountAddress)

	// Default destination to funder address
	if msg.DestAddress == "" {
		dest, _ = sdk.AccAddressFromBech32(msg.FunderAddress)
	}

	if bk.BlockedAddr(dest) {
		return nil, errorsmod.Wrapf(errortypes.ErrUnauthorized,
			"%s is not allowed to receive funds", msg.DestAddress,
		)
	}

	// Check if account exists
	acc := ak.GetAccount(ctx, addr)
	if acc == nil {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", msg.AccountAddress)
	}

	// Check if account has a clawback account
	va, ok := acc.(*types.ClawbackVestingAccount)
	if !ok {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account not subject to clawback: %s", msg.AccountAddress)
	}

	// Check if account funder is same as in msg
	if va.FunderAddress != msg.FunderAddress {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "clawback can only be requested by original funder %s", va.FunderAddress)
	}

	// Return error if clawback is attempted before start time
	if ctx.BlockTime().Before(va.StartTime) {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "clawback can only be executed after vesting begins: %s", va.FunderAddress)
	}

	// Perform clawback transfer
	if err := k.transferClawback(ctx, *va, dest); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeClawback,
				sdk.NewAttribute(types.AttributeKeyFunder, msg.FunderAddress),
				sdk.NewAttribute(types.AttributeKeyAccount, msg.AccountAddress),
				sdk.NewAttribute(types.AttributeKeyDestination, msg.DestAddress),
			),
		},
	)

	return &types.MsgClawbackResponse{}, nil
}

// UpdateVestingFunder updates the funder account of a ClawbackVestingAccount.
func (k Keeper) UpdateVestingFunder(
	goCtx context.Context,
	msg *types.MsgUpdateVestingFunder,
) (*types.MsgUpdateVestingFunderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ak := k.accountKeeper
	bk := k.bankKeeper

	// NOTE: errors checked during msg validation
	newFunder := sdk.MustAccAddressFromBech32(msg.NewFunderAddress)
	vesting := sdk.MustAccAddressFromBech32(msg.VestingAddress)

	// Need to check if new funder can receive funds because in
	// Clawback function, destination defaults to funder address
	if bk.BlockedAddr(newFunder) {
		return nil, errorsmod.Wrapf(errortypes.ErrUnauthorized,
			"%s is not allowed to receive funds", msg.NewFunderAddress,
		)
	}

	// Check if vesting account exists
	vestingAcc := ak.GetAccount(ctx, vesting)
	if vestingAcc == nil {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", msg.VestingAddress)
	}

	// Check if account is a clawback vesting account
	va, ok := vestingAcc.(*types.ClawbackVestingAccount)
	if !ok {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account not subject to clawback: %s", msg.VestingAddress)
	}

	// Check if account current funder is same as in msg
	if va.FunderAddress != msg.FunderAddress {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "clawback can only be requested by original funder %s", va.FunderAddress)
	}

	// Perform clawback account update
	va.FunderAddress = msg.NewFunderAddress
	// set the account with the updated funder
	ak.SetAccount(ctx, va)

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeUpdateVestingFunder,
				sdk.NewAttribute(types.AttributeKeyFunder, msg.FunderAddress),
				sdk.NewAttribute(types.AttributeKeyAccount, msg.VestingAddress),
				sdk.NewAttribute(types.AttributeKeyNewFunder, msg.NewFunderAddress),
			),
		},
	)

	return &types.MsgUpdateVestingFunderResponse{}, nil
}

// ConvertVestingAccount converts a ClawbackVestingAccount to the default chain account
// after its lock and vesting periods have concluded.
func (k Keeper) ConvertVestingAccount(
	goCtx context.Context,
	msg *types.MsgConvertVestingAccount,
) (*types.MsgConvertVestingAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	address := sdk.MustAccAddressFromBech32(msg.VestingAddress)
	account := k.accountKeeper.GetAccount(ctx, address)

	if account == nil {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", msg.VestingAddress)
	}

	// Check if account is of VestingAccount interface
	if _, ok := account.(vestingexported.VestingAccount); !ok {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account not subject to vesting: %s", msg.VestingAddress)
	}

	// check if account is of type ClawbackVestingAccount
	vestingAcc, ok := account.(*types.ClawbackVestingAccount)
	if !ok {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s is not a ClawbackVestingAccount", msg.VestingAddress)
	}

	// check if account  has any vesting coins left
	if vestingAcc.GetVestingCoins(ctx.BlockTime()) != nil {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "vesting coins still left in account: %s", msg.VestingAddress)
	}

	ethAccount := ethtypes.ProtoAccount().(*ethtypes.EthAccount)
	ethAccount.BaseAccount = vestingAcc.BaseAccount
	k.accountKeeper.SetAccount(ctx, ethAccount)

	return &types.MsgConvertVestingAccountResponse{}, nil
}

// ConvertIntoVestingAccount converts a default chain account to the ClawbackVestingAccount
// after its lock and vesting periods have started.
func (k Keeper) ConvertIntoVestingAccount(
	goCtx context.Context,
	msg *types.MsgConvertIntoVestingAccount,
) (*types.MsgConvertIntoVestingAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	funder := sdk.MustAccAddressFromBech32(msg.FromAddress)
	hexTargetAddr := common.HexToAddress(msg.EthAddress)
	targetAddress := sdk.AccAddress(hexTargetAddr.Bytes())
	bondDenom := k.stakingKeeper.BondDenom(ctx)
	targetAccount := k.accountKeeper.GetAccount(ctx, targetAddress)

	if targetAccount == nil {
		return nil, errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", msg.EthAddress)
	}

	if k.bankKeeper.BlockedAddr(targetAddress) {
		return nil, errorsmod.Wrapf(errortypes.ErrUnauthorized,
			"%s is not allowed to be converted into vesting", msg.EthAddress,
		)
	}

	// Check denom
	if msg.Amount.Denom != bondDenom {
		return nil, errorsmod.Wrapf(
			errortypes.ErrInvalidRequest,
			"vesting denom %s is not equal to bond denom %s",
			msg.Amount.Denom,
			bondDenom,
		)
	}

	// Get all balances of target account
	balances := k.bankKeeper.GetAllBalances(ctx, targetAddress)
	bondDenomBalance := balances.AmountOf(bondDenom)
	oneISLM := sdk.NewCoin(bondDenom, sdk.NewInt(1000000000000000000))

	if bondDenomBalance.GTE(oneISLM.Amount) {
		return nil, errorsmod.Wrapf(
			errortypes.ErrInvalidRequest,
			"account balance [%s] is greater than %s %s",
			msg.EthAddress,
			oneISLM.Amount,
			oneISLM.Denom,
		)
	}

	madeNewAcc := false
	var vestingAcc *types.ClawbackVestingAccount
	var isClawback bool

	// TODO Set startDate or lockLength based on msg.LongTerm value
	lockLength := time.Minute * 5
	vestingCoins := sdk.NewCoins(msg.Amount)

	// Setup default locking and vesting periods
	lockingPeriods := sdkvesting.Periods{
		{Length: lockLength.Milliseconds(), Amount: vestingCoins},
	}
	vestingPeriods := sdkvesting.Periods{
		{Length: 0, Amount: vestingCoins},
	}

	vestingAcc, isClawback = targetAccount.(*types.ClawbackVestingAccount)
	if !isClawback {
		baseAcc := authtypes.NewBaseAccountWithAddress(targetAddress)
		vestingAcc = types.NewClawbackVestingAccount(
			baseAcc,
			funder,
			vestingCoins,
			msg.StartTime,
			lockingPeriods,
			vestingPeriods,
		)
		acc := k.accountKeeper.NewAccount(ctx, vestingAcc)
		k.accountKeeper.SetAccount(ctx, acc)

		madeNewAcc = true
	} else {
		if msg.FromAddress != vestingAcc.FunderAddress {
			return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s can only accept grants from account %s", msg.EthAddress, vestingAcc.FunderAddress)
		}

		err := k.addGrant(ctx, vestingAcc, types.Min64(msg.StartTime.Unix(), vestingAcc.StartTime.Unix()), lockingPeriods, vestingPeriods, vestingCoins)
		if err != nil {
			return nil, err
		}
		k.accountKeeper.SetAccount(ctx, vestingAcc)
	}

	if madeNewAcc {
		defer func() {
			telemetry.IncrCounter(1, "new", "account")

			//for _, a := range balances {
			//	if a.Amount.IsInt64() {
			//		telemetry.SetGaugeWithLabels(
			//			[]string{"tx", "msg", "create_clawback_vesting_account"},
			//			float32(a.Amount.Int64()),
			//			[]metrics.Label{telemetry.NewLabel("denom", a.Denom)},
			//		)
			//	}
			//}
		}()
	}

	// Send coins from the funder to vesting account
	if err := k.bankKeeper.SendCoins(ctx, funder, targetAddress, vestingCoins); err != nil {
		return nil, err
	}

	// TODO Set certain validator
	validators := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
	// NOTE: source funds are always unbonded

	if len(validators) > 25 {
		validators = validators[:25]
	}

	newShares, err := k.stakingKeeper.Delegate(ctx, targetAddress, vestingCoins.AmountOf(bondDenom), stakingtypes.Unbonded, validators[len(validators)-1], true)
	if err != nil {
		return nil, err
	}

	defer func() {
		telemetry.IncrCounter(1, stakingtypes.ModuleName, "delegate")
		//	telemetry.SetGaugeWithLabels(
		//		[]string{"tx", "msg", stakingtypes.TypeMsgDelegate},
		//		float32(balances.AmountOf(bondDenom).Int64()),
		//		[]metrics.Label{telemetry.NewLabel("denom", bondDenom)},
		//	)
	}()

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeCreateClawbackVestingAccount,
				sdk.NewAttribute(sdk.AttributeKeySender, funder.String()),
				sdk.NewAttribute(types.AttributeKeyCoins, vestingCoins.String()),
				sdk.NewAttribute(types.AttributeKeyStartTime, vestingAcc.StartTime.String()),
				sdk.NewAttribute(types.AttributeKeyMerge, strconv.FormatBool(isClawback)),
				sdk.NewAttribute(types.AttributeKeyAccount, vestingAcc.Address),
			),
			sdk.NewEvent(
				stakingtypes.EventTypeDelegate,
				sdk.NewAttribute(stakingtypes.AttributeKeyValidator, validators[0].OperatorAddress),
				sdk.NewAttribute(sdk.AttributeKeyAmount, vestingCoins.AmountOf(bondDenom).String()),
				sdk.NewAttribute(stakingtypes.AttributeKeyNewShares, newShares.String()),
			),
			sdk.NewEvent(
				sdk.EventTypeMessage,
				sdk.NewAttribute(sdk.AttributeKeyModule, stakingtypes.AttributeValueCategory),
				sdk.NewAttribute(sdk.AttributeKeySender, targetAddress.String()),
			),
		},
	)

	return &types.MsgConvertIntoVestingAccountResponse{}, nil
}

// addGrant merges a new clawback vesting grant into an existing
// ClawbackVestingAccount.
func (k Keeper) addGrant(
	ctx sdk.Context,
	va *types.ClawbackVestingAccount,
	grantStartTime int64,
	grantLockupPeriods, grantVestingPeriods sdkvesting.Periods,
	grantCoins sdk.Coins,
) error {
	// how much is really delegated?
	bondedAmt := k.stakingKeeper.GetDelegatorBonded(ctx, va.GetAddress())
	unbondingAmt := k.stakingKeeper.GetDelegatorUnbonding(ctx, va.GetAddress())
	delegatedAmt := bondedAmt.Add(unbondingAmt)
	delegated := sdk.NewCoins(sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), delegatedAmt))

	// modify schedules for the new grant
	newLockupStart, newLockupEnd, newLockupPeriods := types.DisjunctPeriods(va.GetStartTime(), grantStartTime, va.LockupPeriods, grantLockupPeriods)
	newVestingStart, newVestingEnd, newVestingPeriods := types.DisjunctPeriods(va.GetStartTime(), grantStartTime,
		va.GetVestingPeriods(), grantVestingPeriods)

	if newLockupStart != newVestingStart {
		return errorsmod.Wrapf(
			evmosvestingtypes.ErrVestingLockup,
			"vesting start time calculation should match lockup start (%d ≠ %d)",
			newVestingStart, newLockupStart,
		)
	}

	va.StartTime = time.Unix(newLockupStart, 0)
	va.EndTime = types.Max64(newLockupEnd, newVestingEnd)
	va.LockupPeriods = newLockupPeriods
	va.VestingPeriods = newVestingPeriods
	va.OriginalVesting = va.OriginalVesting.Add(grantCoins...)

	// cap DV at the current unvested amount, DF rounds out to current delegated
	unvested := va.GetVestingCoins(ctx.BlockTime())
	va.DelegatedVesting = delegated.Min(unvested)
	va.DelegatedFree = delegated.Sub(va.DelegatedVesting...)
	return nil
}

// transferClawback transfers unvested tokens in a ClawbackVestingAccount to
// dest address, updates the lockup schedule and removes future vesting events.
func (k Keeper) transferClawback(
	ctx sdk.Context,
	va types.ClawbackVestingAccount,
	dest sdk.AccAddress,
) error {
	// Compute clawback amount, unlock unvested tokens and remove future vesting events
	updatedAcc, toClawBack := va.ComputeClawback(ctx.BlockTime().Unix())
	if toClawBack.IsZero() {
		// no-op, nothing to transfer
		return nil
	}

	// set the account with the updated values of the vesting schedule
	k.accountKeeper.SetAccount(ctx, &updatedAcc)

	addr := updatedAcc.GetAddress()

	// NOTE: don't use `SpendableCoins` to get the minimum value to clawback since
	// the amount is retrieved from `ComputeClawback`, which ensures correctness.
	// `SpendableCoins` can result in gas exhaustion if the user has too many
	// different denoms (because of store iteration).

	// Transfer clawback to the destination (funder)
	return k.bankKeeper.SendCoins(ctx, addr, dest, toClawBack)
}
