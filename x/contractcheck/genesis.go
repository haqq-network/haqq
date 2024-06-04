package contractcheck

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/haqq-network/haqq/x/contractcheck/keeper"
	"github.com/haqq-network/haqq/x/contractcheck/types"
)

// InitGenesis import module genesis
func InitGenesis(ctx sdk.Context,
	_ keeper.Keeper,
	accountKeeper authkeeper.AccountKeeper,
	_ types.GenesisState,
) {
	// ensure erc20 module account is set on genesis
	if acc := accountKeeper.GetModuleAccount(ctx, types.ModuleName); acc == nil {
		// NOTE: shouldn't occur
		panic("the erc20 module account has not been set")
	}
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(_ sdk.Context, _ keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{}
}
