package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
	evmkeeper "github.com/haqq-network/haqq/x/evm/keeper"
)

type mockSubspace struct {
	ps evmtypes.Params
}

func newMockSubspace(ps evmtypes.Params) mockSubspace {
	return mockSubspace{ps: ps}
}

func (ms mockSubspace) GetParamSetIfExists(_ sdk.Context, ps evmtypes.LegacyParams) {
	*ps.(*evmtypes.Params) = ms.ps
}

func (suite *KeeperTestSuite) TestMigrations() {
	legacySubspace := newMockSubspace(evmtypes.DefaultParams())
	migrator := evmkeeper.NewMigrator(*suite.app.EvmKeeper, legacySubspace)

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
			err := tc.migrateFunc(suite.ctx)
			suite.Require().NoError(err)
		})
	}
}
