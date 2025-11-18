package bank

import (
	"fmt"

	"cosmossdk.io/core/appmodule"
    sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/bank/exported"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	haqqbankkeeper "github.com/haqq-network/haqq/x/bank/keeper"
)

var (
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
	_ module.HasGenesis          = AppModule{}
	_ module.HasServices         = AppModule{}
	_ module.HasInvariants       = AppModule{}

	_ appmodule.AppModule = AppModule{}
)

type AppModuleBasic struct {
	*bank.AppModuleBasic
}

// AppModule represents a wrapper around the Cosmos SDK staking module AppModule and
// the Evmos custom staking module keeper.
type AppModule struct {
	*bank.AppModule

	keeper        *haqqbankkeeper.Keeper
	accountKeeper banktypes.AccountKeeper

	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace exported.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	cdc codec.Codec,
	k *haqqbankkeeper.Keeper,
	ak banktypes.AccountKeeper,
	ss exported.Subspace,
) AppModule {
	am := bank.NewAppModule(cdc, k.BaseKeeper, ak, ss)
	return AppModule{
		AppModule:      &am,
		keeper:         k,
		accountKeeper:  ak,
		legacySubspace: ss,
	}
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	banktypes.RegisterMsgServer(cfg.MsgServer(), bankkeeper.NewMsgServerImpl(am.keeper.BaseKeeper))
	banktypes.RegisterQueryServer(cfg.QueryServer(), am.keeper.BaseKeeper)

	m := bankkeeper.NewMigrator(am.keeper.BaseKeeper, am.legacySubspace)
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

// RegisterInvariants registers module invariants (delegates to embedded bank module).
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}
