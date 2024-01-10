package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrInvalidSigner = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
)
