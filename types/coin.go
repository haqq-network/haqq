package types

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// AttoDenom defines the default coin denomination used in Haqq Network in:
	//
	// - Staking parameters: denomination used as stake in the dPoS chain
	// - Mint parameters: denomination minted due to fee distribution rewards
	// - Governance parameters: denomination used for spam prevention in proposal deposits
	// - Crisis parameters: constant fee denomination used for spam prevention to check broken invariant
	// - EVM parameters: denomination used for running EVM state transitions in Haqq Network.
	AttoDenom string = "aISLM"

	// BaseDenomUnit defines the base denomination unit for Haqq Network.
	// 1 ISLM = 1x10^{BaseDenomUnit} aISLM
	BaseDenomUnit = 18

	// DefaultGasPrice is default gas price for evm transactions
	DefaultGasPrice = 20
)

// PowerReduction defines the default power reduction value for staking
var PowerReduction = sdkmath.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(BaseDenomUnit), nil))

// NewIslmCoin is a utility function that returns an "aISLM" coin with the given sdkmath.Int amount.
// The function will panic if the provided amount is negative.
func NewIslmCoin(amount sdkmath.Int) sdk.Coin {
	return sdk.NewCoin(AttoDenom, amount)
}

// NewIslmDecCoin is a utility function that returns an "aISLM" decimal coin with the given sdkmath.Int amount.
// The function will panic if the provided amount is negative.
func NewIslmDecCoin(amount sdkmath.Int) sdk.DecCoin {
	return sdk.NewDecCoin(AttoDenom, amount)
}

// NewIslmCoinInt64 is a utility function that returns an "aISLM" coin with the given int64 amount.
// The function will panic if the provided amount is negative.
func NewIslmCoinInt64(amount int64) sdk.Coin {
	return sdk.NewInt64Coin(AttoDenom, amount)
}
