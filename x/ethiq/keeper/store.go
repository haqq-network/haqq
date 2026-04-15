package keeper

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ethiq/types"
)

// GetTotalBurnedAmount returns the total amount of burned coins
func (k Keeper) GetTotalBurnedAmount(ctx sdk.Context) sdk.Coin {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TotalBurnedAmountKey)
	if bz == nil {
		return sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt())
	}

	var coin sdk.Coin
	k.cdc.MustUnmarshal(bz, &coin)
	return coin
}

// SetTotalBurnedAmount sets the total amount of burned coins
func (k Keeper) SetTotalBurnedAmount(ctx sdk.Context, coin sdk.Coin) {
	if coin.Denom != utils.BaseDenom {
		panic("the total burned amount must be aISLM")
	}

	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&coin)
	store.Set(types.TotalBurnedAmountKey, bz)
}

// AddToTotalBurnedAmount adds to the total burned amount
func (k Keeper) AddToTotalBurnedAmount(ctx sdk.Context, amount sdkmath.Int) {
	current := k.GetTotalBurnedAmount(ctx)
	newAmount := current.Amount.Add(amount)
	k.SetTotalBurnedAmount(ctx, sdk.NewCoin(utils.BaseDenom, newAmount))
}

// GetTotalBurnedFromApplicationsAmount returns the total amount of burned coins
func (k Keeper) GetTotalBurnedFromApplicationsAmount(ctx sdk.Context) sdk.Coin {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TotalBurnedFromApplicationsAmountKey)
	if bz == nil {
		return sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt())
	}

	var coin sdk.Coin
	k.cdc.MustUnmarshal(bz, &coin)
	return coin
}

// SetTotalBurnedFromApplicationsAmount sets the total amount of burned coins
func (k Keeper) SetTotalBurnedFromApplicationsAmount(ctx sdk.Context, coin sdk.Coin) {
	if coin.Denom != utils.BaseDenom {
		panic("the total burned from applications amount must be aISLM")
	}

	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&coin)
	store.Set(types.TotalBurnedFromApplicationsAmountKey, bz)
}

// AddToTotalBurnedFromApplicationsAmount adds to the total burned amount
func (k Keeper) AddToTotalBurnedFromApplicationsAmount(ctx sdk.Context, amount sdkmath.Int) {
	current := k.GetTotalBurnedFromApplicationsAmount(ctx)
	newAmount := current.Amount.Add(amount)
	k.SetTotalBurnedFromApplicationsAmount(ctx, sdk.NewCoin(utils.BaseDenom, newAmount))
}

func (k Keeper) IsApplicationExecuted(ctx sdk.Context, appID uint64) bool {
	key := sdkmath.NewIntFromUint64(appID).BigInt().Bytes()
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ExecutedApplicationsPrefix)
	return store.Has(key)
}

func (k Keeper) SetApplicationAsExecuted(ctx sdk.Context, appID uint64) {
	key := sdkmath.NewIntFromUint64(appID).BigInt().Bytes()
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ExecutedApplicationsPrefix)
	store.Set(key, []byte{0})
}

func (k Keeper) ResetApplicationByID(ctx sdk.Context, appID uint64) {
	key := sdkmath.NewIntFromUint64(appID).BigInt().Bytes()
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ExecutedApplicationsPrefix)
	store.Delete(key)
}

func (k Keeper) GetAllExecutedApplicationsIDs(ctx sdk.Context) []uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ExecutedApplicationsPrefix)
	apps := make([]uint64, 0)

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bigIntKey := (&big.Int{}).SetBytes(iterator.Key())
		apps = append(apps, bigIntKey.Uint64())
	}

	return apps
}
