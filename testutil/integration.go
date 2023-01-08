package testutil

// import (
// 	"strconv"

// 	errorsmod "cosmossdk.io/errors"
// 	"github.com/cosmos/cosmos-sdk/client/tx"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
// 	"github.com/cosmos/cosmos-sdk/types/tx/signing"

// 	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
// 	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
// 	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
// 	abci "github.com/tendermint/tendermint/abci/types"

// 	"github.com/evmos/ethermint/crypto/ethsecp256k1"
// 	"github.com/evmos/ethermint/encoding"

// 	"github.com/haqq-network/haqq/app"
// )

// // SubmitProposal delivers a submit proposal tx for a given gov content.
// // Depending on the content type, the eventNum needs to specify submit_proposal
// // event.
// func SubmitProposal(
// 	ctx sdk.Context,
// 	appHaqq *app.Haqq,
// 	pk *ethsecp256k1.PrivKey,
// 	content govtypes.Content,
// 	eventNum int,
// ) (id uint64, err error) {
// 	accountAddress := sdk.AccAddress(pk.PubKey().Address().Bytes())
// 	stakeDenom := stakingtypes.DefaultParams().BondDenom

// 	deposit := sdk.NewCoins(sdk.NewCoin(stakeDenom, sdk.NewInt(100000000)))
// 	msg, err := govtypes.NewMsgSubmitProposal(content, deposit, accountAddress)
// 	if err != nil {
// 		return id, err
// 	}
// 	res, err := DeliverTx(ctx, appHaqq, pk, msg)
// 	if err != nil {
// 		return id, err
// 	}

// 	submitEvent := res.GetEvents()[eventNum]
// 	if submitEvent.Type != "submit_proposal" || string(submitEvent.Attributes[0].Key) != "proposal_id" {
// 		return id, errorsmod.Wrapf(errorsmod.Error{}, "eventNumber %d in SubmitProposal calls %s instead of submit_proposal", eventNum, submitEvent.Type)
// 	}

// 	return strconv.ParseUint(string(submitEvent.Attributes[0].Value), 10, 64)
// }

// // Delegate delivers a delegate tx
// func Delegate(
// 	ctx sdk.Context,
// 	appHaqq *app.Haqq,
// 	priv *ethsecp256k1.PrivKey,
// 	delegateAmount sdk.Coin,
// 	validator stakingtypes.Validator,
// ) (abci.ResponseDeliverTx, error) {
// 	accountAddress := sdk.AccAddress(priv.PubKey().Address().Bytes())

// 	val, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
// 	if err != nil {
// 		return abci.ResponseDeliverTx{}, err
// 	}

// 	delegateMsg := stakingtypes.NewMsgDelegate(accountAddress, val, delegateAmount)
// 	return DeliverTx(ctx, appHaqq, priv, delegateMsg)
// }

// // Vote delivers a vote tx with the VoteOption "yes"
// func Vote(
// 	ctx sdk.Context,
// 	appHaqq *app.Haqq,
// 	priv *ethsecp256k1.PrivKey,
// 	proposalID uint64,
// 	voteOption govtypes.VoteOption,
// ) (abci.ResponseDeliverTx, error) {
// 	accountAddress := sdk.AccAddress(priv.PubKey().Address().Bytes())

// 	voteMsg := govtypes.NewMsgVote(accountAddress, proposalID, voteOption)
// 	return DeliverTx(ctx, appHaqq, priv, voteMsg)
// }

// // DeliverTx delivers a tx for a given set of msgs
// func DeliverTx(
// 	ctx sdk.Context,
// 	appHaqq *app.Haqq,
// 	priv *ethsecp256k1.PrivKey,
// 	msgs ...sdk.Msg,
// ) (abci.ResponseDeliverTx, error) {
// 	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
// 	accountAddress := sdk.AccAddress(priv.PubKey().Address().Bytes())
// 	denom := appHaqq.StakingKeeper.GetParams(ctx).BondDenom

// 	txBuilder := encodingConfig.TxConfig.NewTxBuilder()

// 	txBuilder.SetGasLimit(100_000_000)
// 	txBuilder.SetFeeAmount(sdk.Coins{{Denom: denom, Amount: sdk.NewInt(1)}})
// 	if err := txBuilder.SetMsgs(msgs...); err != nil {
// 		return abci.ResponseDeliverTx{}, err
// 	}

// 	seq, err := appHaqq.AccountKeeper.GetSequence(ctx, accountAddress)
// 	if err != nil {
// 		return abci.ResponseDeliverTx{}, err
// 	}

// 	// First round: we gather all the signer infos. We use the "set empty
// 	// signature" hack to do that.
// 	sigV2 := signing.SignatureV2{
// 		PubKey: priv.PubKey(),
// 		Data: &signing.SingleSignatureData{
// 			SignMode:  encodingConfig.TxConfig.SignModeHandler().DefaultMode(),
// 			Signature: nil,
// 		},
// 		Sequence: seq,
// 	}

// 	sigsV2 := []signing.SignatureV2{sigV2}

// 	if err := txBuilder.SetSignatures(sigsV2...); err != nil {
// 		return abci.ResponseDeliverTx{}, err
// 	}

// 	// Second round: all signer infos are set, so each signer can sign.
// 	accNumber := appHaqq.AccountKeeper.GetAccount(ctx, accountAddress).GetAccountNumber()
// 	signerData := authsigning.SignerData{
// 		ChainID:       ctx.ChainID(),
// 		AccountNumber: accNumber,
// 		Sequence:      seq,
// 	}
// 	sigV2, err = tx.SignWithPrivKey(
// 		encodingConfig.TxConfig.SignModeHandler().DefaultMode(), signerData,
// 		txBuilder, priv, encodingConfig.TxConfig,
// 		seq,
// 	)
// 	if err != nil {
// 		return abci.ResponseDeliverTx{}, err
// 	}

// 	sigsV2 = []signing.SignatureV2{sigV2}
// 	if err = txBuilder.SetSignatures(sigsV2...); err != nil {
// 		return abci.ResponseDeliverTx{}, err
// 	}

// 	// bz are bytes to be broadcasted over the network
// 	bz, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
// 	if err != nil {
// 		return abci.ResponseDeliverTx{}, err
// 	}

// 	req := abci.RequestDeliverTx{Tx: bz}
// 	res := appHaqq.BaseApp.DeliverTx(req)
// 	if res.Code != 0 {
// 		return abci.ResponseDeliverTx{}, errorsmod.Wrapf(errortypes.ErrInvalidRequest, res.Log)
// 	}

// 	return res, nil
// }
