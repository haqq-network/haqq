package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrInvalidKey     = sdkerrors.Register(ModuleName, 1, "invalid key")
	ErrModuleDisabled = sdkerrors.Register(ModuleName, 2, "module is disabled")
	ErrInvalidDenom   = sdkerrors.Register(ModuleName, 3, "invalid denom")
)
