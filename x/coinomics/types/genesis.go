package types

import (
	fmt "fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewGenesisState(
	params Params,
	inflation sdk.Dec,
	era uint64,
	eraStartedAtBlock uint64,
	eraTargetMint sdk.Coin,
	eraClosingSupply sdk.Coin,
	maxSupply sdk.Coin,
) GenesisState {
	return GenesisState{
		Params:            params,
		Inflation:         inflation,
		Era:               era,
		EraStartedAtBlock: eraStartedAtBlock,
		EraTargetMint:     eraTargetMint,
		EraClosingSupply:  eraClosingSupply,
		MaxSupply:         maxSupply,
	}
}

func DefaultGenesisState() *GenesisState {
	params := DefaultParams()

	maxSupply := math.NewIntWithDecimal(100_000_000_000, 18)

	return &GenesisState{
		Params:            params,
		Inflation:         sdk.NewDec(0),
		Era:               uint64(0),
		EraStartedAtBlock: uint64(0),
		EraTargetMint:     sdk.NewCoin(params.MintDenom, sdk.NewInt(0)),
		EraClosingSupply:  sdk.NewCoin(params.MintDenom, sdk.NewInt(0)),
		MaxSupply:         sdk.NewCoin(params.MintDenom, maxSupply),
	}
}

// Validate genesis state
func (gs GenesisState) Validate() error {
	if err := validateInflationRate(gs.Inflation); err != nil {
		return err
	}

	if err := validateEraNumber(gs.Era); err != nil {
		return err
	}

	if err := validateEraStartedAtBlock(gs.EraStartedAtBlock); err != nil {
		return err
	}

	if err := validateEraTargetMint(gs.EraTargetMint); err != nil {
		return err
	}

	if err := validateEraClosingSupply(gs.EraClosingSupply); err != nil {
		return err
	}

	if err := validateMaxSupply(gs.MaxSupply); err != nil {
		return err
	}

	return gs.Params.Validate()
}

func validateInflationRate(i interface{}) error {
	_, ok := i.(sdk.Dec)

	if !ok {
		return fmt.Errorf("inflation rate: invalid genesis state type: %T", i)
	}

	return nil
}

func validateEraNumber(i interface{}) error {
	_, ok := i.(uint64)

	if !ok {
		return fmt.Errorf("era number: invalid genesis state type: %T", i)
	}

	return nil
}

func validateEraStartedAtBlock(i interface{}) error {
	_, ok := i.(uint64)

	if !ok {
		return fmt.Errorf("start era block: invalid genesis state type: %T", i)
	}

	return nil
}

func validateEraTargetMint(i interface{}) error {
	targetMint, ok := i.(sdk.Coin)

	if !ok {
		return fmt.Errorf("era mint: invalid genesis state type: %T", i)
	}

	if targetMint.IsNil() {
		return fmt.Errorf("era mint: can't be nil")
	}

	return nil
}

func validateEraClosingSupply(i interface{}) error {
	eraTargetSupply, ok := i.(sdk.Coin)

	if !ok {
		return fmt.Errorf("era closing supply: invalid genesis state type: %T", i)
	}

	if eraTargetSupply.IsNil() {
		return fmt.Errorf("era closing supply: can't be nil")
	}

	return nil
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
