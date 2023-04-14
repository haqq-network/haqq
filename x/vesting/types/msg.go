package types

import (
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/common"
)

var (
	_ sdk.Msg = &MsgCreateClawbackVestingAccount{}
	_ sdk.Msg = &MsgClawback{}
	_ sdk.Msg = &MsgConvertVestingAccount{}
	_ sdk.Msg = &MsgUpdateVestingFunder{}
	_ sdk.Msg = &MsgConvertIntoVestingAccount{}
	_ sdk.Msg = &MsgUpdateVestingSchedule{}
)

const (
	TypeMsgCreateClawbackVestingAccount = "create_clawback_vesting_account"
	TypeMsgClawback                     = "clawback"
	TypeMsgUpdateVestingFunder          = "update_vesting_funder"
	TypeMsgConvertVestingAccount        = "convert_vesting_account"
	TypeMsgConvertIntoVestingAccount    = "convert_into_vesting_account"
	TypeMsgUpdateVestingSchedule        = "update_vesting_schedule"
)

// NewMsgCreateClawbackVestingAccount creates new instance of MsgCreateClawbackVestingAccount
func NewMsgCreateClawbackVestingAccount(
	fromAddr, toAddr sdk.AccAddress,
	startTime time.Time,
	lockupPeriods,
	vestingPeriods sdkvesting.Periods,
	merge bool,
) *MsgCreateClawbackVestingAccount {
	return &MsgCreateClawbackVestingAccount{
		FromAddress:    fromAddr.String(),
		ToAddress:      toAddr.String(),
		StartTime:      startTime,
		LockupPeriods:  lockupPeriods,
		VestingPeriods: vestingPeriods,
		Merge:          merge,
	}
}

// Route returns the name of the module
func (msg MsgCreateClawbackVestingAccount) Route() string { return RouterKey }

// Type returns the the action
func (msg MsgCreateClawbackVestingAccount) Type() string { return TypeMsgCreateClawbackVestingAccount }

// ValidateBasic runs stateless checks on the message
func (msg MsgCreateClawbackVestingAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.FromAddress); err != nil {
		return errorsmod.Wrapf(err, "invalid from address")
	}

	if _, err := sdk.AccAddressFromBech32(msg.ToAddress); err != nil {
		return errorsmod.Wrapf(err, "invalid to address")
	}

	lockupCoins := sdk.NewCoins()
	for i, period := range msg.LockupPeriods {
		if period.Length < 1 {
			return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "invalid period length of %d in period %d, length must be greater than 0", period.Length, i)
		}
		lockupCoins = lockupCoins.Add(period.Amount...)
	}

	vestingCoins := sdk.NewCoins()
	for i, period := range msg.VestingPeriods {
		if period.Length < 1 {
			return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "invalid period length of %d in period %d, length must be greater than 0", period.Length, i)
		}
		vestingCoins = vestingCoins.Add(period.Amount...)
	}

	// If both schedules are present, the must describe the same total amount.
	// IsEqual can panic, so use (a == b) <=> (a <= b && b <= a).
	if len(msg.LockupPeriods) > 0 && len(msg.VestingPeriods) > 0 &&
		!(lockupCoins.IsAllLTE(vestingCoins) && vestingCoins.IsAllLTE(lockupCoins)) {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "vesting and lockup schedules must have same total coins")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgCreateClawbackVestingAccount) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgCreateClawbackVestingAccount) GetSigners() []sdk.AccAddress {
	from := sdk.MustAccAddressFromBech32(msg.FromAddress)
	return []sdk.AccAddress{from}
}

// NewMsgClawback creates new instance of MsgClawback. The dest address may be
// nil - defaulting to the funder.
func NewMsgClawback(funder, addr, dest sdk.AccAddress) *MsgClawback {
	destString := ""
	if dest != nil {
		destString = dest.String()
	}
	return &MsgClawback{
		FunderAddress:  funder.String(),
		AccountAddress: addr.String(),
		DestAddress:    destString,
	}
}

