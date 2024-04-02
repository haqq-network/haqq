package utils

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

func ValidatorConsAddressToHex(valAddress string) common.Address {
	coinbaseAddressBytes := sdk.ConsAddress(valAddress).Bytes()
	return common.BytesToAddress(coinbaseAddressBytes)
}
