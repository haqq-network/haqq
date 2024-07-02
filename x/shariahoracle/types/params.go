package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/common"
)

var ParamStoreKeyCACContractAddress = []byte("CACContractAddress")

// NewParams creates a new Params object
func NewParams(
	cacContractAddress string,
) Params {
	return Params{
		CacContractAddress: cacContractAddress,
	}
}

func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns default params for shariahoracle module
func DefaultParams() Params {
	return Params{
		CacContractAddress: common.Address{}.String(),
	}
}

// ParamSetPairs Implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyCACContractAddress, &p.CacContractAddress, ValidateAddress),
	}
}

// ValidateAddress validate string is a valid ethereum address
func ValidateAddress(i interface{}) error {
	addr, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if !common.IsHexAddress(addr) {
		return fmt.Errorf("invalid ethereum address: %s", addr)
	}

	return nil
}

// Validate all params
func (p Params) Validate() error {
	return ValidateAddress(p.CacContractAddress)
}
