// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ucdao

const (
	// ErrDifferentOriginFromOwner is raised when the origin address is not the same as the owner address.
	ErrDifferentOriginFromOwner = "origin address %s is not the same as owner address %s"
	// ErrAuthorizationNotFound is raised when the authorization is not found.
	ErrAuthorizationNotFound = "authorization not found for spender %s"
	// ErrInsufficientAllowance is raised when the allowance is insufficient.
	ErrInsufficientAllowance = "insufficient allowance: requested %s, available %s"
	// ErrDecreaseAmountTooBig is raised when the decrease amount is bigger than the allowance.
	ErrDecreaseAmountTooBig = "decrease amount %s is bigger than the allowance %s"
)
