package contractcheck

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/contractcheck/keeper"
	"github.com/haqq-network/haqq/x/contractcheck/types"
)

// InitGenesis import module genesis
func InitGenesis(_ sdk.Context, _ keeper.Keeper, _ types.GenesisState) {

}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{}
}
