// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package distribution

const (
	// ErrSetWithdrawAddrAuth is raised when no authorization to set the withdraw address exists.
	ErrSetWithdrawAddrAuth = "set withdrawer address authorization for address %s does not exist"
	// ErrWithdrawDelRewardsAuth is raised when no authorization to withdraw delegation rewards exists.
	ErrWithdrawDelRewardsAuth = "withdraw delegation rewards authorization for address %s does not exist"
	// ErrWithdrawValCommissionAuth is raised when no authorization to withdraw validator commission exists.
	ErrWithdrawValCommissionAuth = "withdraw validator commission authorization for address %s does not exist"
	// ErrDifferentValidator is raised when the origin address is not the same as the validator address.
	ErrDifferentValidator = "origin address %s is not the same as validator address %s"
	// ErrCallerNotDelegator is raised when a caller other than the delegator itself tries to
	// change the delegator's withdraw address. x/distribution has no authorization type, so
	// there is nothing a delegator could grant for this and tx.origin is not accepted.
	ErrCallerNotDelegator = "caller address %s is not the delegator address %s: redirecting a reward stream requires a direct call from the delegator"
	// ErrCallerNotDepositor is raised when a caller other than the depositor itself tries to
	// fund the community pool from the depositor's balance.
	ErrCallerNotDepositor = "caller address %s is not the depositor address %s: funding the community pool requires a direct call from the depositor"
	// ErrWithdrawAddressLength is raised when a withdraw address set through the precompile is
	// not 20 bytes long. Longer Cosmos addresses are legitimate on the Cosmos path, but they
	// have no EVM representation, so rewards sent there are unrecoverable from the EVM side.
	ErrWithdrawAddressLength = "withdraw address %s must be a 20-byte address"
	// ErrValidatorAddressLength is raised when the validator operator address passed to the
	// precompile does not decode to 20 bytes. The EVM address derived from it is what the
	// authorization check compares the caller and the origin against, so it has to be the
	// address the input actually encodes rather than its trailing 20 bytes.
	ErrValidatorAddressLength = "validator address %s must decode to a 20-byte address"
)
