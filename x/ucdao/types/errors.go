package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidKey        = errorsmod.Register(ModuleName, 1, "invalid key")
	ErrModuleDisabled    = errorsmod.Register(ModuleName, 2, "module is disabled")
	ErrInvalidDenom      = errorsmod.Register(ModuleName, 3, "invalid denom")
	ErrNotEligible       = errorsmod.Register(ModuleName, 4, "not eligible")
	ErrInvalidRatio      = errorsmod.Register(ModuleName, 5, "invalid ratio")
	ErrInsufficientFunds = errorsmod.Register(ModuleName, 6, "insufficient funds")
)
