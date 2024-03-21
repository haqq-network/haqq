package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	evmkeeper "github.com/haqq-network/haqq/x/evm/keeper"
	"github.com/haqq-network/haqq/x/evm/types"
)

type mockSubspace struct {
	ps types.Params
}

func newMockSubspace(ps types.Params) mockSubspace {
	return mockSubspace{ps: ps}
}

func (ms mockSubspace) GetParamSetIfExists(_ sdk.Context, ps types.LegacyParams) {
	*ps.(*types.Params) = ms.ps
}

func (suite *KeeperTestSuite) TestMigrations() {
	legacySubspace := newMockSubspace(types.DefaultParams())
	migrator := evmkeeper.NewMigrator(*suite.network.App.EvmKeeper, legacySubspace)

	testCases := []struct {
		name        string
		migrateFunc func(ctx sdk.Context) error
	}{
		{
			"Run Migrate3to4",
			migrator.Migrate3to4,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.migrateFunc(suite.network.GetContext())
			suite.Require().NoError(err)
		})
	}
}
