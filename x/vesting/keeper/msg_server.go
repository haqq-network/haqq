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
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	ethtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/utils"
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
	funderAddress := sdk.MustAccAddressFromBech32(msg.FromAddress)
	vestingAddress := sdk.MustAccAddressFromBech32(msg.ToAddress)

	if bk.BlockedAddr(vestingAddress) {
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
	acc := ak.GetAccount(goCtx, vestingAddress)
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

		err := k.addGrant(goCtx, vestingAcc, msg.GetStartTime().Unix(), msg.GetLockupPeriods(), msg.GetVestingPeriods(), vestingCoins)
		if err != nil {
			return nil, err
		}
		ak.SetAccount(goCtx, vestingAcc)
	} else {
		codeHash := common.BytesToHash(crypto.Keccak256(nil))
		baseAcc := authtypes.NewBaseAccountWithAddress(vestingAddress)
		vestingAcc = types.NewClawbackVestingAccount(
			baseAcc,
			funderAddress,
			vestingCoins,
			msg.StartTime,
			msg.LockupPeriods,
			msg.VestingPeriods,
			&codeHash,
		)
		acc := ak.NewAccount(goCtx, vestingAcc)
		ak.SetAccount(goCtx, acc)
		madeNewAcc = true
	}

	if madeNewAcc {
		defer func() {
			telemetry.IncrCounter(1, "new", "account")
		}()
	}

	// Send coins from the funder to vesting account
	if err := bk.SendCoins(goCtx, funderAddress, vestingAddress, vestingCoins); err != nil {
		return nil, err
	}

	telemetry.IncrCounter(
		float32(ctx.GasMeter().GasConsumed()),
		"tx", "fund_vesting_account", "gas_used",
	)
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
//
// Checks performed on the ValidateBasic include:
//   - funder and vesting addresses are correct bech32 format
//   - if destination address is not empty it is also correct bech32 format
func (k Keeper) Clawback(
	goCtx context.Context,
	msg *types.MsgClawback,
) (*types.MsgClawbackResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	bk := k.bankKeeper

	// NOTE: errors checked during msg validation
	addr := sdk.MustAccAddressFromBech32(msg.AccountAddress)
	funder := sdk.MustAccAddressFromBech32(msg.FunderAddress)

	// NOTE: ignore error in case dest address is not defined and default to funder address
	//#nosec G703 -- error is checked during ValidateBasic already.
	dest, _ := sdk.AccAddressFromBech32(msg.DestAddress)
	if msg.DestAddress == "" {
		dest = funder
	}

	if bk.BlockedAddr(dest) {
		return nil, errorsmod.Wrapf(errortypes.ErrUnauthorized,
			"%s is a blocked address and not allowed to receive funds", msg.DestAddress,
		)
	}

	// Get clawback vesting account
	va, err := k.GetClawbackVestingAccount(goCtx, addr)
	if err != nil {
		return nil, err
	}

	// Check if account has any vesting or lockup periods
	if len(va.VestingPeriods) == 0 && len(va.LockupPeriods) == 0 {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s has no vesting or lockup periods", msg.AccountAddress)
	}

	// Check if account funder is same as in msg
	if va.FunderAddress != funder.String() {
		return nil, errorsmod.Wrapf(errortypes.ErrUnauthorized, "clawback can only be requested by original funder: %s", va.FunderAddress)
	}

	// Perform clawback transfer
	if err := k.transferClawback(goCtx, *va, dest); err != nil {
		return nil, err
	}

	telemetry.IncrCounter(
		float32(ctx.GasMeter().GasConsumed()),
		"tx", "clawback", "gas_used",
	)

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeClawback,
				sdk.NewAttribute(types.AttributeKeyFunder, msg.FunderAddress),
				sdk.NewAttribute(types.AttributeKeyAccount, msg.AccountAddress),
				sdk.NewAttribute(types.AttributeKeyDestination, dest.String()),
			),
		},
	)

	return &types.MsgClawbackResponse{}, nil
}

