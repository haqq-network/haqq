package types

import (
	gethtypes "github.com/ethereum/go-ethereum/core/types"
)

func GetTxTypeName(txType int) string {
	switch txType {
	case gethtypes.DynamicFeeTxType:
		return "DynamicFeeTxType"
	case gethtypes.LegacyTxType:
		return "LegacyTxType"
	case gethtypes.AccessListTxType:
		return "AccessListTxType"
	default:
		panic("unknown tx type")
	}
}
