// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package keeper

import (
	"fmt"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	haqqtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/x/evm/statedb"
	"github.com/haqq-network/haqq/x/evm/types"
)

var _ statedb.Keeper = &Keeper{}

// ----------------------------------------------------------------------------
// StateDB Keeper implementation
// ----------------------------------------------------------------------------

// GetAccount returns nil if account is not exist, returns error if it's not `EthAccountI`
func (k *Keeper) GetAccount(ctx sdk.Context, addr common.Address) *statedb.Account {
	acct := k.GetAccountWithoutBalance(ctx, addr)
	if acct == nil {
		return nil
	}

	acct.Balance = k.GetBalance(ctx, addr)
	return acct
}

// GetState loads contract state from database.
func (k *Keeper) GetState(ctx sdk.Context, addr common.Address, key common.Hash) common.Hash {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.AddressStoragePrefix(addr))

	value := store.Get(key.Bytes())
	if len(value) == 0 {
		return common.Hash{}
	}

	return common.BytesToHash(value)
}

// GetCode loads contract code from database, implements `statedb.Keeper` interface.
func (k *Keeper) GetCode(ctx sdk.Context, codeHash common.Hash) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixCode)
	return store.Get(codeHash.Bytes())
}

// ForEachStorage iterate contract storage, callback return false to break early
func (k *Keeper) ForEachStorage(ctx sdk.Context, addr common.Address, cb func(key, value common.Hash) bool) {
	store := ctx.KVStore(k.storeKey)
	prefix := types.AddressStoragePrefix(addr)

	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := common.BytesToHash(iterator.Key())
		value := common.BytesToHash(iterator.Value())

		// check if iteration stops
		if !cb(key, value) {
			return
		}
	}
}

// SetBalance update account's balance, compare with current balance first, then decide to mint or burn.
func (k *Keeper) SetBalance(ctx sdk.Context, addr common.Address, amount *big.Int) error {
	cosmosAddr := sdk.AccAddress(addr.Bytes())

	params := k.GetParams(ctx)
	coin := k.bankKeeper.GetBalance(ctx, cosmosAddr, params.EvmDenom)
	balance := coin.Amount.BigInt()
	delta := new(big.Int).Sub(amount, balance)
	switch delta.Sign() {
	case 1:
		// mint
		coins := sdk.NewCoins(sdk.NewCoin(params.EvmDenom, sdkmath.NewIntFromBigInt(delta)))
		if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
			return err
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, cosmosAddr, coins); err != nil {
			return err
		}
	case -1:
		// burn
		coins := sdk.NewCoins(sdk.NewCoin(params.EvmDenom, sdkmath.NewIntFromBigInt(new(big.Int).Neg(delta))))
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, cosmosAddr, types.ModuleName, coins); err != nil {
			return err
		}
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
			return err
		}
	default:
		// not changed
	}
	return nil
}

// SetAccount updates nonce/balance/codeHash together.
func (k *Keeper) SetAccount(ctx sdk.Context, addr common.Address, account statedb.Account) error {
	// update account
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		acct = k.accountKeeper.NewAccountWithAddress(ctx, cosmosAddr)
	}

	if _, isModuleAccount := acct.(sdk.ModuleAccountI); isModuleAccount {
		return k.checkModuleAccountUnchanged(ctx, addr, cosmosAddr, account.Balance)
	}

	if err := acct.SetSequence(account.Nonce); err != nil {
		return err
	}

	codeHash := common.BytesToHash(account.CodeHash)

	if ethAcct, ok := acct.(haqqtypes.EthAccountI); ok {
		if err := ethAcct.SetCodeHash(codeHash); err != nil {
			return err
		}
	}

	k.accountKeeper.SetAccount(ctx, acct)

	if err := k.SetBalance(ctx, addr, account.Balance); err != nil {
		return err
	}

	k.Logger(ctx).Debug(
		"account updated",
		"ethereum-address", addr.Hex(),
		"nonce", account.Nonce,
		"codeHash", codeHash.Hex(),
		"balance", account.Balance,
	)
	return nil
}

