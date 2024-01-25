package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrLiquidationFailed = sdkerrors.Register(ModuleName, 1102, "liquidation failed")
	ErrDenomNotFound     = sdkerrors.Register(ModuleName, 1103, "denom not found")
)
