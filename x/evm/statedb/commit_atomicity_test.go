package statedb_test

import (
	"errors"
	"math/big"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/haqq-network/haqq/x/evm/statedb"
)

// atomicTestKeeper routes writes through ctx's real KVStore, unlike the
// in-memory-map mocks elsewhere in this package, so a write discarded via
// CacheContext is actually observable as absent.
type atomicTestKeeper struct {
	key     *storetypes.KVStoreKey
	errAddr common.Address
}

var _ statedb.Keeper = &atomicTestKeeper{}

func (k *atomicTestKeeper) store(ctx sdk.Context) storetypes.KVStore { return ctx.KVStore(k.key) }

func (k *atomicTestKeeper) GetAccount(ctx sdk.Context, addr common.Address) *statedb.Account {
	bz := k.store(ctx).Get(addr.Bytes())
	if bz == nil {
		return nil
	}
	return &statedb.Account{Balance: new(big.Int).SetBytes(bz)}
}

func (k *atomicTestKeeper) SetAccount(ctx sdk.Context, addr common.Address, acc statedb.Account) error {
	if addr == k.errAddr {
		return errors.New("blocked")
	}
	k.store(ctx).Set(addr.Bytes(), acc.Balance.Bytes())
	if em := ctx.EventManager(); em != nil {
		em.EmitEvent(sdk.NewEvent(
			"statedb_set_account",
			sdk.NewAttribute("addr", addr.Hex()),
		))
	}
	return nil
}

func (k *atomicTestKeeper) DeleteAccount(ctx sdk.Context, addr common.Address) error {
	if addr == k.errAddr {
		return errors.New("blocked")
	}
	k.store(ctx).Delete(addr.Bytes())
	return nil
}

func (k *atomicTestKeeper) GetState(sdk.Context, common.Address, common.Hash) common.Hash {
	return common.Hash{}
}
func (k *atomicTestKeeper) GetCode(sdk.Context, common.Hash) []byte { return nil }
func (k *atomicTestKeeper) ForEachStorage(sdk.Context, common.Address, func(common.Hash, common.Hash) bool) {
}
func (k *atomicTestKeeper) SetState(sdk.Context, common.Address, common.Hash, []byte) {}
func (k *atomicTestKeeper) SetCode(sdk.Context, []byte, []byte)                       {}

