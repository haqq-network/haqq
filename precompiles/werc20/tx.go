package werc20

const (
	// DepositMethod defines the ABI method name for the IWERC20 deposit
	// transaction.
	DepositMethod = "deposit"
	// WithdrawMethod defines the ABI method name for the IWERC20 withdraw
	// transaction.
	WithdrawMethod = "withdraw"
)

// Deposit is a no-op and mock function that provides the same interface as the
// WETH contract to support equality between the native coin and its wrapped
// ERC-20 (eg. ISLM and WISLM). It only emits the Deposit event.
func (p Precompile) Deposit() ([]byte, error) {
	return nil, nil
}

// Withdraw is a no-op and mock function that provides the same interface as the
// WETH contract to support equality between the native coin and its wrapped
// ERC-20 (eg. ISLM and WISLM). It only emits the Withdraw event.
func (p Precompile) Withdraw() ([]byte, error) {
	return nil, nil
}
