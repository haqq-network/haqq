package types

func NewGenesisState() GenesisState {
	return GenesisState{}
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{}
}

// Validate genesis state
func (gs GenesisState) Validate() error {
	return nil
}
