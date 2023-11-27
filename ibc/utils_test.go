package ibc

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"

	teststypes "github.com/haqq-network/haqq/types/tests"
)

func init() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("haqq", "haqqpub")
}

func TestGetTransferSenderRecipient(t *testing.T) {
	testCases := []struct {
		name         string
		packet       channeltypes.Packet
		expSender    string
		expRecipient string
		expError     bool
	}{
		{
			"empty packet",
			channeltypes.Packet{},
			"", "",
			true,
		},
		{
			"invalid packet data",
			channeltypes.Packet{
				Data: ibctesting.MockFailPacketData,
			},
			"", "",
			true,
		},
		{
			"empty FungibleTokenPacketData",
			channeltypes.Packet{
				Data: transfertypes.ModuleCdc.MustMarshalJSON(
					&transfertypes.FungibleTokenPacketData{},
				),
			},
			"", "",
			true,
		},
		{
			"invalid sender",
			channeltypes.Packet{
				Data: transfertypes.ModuleCdc.MustMarshalJSON(
					&transfertypes.FungibleTokenPacketData{
						Sender:   "cosmos1",
						Receiver: "haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
						Amount:   "123456",
					},
				),
			},
			"", "",
			true,
		},
		{
			"invalid recipient",
			channeltypes.Packet{
				Data: transfertypes.ModuleCdc.MustMarshalJSON(
					&transfertypes.FungibleTokenPacketData{
						Sender:   "cosmos1tjdjfavsy956d25hvhs3p0nw9a7pfghqegfjmu",
						Receiver: "haqq1",
						Amount:   "123456",
					},
				),
			},
			"", "",
			true,
		},
		{
			"valid - cosmos sender, haqq recipient",
			channeltypes.Packet{
				Data: transfertypes.ModuleCdc.MustMarshalJSON(
					&transfertypes.FungibleTokenPacketData{
						Sender:   "cosmos1tjdjfavsy956d25hvhs3p0nw9a7pfghqegfjmu",
						Receiver: "haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
						Amount:   "123456",
					},
				),
			},
			"haqq1tjdjfavsy956d25hvhs3p0nw9a7pfghqm0up92",
			"haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
			false,
		},
		{
			"valid - haqq sender, cosmos recipient",
			channeltypes.Packet{
				Data: transfertypes.ModuleCdc.MustMarshalJSON(
					&transfertypes.FungibleTokenPacketData{
						Sender:   "haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
						Receiver: "cosmos1tjdjfavsy956d25hvhs3p0nw9a7pfghqegfjmu",
						Amount:   "123456",
					},
				),
			},
			"haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
			"haqq1tjdjfavsy956d25hvhs3p0nw9a7pfghqm0up92",
			false,
		},
		{
			"valid - osmosis sender, haqq recipient",
			channeltypes.Packet{
				Data: transfertypes.ModuleCdc.MustMarshalJSON(
					&transfertypes.FungibleTokenPacketData{
						Sender:   "osmo1tjdjfavsy956d25hvhs3p0nw9a7pfghq3n6zdw",
						Receiver: "haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
						Amount:   "123456",
					},
				),
			},
			"haqq1tjdjfavsy956d25hvhs3p0nw9a7pfghqm0up92",
			"haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
			false,
		},
	}

	for _, tc := range testCases {
		sender, recipient, _, _, err := GetTransferSenderRecipient(tc.packet)
		if tc.expError {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
			require.Equal(t, tc.expSender, sender.String())
			require.Equal(t, tc.expRecipient, recipient.String())
		}
	}
}

