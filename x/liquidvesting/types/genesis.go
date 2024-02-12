package types

func NewGenesisState(
	params Params,
) GenesisState {
	return GenesisState{
		Params: params,
	}
}

func DefaultGenesisState() *GenesisState {
	params := DefaultParams()
	return &GenesisState{
		Params: params,
	}
}

// Validate genesis state
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
