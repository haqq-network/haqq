package types

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/utils"
)

// DefaultGenesisState returns a default ethiq module genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:            DefaultParams(),
		TotalBurnedAmount: sdk.NewCoin(utils.BaseDenom, sdkmath.ZeroInt()),
	}
}

// Validate performs basic validation of genesis data returning an
// error for any failed validation criteria.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	if gs.TotalBurnedAmount.Denom != utils.BaseDenom {
		return ErrInvalidAmount
	}

	return nil
}
