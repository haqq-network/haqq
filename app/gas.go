package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	MinGasPrices     = sdk.NewDec(20_000_000_000)
	MinGasMultiplier = sdk.NewDecWithPrec(5, 1)
)
