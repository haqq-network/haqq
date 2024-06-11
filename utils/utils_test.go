package utils

import (
	"fmt"
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
			"haqq1hd29v0xamy30f8us966t0eefswecrfdntcfzv3",
			"haqq1hd29v0xamy30f8us966t0eefswecrfdntcfzv3",
			false,
		},
		{
			"cosmos address",
			"cosmos1hd29v0xamy30f8us966t0eefswecrfdnflu3j8",
			"haqq1hd29v0xamy30f8us966t0eefswecrfdntcfzv3",
			false,
		},
		{
			"osmosis address",
			"osmo1hd29v0xamy30f8us966t0eefswecrfdnpy0py4",
			"haqq1hd29v0xamy30f8us966t0eefswecrfdntcfzv3",
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

func TestEvmosCoinDenom(t *testing.T) {
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
			"cosmos1hd29v0xamy30f8us966t0eefswecrfdnflu3j8",
			"stride",
			true,
			"invalid Bech32 prefix; expected stride, got cosmos",
		},
		{
			"cosmos1hd29v0xamy30f8us966t0eefswecrfdnf",
			"cosmos",
			true,
			"decoding bech32 failed: invalid checksum",
		},
		{
			"stride1hd29v0xamy30f8us966t0eefswecrfdn25udxt",
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
