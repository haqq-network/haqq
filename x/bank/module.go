package bank

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	haqqbankkeeper "github.com/haqq-network/haqq/x/bank/keeper"
	erc20keeper "github.com/haqq-network/haqq/x/erc20/keeper"
)

type AppModule struct {
	bank.AppModule

	keeper      bankkeeper.Keeper
	erc20keeper erc20keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(module bank.AppModule, keeper bankkeeper.Keeper, erc20keeper erc20keeper.Keeper) AppModule {
	return AppModule{
		AppModule:   module,
		keeper:      keeper,
		erc20keeper: erc20keeper,
	}
}

// LegacyQuerierHandler returns the bank module sdk.Querier.
func (am AppModule) LegacyQuerierHandler(legacyQuerierCdc *codec.LegacyAmino) sdk.Querier {
	return haqqbankkeeper.NewQuerier(am.keeper, am.erc20keeper, legacyQuerierCdc)
}
