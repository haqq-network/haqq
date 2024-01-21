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
	startTime int64,
	periods sdkvesting.Periods,
) (types.Denom, error) {
	denom := types.Denom{
		StartTime:     time.Unix(startTime, 0),
		LockupPeriods: periods,
	}

	counter := k.GetDenomCounter(ctx)
	denom.Denom = types.DenomNameFromID(counter)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.DenomKeyPrefix))
	appendedValue := k.cdc.MustMarshal(&denom)
	store.Set([]byte(denom.Denom), appendedValue)

	// Update chain counter
	k.SetDenomCounter(ctx, counter+1)

	return denom, nil
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