// UpdateVestingFunder updates the funder account of a ClawbackVestingAccount.
//
// Checks performed on the ValidateBasic include:
//   - new funder and vesting addresses are correct bech32 format
//   - new funder address is not the zero address
//   - new funder address is not the same as the current funder address
func (k Keeper) UpdateVestingFunder(
	goCtx context.Context,
	msg *types.MsgUpdateVestingFunder,
) (*types.MsgUpdateVestingFunderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ak := k.accountKeeper
	bk := k.bankKeeper

	// NOTE: errors checked during msg validation
	newFunder := sdk.MustAccAddressFromBech32(msg.NewFunderAddress)
	vestingAccAddr := sdk.MustAccAddressFromBech32(msg.VestingAddress)

	// Need to check if new funder can receive funds because in
	// Clawback function, destination defaults to funder address
	if bk.BlockedAddr(newFunder) {
		return nil, errorsmod.Wrapf(errortypes.ErrUnauthorized,
			"%s is a blocked address and not allowed to fund vesting accounts", msg.NewFunderAddress,
		)
	}

	// Check if vesting account exists
	va, err := k.GetClawbackVestingAccount(goCtx, vestingAccAddr)
	if err != nil {
		return nil, err
	}

	// Check if current funder is same as in msg
	if va.FunderAddress != msg.FunderAddress {
		return nil, errorsmod.Wrapf(errortypes.ErrUnauthorized, "%s is not the current funder and cannot update the funder address", va.FunderAddress)
	}

	// Perform clawback account update
	va.FunderAddress = msg.NewFunderAddress
	ak.SetAccount(ctx, va)

	telemetry.IncrCounter(
		float32(ctx.GasMeter().GasConsumed()),
		"tx", "update_vesting_funder", "gas_used",
	)

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
// after its lockup and vesting periods have concluded.
func (k Keeper) ConvertVestingAccount(
	goCtx context.Context,
	msg *types.MsgConvertVestingAccount,
) (*types.MsgConvertVestingAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	address := sdk.MustAccAddressFromBech32(msg.VestingAddress)

	vestingAcc, err := k.GetClawbackVestingAccount(goCtx, address)
	if err != nil {
		return nil, err
	}

	// check if account has any vesting coins left
	if !vestingAcc.GetVestingCoins(ctx.BlockTime()).IsZero() {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "vesting coins still left in account: %s", msg.VestingAddress)
	}

	// check if account has any locked up coins left
	if vestingAcc.HasLockedCoins(ctx.BlockTime()) {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "locked up coins still left in account: %s", msg.VestingAddress)
	}

	ethAccount := ethtypes.ProtoAccount().(*ethtypes.EthAccount)
	ethAccount.BaseAccount = vestingAcc.BaseAccount
	ethAccount.SetCodeHash(vestingAcc.GetCodeHash()) //nolint: errcheck // There's always nil error
	k.accountKeeper.SetAccount(goCtx, ethAccount)

	return &types.MsgConvertVestingAccountResponse{}, nil
}

// ConvertIntoVestingAccount converts a default chain account to the ClawbackVestingAccount
// after its lock and vesting periods have started.
func (k Keeper) ConvertIntoVestingAccount(
	goCtx context.Context,
	msg *types.MsgConvertIntoVestingAccount,
) (*types.MsgConvertIntoVestingAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
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

	// Disable clawback vesting account creation for contract accounts
	acc := k.accountKeeper.GetAccount(goCtx, to)
	ethAcc, ok := acc.(*ethtypes.EthAccount)
	if ok {
		if err := utils.IsContractAccount(ethAcc); err == nil {
			return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest,
				"account %s is a contract account and cannot be converted in a clawback vesting account", msg.ToAddress,
			)
		}
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

	vestingAcc, newAccountCreated, wasMerged, err := k.ApplyVestingSchedule(
		goCtx,
		from, to,
		vestingCoins,
		msg.StartTime,
		msg.LockupPeriods, msg.VestingPeriods,
		msg.Merge,
	)
	if err != nil {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to apply schedule: %s", err.Error())
	}

	if newAccountCreated {
		defer func() {
			telemetry.IncrCounter(1, "new", "account")
		}()
	}

	// Send coins from the funder to vesting account
	if err := k.bankKeeper.SendCoins(goCtx, from, to, vestingCoins); err != nil {
		return nil, err
	}

	events := sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateClawbackVestingAccount,
			sdk.NewAttribute(sdk.AttributeKeySender, from.String()),
			sdk.NewAttribute(types.AttributeKeyCoins, vestingCoins.String()),
			sdk.NewAttribute(types.AttributeKeyStartTime, vestingAcc.StartTime.String()),
			sdk.NewAttribute(types.AttributeKeyMerge, strconv.FormatBool(wasMerged)),
			sdk.NewAttribute(types.AttributeKeyAccount, vestingAcc.Address),
		),
		// sdk.NewEvent(
		//	sdk.EventTypeMessage,
		//	sdk.NewAttribute(sdk.AttributeKeyModule, stakingtypes.AttributeValueCategory),
		//	sdk.NewAttribute(sdk.AttributeKeySender, to.String()),
		// ),
	}

	if msg.Stake {
		events, err = k.delegateVestedCoins(goCtx, to, msg, events)
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
	goCtx context.Context,
	va *types.ClawbackVestingAccount,
	grantStartTime int64,
	grantLockupPeriods, grantVestingPeriods sdkvesting.Periods,
	grantCoins sdk.Coins,
) error {
	// how much is really delegated?
	vestingAddr := va.GetAddress()
	bondedAmt, err := k.stakingKeeper.GetDelegatorBonded(goCtx, vestingAddr)
	if err != nil {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to get bonded amount: %e", err)
	}
	unbondingAmt, err := k.stakingKeeper.GetDelegatorUnbonding(goCtx, vestingAddr)
	if err != nil {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to get unbonding amount: %e", err)
	}
	delegatedAmt := bondedAmt.Add(unbondingAmt)

	// modify schedules for the new grant
	accStartTime := va.GetStartTime()
	newLockupStart, newLockupEnd, newLockupPeriods := types.DisjunctPeriods(accStartTime, grantStartTime, va.LockupPeriods, grantLockupPeriods)
	newVestingStart, newVestingEnd, newVestingPeriods := types.DisjunctPeriods(accStartTime, grantStartTime, va.GetVestingPeriods(), grantVestingPeriods)

	if newLockupStart != newVestingStart {
		return errorsmod.Wrapf(
			types.ErrVestingLockup,
			"vesting start time calculation should match lockup start (%d ≠ %d)",
			newVestingStart, newLockupStart,
		)
	}

	va.StartTime = time.Unix(newLockupStart, 0).UTC()
	va.EndTime = types.Max64(newLockupEnd, newVestingEnd)
	va.LockupPeriods = newLockupPeriods
	va.VestingPeriods = newVestingPeriods
	va.OriginalVesting = va.OriginalVesting.Add(grantCoins...)

	// DF rounds out to current delegated
	bondDenom, err := k.stakingKeeper.BondDenom(goCtx)
	if err != nil {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to get bond denom: %e", err)
	}
	va.DelegatedVesting = sdk.NewCoins()
	va.DelegatedFree = sdk.NewCoins(sdk.NewCoin(bondDenom, delegatedAmt))
	return nil
}

