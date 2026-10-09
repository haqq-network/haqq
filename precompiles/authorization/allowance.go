package authorization

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/haqq-network/haqq/precompiles/common"
)

// CheckApprovalArgs encodes "unlimited" as a nil *sdk.Coin (see the MaxUint256
// branch there), which the allowance helpers below cannot act on: increasing or
// decreasing a finite limit by "unlimited" has no meaning, and dereferencing the
// nil coin panics out of the precompile - HandleGasError only recovers
// ErrorOutOfGas and re-raises everything else.
//
// RequireLimitedAmount rejects that sentinel with an error instead. method is the
// ABI method name, used only for the message.
func RequireLimitedAmount(coin *sdk.Coin, method string) error {
	if coin == nil || coin.IsNil() {
		return fmt.Errorf("%s does not support unlimited amount (MaxUint256)", method)
	}
	return nil
}

// AddAllowance returns current + delta, rejecting a result that does not fit in
// sdkmath.Int. Both operands originate from uint256 ABI arguments, so the sum can
// need 257 bits, and sdkmath.Int.Add panics above sdkmath.MaxBitLen. Computing on
// big.Int first turns an out-of-range allowance into an error rather than a panic
// unwinding through the EVM.
func AddAllowance(current, delta sdkmath.Int) (sdkmath.Int, error) {
	sum, overflow := cmn.SafeAdd(current, delta)
	if overflow {
		return sdkmath.Int{}, fmt.Errorf(
			"increased allowance does not fit in %d bits: %s + %s",
			sdkmath.MaxBitLen, current, delta,
		)
	}
	return sdkmath.NewIntFromBigInt(sum), nil
}
