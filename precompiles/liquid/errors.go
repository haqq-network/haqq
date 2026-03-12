package liquid

const (
	// ErrInvalidSender is raised when the sender address is not valid.
	ErrInvalidSender = "invalid sender address: %v"
	// ErrInvalidReceiver is raised when the receiver address is not valid.
	ErrInvalidReceiver = "invalid receiver address: %v"
	// ErrInvalidDenom is raised when the provided denom cannot be unpacked.
	ErrInvalidDenom = "invalid denom: %v"
	// ErrInvalidAmount is raised when the amount cannot be unpacked.
	ErrInvalidAmount = "invalid amount: %v"
	// ErrDifferentOriginFromSender is raised when the origin is different from the sender.
	ErrDifferentOriginFromSender = "origin address %s must match sender address %s"
	// ErrAuthzDoesNotExistOrExpired is raised when the authorization does not exist or is expired.
	ErrAuthzDoesNotExistOrExpired = "authorization to %s for address %s does not exist or is expired"
	// ErrRedeemRequiresAuthorization is raised when redeem is called without authorization.
	ErrRedeemRequiresAuthorization = "redeem can only be executed via authorization"
)
