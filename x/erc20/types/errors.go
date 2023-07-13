package types

import (
	errorsmod "cosmossdk.io/errors"
)

// errors
var (
	ErrERC20Disabled          = errorsmod.Register(ModuleName, 332, "erc20 module is disabled")
	ErrInternalTokenPair      = errorsmod.Register(ModuleName, 333, "internal ethereum token mapping error")
	ErrTokenPairNotFound      = errorsmod.Register(ModuleName, 334, "token pair not found")
	ErrTokenPairAlreadyExists = errorsmod.Register(ModuleName, 335, "token pair already exists")
	ErrUndefinedOwner         = errorsmod.Register(ModuleName, 336, "undefined owner of contract pair")
	ErrBalanceInvariance      = errorsmod.Register(ModuleName, 337, "post transfer balance invariant failed")
	ErrUnexpectedEvent        = errorsmod.Register(ModuleName, 338, "unexpected event")
	ErrABIPack                = errorsmod.Register(ModuleName, 339, "contract ABI pack failed")
	ErrABIUnpack              = errorsmod.Register(ModuleName, 3310, "contract ABI unpack failed")
	ErrEVMDenom               = errorsmod.Register(ModuleName, 3311, "EVM denomination registration")
	ErrEVMCall                = errorsmod.Register(ModuleName, 3312, "EVM call unexpected error")
	ErrERC20TokenPairDisabled = errorsmod.Register(ModuleName, 3313, "erc20 token pair is disabled")
)
