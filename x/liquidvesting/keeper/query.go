package keeper

import (
	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

var _ types.QueryServer = Keeper{}
