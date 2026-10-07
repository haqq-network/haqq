package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/haqq-network/haqq/x/ucdao/types"
)

// InitGenesis initializes the ucdao module's state from a given genesis state.
func (k BaseKeeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	totalBalance := sdk.Coins{}
	genState.Balances = types.SanitizeGenesisBalances(genState.Balances)

	for _, balance := range genState.Balances {
		addr := balance.GetAddress()

		if err := k.initBalances(ctx, addr, balance.Coins); err != nil {
			panic(fmt.Errorf("error on setting balances %w", err))
		}

		totalBalance = totalBalance.Add(balance.Coins...)

		// Set the holders store if the balance is not zero
		k.SetHoldersIndex(ctx, addr)
	}

	if !genState.TotalBalance.Empty() && !genState.TotalBalance.Equal(totalBalance) {
		panic(fmt.Errorf("genesis total balance is incorrect, expected %v, got %v", genState.TotalBalance, totalBalance))
	}

	for _, supply := range totalBalance {
		k.setTotalBalanceOfCoin(ctx, supply)
	}
}

// ExportGenesis returns the ucdao module's genesis state.
//
// TotalBalance is the sum of the exported Balances, not the stored counter.
// Balances are the bank balances of the holders' escrow accounts, and anyone can
// credit an escrow with a plain transfer the counter never sees (see
// TrackSubBalance) - in aISLM or in any other denom. InitGenesis rejects a total
// that does not match the balances, so exporting the counter produced a genesis
// this module refused to import once a single coin had been sent to a holder's
// escrow. InitGenesis rebuilds the counter from the balances in any case.
func (k BaseKeeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	balances := k.GetAccountsBalances(ctx)

	total := sdk.NewCoins()
	for _, balance := range balances {
		total = total.Add(balance.Coins...)
	}

	return types.NewGenesisState(k.GetParams(ctx), balances, total)
}

// initBalances sets the balance (multiple coins) for an account by address.
// An error is returned upon failure.
// CONTRACT: SDK bank module must be initialized first as this module relies on bank balances of escrow accounts
func (k BaseKeeper) initBalances(ctx sdk.Context, addr sdk.AccAddress, balances sdk.Coins) error {
	accountStore := k.getAccountStore(ctx, addr)
	denomPrefixStores := make(map[string]prefix.Store) // memoize prefix stores

	for i := range balances {
		balance := balances[i]
		if !balance.IsValid() {
			return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, balance.String())
		}

		// x/dao invariants prohibit persistence of zero balances
		if !balance.IsZero() {
			amount, err := balance.Amount.Marshal()
			if err != nil {
				return err
			}
			accountStore.Set([]byte(balance.Denom), amount)

			denomPrefixStore, ok := denomPrefixStores[balance.Denom]
			if !ok {
				denomPrefixStore = k.getDenomAddressPrefixStore(ctx, balance.Denom)
				denomPrefixStores[balance.Denom] = denomPrefixStore
			}

			// Store a reverse index from denomination to account address with a
			// sentinel value.
			denomAddrKey := address.MustLengthPrefix(addr)
			if !denomPrefixStore.Has(denomAddrKey) {
				denomPrefixStore.Set(denomAddrKey, []byte{0})
			}
		}
	}

	return nil
}
