package ethiq_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/haqq-network/haqq/encoding"
	"github.com/haqq-network/haqq/x/ethiq"
)

// TestValidateGenesisIncompleteParams checks the module-level entry point that operators hit
// with `haqqd validate-genesis`.
//
// A genesis document that omits an amount leaves the zero value of sdkmath.Int, which wraps a
// nil big.Int. Validation must report that as an error; before the nil guard it dereferenced
// the nil pointer and took the whole command down with a runtime panic.
func TestValidateGenesisIncompleteParams(t *testing.T) {
	encCfg := encoding.MakeConfig()

	testCases := []struct {
		name        string
		genesis     string
		expErr      bool
		errContains string
	}{
		{
			name:    "success - complete genesis",
			genesis: `{"params":{"enabled":true,"min_mint_per_tx":"1","max_mint_per_tx":"100","max_supply":"1000"},"total_burned_amount":{"denom":"aISLM","amount":"0"}}`,
		},
		{
			name:        "fail - amounts omitted",
			genesis:     `{"params":{"enabled":true},"total_burned_amount":{"denom":"aISLM","amount":"0"}}`,
			expErr:      true,
			errContains: "parameter must be set",
		},
		{
			name:        "fail - max_supply omitted",
			genesis:     `{"params":{"enabled":true,"min_mint_per_tx":"1","max_mint_per_tx":"100"},"total_burned_amount":{"denom":"aISLM","amount":"0"}}`,
			expErr:      true,
			errContains: "parameter must be set",
		},
		{
			name:        "fail - max_mint_per_tx above max_supply",
			genesis:     `{"params":{"enabled":true,"min_mint_per_tx":"1","max_mint_per_tx":"1001","max_supply":"1000"},"total_burned_amount":{"denom":"aISLM","amount":"0"}}`,
			expErr:      true,
			errContains: "must be less or equal to max_supply",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("ValidateGenesis panicked instead of returning an error: %v", r)
				}
			}()

			err := ethiq.AppModuleBasic{}.ValidateGenesis(
				encCfg.Codec,
				encCfg.TxConfig,
				json.RawMessage(tc.genesis),
			)

			if !tc.expErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Fatalf("expected error containing %q, got: %v", tc.errContains, err)
			}
		})
	}
}
