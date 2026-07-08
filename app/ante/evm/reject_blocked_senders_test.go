package evm_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	evmante "github.com/haqq-network/haqq/app/ante/evm"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// buildEthTxFrom builds a MsgEthereumTx (which itself implements sdk.Tx) with the
// sender already set, mimicking the state after EthSigVerificationDecorator runs.
func buildEthTxFrom(from common.Address) *evmtypes.MsgEthereumTx {
	to := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	msg := evmtypes.NewTx(&evmtypes.EvmTxArgs{
		ChainID:  big.NewInt(1),
		Nonce:    0,
		GasLimit: 21000,
		GasPrice: big.NewInt(1),
		Amount:   big.NewInt(0),
		To:       &to,
	})
	msg.From = from.Hex()
	return msg
}

func TestEthRejectBlockedSendersDecorator(t *testing.T) {
	blocked := common.HexToAddress("0x6185eEF136668F4fc5c86C0284868E69e3296545")
	allowed := common.HexToAddress("0x1111111111111111111111111111111111111111")
	// The blacklist is keyed by bech32; derive it from the sender's address bytes.
	blacklist := map[string]bool{sdk.AccAddress(blocked.Bytes()).String(): true}

	testCases := []struct {
		name      string
		blacklist map[string]bool
		from      common.Address
		expReject bool
	}{
		{"nil blacklist - passes", nil, blocked, false},
		{"empty blacklist - passes", map[string]bool{}, blocked, false},
		{"allowed sender - passes", blacklist, allowed, false},
		{"blocked sender - rejected", blacklist, blocked, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dec := evmante.NewEthRejectBlockedSendersDecorator(tc.blacklist)
			nextCalled := false
			next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
				nextCalled = true
				return ctx, nil
			}

			_, err := dec.AnteHandle(sdk.Context{}, buildEthTxFrom(tc.from), false, next)

			if tc.expReject {
				require.Error(t, err)
				require.ErrorIs(t, err, errortypes.ErrUnauthorized)
				require.False(t, nextCalled, "next handler must not run for a blocked tx")
			} else {
				require.NoError(t, err)
				require.True(t, nextCalled, "next handler must run for an allowed tx")
			}
		})
	}
}
