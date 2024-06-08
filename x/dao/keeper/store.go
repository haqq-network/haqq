package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/dao/types"
)

// getAccountStore gets the account store of the given address.
func (k BaseKeeper) getAccountStore(ctx sdk.Context, addr sdk.AccAddress) prefix.Store {
	store := ctx.KVStore(k.storeKey)
	return prefix.NewStore(store, types.CreateAccountBalancesPrefix(addr))
}

// getDenomAddressPrefixStore returns a prefix store that acts as a reverse index
// between a denomination and account balance for that denomination.
func (k BaseKeeper) getDenomAddressPrefixStore(ctx sdk.Context, denom string) prefix.Store {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.CreateDenomAddressPrefix(denom))
}
