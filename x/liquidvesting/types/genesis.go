package types

import (
	"fmt"
)

func NewGenesisState(
	params Params,
	denomCounter uint64,
	denoms []Denom,
) GenesisState {
	return GenesisState{
		Params:       params,
		DenomCounter: denomCounter,
		Denoms:       denoms,
	}
}

func DefaultGenesisState() *GenesisState {
	params := DefaultParams()
	return &GenesisState{
		Params:       params,
		DenomCounter: 0,
		Denoms:       []Denom{},
	}
}

// Validate genesis state
func (gs GenesisState) Validate() error {
	err := gs.Params.Validate()
	if err != nil {
		return err
	}

	err = validateDenoms(gs)
	if err != nil {
		return err
	}

	return nil
}

func validateDenoms(gs GenesisState) error {
	// Check for duplicated index in chain
	counter := gs.GetDenomCounter()
	denomBaseNameMap := make(map[string]struct{})
	for _, elem := range gs.Denoms {

		if elem.EndTime.Before(elem.StartTime) {
			return fmt.Errorf("denom start time cannot be after end time")
		}

		for _, period := range elem.LockupPeriods {
			if period.GetAmount().IsAnyNegative() {
				return fmt.Errorf("denom periods cannot contain negative amount")
			}
		}

		baseName := elem.BaseDenom
		if _, ok := denomBaseNameMap[baseName]; ok {
			return fmt.Errorf("duplicated denom base name for liquid denom")
		}

		denomBaseNameMap[baseName] = struct{}{}

		denomID, err := DenomIDFromBaseName(elem.GetBaseDenom())
		if err != nil {
			return fmt.Errorf("invalid denom base name")
		}

		if denomID >= counter {
			return fmt.Errorf("denom id %s(%d) should be lower or equal than the last id %d",
				elem.GetBaseDenom(), denomID, counter)
		}
	}

	return nil
}
