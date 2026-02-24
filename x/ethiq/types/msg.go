package types

import (
	"math"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgMintHaqq{}
	_ sdk.Msg = &MsgMintHaqqByApplication{}
)

// GetSigners returns the signers of the message
func (msg *MsgMintHaqq) GetSigners() []sdk.AccAddress {
	fromAddress, _ := sdk.AccAddressFromBech32(msg.FromAddress)
	return []sdk.AccAddress{fromAddress}
}

// ValidateBasic performs basic validation on the message
func (msg *MsgMintHaqq) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ToAddress); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid to_address: %v", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.FromAddress); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid from_address: %v", err)
	}

	if msg.IslmAmount.LT(sdkmath.OneInt()) {
		return errorsmod.Wrap(ErrInvalidAmount, "islm_amount must be positive and greater than zero")
	}

	return nil
}

// GetSigners returns the signers of the message
func (msg *MsgMintHaqqByApplication) GetSigners() []sdk.AccAddress {
	fromAddress, _ := sdk.AccAddressFromBech32(msg.FromAddress)
	return []sdk.AccAddress{fromAddress}
}

// ValidateBasic performs basic validation on the message
func (msg *MsgMintHaqqByApplication) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.FromAddress); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid from_address: %v", err)
	}

	if msg.ApplicationId > math.MaxUint64 {
		return errorsmod.Wrapf(ErrInvalidApplicationID, "uint64 overflow; application_id: %d", msg.ApplicationId)
	}

	return nil
}
