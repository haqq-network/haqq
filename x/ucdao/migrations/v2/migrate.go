package v2

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"

	"github.com/haqq-network/haqq/x/ucdao/types"
)

// Migrate1to2 migrates the x/ucdao module state from the consensus version 1 to
// version 2. Specifically, it takes balances that are currently stored directly in the module state
// and move them into native Cosmos SDK Bank module state.
func Migrate1to2(
	ctx sdk.Context,
	storeKey storetypes.StoreKey,
	bk types.BankKeeper,
	cdc codec.BinaryCodec,
) error {
	// Iterate all holders and migrate to escrow accounts
	store := ctx.KVStore(storeKey)
	balancesStore := prefix.NewStore(store, types.BalancesPrefix)

	iterator := balancesStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		addr, denom, err := types.AddressAndDenomFromBalancesStore(iterator.Key())
		if err != nil {
			// TODO: revisit, for now, panic here to keep same behavior as in 0.42
			// ref: https://github.com/cosmos/cosmos-sdk/issues/7409
			return errorsmod.Wrapf(err, "failed to get balances from balances store: key %s", iterator.Key())
		}

		coin, err := unmarshalBalanceCompat(cdc, iterator.Value(), denom)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal balance from balances store: account %s", addr.String())
		}

		escrow := types.GetEscrowAddress(addr)

		// Transfer coins from module to escrow account
		if err := bk.SendCoinsFromModuleToAccount(ctx, types.ModuleName, escrow, sdk.NewCoins(coin)); err != nil {
			return errorsmod.Wrapf(err, "failed to migrate account balance: %s for %s", coin.String(), addr.String())
		}

		// Update holders index
		setHoldersIndex(ctx, storeKey, bk, addr)
		// Remove redundant data that going to be unused after migration
		balancesStore.Delete(iterator.Key())
	}

	return nil
}

// unmarshalBalanceCompat unmarshal balance amount from storage, it's backward-compatible with the legacy format.
func unmarshalBalanceCompat(cdc codec.BinaryCodec, bz []byte, denom string) (sdk.Coin, error) {
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

func setHoldersIndex(ctx sdk.Context, storeKey storetypes.StoreKey, bk types.BankKeeper, addr sdk.AccAddress) {
	holdersStore := prefix.NewStore(ctx.KVStore(storeKey), types.HoldersPrefix)
	addrKey := address.MustLengthPrefix(addr)

	escrow := types.GetEscrowAddress(addr)

	// Delete value from holders store if all balances is zero.
	allBalances := bk.GetAllBalances(ctx, escrow)
	if allBalances.IsZero() && holdersStore.Has(addrKey) {
		holdersStore.Delete(addrKey)
	}

	// Store an index of account address with a sentinel value.
	if !holdersStore.Has(addrKey) && !allBalances.IsZero() {
		holdersStore.Set(addrKey, []byte{0})
	}
}
