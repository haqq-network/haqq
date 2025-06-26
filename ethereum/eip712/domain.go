package eip712

import (
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// createEIP712Domain creates the typed data domain for the given chainID.
func createEIP712Domain(chainID uint64) apitypes.TypedDataDomain {
	if chainID > math.MaxInt64 {
		panic("chainID too large")
	}

	domain := apitypes.TypedDataDomain{
		Name:              "Cosmos Web3",
		Version:           "1.0.0",
		ChainId:           math.NewHexOrDecimal256(int64(chainID)),
		VerifyingContract: "cosmos",
		Salt:              "0",
	}

	return domain
}
