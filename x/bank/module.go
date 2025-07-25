package bank

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/bank/exported"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	haqqbankkeeper "github.com/haqq-network/haqq/x/bank/keeper"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
)

type AppModule struct {
	bank.AppModule

	keeper      bankkeeper.Keeper
	wKeeper     haqqbankkeeper.WrappedBaseKeeper
	erc20keeper haqqbankkeeper.ERC20Keeper
	subspace    exported.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(module bank.AppModule, keeper bankkeeper.Keeper, evmKeeper erc20types.EVMKeeper, erc20keeper haqqbankkeeper.ERC20Keeper, accountKeeper haqqbankkeeper.AccountKeeper, ss exported.Subspace) AppModule {
	wrappedBankKeeper := haqqbankkeeper.NewWrappedBaseKeeper(keeper, evmKeeper, erc20keeper, accountKeeper)
	return AppModule{
		AppModule:   module,
		keeper:      keeper,
		wKeeper:     wrappedBankKeeper,
		erc20keeper: erc20keeper,
		subspace:    ss,
	}
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	banktypes.RegisterMsgServer(cfg.MsgServer(), haqqbankkeeper.NewMsgServerImpl(am.wKeeper))
	banktypes.RegisterQueryServer(cfg.QueryServer(), am.wKeeper)

	m := bankkeeper.NewMigrator(am.keeper.(bankkeeper.BaseKeeper), am.subspace)
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
