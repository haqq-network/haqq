package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	// BaseDenom is the base denomination for aHAQQ token
	BaseDenom = "aHAQQ"
	// DisplayDenom is the display denomination for HAQQ token (with exponent 18)
	DisplayDenom = "HAQQ"
)

// Parameter store keys
var (
	ParamStoreKeyEnabled      = []byte("Enabled")
	ParamStoreKeyMinMintPerTx = []byte("MinMintPerTx")
	ParamStoreKeyMaxMintPerTx = []byte("MaxMintPerTx")
	ParamStoreKeyMaxSupply    = []byte("MaxSupply")
)

func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func DefaultParams() Params {
	return Params{
		Enabled:      true,
		MinMintPerTx: sdkmath.OneInt().MulRaw(1e18),             // 1 * 10^18
		MaxMintPerTx: sdkmath.OneInt().MulRaw(1e18).MulRaw(1e8), // 1 * 10^8 * 10^18 = 100m HAQQ
		MaxSupply:    sdkmath.OneInt().MulRaw(1e18).MulRaw(1e8), // 1 * 10^8 * 10^18 = 100m HAQQ
	}
}

// ParamSetPairs Implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyEnabled, &p.Enabled, validateBool),
		paramtypes.NewParamSetPair(ParamStoreKeyMinMintPerTx, &p.MinMintPerTx, validateInt),
		paramtypes.NewParamSetPair(ParamStoreKeyMaxMintPerTx, &p.MaxMintPerTx, validateInt),
		paramtypes.NewParamSetPair(ParamStoreKeyMaxSupply, &p.MaxSupply, validateInt),
	}
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
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
	if err := validateInt(p.MinMintPerTx); err != nil {
		return err
	}
	if err := validateInt(p.MaxMintPerTx); err != nil {
		return err
	}
	if err := validateInt(p.MaxSupply); err != nil {
		return err
	}

	// Validate that MinMintPerTx < MaxMintPerTx
	if p.MinMintPerTx.GTE(p.MaxMintPerTx) {
		return fmt.Errorf("min_mint_per_tx (%s) must be less than max_mint_per_tx (%s)", p.MinMintPerTx, p.MaxMintPerTx)
	}

	// Validate that MinMintPerTx < MaxSupply
	if p.MinMintPerTx.GTE(p.MaxSupply) {
		return fmt.Errorf("min_mint_per_tx (%s) must be less than max_supply (%s)", p.MinMintPerTx, p.MaxSupply)
	}

	// Validate that MaxMintPerTx <= MaxSupply
	if p.MaxMintPerTx.GT(p.MaxSupply) {
		return fmt.Errorf("max_mint_per_tx (%s) must be less or equal to max_supply (%s)", p.MaxMintPerTx, p.MaxSupply)
	}

	return nil
}
