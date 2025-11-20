package keeper

import (
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"

	"github.com/haqq-network/haqq/x/ucdao/types"
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

// getHoldersStore gets the holders store.
func (k BaseKeeper) getHoldersStore(ctx sdk.Context) prefix.Store {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.HoldersPrefix)
}

// setHoldersIndex registers account as shareholder or remove it from index
// if there are no coins on designated escrow account.
func (k BaseKeeper) setHoldersIndex(ctx sdk.Context, addr sdk.AccAddress) {
	holdersStore := k.getHoldersStore(ctx)
	addrKey := address.MustLengthPrefix(addr)

	// Delete value from holders store if all balances is zero.
	allBalances := k.GetAccountBalances(ctx, addr)
	if allBalances.IsZero() && holdersStore.Has(addrKey) {
		holdersStore.Delete(addrKey)
	}

	// Store an index of account address with a sentinel value.
	if !holdersStore.Has(addrKey) && !allBalances.IsZero() {
		holdersStore.Set(addrKey, []byte{0})
	}
}
