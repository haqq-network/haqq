package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	v2 "github.com/haqq-network/haqq/x/ucdao/migrations/v2"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper BaseKeeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper BaseKeeper) Migrator {
	return Migrator{
		keeper: keeper,
	}
}

// Migrate1to2 migrates the store from consensus version 1 to 2
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.Migrate1to2(ctx, m.keeper.storeKey, m.keeper.bk, m.keeper.cdc)
}
