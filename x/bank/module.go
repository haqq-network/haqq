package bank

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	haqqbankkeeper "github.com/haqq-network/haqq/x/bank/keeper"
)

type AppModule struct {
	bank.AppModule

	keeper      bankkeeper.Keeper
	wKeeper     haqqbankkeeper.WrappedBaseKeeper
	erc20keeper haqqbankkeeper.ERC20Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(module bank.AppModule, keeper bankkeeper.Keeper, erc20keeper haqqbankkeeper.ERC20Keeper, accountKeeper haqqbankkeeper.AccountKeeper) AppModule {
	wrappedBankKeeper := haqqbankkeeper.NewWrappedBaseKeeper(keeper, erc20keeper, accountKeeper)
	return AppModule{
		AppModule:   module,
		keeper:      keeper,
		wKeeper:     wrappedBankKeeper,
		erc20keeper: erc20keeper,
	}
}

// LegacyQuerierHandler returns the bank module sdk.Querier.
func (am AppModule) LegacyQuerierHandler(legacyQuerierCdc *codec.LegacyAmino) sdk.Querier {
	return haqqbankkeeper.NewQuerier(am.keeper, am.erc20keeper, legacyQuerierCdc)
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	banktypes.RegisterMsgServer(cfg.MsgServer(), haqqbankkeeper.NewMsgServerImpl(am.wKeeper))
	banktypes.RegisterQueryServer(cfg.QueryServer(), am.wKeeper)

	m := bankkeeper.NewMigrator(am.keeper.(bankkeeper.BaseKeeper))
	if err := cfg.RegisterMigration(banktypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 1 to 2: %v", err))
	}

	if err := cfg.RegisterMigration(banktypes.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 2 to 3: %v", err))
	}
}
