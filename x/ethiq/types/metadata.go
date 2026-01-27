package types

import banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

// HaqqDenomMetaData HAQQ token metadata with base denom (exponent 0) and display denom (exponent 18)
var HaqqDenomMetaData = banktypes.Metadata{
	Description: "HAQQ is an ecosystem token of Ethiq Network (L2)",
	Base:        BaseDenom,
	DenomUnits: []*banktypes.DenomUnit{
		{
			Denom:    BaseDenom,
			Exponent: 0,
		},
		{
			Denom:    DisplayDenom,
			Exponent: 18,
		},
	},
	Name:    "HAQQ token",
	Symbol:  DisplayDenom,
	Display: DisplayDenom,
}
