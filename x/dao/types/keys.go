package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/cosmos-sdk/types/kv"
)

const (
	// ModuleName is the module name constant used in many places
	ModuleName = "dao"

	// StoreKey is the store key string for distribution
	StoreKey = ModuleName

	// RouterKey is the message route for distribution
	RouterKey = ModuleName
)

var (
	TotalBalanceKey = []byte{0x00} // key for global dao state

	// BalancesPrefix is the prefix for the account balances store. We use a byte
	// (instead of `[]byte("balances")` to save some disk space).
	BalancesPrefix     = []byte{0x01}
	DenomAddressPrefix = []byte{0x02}

	ParamsKey = []byte{0x03} // key for dao module params
)

// CreateAccountBalancesPrefix creates the prefix for an account's balances.
func CreateAccountBalancesPrefix(addr []byte) []byte {
	return append(BalancesPrefix, address.MustLengthPrefix(addr)...)
}

// CreateDenomAddressPrefix creates a prefix for a reverse index of denomination
// to account balance for that denomination.
func CreateDenomAddressPrefix(denom string) []byte {
	// we add a "zero" byte at the end - null byte terminator, to allow prefix denom prefix
	// scan. Setting it is not needed (key[last] = 0) - because this is the default.
	key := make([]byte, len(DenomAddressPrefix)+len(denom)+1)
	copy(key, DenomAddressPrefix)
	copy(key[len(DenomAddressPrefix):], denom)

	return key
}

// AddressAndDenomFromBalancesStore returns an account address and denom from a balances prefix
// store. The key must not contain the prefix BalancesPrefix as the prefix store
// iterator discards the actual prefix.
//
// If invalid key is passed, AddressAndDenomFromBalancesStore returns ErrInvalidKey.
func AddressAndDenomFromBalancesStore(key []byte) (sdk.AccAddress, string, error) {
	if len(key) == 0 {
		return nil, "", ErrInvalidKey
	}

	kv.AssertKeyAtLeastLength(key, 1)

	addrBound := int(key[0])

	if len(key)-1 < addrBound {
		return nil, "", ErrInvalidKey
	}

	return key[1 : addrBound+1], string(key[addrBound+1:]), nil
}
