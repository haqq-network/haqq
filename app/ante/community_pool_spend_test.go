package ante_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/haqq-network/haqq/app/ante"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
)

// buildTx packs the given messages into a transaction and round-trips it through
// the wire format, so the ante handler sees exactly the message types that the
// tx decoder produces at runtime.
func buildTx(t *testing.T, txConfig client.TxConfig, msgs ...sdk.Msg) sdk.Tx {
	t.Helper()

	builder := txConfig.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(msgs...))

	bz, err := txConfig.TxEncoder()(builder.GetTx())
	require.NoError(t, err)

	tx, err := txConfig.TxDecoder()(bz)
	require.NoError(t, err)

	return tx
}

func TestCommunityPoolSpendAnteHandler(t *testing.T) {
	nw := network.NewUnitTestNetwork()
	ctx := nw.GetContext()
	txConfig := nw.GetEncodingConfig().TxConfig

	proposer, _ := utiltx.NewAccAddressAndKey()
	recipient, _ := utiltx.NewAccAddressAndKey()
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	coins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(100)))

	packContent := func(content proto.Message) *codectypes.Any {
		anyContent, err := codectypes.NewAnyWithValue(content)
		require.NoError(t, err)
		return anyContent
	}

	legacySubmit := func(content govv1beta1.Content) sdk.Msg {
		msg, err := govv1beta1.NewMsgSubmitProposal(content, coins, proposer)
		require.NoError(t, err)
		return msg
	}

	v1Submit := func(msgs ...sdk.Msg) sdk.Msg {
		msg, err := govv1.NewMsgSubmitProposal(msgs, coins, proposer.String(), "", "Test", "summary", false)
		require.NoError(t, err)
		return msg
	}

	spendContent := &distrtypes.CommunityPoolSpendProposal{
		Title:       "Test",
		Description: "description",
		Recipient:   recipient.String(),
		Amount:      coins,
	}
	textContent := &govv1beta1.TextProposal{Title: "Test", Description: "description"}

	spendMsg := &distrtypes.MsgCommunityPoolSpend{
		Authority: authority,
		Recipient: recipient.String(),
		Amount:    coins,
	}
	benignMsg := &banktypes.MsgSend{
		FromAddress: proposer.String(),
		ToAddress:   recipient.String(),
		Amount:      coins,
	}

	testCases := []struct {
		name     string
		msgs     []sdk.Msg
		expBlock bool
	}{
		{
			"block - legacy MsgSubmitProposal with CommunityPoolSpendProposal",
			[]sdk.Msg{legacySubmit(spendContent)},
			true,
		},
		{
			"block - MsgExecLegacyContent with CommunityPoolSpendProposal",
			[]sdk.Msg{govv1.NewMsgExecLegacyContent(packContent(spendContent), authority)},
			true,
		},
		{
			"block - v1 MsgSubmitProposal with MsgCommunityPoolSpend",
			[]sdk.Msg{v1Submit(spendMsg)},
			true,
		},
		{
			"block - v1 MsgSubmitProposal with MsgCommunityPoolSpend among other messages",
			[]sdk.Msg{v1Submit(benignMsg, spendMsg)},
			true,
		},
		{
			"block - v1 spend proposal alongside an unrelated message",
			[]sdk.Msg{v1Submit(spendMsg), benignMsg},
			true,
		},
		{
			"block - spend proposal is not the first message in the tx",
			[]sdk.Msg{benignMsg, legacySubmit(spendContent)},
			true,
		},
		{
			"pass - legacy MsgSubmitProposal with TextProposal",
			[]sdk.Msg{legacySubmit(textContent)},
			false,
		},
		{
			"pass - MsgExecLegacyContent with TextProposal",
			[]sdk.Msg{govv1.NewMsgExecLegacyContent(packContent(textContent), authority)},
			false,
		},
		{
			"pass - v1 MsgSubmitProposal without a community pool spend",
			[]sdk.Msg{v1Submit(benignMsg)},
			false,
		},
		{
			"pass - unrelated message",
			[]sdk.Msg{benignMsg},
			false,
		},
		{
			// The handler only guards the governance entry points; a bare spend
			// message is rejected later by the msg server's authority check.
			"pass - bare MsgCommunityPoolSpend outside of a proposal",
			[]sdk.Msg{spendMsg},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nextCalled := false
			handler := ante.NewCommunityPoolSpendAnteHandler(
				func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
					nextCalled = true
					return ctx, nil
				},
			)

			_, err := handler(ctx, buildTx(t, txConfig, tc.msgs...), false)

			if tc.expBlock {
				require.ErrorIs(t, err, ante.ErrCommunitySpendingComingLater)
				require.False(t, nextCalled, "blocked tx must not reach the wrapped ante handler")
				return
			}

			require.NoError(t, err)
			require.True(t, nextCalled, "allowed tx must reach the wrapped ante handler")
		})
	}
}

// TestCommunityPoolSpendAnteHandlerNestedLegacyContent documents a gap in
// NewCommunityPoolSpendAnteHandler: a v1 MsgSubmitProposal that wraps a
// MsgExecLegacyContent carrying a CommunityPoolSpendProposal passes the ante
// chain, because the handler only scans the nested messages for
// *distrtypes.MsgCommunityPoolSpend and only matches MsgExecLegacyContent at
// the top level of the tx.
//
// Today this is not a way to actually drain the community pool: the app wires
// no legacy gov router (no SetLegacyRouter call), so MsgExecLegacyContent can
// never execute. The proposal would still reach the voting stage instead of
// being rejected up front, and the gap becomes live if a legacy router is ever
// registered.
func TestCommunityPoolSpendAnteHandlerNestedLegacyContent(t *testing.T) {
	nw := network.NewUnitTestNetwork()
	ctx := nw.GetContext()
	txConfig := nw.GetEncodingConfig().TxConfig

	proposer, _ := utiltx.NewAccAddressAndKey()
	recipient, _ := utiltx.NewAccAddressAndKey()
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	coins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, sdkmath.NewInt(100)))

	spendContent, err := codectypes.NewAnyWithValue(&distrtypes.CommunityPoolSpendProposal{
		Title:       "Test",
		Description: "description",
		Recipient:   recipient.String(),
		Amount:      coins,
	})
	require.NoError(t, err)

	nested := govv1.NewMsgExecLegacyContent(spendContent, authority)
	proposal, err := govv1.NewMsgSubmitProposal([]sdk.Msg{nested}, coins, proposer.String(), "", "Test", "summary", false)
	require.NoError(t, err)

	nextCalled := false
	handler := ante.NewCommunityPoolSpendAnteHandler(
		func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			nextCalled = true
			return ctx, nil
		},
	)

	_, err = handler(ctx, buildTx(t, txConfig, proposal), false)

	// Change these assertions to expect ErrCommunitySpendingComingLater once the
	// handler recurses into the messages of a v1 MsgSubmitProposal.
	require.NoError(t, err)
	require.True(t, nextCalled)
}
