package types

import (
	errorsmod "cosmossdk.io/errors"
)

// RootCodespace is the codespace for all errors defined in this package
const RootCodespace = "haqq"

// ErrInvalidChainID returns an error resulting from an invalid chain ID.
var ErrInvalidChainID = errorsmod.Register(RootCodespace, 3, "invalid chain ID")
