package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

var ErrABIPack = errorsmod.Register(ModuleName, 9, "contract ABI pack failed")
var ErrCACAlreadyMinted = errorsmod.Register(ModuleName, 10, "CAC already minted")
var ErrCACNotMinted = errorsmod.Register(ModuleName, 11, "CAC is not minted")
