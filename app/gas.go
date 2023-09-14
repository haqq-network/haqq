package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// MinGasPrices defines 20B aISLM as the minimum gas price value on the fee market module.
	MinGasPrices = sdk.NewDec(20_000_000_000)
	// MinGasMultiplier defines the min gas multiplier value on the fee market module.
	// 50% of the leftover gas will be refunded
	MinGasMultiplier = sdk.NewDecWithPrec(5, 1)
)
