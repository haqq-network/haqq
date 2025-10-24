package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/haqq-network/haqq/x/ucdao/types"
)

// GetBalance returns the balance of a specific denomination for a given account
// by address.
func (k BaseKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	accountStore := k.getAccountStore(ctx, addr)
	bz := accountStore.Get([]byte(denom))
	balance, err := UnmarshalBalanceCompat(k.cdc, bz, denom)
	if err != nil {
		panic(err)
	}

	return balance
}

// HasBalance returns whether or not an account has at least amt balance.
func (k BaseKeeper) HasBalance(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coin) bool {
	return k.GetBalance(ctx, addr, amt.Denom).IsGTE(amt)
}

// IterateAccountBalances iterates over the balances of a single account and
// provides the token balance to a callback. If true is returned from the
// callback, iteration is halted.
func (k BaseKeeper) IterateAccountBalances(ctx sdk.Context, addr sdk.AccAddress, cb func(sdk.Coin) bool) {
	accountStore := k.getAccountStore(ctx, addr)

	iterator := accountStore.Iterator(nil, nil)
	defer sdk.LogDeferred(ctx.Logger(), func() error { return iterator.Close() })

	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Key())
		balance, err := UnmarshalBalanceCompat(k.cdc, iterator.Value(), denom)
		if err != nil {
			panic(err)
		}

		if cb(balance) {
			break
		}
	}
}

func (k BaseKeeper) GetAccountBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	balances := sdk.NewCoins()

	k.IterateAccountBalances(ctx, addr, func(balance sdk.Coin) bool {
		balances = balances.Add(balance)
		return false
	})

	return balances
}

// IterateAllBalances iterates over all the balances of all accounts and
// denominations that are provided to a callback. If true is returned from the
// callback, iteration is halted.
func (k BaseKeeper) IterateAllBalances(ctx sdk.Context, cb func(sdk.AccAddress, sdk.Coin) bool) {
	store := ctx.KVStore(k.storeKey)
	balancesStore := prefix.NewStore(store, types.BalancesPrefix)

	iterator := balancesStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		address, denom, err := types.AddressAndDenomFromBalancesStore(iterator.Key())
		if err != nil {
			k.Logger(ctx).With("key", iterator.Key(), "err", err).Error("failed to get address from balances store")
			// TODO: revisit, for now, panic here to keep same behavior as in 0.42
			// ref: https://github.com/cosmos/cosmos-sdk/issues/7409
			panic(err)
		}

		balance, err := UnmarshalBalanceCompat(k.cdc, iterator.Value(), denom)
		if err != nil {
			panic(err)
		}

		if cb(address, balance) {
			break
		}
	}
}

// GetAccountsBalances returns all the accounts balances from the store.
func (k BaseKeeper) GetAccountsBalances(ctx sdk.Context) []types.Balance {
	balances := make([]types.Balance, 0)
	mapAddressToBalancesIdx := make(map[string]int)

	k.IterateAllBalances(ctx, func(addr sdk.AccAddress, balance sdk.Coin) bool {
		idx, ok := mapAddressToBalancesIdx[addr.String()]
		if ok {
			// address is already on the set of accounts balances
			balances[idx].Coins = balances[idx].Coins.Add(balance)
			balances[idx].Coins.Sort()
			return false
		}

		accountBalance := types.Balance{
			Address: addr.String(),
			Coins:   sdk.NewCoins(balance),
		}
		balances = append(balances, accountBalance)
		mapAddressToBalancesIdx[addr.String()] = len(balances) - 1
		return false
	})

	return balances
}

// GetPaginatedAccountsBalances returns all the accounts balances from the store paginated by accounts.
func (k BaseKeeper) GetPaginatedAccountsBalances(ctx sdk.Context, pagination *query.PageRequest) ([]types.Balance, *query.PageResponse, error) {
	holdersStore := k.getHoldersStore(ctx)
	balances := make([]types.Balance, 0)
	pageRes, err := query.Paginate(holdersStore, pagination, func(key, _ []byte) error {
		addr, err := types.AddressFromHoldersStore(key)
		if err != nil {
			k.Logger(ctx).With("key", key, "err", err).Error("failed to get address from holders store key")
			// TODO: revisit, for now, panic here to keep same behavior as in 0.42
			// ref: https://github.com/cosmos/cosmos-sdk/issues/7409
			panic(err)
		}

		coins := k.GetAccountBalances(ctx, addr)
		balance := types.Balance{
			Address: addr.String(),
			Coins:   coins,
		}

		balances = append(balances, balance)

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return balances, pageRes, nil
}

// UnmarshalBalanceCompat unmarshal balance amount from storage, it's backward-compatible with the legacy format.
func UnmarshalBalanceCompat(cdc codec.BinaryCodec, bz []byte, denom string) (sdk.Coin, error) {
	if err := sdk.ValidateDenom(denom); err != nil {
		return sdk.Coin{}, err
	}

	amount := math.ZeroInt()
	if bz == nil {
		return sdk.NewCoin(denom, amount), nil
	}

	if err := amount.Unmarshal(bz); err != nil {
		// try to unmarshal with the legacy format.
		var balance sdk.Coin
		if cdc.Unmarshal(bz, &balance) != nil {
			// return with the original error
			return sdk.Coin{}, err
		}
		return balance, nil
	}

	return sdk.NewCoin(denom, amount), nil
}

// setBalance sets the coin balance for an account by address.
func (k BaseKeeper) setBalance(ctx sdk.Context, addr sdk.AccAddress, balance sdk.Coin) error {
	if !balance.IsValid() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, balance.String())
	}

	accountStore := k.getAccountStore(ctx, addr)
	denomPrefixStore := k.getDenomAddressPrefixStore(ctx, balance.Denom)
	addrKey := address.MustLengthPrefix(addr)

	// x/bank invariants prohibit persistence of zero balances
	if balance.IsZero() {
		accountStore.Delete([]byte(balance.Denom))
		denomPrefixStore.Delete(addrKey)
	} else {
		amount, err := balance.Amount.Marshal()
		if err != nil {
			return err
		}

		accountStore.Set([]byte(balance.Denom), amount)

		// Store a reverse index from denomination to account address with a
		// sentinel value.
		if !denomPrefixStore.Has(addrKey) {
			denomPrefixStore.Set(addrKey, []byte{0})
		}
	}

	return nil
}

// addCoinsToAccount increase the addr balance by the given amt. Fails if the provided
// amt is invalid. It emits a coin received event.
func (k BaseKeeper) addCoinsToAccount(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coins) error {
	if !amt.IsValid() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, amt.String())
	}

	for _, coin := range amt {
		balance := k.GetBalance(ctx, addr, coin.Denom)
		newBalance := balance.Add(coin)

		err := k.setBalance(ctx, addr, newBalance)
		if err != nil {
			return err
		}
	}

	// TODO emit coin received event

	return nil
}

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
