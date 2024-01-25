package keeper

import (
	"encoding/binary"
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

// CreateDenom
func (k Keeper) CreateDenom(
	ctx sdk.Context,
	originalDenom string,
	startTime int64,
	periods sdkvesting.Periods,
) (types.Denom, error) {
	denom := types.Denom{
		StartTime:     time.Unix(startTime, 0),
		LockupPeriods: periods,
		OriginalDenom: originalDenom,
	}

	counter := k.GetDenomCounter(ctx)
	denom.LiquidDenom = types.DenomNameFromID(counter)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DenomKeyPrefix))
	appendedValue := k.cdc.MustMarshal(&denom)
	store.Set([]byte(denom.LiquidDenom), appendedValue)

	// Update chain counter
	k.SetDenomCounter(ctx, counter+1)

	return denom, nil
}

func (k Keeper) UpdateDenomPeriods(ctx sdk.Context, liquidDenom string, newPeriods sdkvesting.Periods) error {
	d, found := k.GetDenom(ctx, liquidDenom)
	if !found {
		return types.ErrDenomNotFound
	}
	d.LockupPeriods = newPeriods
	k.SetDenom(ctx, d)
	return nil
}

func (k Keeper) GetDenom(ctx sdk.Context, liquidDenom string) (val types.Denom, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DenomKeyPrefix))

	b := store.Get([]byte(liquidDenom))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

func (k Keeper) SetDenom(ctx sdk.Context, denom types.Denom) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DenomKeyPrefix))
	b := k.cdc.MustMarshal(&denom)
	store.Set([]byte(denom.LiquidDenom), b)
}

// GetDenomCounter get the counter for denoms
func (k Keeper) GetDenomCounter(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.DenomKeyPrefix)
	bz := store.Get(byteKey)

	// Counter doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetDenomCounter set the counter for chains
func (k Keeper) SetDenomCounter(ctx sdk.Context, counter uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.DenomCounterKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, counter)
	store.Set(byteKey, bz)
}
