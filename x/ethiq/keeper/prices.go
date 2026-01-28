package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type priceLevel string

// priceLevels determine the price levels of total burned original coins in ISLM.
// Price is a 2 powered by key of current level:
//   - level 0, total burnt ISLM is less than 10000000, HAQQ price is 2^0 = 1
//   - level 1, total burnt ISLM is less than 2000000, HAQQ price is 2^1 = 2
//   - level 2, total burnt ISLM is less than 3000000, HAQQ price is 2^2 = 4
//   - etc.
//
// NOTE: This is testing numbers. Final price levels will be announced and set up later
var priceLevels = []priceLevel{
	"1000000", // 1
	"2000000", // 2
	"3000000", // 4
}

func (pl priceLevel) Amount() (sdkmath.Int, error) {
	lCoin, err := sdk.ParseCoinNormalized(string(pl) + "ISLM")
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	return lCoin.Amount, nil
}
