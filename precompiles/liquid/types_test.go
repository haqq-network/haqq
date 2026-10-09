package liquid_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/haqq-network/haqq/precompiles/liquid"
)

// TestNewRedeemMsgRejectsInvalidDenom pins down that a denom coming from calldata is rejected
// with an error rather than a panic.
//
// NewRedeemMsg runs before ValidateBasic, before the origin/sender check and before the authz
// check, so any caller reaches it with no grant and no balance. sdk.NewCoin panics on a denom
// the regex rejects, and that panic escapes cmn.HandleGasError - which re-raises anything that
// is not ErrorOutOfGas - so RunAtomic never reverts and the tx dies outside the normal path.
func TestNewRedeemMsgRejectsInvalidDenom(t *testing.T) {
	from := common.HexToAddress("0x1")
	to := common.HexToAddress("0x2")

	invalid := []struct {
		name  string
		denom string
	}{
		{"empty", ""},
		{"too short", "A"},
		{"leading digit", "1abc"},
		{"path traversal shaped", "../../etc"},
		{"whitespace", "a b c"},
		{"over length", string(make([]byte, 200))},
	}

	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				msg, _, _, err := liquid.NewRedeemMsg([]interface{}{from, to, tc.denom, big.NewInt(1)})
				require.Error(t, err, "invalid denom must be rejected")
				require.Nil(t, msg)
			})
		})
	}
}

// TestNewRedeemMsgAcceptsLiquidDenom keeps the guard from rejecting the denoms the module
// actually issues.
func TestNewRedeemMsgAcceptsLiquidDenom(t *testing.T) {
	from := common.HexToAddress("0x1")
	to := common.HexToAddress("0x2")

	for _, denom := range []string{"aLIQUID0", "aLIQUID42", "aISLM"} {
		msg, _, _, err := liquid.NewRedeemMsg([]interface{}{from, to, denom, big.NewInt(1)})
		require.NoError(t, err, "denom %q must be accepted", denom)
		require.NotNil(t, msg)
		require.Equal(t, denom, msg.Amount.Denom)
	}
}

// TestNewRedeemMsgMaxUint256Amount covers the other value that arrives raw from calldata: a
// uint256 is exactly MaxBitLen wide, so it must not trip the sdkmath bound check.
func TestNewRedeemMsgMaxUint256Amount(t *testing.T) {
	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

	require.NotPanics(t, func() {
		msg, _, _, err := liquid.NewRedeemMsg([]interface{}{
			common.HexToAddress("0x1"), common.HexToAddress("0x2"), "aLIQUID0", maxUint256,
		})
		require.NoError(t, err)
		require.Equal(t, maxUint256.String(), msg.Amount.Amount.String())
	})
}
