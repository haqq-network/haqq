package testutil

import (
	"fmt"
	"math/big"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/testutil/tx"
	evm "github.com/haqq-network/haqq/x/evm/types"
)

// DeployContract deploys a contract with the provided private key,
// compiled contract data and constructor arguments
func DeployContract(
	ctx sdk.Context,
	appHaqq *app.Haqq,
	priv cryptotypes.PrivKey,
	queryClientEvm evm.QueryClient,
	contract evm.CompiledContract,
	constructorArgs ...interface{},
) (common.Address, error) {
	chainID := appHaqq.EvmKeeper.ChainID()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	nonce := appHaqq.EvmKeeper.GetNonce(ctx, from)

	ctorArgs, err := contract.ABI.Pack("", constructorArgs...)
	if err != nil {
		return common.Address{}, err
	}

	data := append(contract.Bin, ctorArgs...) //nolint:gocritic
	gas, err := tx.GasLimit(ctx, from, data, queryClientEvm)
	if err != nil {
		return common.Address{}, err
	}

	msgEthereumTx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:   chainID,
		Nonce:     nonce,
		GasLimit:  gas,
		GasFeeCap: appHaqq.FeeMarketKeeper.GetBaseFee(ctx),
		GasTipCap: big.NewInt(1),
		Input:     data,
		Accesses:  &ethtypes.AccessList{},
	})
	msgEthereumTx.From = from.String()

	res, err := DeliverEthTx(appHaqq, priv, msgEthereumTx)
	if err != nil {
		return common.Address{}, err
	}

	if _, err := CheckEthTxResponse(res, appHaqq.AppCodec()); err != nil {
		return common.Address{}, err
	}

	return crypto.CreateAddress(from, nonce), nil
}

// DeployContractWithFactory deploys a contract using a contract factory
// with the provided factoryAddress
func DeployContractWithFactory(
	ctx sdk.Context,
	appHaqq *app.Haqq,
	priv cryptotypes.PrivKey,
	factoryAddress common.Address,
) (common.Address, abci.ResponseDeliverTx, error) {
	chainID := appHaqq.EvmKeeper.ChainID()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	factoryNonce := appHaqq.EvmKeeper.GetNonce(ctx, factoryAddress)
	nonce := appHaqq.EvmKeeper.GetNonce(ctx, from)

	msgEthereumTx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:  chainID,
		Nonce:    nonce,
		To:       &factoryAddress,
		GasLimit: uint64(100000),
		GasPrice: big.NewInt(1000000000),
	})
	msgEthereumTx.From = from.String()

	res, err := DeliverEthTx(appHaqq, priv, msgEthereumTx)
	if err != nil {
		return common.Address{}, abci.ResponseDeliverTx{}, err
	}

	if _, err := CheckEthTxResponse(res, appHaqq.AppCodec()); err != nil {
		return common.Address{}, abci.ResponseDeliverTx{}, err
	}

	return crypto.CreateAddress(factoryAddress, factoryNonce), res, err
}

// CheckEthTxResponse checks that the transaction was executed successfully
func CheckEthTxResponse(r abci.ResponseDeliverTx, cdc codec.Codec) (*evm.MsgEthereumTxResponse, error) {
	if !r.IsOK() {
		return nil, fmt.Errorf("tx failed. Code: %d, Logs: %s", r.Code, r.Log)
	}
	var txData sdk.TxMsgData
	if err := cdc.Unmarshal(r.Data, &txData); err != nil {
		return nil, err
	}

	var res evm.MsgEthereumTxResponse
	if err := proto.Unmarshal(txData.MsgResponses[0].Value, &res); err != nil {
		return nil, err
	}

	if res.Failed() {
		return nil, fmt.Errorf("tx failed. VmError: %s", res.VmError)
	}

	return &res, nil
}
