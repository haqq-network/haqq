package types

import (
	_ "embed"
	"encoding/json"

	sdkmath "cosmossdk.io/math"
)

// Embed prices json file to the executable binary. Needed when importing as dependency.
//
//go:embed prices.json
var pricesJSON []byte

type PriceLevel struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Price string `json:"price"`
}

type PriceLevels []PriceLevel

var Prices PriceLevels

func init() {
	err := json.Unmarshal(pricesJSON, &Prices)
	if err != nil {
		panic(err)
	}

	// Validate all price levels eagerly so that any invalid numeric strings
	// cause a panic at startup rather than at runtime during tx execution.
	for _, pl := range Prices {
		pl.FromAmount()
		pl.ToAmount()
		pl.UnitPrice()
	}
}

func (pl PriceLevel) FromAmount() sdkmath.Int {
	amt, ok := sdkmath.NewIntFromString(pl.From)
	if !ok {
		// should never happen as we know the original data
		panic("invalid price level start amount")
	}

	return amt
}

func (pl PriceLevel) ToAmount() sdkmath.Int {
	amt, ok := sdkmath.NewIntFromString(pl.To)
	if !ok {
		// should never happen as we know the original data
		panic("invalid price level end amount")
	}

	return amt
}

func (pl PriceLevel) UnitPrice() sdkmath.LegacyDec {
	amt, err := sdkmath.LegacyNewDecFromStr(pl.Price)
	if err != nil {
		// should never happen as we know the original data
		panic("invalid price")
	}

	return amt
}
