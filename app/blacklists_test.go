package app

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TestBlockedAccountsValid guards the single blacklist against malformed
// entries. A bad bech32 key would silently never match a real signer/sender, so
// a typo would leave the account unfrozen without any obvious failure.
func TestBlockedAccountsValid(t *testing.T) {
	require.NotEmpty(t, blockedAccounts)

	for bech := range blockedAccounts {
		acc, err := sdk.AccAddressFromBech32(bech)
		require.NoErrorf(t, err, "invalid bech32 address in blacklist: %s", bech)
		// Round-trip must be canonical so runtime lookups (which key on
		// AccAddress.String()) match exactly.
		require.Equalf(t, bech, acc.String(), "non-canonical bech32 address in blacklist: %s", bech)
	}
}