// transferClawback transfers unvested tokens in a ClawbackVestingAccount to
// the destination address. Then, it updates the lockup schedule, and removes future
// vesting events.
func (k Keeper) transferClawback(
	goCtx context.Context,
	va types.ClawbackVestingAccount,
	dest sdk.AccAddress,
) error {
	// Compute clawback amount, unlock unvested tokens and remove future vesting events
	ctx := sdk.UnwrapSDKContext(goCtx)
	updatedAcc, toClawBack := va.ComputeClawback(ctx.BlockTime().Unix())
	if toClawBack.IsZero() {
		// no-op, nothing to transfer
		return nil
	}

	// set the account with the updated values of the vesting schedule
	k.accountKeeper.SetAccount(goCtx, &updatedAcc)

	addr := updatedAcc.GetAddress()

	// In case destination is community pool (e.g. Gov Clawback)
	// call the corresponding function
	if dest.String() == k.accountKeeper.GetModuleAddress(distributiontypes.ModuleName).String() {
		return k.distributionKeeper.FundCommunityPool(goCtx, toClawBack, addr)
	}

	// NOTE: don't use `SpendableCoins` to get the minimum value to clawback since
	// the amount is retrieved from `ComputeClawback`, which ensures correctness.
	// `SpendableCoins` can result in gas exhaustion if the user has too many
	// different denoms (because of store iteration).

	// Transfer clawback to the destination (funder)
	return k.bankKeeper.SendCoins(goCtx, addr, dest, toClawBack)
}

func (k Keeper) delegateVestedCoins(goCtx context.Context, delegator sdk.AccAddress, msg *types.MsgConvertIntoVestingAccount, events sdk.Events) (sdk.Events, error) {
	valAddress, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return events, errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid validator address: %s", msg.ValidatorAddress)
	}

	validator, err := k.stakingKeeper.GetValidator(goCtx, valAddress)
	if err != nil {
		return events, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to get validator %s: %e", msg.ValidatorAddress, err)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
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
		return events, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "no vested coins to delegate immediately, check your vesting schedule")
	}

	bondDenom, err := k.stakingKeeper.BondDenom(goCtx)
	if err != nil {
		return events, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "failed to get bond denom: %e", err)
	}
	found, amountToDelegate := vestedCoins.Find(bondDenom)
	if !found || amountToDelegate.IsZero() {
		// Nothing to delegate
		return events, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "coin denom not match bonding denom '%s', check your vesting schedule", bondDenom)
	}

	newShares, err := k.stakingKeeper.Delegate(goCtx, delegator, amountToDelegate.Amount, stakingtypes.Unbonded, validator, true)
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
