package types

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type MsgTestSuite struct {
	suite.Suite
}

func TestMsgSuite(t *testing.T) {
	suite.Run(t, new(MsgTestSuite))
}

func (suite *MsgTestSuite) TestNewMsgLiquidate() {
	from := sdk.AccAddress([]byte("liquidatefrom_______"))
	to := sdk.AccAddress([]byte("liquidateto_________"))
	amount := sdk.NewCoin("aISLM", sdkmath.NewInt(1000))

	msg := NewMsgLiquidate(from, to, amount)

	suite.Require().NotNil(msg)
	suite.Require().Equal(from.String(), msg.LiquidateFrom)
	suite.Require().Equal(to.String(), msg.LiquidateTo)
	suite.Require().Equal(amount, msg.Amount)
}

func (suite *MsgTestSuite) TestNewMsgRedeem() {
	from := sdk.AccAddress([]byte("redeemfrom__________"))
	to := sdk.AccAddress([]byte("redeemto____________"))
	amount := sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500))

	msg := NewMsgRedeem(from, to, amount)

	suite.Require().NotNil(msg)
	suite.Require().Equal(from.String(), msg.RedeemFrom)
	suite.Require().Equal(to.String(), msg.RedeemTo)
	suite.Require().Equal(amount, msg.Amount)
}

func (suite *MsgTestSuite) TestMsgLiquidateRoute() {
	from := sdk.AccAddress([]byte("liquidatefrom_______"))
	to := sdk.AccAddress([]byte("liquidateto_________"))
	msg := NewMsgLiquidate(from, to, sdk.NewCoin("aISLM", sdkmath.NewInt(100)))
	suite.Require().Equal("liquidvesting", msg.Route())
}

func (suite *MsgTestSuite) TestMsgLiquidateType() {
	from := sdk.AccAddress([]byte("liquidatefrom_______"))
	to := sdk.AccAddress([]byte("liquidateto_________"))
	msg := NewMsgLiquidate(from, to, sdk.NewCoin("aISLM", sdkmath.NewInt(100)))
	suite.Require().Equal("liquidate", msg.Type())
}

func (suite *MsgTestSuite) TestMsgLiquidateGetSigners() {
	from := sdk.AccAddress([]byte("liquidatefrom_______"))
	to := sdk.AccAddress([]byte("liquidateto_________"))
	msg := NewMsgLiquidate(from, to, sdk.NewCoin("aISLM", sdkmath.NewInt(100)))
	signers := msg.GetSigners()
	suite.Require().Len(signers, 1)
	suite.Require().Equal(from, signers[0])
}

func (suite *MsgTestSuite) TestMsgRedeemRoute() {
	from := sdk.AccAddress([]byte("redeemfrom__________"))
	to := sdk.AccAddress([]byte("redeemto____________"))
	msg := NewMsgRedeem(from, to, sdk.NewCoin("aLIQUID1", sdkmath.NewInt(100)))
	suite.Require().Equal("liquidvesting", msg.Route())
}

func (suite *MsgTestSuite) TestMsgRedeemType() {
	from := sdk.AccAddress([]byte("redeemfrom__________"))
	to := sdk.AccAddress([]byte("redeemto____________"))
	msg := NewMsgRedeem(from, to, sdk.NewCoin("aLIQUID1", sdkmath.NewInt(100)))
	suite.Require().Equal("redeem", msg.Type())
}

func (suite *MsgTestSuite) TestMsgRedeemGetSigners() {
	from := sdk.AccAddress([]byte("redeemfrom__________"))
	to := sdk.AccAddress([]byte("redeemto____________"))
	msg := NewMsgRedeem(from, to, sdk.NewCoin("aLIQUID1", sdkmath.NewInt(100)))
	signers := msg.GetSigners()
	suite.Require().Len(signers, 1)
	suite.Require().Equal(from, signers[0])
}

