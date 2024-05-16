package types

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// DefaultMinimumLiquidationAmountISLM default parameter value in ISLM
const DefaultMinimumLiquidationAmountISLM = 1000

// AttoMultiplier 10^18
var AttoMultiplier = math.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

// DefaultMinimumLiquidationAmount default atto parameter value
var (
	DefaultMinimumLiquidationAmount = math.NewInt(DefaultMinimumLiquidationAmountISLM).Mul(AttoMultiplier)
	DefaultEnableLiquidVesting      = true
)

// ParamStoreKeyMinimumLiquidationAmount Parameter store keys
var (
	ParamStoreKeyMinimumLiquidationAmount = []byte("MinimumLiquidationAmount")
	ParamStoreKeyEnableLiquidVesting      = []byte("EnableLiquidVesting")
)

func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(
	minimumLiquidationAmount math.Int,
	enableLiquidVesting bool,
) Params {
	return Params{
		MinimumLiquidationAmount: minimumLiquidationAmount,
		EnableLiquidVesting:      enableLiquidVesting,
	}
}

func DefaultParams() Params {
	return Params{
		MinimumLiquidationAmount: DefaultMinimumLiquidationAmount,
		EnableLiquidVesting:      DefaultEnableLiquidVesting,
	}
}

// ParamSetPairs Implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyMinimumLiquidationAmount, &p.MinimumLiquidationAmount, validateMathIntPositive),
		paramtypes.NewParamSetPair(ParamStoreKeyEnableLiquidVesting, &p.EnableLiquidVesting, validateBool),
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

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func (p Params) Validate() error {
	return validateMathIntPositive(p.MinimumLiquidationAmount)
}
