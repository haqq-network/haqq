package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrLiquidationFailed = sdkerrors.Register(ModuleName, 1102, "liquidation failed")
	ErrRedeemFailed      = sdkerrors.Register(ModuleName, 1103, "redeem failed")
	ErrDenomNotFound     = sdkerrors.Register(ModuleName, 1104, "denom not found")
)