// Route returns the message route for a MsgClawback.
func (msg MsgClawback) Route() string { return RouterKey }

// Type returns the message type for a MsgClawback.
func (msg MsgClawback) Type() string { return TypeMsgClawback }

// ValidateBasic runs stateless checks on the message
func (msg MsgClawback) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.GetFunderAddress()); err != nil {
		return errorsmod.Wrapf(err, "invalid funder address")
	}

	if _, err := sdk.AccAddressFromBech32(msg.GetAccountAddress()); err != nil {
		return errorsmod.Wrapf(err, "invalid account address")
	}

	if msg.GetDestAddress() != "" {
		if _, err := sdk.AccAddressFromBech32(msg.GetDestAddress()); err != nil {
			return errorsmod.Wrapf(err, "invalid dest address")
		}
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgClawback) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgClawback) GetSigners() []sdk.AccAddress {
	funder := sdk.MustAccAddressFromBech32(msg.FunderAddress)
	return []sdk.AccAddress{funder}
}

// NewMsgUpdateVestingFunder creates new instance of MsgUpdateVestingFunder
func NewMsgUpdateVestingFunder(funder, newFunder, vesting sdk.AccAddress) *MsgUpdateVestingFunder {
	return &MsgUpdateVestingFunder{
		FunderAddress:    funder.String(),
		NewFunderAddress: newFunder.String(),
		VestingAddress:   vesting.String(),
	}
}

// Route returns the message route for a MsgUpdateVestingFunder.
func (msg MsgUpdateVestingFunder) Route() string { return RouterKey }

// Type returns the message type for a MsgUpdateVestingFunder.
func (msg MsgUpdateVestingFunder) Type() string { return TypeMsgUpdateVestingFunder }

// ValidateBasic runs stateless checks on the MsgUpdateVestingFunder message
func (msg MsgUpdateVestingFunder) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.GetFunderAddress()); err != nil {
		return errorsmod.Wrapf(err, "invalid funder address")
	}

	if _, err := sdk.AccAddressFromBech32(msg.GetNewFunderAddress()); err != nil {
		return errorsmod.Wrapf(err, "invalid new funder address")
	}

	// New funder address can not be equal to current funder address
	if msg.FunderAddress == msg.NewFunderAddress {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "new funder address is equal to current funder address")
	}

	if _, err := sdk.AccAddressFromBech32(msg.GetVestingAddress()); err != nil {
		return errorsmod.Wrapf(err, "invalid vesting account address")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgUpdateVestingFunder) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgUpdateVestingFunder) GetSigners() []sdk.AccAddress {
	funder := sdk.MustAccAddressFromBech32(msg.FunderAddress)
	return []sdk.AccAddress{funder}
}

// NewMsgConvertVestingAccount creates new instance of MsgConvertVestingAccount
func NewMsgConvertVestingAccount(vestingAcc sdk.AccAddress) *MsgConvertVestingAccount {
	return &MsgConvertVestingAccount{
		VestingAddress: vestingAcc.String(),
	}
}

// Route returns the name of the module
func (msg MsgConvertVestingAccount) Route() string { return RouterKey }

// Type returns the action
func (msg MsgConvertVestingAccount) Type() string { return TypeMsgConvertVestingAccount }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertVestingAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.GetVestingAddress()); err != nil {
		return errorsmod.Wrapf(err, "invalid vesting address")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgConvertVestingAccount) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertVestingAccount) GetSigners() []sdk.AccAddress {
	vesting := sdk.MustAccAddressFromBech32(msg.VestingAddress)
	return []sdk.AccAddress{vesting}
}

