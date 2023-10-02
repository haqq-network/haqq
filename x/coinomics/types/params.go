package types

import (
	"errors"
	"fmt"
	"math"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/haqq-network/haqq/utils"
)

var DefaultMintDenom = utils.BaseDenom

// Parameter store keys
var (
	ParamStoreKeyMintDenom       = []byte("ParamStoreKeyMintDenom")
	ParamStoreKeyBlockPerEra     = []byte("ParamStoreKeyBlockPerEra")
	ParamStoreKeyEnableCoinomics = []byte("ParamStoreKeyEnableCoinomics")
)

func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(
	mintDenom string,
	blockPerEra uint64,
	enableCoinomics bool,
) Params {
	return Params{
		MintDenom:       mintDenom,
		BlocksPerEra:    blockPerEra,
		EnableCoinomics: enableCoinomics,
	}
}

func DefaultParams() Params {
	return Params{
		MintDenom:       DefaultMintDenom,
		BlocksPerEra:    5259600 * 2, // 2 years
		EnableCoinomics: true,
	}
}

// Implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyMintDenom, &p.MintDenom, validateMintDenom),
		paramtypes.NewParamSetPair(ParamStoreKeyBlockPerEra, &p.BlocksPerEra, validateBlockPerEra),
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

func validateBlockPerEra(i interface{}) error {
	v, ok := i.(uint64)

	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	// Check if BlocksPerEra is within the uint64 range
	if v > math.MaxUint64 {
		return errors.New("BlocksPerEra is out of uint64 range")
	}

	if v == 0 {
		return errors.New("block per era must not be zero")
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
	if err := validateBlockPerEra(p.BlocksPerEra); err != nil {
		return err
	}

	return validateBool(p.EnableCoinomics)
}
