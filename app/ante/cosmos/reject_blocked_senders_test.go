package cosmos_test

import (
	"bytes"
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	cosmosante "github.com/haqq-network/haqq/app/ante/cosmos"
)

// mockSigVerifiableTx is a minimal authsigning.SigVerifiableTx used to exercise
// RejectBlockedSendersDecorator without building a full signed tx.
type mockSigVerifiableTx struct {
	signers [][]byte
}

func (m mockSigVerifiableTx) GetMsgs() []sdk.Msg                              { return nil }
func (m mockSigVerifiableTx) GetMsgsV2() ([]protov2.Message, error)           { return nil, nil }
func (m mockSigVerifiableTx) GetSigners() ([][]byte, error)                   { return m.signers, nil }
func (m mockSigVerifiableTx) GetPubKeys() ([]cryptotypes.PubKey, error)       { return nil, nil }
func (m mockSigVerifiableTx) GetSignaturesV2() ([]signing.SignatureV2, error) { return nil, nil }

func TestRejectBlockedSendersDecorator(t *testing.T) {
	blocked := sdk.AccAddress(bytes.Repeat([]byte{0x01}, 20))
	allowed := sdk.AccAddress(bytes.Repeat([]byte{0x02}, 20))
	blacklist := map[string]bool{blocked.String(): true}

	testCases := []struct {
		name      string
		blacklist map[string]bool
		signers   [][]byte
		expReject bool
	}{
		{"nil blacklist - passes", nil, [][]byte{blocked.Bytes()}, false},
		{"empty blacklist - passes", map[string]bool{}, [][]byte{blocked.Bytes()}, false},
		{"allowed signer - passes", blacklist, [][]byte{allowed.Bytes()}, false},
		{"blocked signer - rejected", blacklist, [][]byte{blocked.Bytes()}, true},
		{"blocked co-signer - rejected", blacklist, [][]byte{allowed.Bytes(), blocked.Bytes()}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dec := cosmosante.NewRejectBlockedSendersDecorator(tc.blacklist)
			nextCalled := false
			next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
				nextCalled = true
				return ctx, nil
			}

			_, err := dec.AnteHandle(sdk.Context{}, mockSigVerifiableTx{signers: tc.signers}, false, next)

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
