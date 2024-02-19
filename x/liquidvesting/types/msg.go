package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgLiquidate = "liquidate"
	TypeMsgRedeem    = "redeem"
)

// NewMsgLiquidate creates new instance of MsgLiquidate
func NewMsgLiquidate(liquidateFrom, liquidateTo sdk.AccAddress, amount sdk.Coin) *MsgLiquidate {
	return &MsgLiquidate{
		LiquidateFrom: liquidateFrom.String(),
		LiquidateTo:   liquidateTo.String(),
		Amount:        amount,
	}
}

// Route returns the name of the module
func (msg MsgLiquidate) Route() string { return RouterKey }

// Type returns the action type
func (msg MsgLiquidate) Type() string { return TypeMsgLiquidate }

// GetSignBytes encodes the message for signing
func (msg *MsgLiquidate) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(msg))
}

// ValidateBasic runs stateless checks on the message
func (msg MsgLiquidate) ValidateBasic() error {
	if !msg.Amount.Amount.IsPositive() {
		return errorsmod.Wrapf(errortypes.ErrInvalidCoins, "cannot liquidate non-positive amount")
	}

	_, err := sdk.AccAddressFromBech32(msg.LiquidateFrom)
	if err != nil {
		return errorsmod.Wrap(err, "invalid account address liquidateFrom")
	}

	_, err = sdk.AccAddressFromBech32(msg.LiquidateTo)
	if err != nil {
		return errorsmod.Wrap(err, "invalid account address liquidateTo")
	}
	return nil
}

// GetSigners defines whose signature is required
func (msg MsgLiquidate) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(msg.LiquidateFrom)
	return []sdk.AccAddress{addr}
}

// NewMsgRedeem creates new instance of MsgLiquidate
func NewMsgRedeem(redeemFrom, redeemTo sdk.AccAddress, amount sdk.Coin) *MsgRedeem {
	return &MsgRedeem{
		RedeemFrom: redeemFrom.String(),
		RedeemTo:   redeemTo.String(),
		Amount:     amount,
	}
}

// Route returns the name of the module
func (msg MsgRedeem) Route() string { return RouterKey }

// Type returns the action type
func (msg MsgRedeem) Type() string { return TypeMsgRedeem }

// GetSignBytes encodes the message for signing
func (msg *MsgRedeem) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(msg))
}

// ValidateBasic runs stateless checks on the message
func (msg MsgRedeem) ValidateBasic() error {
	if !msg.Amount.Amount.IsPositive() {
		return errorsmod.Wrapf(errortypes.ErrInvalidCoins, "cannot liquidate non-positive amount")
	}

	_, err := sdk.AccAddressFromBech32(msg.RedeemFrom)
	if err != nil {
		return errorsmod.Wrap(err, "invalid account address redeemFrom")
	}

	_, err = sdk.AccAddressFromBech32(msg.RedeemTo)
	if err != nil {
		return errorsmod.Wrap(err, "invalid account address redeemTo")
	}
	return nil
}

// GetSigners defines whose signature is required
func (msg MsgRedeem) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(msg.RedeemFrom)
	return []sdk.AccAddress{addr}
}