// TestCommitAtomicity commits a dirty set sorted [credit, blocked, debit]. A
// late failure on blocked must discard the whole commit, including credit,
// which a non-atomic commit would already have written.
func TestCommitAtomicity(t *testing.T) {
	credit := common.BigToAddress(big.NewInt(10))
	blocked := common.BigToAddress(big.NewInt(50))
	debit := common.BigToAddress(big.NewInt(90))
	precompileAddr := common.BigToAddress(big.NewInt(1)) // written via cacheCtx, bypassing the journal

	setup := func(name string, errAddr common.Address) *statedb.StateDB {
		key := storetypes.NewKVStoreKey(name)
		tkey := storetypes.NewTransientStoreKey(name + "_t")
		ctx := testutil.DefaultContext(key, tkey).WithEventManager(sdk.NewEventManager())
		return statedb.New(ctx, &atomicTestKeeper{key: key, errAddr: errAddr}, emptyTxConfig)
	}
	seed := func(db *statedb.StateDB) {
		db.AddBalance(credit, big.NewInt(1_000_000))
		db.AddBalance(blocked, big.NewInt(1))
		db.AddBalance(debit, big.NewInt(1))
	}
	// persisted reads through db's own keeper/ctx, bypassing db's in-memory
	// cache, to check what actually reached the real store.
	persisted := func(db *statedb.StateDB, addr common.Address) *statedb.Account {
		return db.Keeper().GetAccount(db.GetContext(), addr)
	}
	// stageViaPrecompile writes directly through the cache context the way a
	// real precompile does, bypassing the journal entirely.
	stageViaPrecompile := func(t *testing.T, db *statedb.StateDB) {
		t.Helper()
		cacheCtx, err := db.GetCacheContext()
		require.NoError(t, err)
		require.NoError(t, db.Keeper().SetAccount(cacheCtx, precompileAddr, statedb.Account{Balance: big.NewInt(7)}))
	}

	t.Run("late failure discards the whole commit", func(t *testing.T) {
		db := setup("fail", blocked)
		seed(db)
		require.Error(t, db.Commit())
		require.Nil(t, persisted(db, credit))
	})

	t.Run("late failure discards precompile-staged writes too", func(t *testing.T) {
		db := setup("fail_precompile", blocked)
		stageViaPrecompile(t, db)
		seed(db)
		require.Error(t, db.Commit())
		require.Nil(t, persisted(db, credit))
		require.Nil(t, persisted(db, precompileAddr))
	})

	t.Run("success still persists everything", func(t *testing.T) {
		db := setup("ok", common.Address{})
		seed(db)
		require.NoError(t, db.Commit())
		require.Equal(t, 0, persisted(db, credit).Balance.Cmp(big.NewInt(1_000_000)))
		require.Equal(t, 0, persisted(db, debit).Balance.Cmp(big.NewInt(1)))
	})

	t.Run("success persists precompile-staged writes too", func(t *testing.T) {
		db := setup("ok_precompile", common.Address{})
		stageViaPrecompile(t, db)
		seed(db)
		require.NoError(t, db.Commit())
		require.Equal(t, 0, persisted(db, credit).Balance.Cmp(big.NewInt(1_000_000)))
		require.Equal(t, 0, persisted(db, precompileAddr).Balance.Cmp(big.NewInt(7)))
	})

	// A reverted precompile replaces writeCache with a snapshot flush. Final
	// Commit still folds remaining dirties into cacheCtx; those writes and
	// their events must reach the parent, while reverted precompile work must not.
	t.Run("reverted precompile keeps later commit writes and events", func(t *testing.T) {
		db := setup("revert_events", common.Address{})
		cacheCtx, err := db.GetCacheContext()
		require.NoError(t, err)

		cacheCtx.EventManager().EmitEvent(sdk.NewEvent("before_precompile"))
		snapshot := db.MultiStoreSnapshot()
		snapshotEvents := append(sdk.Events(nil), cacheCtx.EventManager().Events()...)

		require.NoError(t, db.Keeper().SetAccount(cacheCtx, precompileAddr, statedb.Account{Balance: big.NewInt(7)}))
		cacheCtx.EventManager().EmitEvent(sdk.NewEvent("precompile_work"))

		db.RevertMultiStore(snapshot, snapshotEvents)

		db.AddBalance(credit, big.NewInt(1_000_000))
		require.NoError(t, db.Commit())

		require.Nil(t, persisted(db, precompileAddr))
		require.Equal(t, 0, persisted(db, credit).Balance.Cmp(big.NewInt(1_000_000)))

		parentEvents := db.GetContext().EventManager().Events()
		require.True(t, hasEvent(parentEvents, "before_precompile"))
		require.False(t, hasEvent(parentEvents, "precompile_work"))
		require.True(t, hasEventAttr(parentEvents, "statedb_set_account", "addr", credit.Hex()))
		require.False(t, hasEventAttr(parentEvents, "statedb_set_account", "addr", precompileAddr.Hex()))
	})
}

func hasEvent(events sdk.Events, typ string) bool {
	for _, ev := range events {
		if ev.Type == typ {
			return true
		}
	}
	return false
}

func hasEventAttr(events sdk.Events, typ, key, value string) bool {
	for _, ev := range events {
		if ev.Type != typ {
			continue
		}
		for _, attr := range ev.Attributes {
			if attr.Key == key && attr.Value == value {
				return true
			}
		}
	}
	return false
}
