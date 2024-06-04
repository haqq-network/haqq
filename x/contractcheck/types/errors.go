package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

var ErrABIPack = errorsmod.Register(ModuleName, 9, "contract ABI pack failed")
