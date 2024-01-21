package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

var ErrLiquidationFailed = sdkerrors.Register(ModuleName, 1102, "liquidation failed")
