package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Validate performs basic validation of supply genesis data returning an
// error for any failed validation criteria.
func (gs GenesisState) Validate() error {
	// TODO Add custom validation logic

	return nil
}

// NewGenesisState creates a new genesis state.
func NewGenesisState(params Params, balances []Balance, total sdk.Coins) *GenesisState {
	return &GenesisState{
		Params:       params,
		Balances:     balances,
		TotalBalance: total,
	}
}

// DefaultGenesisState returns a default dao module genesis state.
func DefaultGenesisState() *GenesisState {
	return NewGenesisState(DefaultParams(), []Balance{}, sdk.Coins{})
}

// GetGenesisStateFromAppState returns x/dao GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}
