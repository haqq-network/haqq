package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TypeMsgMint = "mint"
)

// NewMsgMint creates new instance of MsgMint
func NewMsgMint(name string) *MsgMint {
	return &MsgMint{
		Name: name,
	}
}

// Route returns the name of the module
func (msg MsgMint) Route() string { return RouterKey }

// Type returns the action type
func (msg MsgMint) Type() string { return TypeMsgMint }

// GetSignBytes encodes the message for signing
func (msg *MsgMint) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(msg))
}

// ValidateBasic runs stateless checks on the message
func (msg MsgMint) ValidateBasic() error {
	return nil
}

// GetSigners defines whose signature is required
func (msg MsgMint) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{}
}
