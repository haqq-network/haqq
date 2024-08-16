package types

import (
	errorsmod "cosmossdk.io/errors"
)

// errors
var (
	ErrInsufficientVestedCoins   = errorsmod.Register(ModuleName, 2, "insufficient vested coins error")
	ErrVestingLockup             = errorsmod.Register(ModuleName, 3, "vesting lockup error")
	ErrInsufficientUnlockedCoins = errorsmod.Register(ModuleName, 4, "insufficient unlocked coins error")
	ErrApplyShedule              = errorsmod.Register(ModuleName, 5, "failed to apply schedule")
	ErrNotSubjectToClawback      = errorsmod.Register(ModuleName, 6, "account is not subject to clawback vesting")
	ErrNothingToClawback         = errorsmod.Register(ModuleName, 7, "nothing to clawback from the account")
)
