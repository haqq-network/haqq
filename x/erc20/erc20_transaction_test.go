package erc20

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/testutil/network"
	erc20types "github.com/evmos/evmos/v7/x/erc20/types"
	haqqnetwork "github.com/haqq-network/haqq/testutil/network"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestTransferETH(t *testing.T) {
	cfg := network.DefaultConfig()
	encCfg := simapp.MakeTestEncodingConfig()
	cfg.AppConstructor = haqqnetwork.NewAppConstructor(encCfg)
	cfg.NumValidators = 1
	cfg.GenesisState[erc20types.ModuleName] = []byte("{\"params\":{\"enable_erc20\":true,\"enable_evm_hook\":true},\"token_pairs\":[]}")

	baseDir := t.TempDir()
	n, err := network.New(t, baseDir, cfg)
	require.NoError(t, err)

	_, err = n.WaitForHeight(1)
	require.NoError(t, err)

	val := n.Validators[0]

	var addr [20]byte
	copy(addr[:], val.Address.Bytes()[:20])

	balanceBefore, err := val.JSONRPCClient.BalanceAt(context.Background(), addr, big.NewInt(5))
	require.NoError(t, err)

	blockNumber, err := val.JSONRPCClient.BlockNumber(context.Background())
	require.NoError(t, err)

	chainId, err := val.JSONRPCClient.ChainID(context.Background())
	require.NoError(t, err)

	nonce, err := val.JSONRPCClient.NonceAt(context.Background(), addr, big.NewInt(int64(blockNumber)))
	require.NoError(t, err)

	var receiveAddr common.Address
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainId,
		Nonce:     nonce,
		GasTipCap: big.NewInt(3869400046),
		GasFeeCap: big.NewInt(38694000460),
		Gas:       uint64(22012),
		To:        &receiveAddr,
		Value:     big.NewInt(100000),
	})

	secretKeyArmored, err := val.ClientCtx.Keyring.ExportPrivKeyArmorByAddress(val.Address, "")
	require.NoError(t, err)

	secretKey, _, err := crypto.UnarmorDecryptPrivKey(secretKeyArmored, "")
	require.NoError(t, err)

	var sk ecdsa.PrivateKey
	sk.D = new(big.Int).SetBytes(secretKey.Bytes())
	sk.PublicKey.Curve = elliptic.P256()
	sk.PublicKey.X, sk.PublicKey.Y = sk.PublicKey.Curve.ScalarBaseMult(sk.D.Bytes())

	signer := types.NewLondonSigner(chainId)
	signedTx, err := types.SignTx(tx, signer, &sk)
	require.NoError(t, err)
	require.NoError(t, val.JSONRPCClient.SendTransaction(context.Background(), signedTx))

	require.NoError(t, n.WaitForNextBlock())
	require.NoError(t, n.WaitForNextBlock())

	_, isPending, err := val.JSONRPCClient.TransactionByHash(context.Background(), signedTx.Hash())
	require.NoError(t, err)
	require.False(t, isPending)

	ethBlockNumber, err := val.JSONRPCClient.BlockNumber(context.Background())
	require.NoError(t, err)

	balanceAfter, err := val.JSONRPCClient.BalanceAt(context.Background(), addr, big.NewInt(int64(ethBlockNumber)))
	require.Equal(t, 1, balanceBefore.CmpAbs(balanceAfter))
}
