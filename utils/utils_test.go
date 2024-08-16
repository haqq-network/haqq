package utils

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
)

func init() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("haqq", "haqqpub")
}

func TestIsSupportedKeys(t *testing.T) {
	testCases := []struct {
		name        string
		pk          cryptotypes.PubKey
		isSupported bool
	}{
		{
			"nil key",
			nil,
			false,
		},
		{
			"ethsecp256k1 key",
			&ethsecp256k1.PubKey{},
			true,
		},
		{
			"ed25519 key",
			&ed25519.PubKey{},
			true,
		},
		{
			"multisig key - no pubkeys",
			&multisig.LegacyAminoPubKey{},
			false,
		},
		{
			"multisig key - valid pubkeys",
			multisig.NewLegacyAminoPubKey(2, []cryptotypes.PubKey{&ed25519.PubKey{}, &ed25519.PubKey{}, &ed25519.PubKey{}}),
			true,
		},
		{
			"multisig key - nested multisig",
			multisig.NewLegacyAminoPubKey(2, []cryptotypes.PubKey{&ed25519.PubKey{}, &ed25519.PubKey{}, &multisig.LegacyAminoPubKey{}}),
			false,
		},
		{
			"multisig key - invalid pubkey",
			multisig.NewLegacyAminoPubKey(2, []cryptotypes.PubKey{&ed25519.PubKey{}, &ed25519.PubKey{}, &secp256k1.PubKey{}}),
			false,
		},
		{
			"cosmos secp256k1",
			&secp256k1.PubKey{},
			false,
		},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.isSupported, IsSupportedKey(tc.pk), tc.name)
	}
}

func TestGetEvmosAddressFromBech32(t *testing.T) {
	testCases := []struct {
		name       string
		address    string
		expAddress string
		expError   bool
	}{
		{
			"blank bech32 address",
			" ",
			"",
			true,
		},
		{
			"invalid bech32 address",
			"haqq",
			"",
			true,
		},
		{
			"invalid address bytes",
			"haqq1123",
			"",
			true,
		},
		{
			"haqq address",
			"haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
			"haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
			false,
		},
		{
			"cosmos address",
			"cosmos1hdr0lhv75vesvtndlh78ck4cez6esz8ugcrufk",
			"haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
			false,
		},
		{
			"osmosis address",
			"osmo1hdr0lhv75vesvtndlh78ck4cez6esz8uqrsvly",
			"haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq",
			false,
		},
	}

	for _, tc := range testCases {
		addr, err := GetHaqqAddressFromBech32(tc.address)
		if tc.expError {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
			require.Equal(t, tc.expAddress, addr.String(), tc.name)
		}
	}
}

func TestCoinDenom(t *testing.T) {
	testCases := []struct {
		name     string
		denom    string
		expError bool
	}{
		{
			"valid denom - native coin",
			"aISLM",
			false,
		},
		{
			"valid denom - ibc coin",
			"ibc/7B2A4F6E798182988D77B6B884919AF617A73503FDAC27C916CD7A69A69013CF",
			false,
		},
		{
			"valid denom - ethereum address (ERC-20 contract)",
			"erc20/0x52908400098527886e0f7030069857D2E4169EE7",
			false,
		},
		{
			"invalid denom - only one character",
			"a",
			true,
		},
		{
			"invalid denom - too large (> 127 chars)",
			"ibc/7B2A4F6E798182988D77B6B884919AF617A73503FDAC27C916CD7A69A69013CF7B2A4F6E798182988D77B6B884919AF617A73503FDAC27C916CD7A69A69013CF",
			true,
		},
		{
			"invalid denom - starts with 0 but not followed by 'x'",
			"0a52908400098527886E0F7030069857D2E4169EE7",
			true,
		},
		{
			"invalid denom - hex address but 19 bytes long",
			"0x52908400098527886E0F7030069857D2E4169E",
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			err := sdk.ValidateDenom(tc.denom)
			if tc.expError {
				require.Error(t, err, tc.name)
			} else {
				require.NoError(t, err, tc.name)
			}
		})
	}
}

