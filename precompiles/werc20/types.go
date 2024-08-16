package werc20

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// EventDepositWithdraw defines the common event data for the WERC20 Deposit
// and Withdraw events.
type EventDepositWithdraw struct {
	// source or destination address
	Address common.Address
	// amount deposited or withdrawn
	Amount *big.Int
}
