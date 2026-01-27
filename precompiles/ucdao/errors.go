// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ucdao

const (
	// ErrAuthorizationNotFound is raised when the authorization is not found.
	ErrAuthorizationNotFound = "authorization not found for spender %s"
	// ErrInsufficientAllowance is raised when the allowance is insufficient.
	ErrInsufficientAllowance = "insufficient allowance: requested %s, available %s"
	// ErrDecreaseAmountTooBig is raised when the decrease amount is bigger than the allowance.
	ErrDecreaseAmountTooBig = "decrease amount %s is bigger than the allowance %s"
)