func TestGetTransferAmount(t *testing.T) {
	testCases := []struct {
		name      string
		packet    channeltypes.Packet
		expAmount string
		expError  bool
	}{
		{
			"empty packet",
			channeltypes.Packet{},
			"",
			true,
		},
		{
			"invalid packet data",
			channeltypes.Packet{
				Data: ibctesting.MockFailPacketData,
			},
			"",
			true,
		},
		{
			"invalid amount - empty",
			channeltypes.Packet{
				Data: transfertypes.ModuleCdc.MustMarshalJSON(
					&transfertypes.FungibleTokenPacketData{
						Sender:   "cosmos1tjdjfavsy956d25hvhs3p0nw9a7pfghqegfjmu",
						Receiver: "haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
						Amount:   "",
					},
				),
			},
			"",
			true,
		},
		{
			"invalid amount - non-int",
			channeltypes.Packet{
				Data: transfertypes.ModuleCdc.MustMarshalJSON(
					&transfertypes.FungibleTokenPacketData{
						Sender:   "cosmos1tjdjfavsy956d25hvhs3p0nw9a7pfghqegfjmu",
						Receiver: "haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
						Amount:   "test",
					},
				),
			},
			"test",
			true,
		},
		{
			"valid",
			channeltypes.Packet{
				Data: transfertypes.ModuleCdc.MustMarshalJSON(
					&transfertypes.FungibleTokenPacketData{
						Sender:   "cosmos1tjdjfavsy956d25hvhs3p0nw9a7pfghqegfjmu",
						Receiver: "haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
						Amount:   "10000",
					},
				),
			},
			"10000",
			false,
		},
	}

	for _, tc := range testCases {
		amt, err := GetTransferAmount(tc.packet)
		if tc.expError {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
			require.Equal(t, tc.expAmount, amt)
		}
	}
}

func TestGetReceivedCoin(t *testing.T) {
	testCases := []struct {
		name       string
		srcPort    string
		srcChannel string
		dstPort    string
		dstChannel string
		rawDenom   string
		rawAmount  string
		expCoin    sdk.Coin
	}{
		{
			"transfer unwrapped coin to destination which is not its source",
			"transfer",
			"channel-0",
			"transfer",
			"channel-0",
			"uosmo",
			"10",
			sdk.Coin{Denom: teststypes.UosmoIbcdenom, Amount: sdk.NewInt(10)},
		},
		{
			"transfer ibc wrapped coin to destination which is its source",
			"transfer",
			"channel-0",
			"transfer",
			"channel-0",
			"transfer/channel-0/aISLM",
			"10",
			sdk.Coin{Denom: "aISLM", Amount: sdk.NewInt(10)},
		},
		{
			"transfer 2x ibc wrapped coin to destination which is its source",
			"transfer",
			"channel-0",
			"transfer",
			"channel-2",
			"transfer/channel-0/transfer/channel-1/uatom",
			"10",
			sdk.Coin{Denom: teststypes.UatomIbcdenom, Amount: sdk.NewInt(10)},
		},
		{
			"transfer ibc wrapped coin to destination which is not its source",
			"transfer",
			"channel-0",
			"transfer",
			"channel-0",
			"transfer/channel-1/uatom",
			"10",
			sdk.Coin{Denom: teststypes.UatomOsmoIbcdenom, Amount: sdk.NewInt(10)},
		},
	}

	for _, tc := range testCases {
		coin := GetReceivedCoin(tc.srcPort, tc.srcChannel, tc.dstPort, tc.dstChannel, tc.rawDenom, tc.rawAmount)
		require.Equal(t, tc.expCoin, coin)
	}
}

func TestGetSentCoin(t *testing.T) {
	testCases := []struct {
		name      string
		rawDenom  string
		rawAmount string
		expCoin   sdk.Coin
	}{
		{
			"get unwrapped aISLM coin",
			"aISLM",
			"10",
			sdk.Coin{Denom: "aISLM", Amount: sdk.NewInt(10)},
		},
		{
			"get ibc wrapped aISLM coin",
			"transfer/channel-0/aISLM",
			"10",
			sdk.Coin{Denom: teststypes.AislmIbcdenom, Amount: sdk.NewInt(10)},
		},
		{
			"get ibc wrapped uosmo coin",
			"transfer/channel-0/uosmo",
			"10",
			sdk.Coin{Denom: teststypes.UosmoIbcdenom, Amount: sdk.NewInt(10)},
		},
		{
			"get ibc wrapped uatom coin",
			"transfer/channel-1/uatom",
			"10",
			sdk.Coin{Denom: teststypes.UatomIbcdenom, Amount: sdk.NewInt(10)},
		},
		{
			"get 2x ibc wrapped uatom coin",
			"transfer/channel-0/transfer/channel-1/uatom",
			"10",
			sdk.Coin{Denom: teststypes.UatomOsmoIbcdenom, Amount: sdk.NewInt(10)},
		},
	}

	for _, tc := range testCases {
		coin := GetSentCoin(tc.rawDenom, tc.rawAmount)
		require.Equal(t, tc.expCoin, coin)
	}
}
