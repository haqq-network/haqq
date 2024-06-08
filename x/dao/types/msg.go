package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	proto "github.com/cosmos/gogoproto/proto"
)

const (
	TypeMsgFund = "fund_dao"
)

// Verify interface at compile time
var (
	_ sdk.Msg = (*MsgFund)(nil)
)

// NewMsgFund returns a new MsgFund with a sender and
// a funding amount.
func NewMsgFund(amount sdk.Coins, depositor sdk.AccAddress) *MsgFund {
	return &MsgFund{
		Amount:    amount,
		Depositor: depositor.String(),
	}
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

func (m *AllowedCollateral) String() string {
	return proto.CompactTextString(m)
}
