package ethiq

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/ethiq/keeper"
	"github.com/haqq-network/haqq/x/ethiq/types"
)

// InitGenesis initializes the ethiq module's state from a provided genesis state
func InitGenesis(ctx sdk.Context, k keeper.Keeper, ak types.AccountKeeper, genState types.GenesisState) {
	// Set params
	k.SetParams(ctx, genState.Params)

	// Set total burned amount
	if !genState.TotalBurnedAmount.IsZero() {
		k.SetTotalBurnedAmount(ctx, genState.TotalBurnedAmount)
	}

	// Ensure metadata is set up
	if err := k.EnsureEthiqMetadata(ctx); err != nil {
		panic(err)
	}

	// Ensure ERC20 registration (only if module is enabled)
	if genState.Params.Enabled {
		if err := k.EnsureEthiqERC20Registration(ctx); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the ethiq module's exported genesis state
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params:            k.GetParams(ctx),
		TotalBurnedAmount: k.GetTotalBurnedAmount(ctx),
	}
}

