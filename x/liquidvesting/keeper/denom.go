package keeper

import (
	"encoding/binary"
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

// CreateDenom creates new liquid denom and stores it
func (k Keeper) CreateDenom(
	ctx sdk.Context,
	originalDenom string,
	startTime int64,
	periods sdkvesting.Periods,
) (types.Denom, error) {
	counter := k.GetDenomCounter(ctx)

	denom := types.Denom{
		StartTime:     time.Unix(startTime, 0),
		LockupPeriods: periods,
		OriginalDenom: originalDenom,
		EndTime:       time.Unix(startTime+periods.TotalLength(), 0),
		BaseDenom:     types.DenomBaseNameFromID(counter),
		DisplayDenom:  types.DenomDisplayNameFromID(counter),
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomKeyPrefix)
	appendedValue := k.cdc.MustMarshal(&denom)
	store.Set([]byte(denom.GetBaseDenom()), appendedValue)

	// Update denom counter
	k.SetDenomCounter(ctx, counter+1)

	return denom, nil
}

// UpdateDenomPeriods updates schedule periods bound to liquid denom
func (k Keeper) UpdateDenomPeriods(ctx sdk.Context, baseDenom string, newPeriods sdkvesting.Periods) error {
	d, found := k.GetDenom(ctx, baseDenom)
	if !found {
		return types.ErrDenomNotFound
	}
	d.LockupPeriods = newPeriods
	k.SetDenom(ctx, d)
	return nil
}

// DeleteDenom deletes denom from the storage
func (k Keeper) DeleteDenom(ctx sdk.Context, baseDenom string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomKeyPrefix)
	store.Delete([]byte(baseDenom))
}

// GetDenom queries denom from the store
func (k Keeper) GetDenom(ctx sdk.Context, baseDenom string) (val types.Denom, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomKeyPrefix)

	b := store.Get([]byte(baseDenom))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// SetDenom sets denom in the store
func (k Keeper) SetDenom(ctx sdk.Context, denom types.Denom) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomKeyPrefix)
	b := k.cdc.MustMarshal(&denom)
	store.Set([]byte(denom.GetBaseDenom()), b)
}

// GetDenomCounter get the counter for denoms
func (k Keeper) GetDenomCounter(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.DenomCounterKey
	bz := store.Get(byteKey)

	// Counter doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetDenomCounter set the counter for denoms
func (k Keeper) SetDenomCounter(ctx sdk.Context, counter uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.DenomCounterKey
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, counter)
	store.Set(byteKey, bz)
}

// IterateDenoms iterates over all the stored denoms and performs a callback function.
// Stops iteration when callback returns true.
func (k Keeper) IterateDenoms(ctx sdk.Context, cb func(account types.Denom) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.DenomKeyPrefix)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var denom types.Denom
		k.cdc.MustUnmarshal(iterator.Value(), &denom)

		if cb(denom) {
			break
		}
	}
}
