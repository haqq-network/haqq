package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrABIPack            = errorsmod.Register(ModuleName, 9, "contract ABI pack failed")
	ErrCACAlreadyGranted  = errorsmod.Register(ModuleName, 10, "CAC already granted")
	ErrCACNotGranted      = errorsmod.Register(ModuleName, 11, "CAC is not granted")
	ErrInvalidEVMResponse = errorsmod.Register(ModuleName, 12, "invalid EVM response")
)