// checkModuleAccountUnchanged refuses a commit that would reconcile a Cosmos module
// account's balance against the EVM's view of it, and is a no-op when the two already
// agree.
//
// A module account's balance belongs to its module and moves through bank calls that
// leave no EVM journal entry - which is what every stateful precompile does. The two
// sides also read from different places: StateDB.getStateObject loads balances from
// the transaction context, while a precompile writes through the StateDB's cache
// context, so an account first loaded into the StateDB *after* a precompile moved it
// carries the balance it had before the call.
//
// SetBalance would then treat that stale number as the truth and mint or burn the
// difference. A contract can delegate X and send 1 wei to the bonded pool in the same
// transaction: the pool loads at its pre-delegation balance B, the journal dirties it
// at B+1, and commit burns X-1 out of the pool while the delegation shares survive -
// leaving staking unable to pay out what it still owes.
//
// There is no correct amount to write here, so nothing is written. The EVM cannot
// legitimately author a module account's balance: a module account has no key, so the
// only EVM-side change is a transfer into it, and bank already refuses that because
// module accounts are blocked recipients. This makes the other direction fail the same
// way instead of silently destroying module funds. Nothing that works today reaches
// this error: a module account only enters the dirty set when an EVM transfer targeted
// it, and mirroring a precompile's bank movement into the journal (as the precompiles
// do for the accounts they move) leaves the two sides equal and this check silent.
func (k *Keeper) checkModuleAccountUnchanged(
	ctx sdk.Context,
	addr common.Address,
	cosmosAddr sdk.AccAddress,
	evmBalance *big.Int,
) error {
	bankBalance := k.GetBalance(ctx, addr)
	if bankBalance.Cmp(evmBalance) == 0 {
		return nil
	}

	return errorsmod.Wrapf(
		types.ErrInvalidAccount,
		"%s is a module account: the EVM may not change its balance (evm view %s, bank %s)",
		cosmosAddr, evmBalance, bankBalance,
	)
}

// SetState update contract storage, delete if value is empty.
func (k *Keeper) SetState(ctx sdk.Context, addr common.Address, key common.Hash, value []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.AddressStoragePrefix(addr))
	action := "updated"
	if len(value) == 0 {
		store.Delete(key.Bytes())
		action = "deleted"
	} else {
		store.Set(key.Bytes(), value)
	}
	k.Logger(ctx).Debug(
		fmt.Sprintf("state %s", action),
		"ethereum-address", addr.Hex(),
		"key", key.Hex(),
	)
}

// SetCode set contract code, delete if code is empty.
func (k *Keeper) SetCode(ctx sdk.Context, codeHash, code []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixCode)

	// store or delete code
	action := "updated"
	if len(code) == 0 {
		store.Delete(codeHash)
		action = "deleted"
	} else {
		store.Set(codeHash, code)
	}
	k.Logger(ctx).Debug(
		fmt.Sprintf("code %s", action),
		"code-hash", common.BytesToHash(codeHash).Hex(),
	)
}

// DeleteAccount handles contract's suicide call:
// - clear balance
// - remove code
// - remove states
// - remove auth account
func (k *Keeper) DeleteAccount(ctx sdk.Context, addr common.Address) error {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		return nil
	}

	// NOTE: only Ethereum accounts (contracts) can be selfdestructed
	_, ok := acct.(haqqtypes.EthAccountI)
	if !ok {
		return errorsmod.Wrapf(types.ErrInvalidAccount, "type %T, address %s", acct, addr)
	}

	// clear balance
	if err := k.SetBalance(ctx, addr, new(big.Int)); err != nil {
		return err
	}

	// clear storage
	k.ForEachStorage(ctx, addr, func(key, _ common.Hash) bool {
		k.SetState(ctx, addr, key, nil)
		return true
	})

	// remove auth account
	k.accountKeeper.RemoveAccount(ctx, acct)

	k.Logger(ctx).Debug(
		"account suicided",
		"ethereum-address", addr.Hex(),
		"cosmos-address", cosmosAddr.String(),
	)

	return nil
}
