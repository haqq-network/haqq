// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package v2

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"

	"github.com/haqq-network/haqq/x/erc20/keeper"
)

var _ ibcapi.IBCModule = &IBCMiddleware{}

// IBCMiddleware implements the ICS26 callbacks for the transfer middleware given
// the erc20 keeper and the underlying application for IBC v2.
type IBCMiddleware struct {
	app    ibcapi.IBCModule
	keeper keeper.Keeper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application
func NewIBCMiddleware(k keeper.Keeper, app ibcapi.IBCModule) *IBCMiddleware {
	return &IBCMiddleware{
		app:    app,
		keeper: k,
	}
}

// OnSendPacket implements the IBCModule interface.
func (im *IBCMiddleware) OnSendPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	signer sdk.AccAddress,
) error {
	return im.app.OnSendPacket(ctx, sourceClient, destinationClient, sequence, payload, signer)
}

// OnRecvPacket implements the IBCModule interface.
// It receives the tokens through the default ICS20 OnRecvPacket callback logic
// and then automatically converts the Cosmos Coin to their ERC20 token
// representation.
// If the acknowledgement fails, this callback will default to the ibc-core
// packet callback.
func (im *IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) channeltypesv2.RecvPacketResult {
	// Call the underlying app's OnRecvPacket
	result := im.app.OnRecvPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)

	// If the result status is not success, return early
	if result.Status != channeltypesv2.PacketStatus_Success {
		return result
	}

	// Extract the acknowledgement from the result
	ackBytes := result.Acknowledgement
	if len(ackBytes) == 0 {
		return result
	}

	// Unmarshal the acknowledgement to check if it's successful
	var ack channeltypes.Acknowledgement
	if err := transfertypes.ModuleCdc.UnmarshalJSON(ackBytes, &ack); err != nil {
		// If we can't unmarshal, return the result as-is
		return result
	}

	// If the acknowledgement is not successful, return early
	if !ack.Success() {
		return result
	}

	// Process ERC20 conversion using v2 payload data directly
	// Use the v2-specific keeper method that works with payloads
	updatedAck := im.keeper.OnRecvPacketV2(ctx, payload.SourcePort, payload.DestinationPort, payload, ack)

	// If keeper returned a different acknowledgement, we need to update the result
	// The keeper's OnRecvPacket returns exported.Acknowledgement, which is typically
	// a channeltypes.Acknowledgement. We need to check if it changed from the original.
	if updatedAck != nil && updatedAck != ack {
		// Type assert to channeltypes.Acknowledgement to marshal it
		if channelAck, ok := updatedAck.(channeltypes.Acknowledgement); ok {
			// Convert the acknowledgement to bytes using the transfer module codec
			updatedAckBytes, err := transfertypes.ModuleCdc.MarshalJSON(&channelAck)
			if err != nil {
				// If marshaling fails, return the original result
				return result
			}

			// Determine the status based on the acknowledgement
			status := channeltypesv2.PacketStatus_Success
			if !updatedAck.Success() {
				status = channeltypesv2.PacketStatus_Failure
			}

			return channeltypesv2.RecvPacketResult{
				Status:          status,
				Acknowledgement: updatedAckBytes,
			}
		}
	}

	// Return the original result if no changes
	return result
}

// OnAcknowledgementPacket implements the IBCModule interface.
// It refunds the token transferred and then automatically converts the
// Cosmos Coin to their ERC20 token representation.
func (im *IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	acknowledgement []byte,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	// Unmarshal the acknowledgement
	var ack channeltypes.Acknowledgement
	if err := transfertypes.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet acknowledgement: %v", err)
	}

	// Call the underlying app's OnAcknowledgementPacket
	if err := im.app.OnAcknowledgementPacket(ctx, sourceClient, destinationClient, sequence, acknowledgement, payload, relayer); err != nil {
		return err
	}

	// Process ERC20 conversion using v2-specific keeper method
	return im.keeper.OnAcknowledgementPacketV2(ctx, payload, ack)
}

// OnTimeoutPacket implements the IBCModule interface.
// It refunds the token transferred and then automatically converts the
// Cosmos Coin to their ERC20 token representation.
func (im *IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	// Call the underlying app's OnTimeoutPacket
	if err := im.app.OnTimeoutPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer); err != nil {
		return err
	}

	// Process ERC20 conversion using v2-specific keeper method
	return im.keeper.OnTimeoutPacketV2(ctx, payload)
}
