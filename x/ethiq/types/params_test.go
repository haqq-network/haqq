package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
)

// TestParamsValidateRejectsNilInt pins the guard on the zero value of sdkmath.Int.
//
// Int{} wraps a nil big.Int, and both IsPositive and String dereference it. A genesis
// document that omits min_mint_per_tx produces exactly that value, so validation has to
// answer with an error rather than a nil pointer dereference.
func TestParamsValidateRejectsNilInt(t *testing.T) {
	testCases := []struct {
		name   string
		params Params
	}{
		{
			name:   "all amounts unset",
			params: Params{Enabled: true},
		},
		{
			name: "min_mint_per_tx unset",
			params: Params{
				Enabled:      true,
				MaxMintPerTx: sdkmath.NewInt(100),
				MaxSupply:    sdkmath.NewInt(1000),
			},
		},
		{
			name: "max_mint_per_tx unset",
			params: Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(1),
				MaxSupply:    sdkmath.NewInt(1000),
			},
		},
		{
			name: "max_supply unset",
			params: Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(1),
				MaxMintPerTx: sdkmath.NewInt(100),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Validate panicked instead of returning an error: %v", r)
				}
			}()

			err := tc.params.Validate()
			if err == nil {
				t.Fatal("expected an error for params with an unset amount")
			}
			if got := err.Error(); got != "parameter must be set, got: nil" {
				t.Fatalf("unexpected error: %s", got)
			}
		})
	}
}

// TestValidateParamsChecksWholeSet covers the validator the params subspace runs on every
// set and on every governance update: it must apply the cross-field rules, not just the
// per-field ones.
func TestValidateParamsChecksWholeSet(t *testing.T) {
	testCases := []struct {
		name   string
		value  interface{}
		expErr bool
	}{
		{"default params", DefaultParams(), false},
		{
			name: "max_mint_per_tx above max_supply",
			value: Params{
				Enabled:      true,
				MinMintPerTx: sdkmath.NewInt(1),
				MaxMintPerTx: sdkmath.NewInt(1001),
				MaxSupply:    sdkmath.NewInt(1000),
			},
			expErr: true,
		},
		{"wrong type", "not params", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateParams(tc.value)
			if tc.expErr && err == nil {
				t.Fatal("expected an error")
			}
			if !tc.expErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
