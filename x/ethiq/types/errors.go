package types

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrInvalidAddress    = sdkerrors.Register(ModuleName, 1, "invalid address")
	ErrInvalidAmount     = sdkerrors.Register(ModuleName, 2, "invalid amount")
	ErrInsufficientFunds = sdkerrors.Register(ModuleName, 3, "insufficient funds")
	ErrModuleDisabled    = sdkerrors.Register(ModuleName, 4, "module is disabled")
	ErrCalculationFailed = sdkerrors.Register(ModuleName, 5, "calculation failed")
	ErrInfiniteResult    = sdkerrors.Register(ModuleName, 6, "infinite result occurred")
)
