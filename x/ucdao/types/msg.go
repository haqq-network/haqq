package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	proto "github.com/cosmos/gogoproto/proto"
)

const (
	TypeMsgFund                        = "fund_dao"
	TypeMsgTransferOwnership           = "transfer_ownership"
	TypeMsgTransferOwnershipWithRatio  = "transfer_ownership_with_ratio"
	TypeMsgTransferOwnershipWithAmount = "transfer_ownership_with_amount"
)

// Verify interface at compile time
var (
	_ sdk.Msg = (*MsgFund)(nil)
	_ sdk.Msg = (*MsgTransferOwnership)(nil)
)

// NewMsgFund returns a new MsgFund with a sender and
// a funding amount.
func NewMsgFund(amount sdk.Coins, depositor sdk.AccAddress) *MsgFund {
	return &MsgFund{
		Amount:    amount,
		Depositor: depositor.String(),
	}
}

func init() {
	// Register the MsgFundLegacy type with the old name to support history data reading.
	proto.RegisterType((*MsgFundLegacy)(nil), "haqq.dao.v1.MsgFund")
}

// Route returns the MsgFundCommunityPool message route.
func (msg MsgFund) Route() string { return ModuleName }

// Type returns the MsgFundCommunityPool message type.
func (msg MsgFund) Type() string { return TypeMsgFund }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgFund) GetSigners() []sdk.AccAddress {
	depositor, _ := sdk.AccAddressFromBech32(msg.Depositor)
	return []sdk.AccAddress{depositor}
}

// GetSignBytes returns the raw bytes for a MsgFundCommunityPool message that
// the expected signer needs to sign.
func (msg MsgFund) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgFundCommunityPool message validation.
func (msg MsgFund) ValidateBasic() error {
	if !msg.Amount.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}
	if _, err := sdk.AccAddressFromBech32(msg.Depositor); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid depositor address: %s", err)
	}
	return nil
}

// NewMsgFundLegacy returns a new MsgFundLegacy with a sender and
// a funding amount.
func NewMsgFundLegacy(amount sdk.Coins, depositor sdk.AccAddress) *MsgFundLegacy {
	return &MsgFundLegacy{
		Amount:    amount,
		Depositor: depositor.String(),
	}
}

// Route returns the MsgFundCommunityPool message route.
func (msg MsgFundLegacy) Route() string { return ModuleName }

// Type returns the MsgFundCommunityPool message type.
func (msg MsgFundLegacy) Type() string { return TypeMsgFund }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgFundLegacy) GetSigners() []sdk.AccAddress {
	depositor, _ := sdk.AccAddressFromBech32(msg.Depositor)
	return []sdk.AccAddress{depositor}
}

// GetSignBytes returns the raw bytes for a MsgFundCommunityPool message that
// the expected signer needs to sign.
func (msg MsgFundLegacy) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgFundCommunityPool message validation.
func (msg MsgFundLegacy) ValidateBasic() error {
	if !msg.Amount.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}
	if _, err := sdk.AccAddressFromBech32(msg.Depositor); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid depositor address: %s", err)
	}
	return nil
}

func (m *AllowedCollateral) String() string {
	return proto.CompactTextString(m)
}

// NewMsgTransferOwnership returns a new MsgTransferOwnership with an old and new owner addresses.
func NewMsgTransferOwnership(owner, newOwner sdk.AccAddress) *MsgTransferOwnership {
	return &MsgTransferOwnership{
		Owner:    owner.String(),
		NewOwner: newOwner.String(),
	}
}

// Route returns the MsgTransferOwnership message route.
func (msg MsgTransferOwnership) Route() string { return ModuleName }

// Type returns the MsgTransferOwnership message type.
func (msg MsgTransferOwnership) Type() string { return TypeMsgTransferOwnership }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgTransferOwnership) GetSigners() []sdk.AccAddress {
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{owner}
}

// GetSignBytes returns the raw bytes for a MsgTransferOwnership message that
// the expected signer needs to sign.
func (msg MsgTransferOwnership) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgTransferOwnership message validation.
func (msg MsgTransferOwnership) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid owner address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.NewOwner); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid new owner address: %s", err)
	}
	return nil
}

// NewMsgTransferOwnershipWithRatio returns a new MsgTransferOwnershipWithRatio with an old and new owner addresses and ratio.
func NewMsgTransferOwnershipWithRatio(owner, newOwner sdk.AccAddress, ratio math.LegacyDec) *MsgTransferOwnershipWithRatio {
	return &MsgTransferOwnershipWithRatio{
		Owner:    owner.String(),
		NewOwner: newOwner.String(),
		Ratio:    ratio,
	}
}

// Route returns the MsgTransferOwnershipWithRatio message route.
func (msg MsgTransferOwnershipWithRatio) Route() string { return ModuleName }

// Type returns the MsgTransferOwnershipWithRatio message type.
func (msg MsgTransferOwnershipWithRatio) Type() string { return TypeMsgTransferOwnershipWithRatio }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgTransferOwnershipWithRatio) GetSigners() []sdk.AccAddress {
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{owner}
}

// GetSignBytes returns the raw bytes for a MsgTransferOwnershipWithRatio message that
// the expected signer needs to sign.
func (msg MsgTransferOwnershipWithRatio) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgTransferOwnership message validation.
func (msg MsgTransferOwnershipWithRatio) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid owner address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.NewOwner); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid new owner address: %s", err)
	}
	if msg.Ratio.LTE(math.LegacyZeroDec()) || msg.Ratio.GT(math.LegacyOneDec()) {
		return ErrInvalidRatio.Wrapf("expected greater than 0 and less or equal than 1, got %s", msg.Ratio)
	}
	return nil
}

// NewMsgTransferOwnershipWithAmount returns a new MsgTransferOwnershipWithAmount with an old and new owner addresses and ratio.
func NewMsgTransferOwnershipWithAmount(owner, newOwner sdk.AccAddress, amount sdk.Coins) *MsgTransferOwnershipWithAmount {
	return &MsgTransferOwnershipWithAmount{
		Owner:    owner.String(),
		NewOwner: newOwner.String(),
		Amount:   amount,
	}
}

// Route returns the MsgTransferOwnershipWithRatio message route.
func (msg MsgTransferOwnershipWithAmount) Route() string { return ModuleName }

// Type returns the MsgTransferOwnershipWithRatio message type.
func (msg MsgTransferOwnershipWithAmount) Type() string { return TypeMsgTransferOwnershipWithAmount }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgTransferOwnershipWithAmount) GetSigners() []sdk.AccAddress {
	owner, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{owner}
}

// GetSignBytes returns the raw bytes for a MsgTransferOwnershipWithRatio message that
// the expected signer needs to sign.
func (msg MsgTransferOwnershipWithAmount) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgTransferOwnership message validation.
func (msg MsgTransferOwnershipWithAmount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid owner address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.NewOwner); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid new owner address: %s", err)
	}
	if !msg.Amount.IsValid() {
		return sdkerrors.ErrInvalidCoins.Wrapf("invalid amount: %s", msg.Amount)
	}
	return nil
}
