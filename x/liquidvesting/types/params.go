package types

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// DefaultMinimumLiquidationAmount Default parameters 1 * 10^24
var DefaultMinimumLiquidationAmount = math.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(21), nil))

// ParamStoreKeyMinimumLiquidationAmount Parameter store keys
var ParamStoreKeyMinimumLiquidationAmount = []byte("ParamStoreKeyMinimumLiquidationAmount")

func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(minimumLiquidationAmount math.Int) Params {
	return Params{MinimumLiquidationAmount: minimumLiquidationAmount}
}

func DefaultParams() Params {
	return Params{MinimumLiquidationAmount: DefaultMinimumLiquidationAmount}
}

// ParamSetPairs Implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyMinimumLiquidationAmount, &p.MinimumLiquidationAmount, validateMathIntPositive),
	}
}

func validateMathIntPositive(i interface{}) error {
	n, ok := i.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if !n.IsPositive() {
		return fmt.Errorf("parameter value must be positive, got: %s", n)
	}

	return nil
}

func (p Params) Validate() error {
	return validateMathIntPositive(p.MinimumLiquidationAmount)
}
