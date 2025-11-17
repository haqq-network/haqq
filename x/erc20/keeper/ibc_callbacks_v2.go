// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/ethereum/go-ethereum/common"
	"github.com/hashicorp/go-metrics"

	"github.com/haqq-network/haqq/ibc"
	"github.com/haqq-network/haqq/x/erc20/types"
)

// OnRecvPacketV2 performs the ICS20 middleware receive callback for automatically
// converting an IBC Coin to their ERC20 representation for IBC v2.
// This version works directly with v2 payloads instead of v1 packets.
//
// CONTRACT: This middleware MUST be executed transfer after the ICS20 OnRecvPacket
// Return acknowledgement and continue with the next layer of the IBC middleware
// stack if:
// - ERC20s are disabled
// - Denomination is native staking token
// - The base denomination is not registered as ERC20
func (k Keeper) OnRecvPacketV2(
	ctx sdk.Context,
	sourcePort string,
	destinationPort string,
	payload channeltypesv2.Payload,
	ack exported.Acknowledgement,
) exported.Acknowledgement {
	// If ERC20 module is disabled no-op
	if !k.IsERC20Enabled(ctx) {
		return ack
	}

	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(payload.Value, &data); err != nil {
		// NOTE: shouldn't happen as the packet has already
		// been decoded on ICS20 transfer logic
		err = errorsmod.Wrapf(errortypes.ErrInvalidType, "cannot unmarshal ICS-20 transfer packet data")
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// use a zero gas config to avoid extra costs for the relayers
	ctx = ctx.
		WithKVGasConfig(storetypes.GasConfig{}).
		WithTransientKVGasConfig(storetypes.GasConfig{})

	sender, recipient, _, _, err := ibc.GetTransferSenderRecipient(data)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// In v2, we don't have channel IDs, so we can't check if it's an EVM channel.
	// However, we still need to validate that sender != recipient for non-EVM scenarios
	// to prevent users from having funds stuck. Since we can't determine EVM channel status
	// in v2, we'll be more conservative and reject sender == recipient cases.
	// This is a safety measure - if needed, this can be relaxed based on v2 routing specifics.
	if sender.Equals(recipient) {
		return channeltypes.NewErrorAcknowledgement(types.ErrInvalidIBC)
	}

	receiverAcc := k.accountKeeper.GetAccount(ctx, recipient)

	// return acknowledgement without conversion if receiver is a module account
	if types.IsModuleAccount(receiverAcc) {
		return ack
	}

	// parse the transferred denom
	// In v2, we use empty strings for channels since they're not available
	coin := ibc.GetReceivedCoin(
		sourcePort, "",
		destinationPort, "",
		data.Denom, data.Amount,
	)

	// If the coin denom starts with `factory/` then it is a token factory coin, and we should not convert it
	// NOTE: Check https://docs.osmosis.zone/osmosis-core/modules/tokenfactory/ for more information
	if strings.HasPrefix(data.Denom, "factory/") {
		return ack
	}

	// check if the coin is a native staking token
	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}
	if coin.Denom == bondDenom {
		// no-op, received coin is the staking denomination
		return ack
	}

	pairID := k.GetTokenPairID(ctx, coin.Denom)
	pair, found := k.GetTokenPair(ctx, pairID)
	switch {
	// Case 1. token pair is not registered and is a single hop IBC Coin
	// by checking the prefix we ensure that only coins not native from this chain are evaluated.
	// IsNativeFromSourceChain will check if the coin is native from the source chain.
	// If the coin denom starts with `factory/` then it is a token factory coin, and we should not convert it
	// NOTE: Check https://docs.osmosis.zone/osmosis-core/modules/tokenfactory/ for more information
	case !found && strings.HasPrefix(coin.Denom, "ibc/") && ibc.IsBaseDenomFromSourceChain(data.Denom):
		tokenPair, err := k.RegisterERC20Extension(ctx, coin.Denom)
		if err != nil {
			return channeltypes.NewErrorAcknowledgement(err)
		}

		ctx.EventManager().EmitEvents(
			sdk.Events{
				sdk.NewEvent(
					types.EventTypeRegisterERC20Extension,
					sdk.NewAttribute(types.AttributeCoinSourceChannel, ""), // Empty in v2
					sdk.NewAttribute(types.AttributeKeyERC20Token, tokenPair.Erc20Address),
					sdk.NewAttribute(types.AttributeKeyCosmosCoin, tokenPair.Denom),
				),
			},
		)
		return ack

	// Case 2. native ERC20 token
	case found && pair.IsNativeERC20():
		// Token pair is disabled -> return
		if !pair.Enabled {
			return ack
		}

		balance := k.bankKeeper.GetBalance(ctx, recipient, coin.Denom)
		if err := k.ConvertCoinNativeERC20(ctx, pair, balance.Amount, common.BytesToAddress(recipient.Bytes()), recipient); err != nil {
			return channeltypes.NewErrorAcknowledgement(err)
		}

		// For now the only case we are interested in adding telemetry is a successful conversion.
		telemetry.IncrCounterWithLabels(
			[]string{types.ModuleName, "ibc", "on_recv", "v2", "total"},
			1,
			[]metrics.Label{
				telemetry.NewLabel("denom", coin.Denom),
				telemetry.NewLabel("source_port", sourcePort),
				telemetry.NewLabel("destination_port", destinationPort),
			},
		)
	}

	return ack
}

// OnAcknowledgementPacketV2 responds to the success or failure of a packet
// acknowledgement written on the receiving chain for IBC v2.
// If the acknowledgement was a success then nothing occurs. If the acknowledgement failed,
// then the sender is refunded and then the IBC Coins are converted to ERC20.
func (k Keeper) OnAcknowledgementPacketV2(
	ctx sdk.Context,
	payload channeltypesv2.Payload,
	ack channeltypes.Acknowledgement,
) error {
	// Unmarshal the packet data from payload
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(payload.Value, &data); err != nil {
		return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet data: %s", err.Error())
	}

	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		// convert the token from Cosmos Coin to its ERC20 representation
		return k.ConvertCoinToERC20FromPacket(ctx, data)
	default:
		// the acknowledgement succeeded on the receiving chain so nothing needs to
		// be executed and no error needs to be returned
		return nil
	}
}

// OnTimeoutPacketV2 converts the IBC coin to ERC20 after refunding the sender
// since the original packet sent was never received and has been timed out for IBC v2.
func (k Keeper) OnTimeoutPacketV2(
	ctx sdk.Context,
	payload channeltypesv2.Payload,
) error {
	// Unmarshal the packet data from payload
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(payload.Value, &data); err != nil {
		return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet data: %s", err.Error())
	}

	return k.ConvertCoinToERC20FromPacket(ctx, data)
}
