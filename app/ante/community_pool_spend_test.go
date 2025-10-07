package ante

import (
	"testing"
)

func TestCommunityPoolSpendAnteHandler(t *testing.T) {
	t.Skip()

	// TODO Cover CommunityPoolSpendAnteHandler with tests
	// TODO Refactor old legacy code below
	// t.Run("try add gov spend proposal", func(t *testing.T) {
	//	coins := sdk.NewCoins(sdk.NewCoin("aISLM", sdk.NewInt(100)))
	//	newAccPkey := ed25519.GenPrivKey()
	//	recipient := sdk.AccAddress(newAccPkey.PubKey().Address())
	//
	//	// generate submit proposal
	//	cpsp := distrtypes.NewCommunityPoolSpendProposal("Test", "description", recipient, coins)
	//	sp, _ := govtypes.NewMsgSubmitProposal(cpsp, coins, recipient)
	//
	//	/// build tx
	//	builder := app.GetTxConfig().NewTxBuilder()
	//	require.NoError(t, builder.SetMsgs(sp))
	//	require.NoError(t, builder.SetSignatures(signing.SignatureV2{
	//		PubKey: newAccPkey.PubKey(),
	//		Data:   nil,
	//	}))
	//
	//	// run tx
	//	ctx := app.NewContext(true, tmproto.Header{Height: 1})
	//	_, err = handler(ctx, builder.GetTx(), true)
	//
	//	require.Error(t, ErrCommunitySpendingComingLater, err)
	// })
}
