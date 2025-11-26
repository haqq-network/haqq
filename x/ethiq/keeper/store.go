package keeper

import (
	sdkmath "cosmossdk.io/math"
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
