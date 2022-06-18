package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	// upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// BeginBlockForks executes any necessary fork logic based upon the current block height.
func BeginBlockForks(ctx sdk.Context, app *Haqq) {
	// switch ctx.BlockHeight() {
	// default:
	// 	// do nothing
	// 	return
	// }
}
