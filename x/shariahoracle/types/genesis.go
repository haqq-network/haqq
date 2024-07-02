package types

// NewGenesisState creates a new GenesisState object
func NewGenesisState(params Params) GenesisState {
	return GenesisState{
		Params: params,
	}
}

// DefaultGenesisState - default GenesisState for shariah oracle
func DefaultGenesisState() *GenesisState {
	params := DefaultParams()
	return &GenesisState{
		Params: params,
	}
}

// Validate genesis state
func (gs GenesisState) Validate() error {
	err := gs.Params.Validate()
	if err != nil {
		return err
	}

	return nil
}
