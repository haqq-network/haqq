package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgLiquidate = "liquidate"

// NewMsgLiquidate creates new instance of MsgLiquidate
func NewMsgLiquidate(address sdk.AccAddress, amount sdk.Coin) *MsgLiquidate {
	return &MsgLiquidate{
		Address: address.String(),
		Amount:  amount,
	}
}

// Route returns the name of the module
func (msg MsgLiquidate) Route() string { return RouterKey }

// Type returns the the action
func (msg MsgLiquidate) Type() string { return TypeMsgLiquidate }

// ValidateBasic runs stateless checks on the message
func (msg MsgLiquidate) ValidateBasic() error {
	if !msg.Amount.Amount.IsPositive() {
		return errorsmod.Wrapf(errortypes.ErrInvalidCoins, "cannot liquidate non-positive amount")
	}
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return errorsmod.Wrap(err, "invalid account address address")
	}
	return nil
}

// GetSigners defines whose signature is required
func (msg MsgLiquidate) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}