// NewMsgConvertIntoVestingAccount creates new instance of MsgConvertIntoVestingAccount
func NewMsgConvertIntoVestingAccount(
	funder, address sdk.AccAddress,
	startTime time.Time,
	amount sdk.Coin,
	longTerm bool,
) *MsgConvertIntoVestingAccount {
	return &MsgConvertIntoVestingAccount{
		FromAddress: funder.String(),
		EthAddress:  common.Bytes2Hex(address.Bytes()),
		StartTime:   startTime,
		Amount:      amount,
		LongTerm:    longTerm,
	}
}

// Route returns the name of the module
func (msg MsgConvertIntoVestingAccount) Route() string { return RouterKey }

// Type returns the action
func (msg MsgConvertIntoVestingAccount) Type() string { return TypeMsgConvertIntoVestingAccount }

// ValidateBasic runs stateless checks on the message
func (msg MsgConvertIntoVestingAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.FromAddress); err != nil {
		return errorsmod.Wrapf(err, "invalid from address")
	}

	if ok := common.IsHexAddress(msg.EthAddress); !ok {
		return errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid eth address")
	}

	coins := sdk.NewCoins()
	coins = coins.Add(msg.Amount)
	zeroCoins := sdk.NewCoins()

	// IsEqual can panic, so use (a == b) <=> (a <= b && b <= a).
	if coins.IsAllLTE(zeroCoins) && zeroCoins.IsAllLTE(coins) {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "amount is not specified")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgConvertIntoVestingAccount) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConvertIntoVestingAccount) GetSigners() []sdk.AccAddress {
	from := sdk.MustAccAddressFromBech32(msg.FromAddress)
	return []sdk.AccAddress{from}
}

// NewMsgUpdateVestingSchedule creates new instance of MsgUpdateVestingSchedule
func NewMsgUpdateVestingSchedule(
	funder, vesting sdk.AccAddress,
	startTime time.Time,
	lockupPeriods,
	vestingPeriods sdkvesting.Periods,
) *MsgUpdateVestingSchedule {
	return &MsgUpdateVestingSchedule{
		FunderAddress:  funder.String(),
		VestingAddress: vesting.String(),
		StartTime:      startTime,
		LockupPeriods:  lockupPeriods,
		VestingPeriods: vestingPeriods,
	}
}

// Route returns the message route for a MsgUpdateVestingFunder.
func (msg MsgUpdateVestingSchedule) Route() string { return RouterKey }

// Type returns the message type for a MsgUpdateVestingFunder.
func (msg MsgUpdateVestingSchedule) Type() string { return TypeMsgUpdateVestingSchedule }

// ValidateBasic runs stateless checks on the MsgUpdateVestingFunder message
func (msg MsgUpdateVestingSchedule) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.GetVestingAddress()); err != nil {
		return errorsmod.Wrapf(err, "invalid vesting account address")
	}

	lockupCoins := sdk.NewCoins()
	for i, period := range msg.LockupPeriods {
		if period.Length < 1 {
			return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "invalid period length of %d in period %d, length must be greater than 0", period.Length, i)
		}
		lockupCoins = lockupCoins.Add(period.Amount...)
	}

	vestingCoins := sdk.NewCoins()
	for i, period := range msg.VestingPeriods {
		if period.Length < 1 {
			return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "invalid period length of %d in period %d, length must be greater than 0", period.Length, i)
		}
		vestingCoins = vestingCoins.Add(period.Amount...)
	}

	// If both schedules are present, the must describe the same total amount.
	// IsEqual can panic, so use (a == b) <=> (a <= b && b <= a).
	if len(msg.LockupPeriods) > 0 && len(msg.VestingPeriods) > 0 &&
		!(lockupCoins.IsAllLTE(vestingCoins) && vestingCoins.IsAllLTE(lockupCoins)) {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "vesting and lockup schedules must have same total coins")
	}

	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgUpdateVestingSchedule) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgUpdateVestingSchedule) GetSigners() []sdk.AccAddress {
	funder := sdk.MustAccAddressFromBech32(msg.FunderAddress)
	return []sdk.AccAddress{funder}
}
