package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/ucdao/types"
)

// GetTotalBalance get the global total balance of dao module
func (k BaseKeeper) GetTotalBalance(ctx sdk.Context) sdk.Coins {
	balance := sdk.NewCoins()

	k.IterateTotalBalance(ctx, func(c sdk.Coin) bool {
		if c.IsZero() {
			return false
		}

		balance = balance.Add(c)

		return false
	})

	return balance
}

// GetPaginatedTotalBalance queries for the supply, ignoring 0 coins, with a given pagination
func (k BaseKeeper) GetPaginatedTotalBalance(ctx sdk.Context, pagination *query.PageRequest) (sdk.Coins, *query.PageResponse, error) {
	store := ctx.KVStore(k.storeKey)
	supplyStore := prefix.NewStore(store, types.TotalBalanceKey)

	supply := sdk.NewCoins()

	pageRes, err := query.Paginate(supplyStore, pagination, func(key, value []byte) error {
		var amount math.Int
		err := amount.Unmarshal(value)
		if err != nil {
			return fmt.Errorf("unable to convert amount string to Int %v", err)
		}

		// `Add` omits the 0 coins addition to the `supply`.
		supply = supply.Add(sdk.NewCoin(string(key), amount))
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return supply, pageRes, nil
}

// IterateTotalBalance iterates over the total balance calling the given cb (callback) function
// with the balance of each coin.
// The iteration stops if the callback returns true.
func (k BaseKeeper) IterateTotalBalance(ctx sdk.Context, cb func(sdk.Coin) bool) {
	store := ctx.KVStore(k.storeKey)
	supplyStore := prefix.NewStore(store, types.TotalBalanceKey)

	iterator := supplyStore.Iterator(nil, nil)
	defer sdk.LogDeferred(ctx.Logger(), func() error { return iterator.Close() })

	for ; iterator.Valid(); iterator.Next() {
		var amount math.Int
		err := amount.Unmarshal(iterator.Value())
		if err != nil {
			panic(fmt.Errorf("unable to unmarshal supply value %v", err))
		}

		balance := sdk.Coin{
			Denom:  string(iterator.Key()),
			Amount: amount,
		}

		if cb(balance) {
			break
		}
	}
}

// GetTotalBalanceOf retrieves the total balance of certain coin from store
func (k BaseKeeper) GetTotalBalanceOf(ctx sdk.Context, denom string) sdk.Coin {
	store := ctx.KVStore(k.storeKey)
	supplyStore := prefix.NewStore(store, types.TotalBalanceKey)

	bz := supplyStore.Get(utils.UnsafeStrToBytes(denom))
	if bz == nil {
		return sdk.Coin{
			Denom:  denom,
			Amount: math.NewInt(0),
		}
	}

	var amount math.Int
	err := amount.Unmarshal(bz)
	if err != nil {
		panic(fmt.Errorf("unable to unmarshal total balance value %v", err))
	}

	return sdk.Coin{
		Denom:  denom,
		Amount: amount,
	}
}

// HasTotalBalanceOf checks if the supply coin exists in store.
func (k BaseKeeper) HasTotalBalanceOf(ctx sdk.Context, denom string) bool {
	store := ctx.KVStore(k.storeKey)
	supplyStore := prefix.NewStore(store, types.TotalBalanceKey)

	return supplyStore.Has(utils.UnsafeStrToBytes(denom))
}

// setSupply sets the supply for the given coin
func (k BaseKeeper) setTotalBalanceOfCoin(ctx sdk.Context, coin sdk.Coin) {
	intBytes, err := coin.Amount.Marshal()
	if err != nil {
		panic(fmt.Errorf("unable to marshal amount value %v", err))
	}

	store := ctx.KVStore(k.storeKey)
	supplyStore := prefix.NewStore(store, types.TotalBalanceKey)

	// Bank invariants and IBC requires to remove zero coins.
	if coin.IsZero() {
		supplyStore.Delete(utils.UnsafeStrToBytes(coin.GetDenom()))
	} else {
		supplyStore.Set([]byte(coin.GetDenom()), intBytes)
	}
}