func (suite *MsgTestSuite) TestMsgLiquidate() {
	validFrom := sdk.AccAddress([]byte("liquidatefrom_______"))
	validTo := sdk.AccAddress([]byte("liquidateto_________"))

	testCases := []struct {
		name        string
		msg         MsgLiquidate
		expectError bool
	}{
		{
			name: "valid message",
			msg: MsgLiquidate{
				LiquidateFrom: validFrom.String(),
				LiquidateTo:   validTo.String(),
				Amount:        sdk.NewCoin("aISLM", sdkmath.NewInt(1000)),
			},
			expectError: false,
		},
		{
			name: "zero amount",
			msg: MsgLiquidate{
				LiquidateFrom: validFrom.String(),
				LiquidateTo:   validTo.String(),
				Amount:        sdk.NewCoin("aISLM", sdkmath.NewInt(0)),
			},
			expectError: true,
		},
		{
			name: "negative amount",
			msg: MsgLiquidate{
				LiquidateFrom: validFrom.String(),
				LiquidateTo:   validTo.String(),
				Amount:        sdk.Coin{Denom: "aISLM", Amount: sdkmath.NewInt(-1)},
			},
			expectError: true,
		},
		{
			name: "invalid from address",
			msg: MsgLiquidate{
				LiquidateFrom: "not-a-valid-bech32",
				LiquidateTo:   validTo.String(),
				Amount:        sdk.NewCoin("aISLM", sdkmath.NewInt(1000)),
			},
			expectError: true,
		},
		{
			name: "invalid to address",
			msg: MsgLiquidate{
				LiquidateFrom: validFrom.String(),
				LiquidateTo:   "not-a-valid-bech32",
				Amount:        sdk.NewCoin("aISLM", sdkmath.NewInt(1000)),
			},
			expectError: true,
		},
		{
			name: "empty from address",
			msg: MsgLiquidate{
				LiquidateFrom: "",
				LiquidateTo:   validTo.String(),
				Amount:        sdk.NewCoin("aISLM", sdkmath.NewInt(1000)),
			},
			expectError: true,
		},
		{
			name: "empty to address",
			msg: MsgLiquidate{
				LiquidateFrom: validFrom.String(),
				LiquidateTo:   "",
				Amount:        sdk.NewCoin("aISLM", sdkmath.NewInt(1000)),
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()
			if tc.expectError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *MsgTestSuite) TestMsgLiquidateGetSignBytes() {
	from := sdk.AccAddress([]byte("liquidatefrom_______"))
	to := sdk.AccAddress([]byte("liquidateto_________"))
	msg := NewMsgLiquidate(from, to, sdk.NewCoin("aISLM", sdkmath.NewInt(100)))
	bz := msg.GetSignBytes()
	suite.Require().NotEmpty(bz)
}

func (suite *MsgTestSuite) TestMsgRedeemGetSignBytes() {
	from := sdk.AccAddress([]byte("redeemfrom__________"))
	to := sdk.AccAddress([]byte("redeemto____________"))
	msg := NewMsgRedeem(from, to, sdk.NewCoin("aLIQUID1", sdkmath.NewInt(100)))
	bz := msg.GetSignBytes()
	suite.Require().NotEmpty(bz)
}

func (suite *MsgTestSuite) TestMsgRedeem() {
	validFrom := sdk.AccAddress([]byte("redeemfrom__________"))
	validTo := sdk.AccAddress([]byte("redeemto____________"))

	testCases := []struct {
		name        string
		msg         MsgRedeem
		expectError bool
	}{
		{
			name: "valid message",
			msg: MsgRedeem{
				RedeemFrom: validFrom.String(),
				RedeemTo:   validTo.String(),
				Amount:     sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500)),
			},
			expectError: false,
		},
		{
			name: "zero amount",
			msg: MsgRedeem{
				RedeemFrom: validFrom.String(),
				RedeemTo:   validTo.String(),
				Amount:     sdk.NewCoin("aLIQUID1", sdkmath.NewInt(0)),
			},
			expectError: true,
		},
		{
			name: "negative amount",
			msg: MsgRedeem{
				RedeemFrom: validFrom.String(),
				RedeemTo:   validTo.String(),
				Amount:     sdk.Coin{Denom: "aLIQUID1", Amount: sdkmath.NewInt(-1)},
			},
			expectError: true,
		},
		{
			name: "invalid from address",
			msg: MsgRedeem{
				RedeemFrom: "not-a-valid-bech32",
				RedeemTo:   validTo.String(),
				Amount:     sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500)),
			},
			expectError: true,
		},
		{
			name: "invalid to address",
			msg: MsgRedeem{
				RedeemFrom: validFrom.String(),
				RedeemTo:   "not-a-valid-bech32",
				Amount:     sdk.NewCoin("aLIQUID1", sdkmath.NewInt(500)),
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()
			if tc.expectError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
