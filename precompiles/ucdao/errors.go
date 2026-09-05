package ucdao

const (
	// ErrDecreaseAmountTooBig is raised when the amount by which the allowance should be decreased is greater
	// than the authorization limit.
	ErrDecreaseAmountTooBig = "amount by which the allowance should be decreased is greater than the authorization limit: %s > %s"
	// ErrDifferentOriginFromSender is raised when the origin address is not the same as the sender address.
	ErrDifferentOriginFromSender = "origin address %s is not the same as sender address %s"
	// ErrTransferOwnershipNotDelegatable is raised when transferOwnership is invoked from a
	// contract. The message moves the whole escrow and carries no amount, so no spend limit
	// can be expressed for it and ucDAO registers no authorization type for it. Use
	// transferOwnershipWithAmount, which is delegatable.
	ErrTransferOwnershipNotDelegatable = "transferOwnership cannot be called on behalf of another account: " +
		"it transfers the entire escrow and has no authorization type; " +
		"use transferOwnershipWithAmount with a grant for %s"
)
