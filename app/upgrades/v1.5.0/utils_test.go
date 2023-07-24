package v150_test

import (
	"encoding/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/server/config"
	evm "github.com/evmos/ethermint/x/evm/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/haqq-network/haqq/testutil/contracts"
	"github.com/stretchr/testify/require"
	"math/big"
)

func depositContract(from, to testAcc, amount *big.Int) (*evmtypes.MsgEthereumTxResponse, error) {
	// deposit contract
	callData, err := contracts.HaqqTestingContract.ABI.Pack("deposit", to.ethAddress)
	require.NoError(s.t, err)

	hexAmount := hexutil.Big(*amount)

	args, err := json.Marshal(&evm.TransactionArgs{
		To:    &s.contractAddress,
		From:  &from.ethAddress,
		Value: &hexAmount,
		Data:  (*hexutil.Bytes)(&callData),
	})
	require.NoError(s.t, err)

	res, err := s.queryClientEvm.EstimateGas(s.ctx, &evm.EthCallRequest{
		Args:   args,
		GasCap: config.DefaultGasCap,
	})
	require.NoError(s.t, err)

	// Mint the max gas to the FeeCollector to ensure balance in case of refund
	gasReq := sdk.NewInt(s.app.FeeMarketKeeper.GetBaseFee(s.ctx).Int64())
	gasReq = gasReq.MulRaw(int64(res.Gas))
	s.MintFeeCollector(sdk.NewCoins(sdk.NewCoin(s.app.EvmKeeper.GetEVMDenom(s.ctx), gasReq)))

	nonce := s.app.EvmKeeper.GetNonce(s.ctx, from.ethAddress)

	ercTransferTx := evm.NewTx(
		s.app.EvmKeeper.ChainID(),
		nonce,
		&s.contractAddress,
		amount,
		res.Gas,
		nil,
		s.app.FeeMarketKeeper.GetBaseFee(s.ctx),
		big.NewInt(1),
		callData,
		&ethtypes.AccessList{}, // accesses
	)

	ercTransferTx.From = from.ethAddress.Hex()
	err = ercTransferTx.Sign(ethtypes.LatestSignerForChainID(s.app.EvmKeeper.ChainID()), from.signer)
	require.NoError(s.t, err)

	return s.app.EvmKeeper.EthereumTx(s.ctx, ercTransferTx)
}

func withdrawContract(to testAcc, amount *big.Int) (*evmtypes.MsgEthereumTxResponse, error) {
	// deposit contract
	callData, err := contracts.HaqqTestingContract.ABI.Pack("withdraw", amount)
	require.NoError(s.t, err)

	args, err := json.Marshal(&evm.TransactionArgs{
		To:   &s.contractAddress,
		From: &to.ethAddress,
		Data: (*hexutil.Bytes)(&callData),
	})
	require.NoError(s.t, err)

	res, err := s.queryClientEvm.EstimateGas(s.ctx, &evm.EthCallRequest{
		Args:   args,
		GasCap: config.DefaultGasCap,
	})
	require.NoError(s.t, err)

	// Mint the max gas to the FeeCollector to ensure balance in case of refund
	gasReq := sdk.NewInt(s.app.FeeMarketKeeper.GetBaseFee(s.ctx).Int64())
	gasReq = gasReq.MulRaw(int64(res.Gas))
	s.MintFeeCollector(sdk.NewCoins(sdk.NewCoin(s.app.EvmKeeper.GetEVMDenom(s.ctx), gasReq)))

	nonce := s.app.EvmKeeper.GetNonce(s.ctx, to.ethAddress)

	ercTransferTx := evm.NewTx(
		s.app.EvmKeeper.ChainID(),
		nonce,
		&s.contractAddress,
		nil,
		res.Gas,
		nil,
		s.app.FeeMarketKeeper.GetBaseFee(s.ctx),
		big.NewInt(1),
		callData,
		&ethtypes.AccessList{}, // accesses
	)

	ercTransferTx.From = to.ethAddress.Hex()
	err = ercTransferTx.Sign(ethtypes.LatestSignerForChainID(s.app.EvmKeeper.ChainID()), to.signer)
	require.NoError(s.t, err)

	return s.app.EvmKeeper.EthereumTx(s.ctx, ercTransferTx)
}
