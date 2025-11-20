package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/haqq-network/haqq/x/ucdao/types"
)

// GetBalance returns the balance of a specific denomination for a given account
// by address.
func (k BaseKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	escrow := types.GetEscrowAddress(addr)
	return k.bk.GetBalance(ctx, escrow, denom)
}

// GetAccountBalances returns all balances for a given account by address.
func (k BaseKeeper) GetAccountBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	escrow := types.GetEscrowAddress(addr)
	return k.bk.GetAllBalances(ctx, escrow)
}

// HasBalance returns whether or not an account has at least amt balance.
func (k BaseKeeper) HasBalance(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coin) bool {
	escrow := types.GetEscrowAddress(addr)
	return k.bk.HasBalance(ctx, escrow, amt)
}

// GetAccountsBalances returns all the accounts balances from the store.
func (k BaseKeeper) GetAccountsBalances(ctx sdk.Context) []types.Balance {
	balances := make([]types.Balance, 0)

	holders := k.GetHolders(ctx)
	for _, holder := range holders {
		holderBalances := k.GetAccountBalances(ctx, holder)
		accountBalance := types.Balance{
			Address: holder.String(),
			Coins:   holderBalances.Sort(),
		}
		balances = append(balances, accountBalance)
	}

	return balances
}

// GetHolders returns list of accounts holding any coins on escrow account and registered in UC DAO module.
// NOTE! Direct transfers to escrow accounts won't be tracked by module.
func (k BaseKeeper) GetHolders(ctx sdk.Context) []sdk.AccAddress {
	holdersStore := k.getHoldersStore(ctx)
	holders := make([]sdk.AccAddress, 0)

	iterator := holdersStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		addr, err := types.AddressFromHoldersStore(iterator.Key())
		if err != nil {
			k.Logger(ctx).With("key", iterator.Key(), "err", err).Error("failed to get address from holders store key")
			// TODO: revisit, for now, panic here to keep same behavior as in 0.42
			// ref: https://github.com/cosmos/cosmos-sdk/issues/7409
			panic(err)
		}
		holders = append(holders, addr)
	}

	return holders
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
