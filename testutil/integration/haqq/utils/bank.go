package utils

import (
	"fmt"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/haqq-network/haqq/testutil/integration/common/grpc"
)

// CheckBalances checks that the given accounts have the expected balances and
// returns an error if that is not the case.
func CheckBalances(handler grpc.Handler, balances []banktypes.Balance) error {
	for _, balance := range balances {
		addr := balance.GetAddress()
		for _, coin := range balance.GetCoins() {
			balance, err := handler.GetBalance(addr, coin.Denom)
			if err != nil {
				return err
			}

			if !balance.Balance.IsEqual(coin) {
				return fmt.Errorf(
					"expected balance %s, got %s for address %s",
					coin, balance.Balance, addr,
				)
			}
		}
	}

	return nil
}
