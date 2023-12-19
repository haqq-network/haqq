package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewGenesisState(
	params Params,
	maxSupply sdk.Coin,
) GenesisState {
	return GenesisState{
		Params:      params,
		PrevBlockTs: math.NewInt(0),
		MaxSupply:   maxSupply,
	}
}

func DefaultGenesisState() *GenesisState {
	params := DefaultParams()

	maxSupply := math.NewIntWithDecimal(100_000_000_000, 18)

	return &GenesisState{
		Params:      params,
		PrevBlockTs: math.NewInt(0),
		MaxSupply:   sdk.NewCoin(params.MintDenom, maxSupply),
	}
}

// Validate genesis state
func (gs GenesisState) Validate() error {
	if err := validateMaxSupply(gs.MaxSupply); err != nil {
		return err
	}

	return gs.Params.Validate()
}

func validateMaxSupply(i interface{}) error {
	totalTargetSupply, ok := i.(sdk.Coin)

	if !ok {
		return fmt.Errorf("max supply: invalid genesis state type: %T", i)
	}

	if totalTargetSupply.IsNil() {
		return fmt.Errorf("max supply: can't be nil")
	}

	return nil
}
