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

// ParamStoreKeyParams is the single parameter store key holding the whole Params set.
//
// The x/params legacy subspace validates one key per ParameterChangeProposal change and
// applies changes one at a time, so per-field keys would let governance set MaxMintPerTx
// above MaxSupply (or MinMintPerTx above MaxMintPerTx) without the cross-field rules in
// Params.Validate ever running. Storing the set under a single key makes every governance
// update carry the complete Params and pass Validate atomically.
//
// NOTE for proposal authors: Subspace.Update loads the stored value before unmarshalling the
// proposal JSON over it, and Enabled is `omitempty`. A proposal that means to disable the
// module has to spell out "enabled":false; omitting the field keeps the current value.
var ParamStoreKeyParams = []byte("Params")

func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func DefaultParams() Params {
	return Params{
		Enabled:      true,
		MinMintPerTx: sdkmath.OneInt(),                          // 1 aHAQQ
		MaxMintPerTx: sdkmath.OneInt().MulRaw(1e18).MulRaw(1e8), // 1 * 10^8 * 10^18 = 100m HAQQ
		MaxSupply:    sdkmath.OneInt().MulRaw(1e18).MulRaw(1e8), // 1 * 10^8 * 10^18 = 100m HAQQ
	}
}

// ParamSetPairs Implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyParams, p, validateParams),
	}
}

// validateParams runs the full Params validation, including the cross-field rules.
// It is the validator the params subspace calls on every set and on every governance update.
func validateParams(i interface{}) error {
	params, ok := i.(Params)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return params.Validate()
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateInt(i interface{}) error {
	val, ok := i.(sdkmath.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	// The zero value of sdkmath.Int wraps a nil big.Int: IsPositive and String both
	// dereference it. A genesis document that omits the field produces exactly that,
	// so the nil case must be rejected before any other method is called on val.
	if val.IsNil() {
		return fmt.Errorf("parameter must be set, got: nil")
	}
	if !val.IsPositive() {
		return fmt.Errorf("parameter must be positive, got: %s", val.String())
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
