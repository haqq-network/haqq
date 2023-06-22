package testutil

import (
	"fmt"
	"math/big"

	"github.com/gogo/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	evm "github.com/evmos/ethermint/x/evm/types"
	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/testutil/tx"
)

// DeployContract deploys a contract with the provided private key,
// compiled contract data and constructor arguments
func DeployContract(
	ctx sdk.Context,
	evmosApp *app.Haqq,
	priv cryptotypes.PrivKey,
	queryClientEvm evm.QueryClient,
	contract evm.CompiledContract,
	constructorArgs ...interface{},
) (common.Address, error) {
	chainID := evmosApp.EvmKeeper.ChainID()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	nonce := evmosApp.EvmKeeper.GetNonce(ctx, from)

	ctorArgs, err := contract.ABI.Pack("", constructorArgs...)
	if err != nil {
		return common.Address{}, err
	}

	data := append(contract.Bin, ctorArgs...) //nolint:gocritic
	gas, err := tx.GasLimit(ctx, from, data, queryClientEvm)
	if err != nil {
		return common.Address{}, err
	}

	msgEthereumTx := evm.NewTx(
		chainID,
		nonce,
		nil,
		nil,
		gas,
		nil,
		evmosApp.FeeMarketKeeper.GetBaseFee(ctx),
		big.NewInt(1),
		data,
		&ethtypes.AccessList{},
	)
	msgEthereumTx.From = from.String()

	res, err := DeliverEthTx(ctx, evmosApp, priv, msgEthereumTx)
	if err != nil {
		return common.Address{}, err
	}

	if _, err := CheckEthTxResponse(res, evmosApp.AppCodec()); err != nil {
		return common.Address{}, err
	}

	return crypto.CreateAddress(from, nonce), nil
}

// DeployContractWithFactory deploys a contract using a contract factory
// with the provided factoryAddress
func DeployContractWithFactory(
	ctx sdk.Context,
	evmosApp *app.Haqq,
	priv cryptotypes.PrivKey,
	factoryAddress common.Address,
	_ evm.QueryClient,
) (common.Address, abci.ResponseDeliverTx, error) {
	chainID := evmosApp.EvmKeeper.ChainID()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	factoryNonce := evmosApp.EvmKeeper.GetNonce(ctx, factoryAddress)
	nonce := evmosApp.EvmKeeper.GetNonce(ctx, from)

	msgEthereumTx := evm.NewTx(
		chainID,
		nonce,
		&factoryAddress,
		nil,
		uint64(100000),
		big.NewInt(1000000000),
		nil,
		nil,
		nil,
		nil,
	)
	msgEthereumTx.From = from.String()

	res, err := DeliverEthTx(ctx, evmosApp, priv, msgEthereumTx)
	if err != nil {
		return common.Address{}, abci.ResponseDeliverTx{}, err
	}

	if _, err := CheckEthTxResponse(res, evmosApp.AppCodec()); err != nil {
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
