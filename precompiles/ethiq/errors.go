package ethiq

const (
	// ErrCalculationFailed is raised when calculation has failed
	ErrCalculationFailed = "calculation failed"
	// ErrInvalidSender is raised when the sender is invalid.
	ErrInvalidSender = "invalid sender: %s"
	// ErrInvalidReceiver is raised when the receiver is invalid.
	ErrInvalidReceiver = "invalid receiver: %s"
	// ErrInvalidApplicationID is raised when the given application ID is invalid.
	ErrInvalidApplicationID = "invalid application id: %s"
	// ErrDifferentOriginFromSender is raised when the origin address is not the same as the sender address.
	ErrDifferentOriginFromSender = "origin address %s is not the same as sender address %s"
)
