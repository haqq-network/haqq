package keeper_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/x/evm/statedb"
)

// TestCommitEmitsBankEventsOnce guards against commitWithCtx promoting the
// events of its staging branch twice: CacheContext's write() already re-emits
// them on the parent, so emitting them by hand as well doubled every bank event
// a commit produced, and indexers counted each mint and transfer twice.
func (suite *KeeperTestSuite) TestCommitEmitsBankEventsOnce() {
	countByType := func(events sdk.Events) map[string]int {
		counts := make(map[string]int)
		for _, e := range events {
			counts[e.Type]++
		}
		return counts
	}
	// one mint into the evm module and one send from it to the account
	expected := map[string]int{"coinbase": 1, "coin_received": 2, "coin_spent": 1, "transfer": 1}

	testCases := []struct {
		name      string
		viaFlush  bool
		recipient common.Address
	}{
		{"plain commit", false, common.BigToAddress(big.NewInt(0xabcdef))},
		{"flush into the precompile cache context, then commit", true, common.BigToAddress(big.NewInt(0xabcdee))},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			k := suite.network.App.EvmKeeper
			ctx := suite.network.GetContext().WithEventManager(sdk.NewEventManager())

			db := statedb.New(ctx, k, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash())))
			db.AddBalance(tc.recipient, big.NewInt(1000))
			if tc.viaFlush {
				_, err := db.GetCacheContext()
				suite.Require().NoError(err)
				suite.Require().NoError(db.CommitWithCacheCtx())
			}
			suite.Require().NoError(db.Commit())

			counts := countByType(ctx.EventManager().Events())
			for typ, n := range expected {
				suite.Require().Equal(n, counts[typ], "%s events", typ)
			}
		})
	}
}
