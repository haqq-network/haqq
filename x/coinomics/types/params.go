package types

import (
	"cosmossdk.io/math"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var DefaultMintDenom = "aISLM"

// Parameter store keys
var (
	ParamStoreKeyMintDenom         = []byte("ParamStoreKeyMintDenom")
	ParamStoreKeyEnableCoinomics   = []byte("ParamStoreKeyEnableCoinomics")
	ParamStoreKeyRewardCoefficient = []byte("ParamStoreKeyRewardCoefficient")
)

func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(
	mintDenom string,
	rewardsCoefficient math.LegacyDec,
	enableCoinomics bool,
) Params {
	return Params{
		MintDenom:         mintDenom,
		RewardCoefficient: rewardsCoefficient,
		EnableCoinomics:   enableCoinomics,
	}
}

func DefaultParams() Params {
	return Params{
		MintDenom:         DefaultMintDenom,
		RewardCoefficient: math.LegacyNewDecWithPrec(78, 1),
		EnableCoinomics:   true,
	}
}

// Implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyMintDenom, &p.MintDenom, validateMintDenom),
		paramtypes.NewParamSetPair(ParamStoreKeyRewardCoefficient, &p.RewardCoefficient, validateRewardCoefficient),
		paramtypes.NewParamSetPair(ParamStoreKeyEnableCoinomics, &p.EnableCoinomics, validateBool),
	}
}

func validateMintDenom(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if strings.TrimSpace(v) == "" {
		return errors.New("mint denom cannot be blank")
	}

	return sdk.ValidateDenom(v)
}

func validateRewardCoefficient(i interface{}) error {
	_, ok := i.(math.LegacyDec)

	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
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
	if err := validateMintDenom(p.MintDenom); err != nil {
		return err
	}
	if err := validateRewardCoefficient(p.RewardCoefficient); err != nil {
		return err
	}

	return validateBool(p.EnableCoinomics)
}
