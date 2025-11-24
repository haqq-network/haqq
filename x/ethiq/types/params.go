package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	// BaseDenom is the base denomination for aethiq
	BaseDenom = "aethiq"
	// DisplayDenom is the display denomination for ethiq (with exponent 18)
	DisplayDenom = "ethiq"
	// ISLMBaseDenom is the base denomination for aISLM
	ISLMBaseDenom = "aISLM"
)

// Parameter store keys
var (
	ParamStoreKeyEnabled          = []byte("Enabled")
	ParamStoreKeyStartRate        = []byte("StartRate")
	ParamStoreKeyCurveCoefficient = []byte("CurveCoefficient")
	ParamStoreKeyPowerCoefficient = []byte("PowerCoefficient")
	ParamStoreKeyMinMintPerTx     = []byte("MinMintPerTx")
	ParamStoreKeyMaxMintPerTx     = []byte("MaxMintPerTx")
)

func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func DefaultParams() Params {
	return Params{
		Enabled:          false,
		StartRate:        sdkmath.LegacyNewDec(1),
		CurveCoefficient: sdkmath.LegacyNewDecWithPrec(5, 9),        // 0.000000005
		PowerCoefficient: sdkmath.LegacyNewDecWithPrec(11, 1),       // 1.1
		MinMintPerTx:     sdkmath.OneInt().MulRaw(1e18),             // 1 * 10^18
		MaxMintPerTx:     sdkmath.OneInt().MulRaw(1e18).MulRaw(1e9), // 1 * 10^9 * 10^18
	}
}

// ParamSetPairs Implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyEnabled, &p.Enabled, validateBool),
		paramtypes.NewParamSetPair(ParamStoreKeyStartRate, &p.StartRate, validateDec),
		paramtypes.NewParamSetPair(ParamStoreKeyCurveCoefficient, &p.CurveCoefficient, validateDec),
		paramtypes.NewParamSetPair(ParamStoreKeyPowerCoefficient, &p.PowerCoefficient, validateDec),
		paramtypes.NewParamSetPair(ParamStoreKeyMinMintPerTx, &p.MinMintPerTx, validateInt),
		paramtypes.NewParamSetPair(ParamStoreKeyMaxMintPerTx, &p.MaxMintPerTx, validateInt),
	}
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateDec(i interface{}) error {
	_, ok := i.(sdkmath.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateInt(i interface{}) error {
	_, ok := i.(sdkmath.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func (p Params) Validate() error {
	if err := validateBool(p.Enabled); err != nil {
		return err
	}
	if err := validateDec(p.StartRate); err != nil {
		return err
	}
	if err := validateDec(p.CurveCoefficient); err != nil {
		return err
	}
	if err := validateDec(p.PowerCoefficient); err != nil {
		return err
	}
	if err := validateInt(p.MinMintPerTx); err != nil {
		return err
	}
	if err := validateInt(p.MaxMintPerTx); err != nil {
		return err
	}

	// Validate that MinMintPerTx < MaxMintPerTx
	if p.MinMintPerTx.GTE(p.MaxMintPerTx) {
		return fmt.Errorf("min_mint_per_tx (%s) must be less than max_mint_per_tx (%s)", p.MinMintPerTx, p.MaxMintPerTx)
	}

	// Validate that coefficients are positive
	if p.StartRate.LTE(sdkmath.LegacyZeroDec()) {
		return fmt.Errorf("start_rate must be positive")
	}
	if p.CurveCoefficient.LTE(sdkmath.LegacyZeroDec()) {
		return fmt.Errorf("curve_coefficient must be positive")
	}
	if p.PowerCoefficient.LTE(sdkmath.LegacyZeroDec()) {
		return fmt.Errorf("power_coefficient must be positive")
	}

	return nil
}
