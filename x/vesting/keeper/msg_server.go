package keeper

import (
	"context"
	"strconv"
	"time"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/evmos/ethermint/types"
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
	ak := k.accountKeeper
	bk := k.bankKeeper

	var (
		to  sdk.AccAddress
		err error
	)

	// to address can be both Bech32 or Hex
	from := sdk.MustAccAddressFromBech32(msg.FromAddress)
	to, err = sdk.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		hexTargetAddr := common.HexToAddress(msg.ToAddress)
		to = hexTargetAddr.Bytes()
	}

	if err := sdk.VerifyAddressFormat(to); err != nil {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidAddress,
			"%s is not valid address", msg.ToAddress,
		)
	}

	if bk.BlockedAddr(to) {
		return nil, errorsmod.Wrapf(errortypes.ErrUnauthorized,
			"%s is not allowed to be converted into vesting", msg.ToAddress,
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

	createNewAcc := false
	targetAccount := k.accountKeeper.GetAccount(ctx, to)

	if targetAccount == nil {
		createNewAcc = true
	}

	madeNewAcc := false
	var vestingAcc *types.ClawbackVestingAccount
	var isClawback bool

	_, isEthAccount := targetAccount.(*ethtypes.EthAccount)
	vestingAcc, isClawback = targetAccount.(*types.ClawbackVestingAccount)

	if !isClawback && !isEthAccount && !createNewAcc {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s already exists but can't be converted into vesting account", msg.ToAddress)
	}

	if !isClawback || createNewAcc {
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
	} else {
		if !msg.Merge {
			return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s already exists; consider using --merge", msg.ToAddress)
		}

		if msg.FromAddress != vestingAcc.FunderAddress {
			return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s can only accept grants from account %s", msg.ToAddress, vestingAcc.FunderAddress)
		}

		err := k.addGrant(ctx, vestingAcc, types.Min64(msg.StartTime.Unix(), vestingAcc.StartTime.Unix()), msg.LockupPeriods, msg.VestingPeriods, vestingCoins)
		if err != nil {
			return nil, err
		}
		ak.SetAccount(ctx, vestingAcc)
	}

	if madeNewAcc {
		defer func() {
			telemetry.IncrCounter(1, "new", "account")
		}()
	}

	// Send coins from the funder to vesting account
	if err := k.bankKeeper.SendCoins(ctx, from, to, vestingCoins); err != nil {
		return nil, err
	}

	events := sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateClawbackVestingAccount,
			sdk.NewAttribute(sdk.AttributeKeySender, from.String()),
			sdk.NewAttribute(types.AttributeKeyCoins, vestingCoins.String()),
			sdk.NewAttribute(types.AttributeKeyStartTime, vestingAcc.StartTime.String()),
			sdk.NewAttribute(types.AttributeKeyMerge, strconv.FormatBool(isClawback)),
			sdk.NewAttribute(types.AttributeKeyAccount, vestingAcc.Address),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, stakingtypes.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, to.String()),
		),
	}

	if msg.Stake {
		events, err = k.delegateVestedCoins(ctx, to, msg, events)
		if err != nil {
			return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to delegate vested coins: %s", err.Error())
		}
	}

	ctx.EventManager().EmitEvents(events)

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

func (k Keeper) delegateVestedCoins(ctx sdk.Context, to sdk.AccAddress, msg *types.MsgConvertIntoVestingAccount, events sdk.Events) (sdk.Events, error) {
	valAddress, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return events, errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid validator address: %s", msg.ValidatorAddress)
	}

	validator, found := k.stakingKeeper.GetValidator(ctx, valAddress)
	if !found {
		return events, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "validator %s not found", msg.ValidatorAddress)
	}

	newVestingStart, newVestingEnd, _ := types.DisjunctPeriods(
		msg.GetStartTime().Unix(),
		msg.GetStartTime().Unix(),
		msg.GetVestingPeriods(),
		msg.GetVestingPeriods(),
	)
	vestedCoins := types.ReadSchedule(
		newVestingStart,
		newVestingEnd,
		msg.GetVestingPeriods(),
		msg.GetVestingPeriods().TotalAmount(),
		ctx.BlockTime().Unix(),
	)
	if vestedCoins.IsZero() {
		// Nothing to delegate
		return events, nil
	}

	found, amountToDelegate := vestedCoins.Find(k.stakingKeeper.BondDenom(ctx))
	if !found || amountToDelegate.IsZero() {
		// Nothing to delegate
		return events, nil
	}

	newShares, err := k.stakingKeeper.Delegate(ctx, to, amountToDelegate.Amount, stakingtypes.Unbonded, validator, true)
	if err != nil {
		return events, err
	}

	telemetry.IncrCounter(1, stakingtypes.ModuleName, "delegate")

	events = append(events, sdk.NewEvent(
		stakingtypes.EventTypeDelegate,
		sdk.NewAttribute(stakingtypes.AttributeKeyValidator, validator.OperatorAddress),
		sdk.NewAttribute(sdk.AttributeKeyAmount, amountToDelegate.String()),
		sdk.NewAttribute(stakingtypes.AttributeKeyNewShares, newShares.String()),
	))

	return events, nil
}
