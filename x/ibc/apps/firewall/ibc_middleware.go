package firewall

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v3/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	utils "github.com/haqq-network/haqq/types"
)

var _ porttypes.Middleware = &IBCMiddleware{}

// IBCMiddleware implements the ICS26 callbacks for the fee middleware given the
// firewall keeper and the underlying application.
type IBCMiddleware struct {
	app         porttypes.IBCModule
	ics4Wrapper porttypes.ICS4Wrapper
}

// NewIBCMiddleware creates new IBCMiddleware with given ICS4 wrapper and underlying application
func NewIBCMiddleware(app porttypes.IBCModule, w porttypes.ICS4Wrapper) IBCMiddleware {
	return IBCMiddleware{
		app:         app,
		ics4Wrapper: w,
	}
}

// NewICS4Wrapper creates new IBCMiddleware with given underlying ICS4 wrapper
func NewICS4Wrapper(w porttypes.ICS4Wrapper) IBCMiddleware {
	return IBCMiddleware{
		ics4Wrapper: w,
	}
}

// IsAllowedAddress checks if given address is allowed to make IBC transfers
//
// Allow only team addresses to ibc-transfer coins outside the Haqq network.
//
// This is required until listing happens so that:
// * team is able to transfer ISLM to Gravity Bridge and create a ERC-20 wrap
//   for listing on exchanges which we agreed to list ERC-20 token
// * presale participants and partners won't be able to transfer ISLM
//   to a network with AMMs or DEXes and define a price before the official listing
//
// After the official listing this restriction will be removed.
func (im IBCMiddleware) IsAllowedAddress(chainID, addr string) bool {
	var wl map[string]bool

	if utils.IsMainNetwork(chainID) {
		wl = map[string]bool{
			// Put here MainNet addresses
			"haqq1uu7epkq75j2qzqvlyzfkljc8h277gz7kxqah0v": true,
		}
	}

	if utils.IsTestEdgeNetwork(chainID) {
		wl = map[string]bool{
			// Put here TestEdge addresses
			"haqq1dz25tp2llzus5mpy0h5nzxzw8r233r8egsjr5v": true,
			"haqq1zmh0d60prm7sqjpayurnsnlw85xrpy73av48ak": true,
			"haqq1zqc0juh5psyek9d4asc828fs3k48ed5xaje3lj": true,
			"haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm": true,
		}
	}

	// By default, all addresses are not allowed
	return wl[addr]
}

// OnChanOpenInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) error {
	// call underlying app's OnChanOpenInit callback with the counterparty app version.
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version)
}

// OnChanOpenTry implements the IBCMiddleware interface
// If the channel is not fee enabled the underlying application version will be returned
// If the channel is fee enabled we merge the underlying application version with the ics29 version
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	// call underlying app's OnChanOpenTry callback with the counterparty app version.
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	// call underlying app's OnChanOpenAck callback with the counterparty app version.
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// call underlying app's OnChanOpenConfirm callback.
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// call underlying app's OnChanCloseInit callback.
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// call underlying app's OnChanCloseConfirm callback.
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCMiddleware interface.
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) exported.Acknowledgement {
	logger := ctx.Logger()
	logger.Debug("run OnRecvPacket from IBC Firewall Middleware")

	// Allow all IBC packets by default, restrict for MainNet and TestEdge
	if !utils.IsMainNetwork(ctx.ChainID()) && !utils.IsTestEdgeNetwork(ctx.ChainID()) {
		return im.app.OnRecvPacket(ctx, packet, relayer)
	}

	// Decode packet data and check if receiver is eligible to use IBC transfer
	var data types.FungibleTokenPacketData
	var ackErr error
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		ackErr = sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "cannot unmarshal ICS-20 transfer packet data")
		return channeltypes.NewErrorAcknowledgement(ackErr.Error())
	}

	logger.Debug("OnRecvPacket -> check allowance of receiver address ", data.Receiver)
	if !im.IsAllowedAddress(ctx.ChainID(), data.Receiver) {
		logger.Debug("OnRecvPacket -> address NOT ALLOWED to receive IBC Transfers", data.Receiver)
		err := sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", data.Receiver)
		return channeltypes.NewErrorAcknowledgement(err.Error())
	}

	logger.Debug("OnRecvPacket -> address ALLOWED to receive IBC Transfers. Proceed!", data.Receiver)
	// call underlying app's OnRecvPacket callback.
	return im.app.OnRecvPacket(ctx, packet, relayer)
}

// OnAcknowledgementPacket implements the IBCMiddleware interface
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	// call underlying app's OnAcknowledgementPacket callback.
	return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	// call underlying app's OnTimeoutPacket callback
	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}

// SendPacket implements the ICS4 Wrapper interface
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
) error {
	logger := ctx.Logger()
	logger.Debug("run SendPacket from IBC Firewall Middleware")

	// Allow all IBC packets by default, restrict for MainNet and TestEdge
	if !utils.IsMainNetwork(ctx.ChainID()) && !utils.IsTestEdgeNetwork(ctx.ChainID()) {
		return im.ics4Wrapper.SendPacket(ctx, chanCap, packet)
	}

	// Decode packet data and check if receiver is eligible to use IBC transfer
	var data types.FungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "cannot unmarshal ICS-20 transfer packet data")
	}

	logger.Debug("SendPacket -> check allowance of sender address ", data.Receiver)
	if !im.IsAllowedAddress(ctx.ChainID(), data.Sender) {
		logger.Debug("SendPacket -> address NOT ALLOWED to send IBC Transfers", data.Sender)
		return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to send funds", data.Sender)
	}

	logger.Debug("SendPacket -> address ALLOWED to send IBC Transfers. Proceed!", data.Sender)
	// call underlying ICS4 Wrapper's SendPacket callback.
	return im.ics4Wrapper.SendPacket(ctx, chanCap, packet)
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
	ack exported.Acknowledgement,
) error {
	// call underlying ICS4 Wrapper's WriteAcknowledgement callback
	return im.ics4Wrapper.WriteAcknowledgement(ctx, chanCap, packet, ack)
}
