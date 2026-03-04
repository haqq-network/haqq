package ethiq

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/ethiq/keeper"
	"github.com/haqq-network/haqq/x/ethiq/types"
)

// InitGenesis initializes the ethiq module's state from a provided genesis state
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set params
	k.SetParams(ctx, genState.Params)

	// Set total burned amount
	if !genState.TotalBurnedAmount.IsZero() {
		k.SetTotalBurnedAmount(ctx, genState.TotalBurnedAmount)
	}

	// Set total burned from applications amount
	for _, appID := range genState.ExecutedApplications {
		if !types.IsApplicationExists(appID) {
			panic(fmt.Sprintf("invalid executed application id, id %d not found", appID))
		}

		application, err := types.GetApplicationByID(appID)
		if err != nil {
			panic(err)
		}

		k.SetApplicationAsExecuted(ctx, appID)
		k.AddToTotalBurnedFromApplicationsAmount(ctx, application.BurnAmount.Amount)
	}

	// Ensure metadata is set up
	if err := k.EnsureHaqqMetadata(ctx); err != nil {
		panic(err)
	}

	// Ensure ERC20 registration
	if err := k.EnsureHaqqERC20Registration(ctx); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the ethiq module's exported genesis state
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params:               k.GetParams(ctx),
		TotalBurnedAmount:    k.GetTotalBurnedAmount(ctx),
		ExecutedApplications: k.GetAllExecutedApplicationsIDs(ctx),
	}
}
