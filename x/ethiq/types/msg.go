package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgMintEthiq{}

// GetSigners returns the signers of the message
func (msg *MsgMintEthiq) GetSigners() []sdk.AccAddress {
	fromAddress, _ := sdk.AccAddressFromBech32(msg.FromAddress)
	return []sdk.AccAddress{fromAddress}
}

// ValidateBasic performs basic validation on the message
func (msg *MsgMintEthiq) ValidateBasic() error {
	if msg.EthiqAmount.LTE(sdkmath.OneInt()) {
		return errorsmod.Wrap(ErrInvalidAmount, "ethiq_amount must be positive and greater than 1")
	}

	if msg.MaxIslmAmount.LTE(sdkmath.OneInt()) {
		return errorsmod.Wrap(ErrInvalidAmount, "max_islm_amount must be positive and greater than 1")
	}

	if _, err := sdk.AccAddressFromBech32(msg.ToAddress); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid to_address: %v", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.FromAddress); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid from_address: %v", err)
	}

	return nil
}
