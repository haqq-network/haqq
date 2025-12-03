package keeper

import (
	"math/big"

	"github.com/haqq-network/haqq/x/ethiq/types"
	"github.com/rsned/bigmath"

	sdkmath "cosmossdk.io/math"
)

// powerDec calculates x^power where power is a decimal
// Uses iterative multiplication for integer part and approximation for fractional part
func powerDec(x sdkmath.LegacyDec, power sdkmath.LegacyDec) (sdkmath.LegacyDec, error) {
	// We gonna work with big.Float
	// To ensure deterministic calculations we'll use strict rules:
	// - constant precision for both values
	// - same rounding mode
	// - string representation before start
	xStr := x.String()
	pStr := power.String()

	xFloat, _, err := new(big.Float).SetPrec(18).SetMode(big.ToNearestEven).Parse(xStr, 10)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}

	pFloat, _, err := new(big.Float).SetPrec(18).SetMode(big.ToNearestEven).Parse(pStr, 10)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}

	zFloat := bigmath.Pow(xFloat, pFloat)
	if zFloat.IsInf() {
		return sdkmath.LegacyDec{}, types.ErrInfiniteResult
	}

	return sdkmath.LegacyNewDecFromStr(zFloat.String())
}
