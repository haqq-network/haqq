package v2

import (
	transferv2 "github.com/cosmos/ibc-go/v10/modules/apps/transfer/v2"
	"github.com/cosmos/ibc-go/v10/modules/core/api"
	"github.com/haqq-network/haqq/x/ibc/transfer/keeper"
)

var _ api.IBCModule = (*IBCModule)(nil)

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(k keeper.Keeper) *IBCModule {
	transferModule := transferv2.NewIBCModule(*k.Keeper)
	return &IBCModule{
		IBCModule: transferModule,
	}
}

type IBCModule struct {
	*transferv2.IBCModule
}