func TestAccAddressFromBech32(t *testing.T) {
	testCases := []struct {
		address      string
		bech32Prefix string
		expErr       bool
		errContains  string
	}{
		{
			"",
			"",
			true,
			"empty address string is not allowed",
		},
		{
			"cosmos1hdr0lhv75vesvtndlh78ck4cez6esz8ugcrufk",
			"stride",
			true,
			"invalid Bech32 prefix; expected stride, got cosmos",
		},
		{
			"cosmos1hdr0lhv75vesvtndlh78ck4cez6eszufk",
			"cosmos",
			true,
			"decoding bech32 failed: invalid checksum",
		},
		{
			"stride1hdr0lhv75vesvtndlh78ck4cez6esz8utnrqa6",
			"stride",
			false,
			"",
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.address, func(t *testing.T) {
			t.Parallel()

			_, err := CreateAccAddressFromBech32(tc.address, tc.bech32Prefix)
			if tc.expErr {
				require.Error(t, err, "expected error while creating AccAddress")
				require.Contains(t, err.Error(), tc.errContains, "expected different error")
			} else {
				require.NoError(t, err, "expected no error while creating AccAddress")
			}
		})
	}
}

func TestParseHexValue(t *testing.T) {
	tests := []struct {
		hexStr string
		want   *big.Int
	}{
		{"0x1", big.NewInt(1)},
		{"0x10", big.NewInt(16)},
		{"0xff", big.NewInt(255)},
		{"0x1234567890abcdef", big.NewInt(0x1234567890abcdef)},
	}

	for _, tt := range tests {
		got := ParseHexValue(tt.hexStr)
		if got.Cmp(tt.want) != 0 {
			t.Errorf("ParseHexValue(%s) = %v, want %v", tt.hexStr, got, tt.want)
		}
	}
}

func TestRemove0xPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0x1", "1"},
		{"0XABC", "ABC"},
		{"123", "123"},
		{"0x123", "123"},
		{"0x1234567890abcdef", "1234567890abcdef"},
	}

	for _, tt := range tests {
		got := Remove0xPrefix(tt.input)
		if got != tt.want {
			t.Errorf("Remove0xPrefix(%s) = %s, want %s", tt.input, got, tt.want)
		}
	}
}

func TestKeccak256(t *testing.T) {
	tests := []struct {
		input []byte
		want  string
	}{
		{[]byte("hello"), "1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8"},
		{[]byte("haqq"), "ede9ccb406cc78631779409e4f3d0946ec6bfc530918f2dc8f63c284d209e724"},
	}

	for _, tt := range tests {
		got := Keccak256(tt.input)
		if hex.EncodeToString(got) != tt.want {
			t.Errorf("Keccak256(%s) = %x, want %s", tt.input, got, tt.want)
		}
	}
}

func TestCalculateStorageKey(t *testing.T) {
	tests := []struct {
		addr string
		i    int
		want string
	}{
		{"0xA367C471fFEdbB3230793e0aaf045c38e57eDf98", 0, "0xece64beae9f44f327fa25deecc04fcb83b8512d3873bc0f6702645d10aaafaad"},
		{"0xA367C471fFEdbB3230793e0aaf045c38e57eDf98", 1, "0x706a64cd6ab6caa25d744643a971945a13ac5b19961a5295e0771dd24711cc34"},
		{"0xA367C471fFEdbB3230793e0aaf045c38e57eDf98", 2, "0xd20799b9ccb19c9d821e349cec115df5cfd391b8d9c5b5ea10f9cc3d4f1e801e"},
		{"0xA367C471fFEdbB3230793e0aaf045c38e57eDf98", 3, "0xd6b77ced29b77d9d8fdab16e04c4ea5d9056bc8f52f1b081d4e80c158d5e91bd"},
	}

	for _, tt := range tests {
		got := CalculateStorageKey(tt.addr, tt.i)
		if got != tt.want {
			t.Errorf("CalculateStorageKey(%s, %d) = %s, want %s", tt.addr, tt.i, got, tt.want)
		}
	}
}
