package keeper_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	haqqtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/x/evm/keeper/testdata"
	"github.com/haqq-network/haqq/x/evm/types"
)

func SetupContract(b *testing.B) (*KeeperTestSuite, common.Address) {
	suite := KeeperTestSuite{}
	suite.SetupTestWithT(b)

	amt := sdk.Coins{haqqtypes.NewIslmCoinInt64(1000000000000000000)}
	err := suite.app.BankKeeper.MintCoins(suite.ctx, types.ModuleName, amt)
	require.NoError(b, err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, suite.address.Bytes(), amt)
	require.NoError(b, err)

	contractAddr := suite.DeployTestContract(b, suite.address, sdkmath.NewIntWithDecimal(1000, 18).BigInt())
	suite.Commit()

	return &suite, contractAddr
}

func SetupTestMessageCall(b *testing.B) (*KeeperTestSuite, common.Address) {
	suite := KeeperTestSuite{}
	suite.SetupTestWithT(b)

	amt := sdk.Coins{haqqtypes.NewIslmCoinInt64(1000000000000000000)}
	err := suite.app.BankKeeper.MintCoins(suite.ctx, types.ModuleName, amt)
	require.NoError(b, err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, suite.address.Bytes(), amt)
	require.NoError(b, err)

	contractAddr := suite.DeployTestMessageCall(b)
	suite.Commit()

	return &suite, contractAddr
}

type TxBuilder func(suite *KeeperTestSuite, contract common.Address) *types.MsgEthereumTx

func DoBenchmark(b *testing.B, txBuilder TxBuilder) {
	suite, contractAddr := SetupContract(b)

	msg := txBuilder(suite, contractAddr)
	msg.From = suite.address.Hex()
	err := msg.Sign(ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID()), suite.signer)
	require.NoError(b, err)

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := suite.ctx.CacheContext()

		// deduct fee first
		txData, err := types.UnpackTxData(msg.Data)
		require.NoError(b, err)

		fees := sdk.Coins{sdk.NewCoin(suite.EvmDenom(), sdkmath.NewIntFromBigInt(txData.Fee()))}
		err = authante.DeductFees(suite.app.BankKeeper, suite.ctx, suite.app.AccountKeeper.GetAccount(ctx, msg.GetFrom()), fees)
		require.NoError(b, err)

		rsp, err := suite.app.EvmKeeper.EthereumTx(sdk.WrapSDKContext(ctx), msg)
		require.NoError(b, err)
		require.False(b, rsp.Failed())
	}
}

func BenchmarkTokenTransfer(b *testing.B) {
	erc20Contract, err := testdata.LoadERC20Contract()
	require.NoError(b, err, "failed to load erc20 contract")

	DoBenchmark(b, func(suite *KeeperTestSuite, contract common.Address) *types.MsgEthereumTx {
		input, err := erc20Contract.ABI.Pack("transfer", common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
		ethTxParams := &types.EvmTxArgs{
			ChainID:  suite.app.EvmKeeper.ChainID(),
			Nonce:    nonce,
			To:       &contract,
			Amount:   big.NewInt(0),
			GasLimit: 410000,
			GasPrice: big.NewInt(1),
			Input:    input,
		}
		return types.NewTx(ethTxParams)
	})
}

func BenchmarkEmitLogs(b *testing.B) {
	erc20Contract, err := testdata.LoadERC20Contract()
	require.NoError(b, err, "failed to load erc20 contract")

	DoBenchmark(b, func(suite *KeeperTestSuite, contract common.Address) *types.MsgEthereumTx {
		input, err := erc20Contract.ABI.Pack("benchmarkLogs", big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
		ethTxParams := &types.EvmTxArgs{
			ChainID:  suite.app.EvmKeeper.ChainID(),
			Nonce:    nonce,
			To:       &contract,
			Amount:   big.NewInt(0),
			GasLimit: 4100000,
			GasPrice: big.NewInt(1),
			Input:    input,
		}
		return types.NewTx(ethTxParams)
	})
}

func BenchmarkTokenTransferFrom(b *testing.B) {
	erc20Contract, err := testdata.LoadERC20Contract()
	require.NoError(b, err)

	DoBenchmark(b, func(suite *KeeperTestSuite, contract common.Address) *types.MsgEthereumTx {
		input, err := erc20Contract.ABI.Pack("transferFrom", suite.address, common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(0))
		require.NoError(b, err)
		nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
		ethTxParams := &types.EvmTxArgs{
			ChainID:  suite.app.EvmKeeper.ChainID(),
			Nonce:    nonce,
			To:       &contract,
			Amount:   big.NewInt(0),
			GasLimit: 410000,
			GasPrice: big.NewInt(1),
			Input:    input,
		}
		return types.NewTx(ethTxParams)
	})
}

func BenchmarkTokenMint(b *testing.B) {
	erc20Contract, err := testdata.LoadERC20Contract()
	require.NoError(b, err, "failed to load erc20 contract")

	DoBenchmark(b, func(suite *KeeperTestSuite, contract common.Address) *types.MsgEthereumTx {
		input, err := erc20Contract.ABI.Pack("mint", common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
		ethTxParams := &types.EvmTxArgs{
			ChainID:  suite.app.EvmKeeper.ChainID(),
			Nonce:    nonce,
			To:       &contract,
			Amount:   big.NewInt(0),
			GasLimit: 410000,
			GasPrice: big.NewInt(1),
			Input:    input,
		}
		return types.NewTx(ethTxParams)
	})
}

func BenchmarkMessageCall(b *testing.B) {
	suite, contract := SetupTestMessageCall(b)

	messageCallContract, err := testdata.LoadMessageCallContract()
	require.NoError(b, err, "failed to load message call contract")

	input, err := messageCallContract.ABI.Pack("benchmarkMessageCall", big.NewInt(10000))
	require.NoError(b, err)
	nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
	ethTxParams := &types.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    nonce,
		To:       &contract,
		Amount:   big.NewInt(0),
		GasLimit: 25000000,
		GasPrice: big.NewInt(1),
		Input:    input,
	}
	msg := types.NewTx(ethTxParams)

	msg.From = suite.address.Hex()
	err = msg.Sign(ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID()), suite.signer)
	require.NoError(b, err)

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := suite.ctx.CacheContext()

		// deduct fee first
		txData, err := types.UnpackTxData(msg.Data)
		require.NoError(b, err)

		fees := sdk.Coins{sdk.NewCoin(suite.EvmDenom(), sdkmath.NewIntFromBigInt(txData.Fee()))}
		err = authante.DeductFees(suite.app.BankKeeper, suite.ctx, suite.app.AccountKeeper.GetAccount(ctx, msg.GetFrom()), fees)
		require.NoError(b, err)

		rsp, err := suite.app.EvmKeeper.EthereumTx(sdk.WrapSDKContext(ctx), msg)
		require.NoError(b, err)
		require.False(b, rsp.Failed())
	}
}
