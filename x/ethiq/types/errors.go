package types

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrModuleDisabled       = sdkerrors.Register(ModuleName, 1, "module is disabled")
	ErrInvalidAddress       = sdkerrors.Register(ModuleName, 2, "invalid address")
	ErrInvalidAmount        = sdkerrors.Register(ModuleName, 3, "invalid amount")
	ErrInsufficientFunds    = sdkerrors.Register(ModuleName, 4, "insufficient funds")
	ErrCalculationFailed    = sdkerrors.Register(ModuleName, 5, "calculation failed")
	ErrInvalidFundsSource   = sdkerrors.Register(ModuleName, 6, "invalid funds source")
	ErrInvalidApplicationID = sdkerrors.Register(ModuleName, 7, "invalid application ID")
	ErrParseApplication     = sdkerrors.Register(ModuleName, 8, "failed to parse application")
	ErrBurnCoins            = sdkerrors.Register(ModuleName, 9, "failed to burn coins")
	ErrMintCoins            = sdkerrors.Register(ModuleName, 10, "failed to mint coins")
)
