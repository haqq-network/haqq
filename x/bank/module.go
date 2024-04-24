package bank

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/bank/exported"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	haqqbankkeeper "github.com/haqq-network/haqq/x/bank/keeper"
)

type AppModule struct {
	bank.AppModule

	keeper         bankkeeper.Keeper
	legacySubspace exported.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper bankkeeper.Keeper, ak haqqbankkeeper.AccountKeeper, ss exported.Subspace) AppModule {
	return AppModule{
		AppModule:      bank.NewAppModule(cdc, keeper.(haqqbankkeeper.BaseKeeper).BaseKeeper, ak, ss),
		keeper:         keeper,
		legacySubspace: ss,
	}
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	banktypes.RegisterMsgServer(cfg.MsgServer(), haqqbankkeeper.NewMsgServerImpl(am.keeper))
	banktypes.RegisterQueryServer(cfg.QueryServer(), am.keeper)

	base, ok := am.keeper.(haqqbankkeeper.BaseKeeper)
	if !ok {
		panic(fmt.Sprintf("invalid keeper: %T", am.keeper))
	}

	m := bankkeeper.NewMigrator(base.BaseKeeper, am.legacySubspace)
	if err := cfg.RegisterMigration(banktypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 1 to 2: %v", err))
	}

	if err := cfg.RegisterMigration(banktypes.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 2 to 3: %v", err))
	}

	if err := cfg.RegisterMigration(banktypes.ModuleName, 3, m.Migrate3to4); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 3 to 4: %v", err))
	}
}
