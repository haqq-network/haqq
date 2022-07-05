package erc20

// import (
// 	"context"
// 	"fmt"
// 	"math/big"
// 	"strings"
// 	"testing"
// 	"time"

// 	"github.com/cosmos/cosmos-sdk/crypto"
// 	"github.com/cosmos/cosmos-sdk/simapp"
// 	"github.com/ethereum/go-ethereum/common"
// 	"github.com/ethereum/go-ethereum/common/hexutil"
// 	"github.com/ethereum/go-ethereum/core/types"
// 	"github.com/stretchr/testify/require"
// 	"github.com/tharsis/ethermint/testutil/network"

// 	"github.com/evmos/ethermint/crypto/ethsecp256k1"

// 	ethcrypto "github.com/ethereum/go-ethereum/crypto"
// 	haqqnetwork "github.com/haqq-network/haqq/testutil/network"
// )

// func TestTransferETH(t *testing.T) {
// 	cfg := network.DefaultConfig()
// 	encCfg := simapp.MakeTestEncodingConfig()
// 	cfg.AppConstructor = haqqnetwork.NewAppConstructor(encCfg)
// 	cfg.NumValidators = 1
// 	baseDir := t.TempDir()
// 	n, err := network.New(t, baseDir, cfg)
// 	require.NoError(t, err)

// 	_, err = n.WaitForHeight(1)
// 	require.NoError(t, err)

// 	val := n.Validators[0]

// 	var addr [20]byte
// 	copy(addr[:], val.Address.Bytes()[:20])

// 	fmt.Printf("APIAddress: %v", val.AppConfig.JSONRPC.Address)

// 	balanceBefore, err := val.JSONRPCClient.BalanceAt(context.Background(), addr, big.NewInt(5))
// 	fmt.Printf("balanceBefore: %s, %b", balanceBefore, balanceBefore.Bytes())

// 	expInt := new(big.Int)
// 	expectedBalance, _ := expInt.SetString("400000000000000000000", 10)

// 	require.Equal(t, expectedBalance, balanceBefore)
// 	require.NoError(t, err)

// 	blockNumber, err := val.JSONRPCClient.BlockNumber(context.Background())
// 	require.NoError(t, err)

// 	chainId, err := val.JSONRPCClient.ChainID(context.Background())
// 	require.NoError(t, err)

// 	fmt.Printf("chainId: %v", chainId)

// 	nonce, err := val.JSONRPCClient.NonceAt(context.Background(), addr, big.NewInt(int64(blockNumber)))
// 	require.NoError(t, err)

// 	fmt.Printf("nonce: %v", nonce)

// 	// var receiveAddr common.Address

// 	// receiveKey1, _ = crypto.GenerateKey()
// 	// receiveAddr1 = crypto.PubkeyToAddress(receiveKey1.PublicKey)

// 	// tx1 := types.SignTx(types.NewTransaction(
// 	// 	nonce,
// 	// ))

// 	gasPrice, _ := val.JSONRPCClient.SuggestGasPrice(context.Background())
// 	fmt.Printf("gasPrice: %v", gasPrice)

// 	toEthAddress := common.HexToAddress("0xD77E96273fe7DB671a35BBB395CDe6E5aA310A34")
// 	fmt.Printf("toEthAddress: %v", toEthAddress)

// 	msgTx := types.NewTransaction(
// 		nonce,
// 		toEthAddress,
// 		big.NewInt(1000000000000000000),
// 		100000,
// 		gasPrice,
// 		nil,
// 	)

// 	armorKey, _ := val.ClientCtx.Keyring.ExportPrivKeyArmorByAddress(val.Address, "")
// 	privKey, algo, err := crypto.UnarmorDecryptPrivKey(armorKey, "")
// 	fmt.Printf("\nunarmorPrivateKey %v+, algo: %v, err: %v", privKey, algo, err)

// 	ethPrivKey := ethsecp256k1.PrivKey{Key: privKey.Bytes()}
// 	fmt.Printf("\nethPrivKey %v+\n", ethPrivKey)

// 	ecdsaKey, _ := ethPrivKey.ToECDSA()
// 	fmt.Printf("ecdsaKey: %v \n", ecdsaKey)

// 	// Formats key for output
// 	privB := ethcrypto.FromECDSA(ecdsaKey)
// 	keyS := strings.ToUpper(hexutil.Encode(privB)[2:])

// 	fmt.Printf("keyS: %v \n", keyS)

// 	signedTx, _ := types.SignTx(
// 		msgTx,
// 		types.NewLondonSigner(chainId),
// 		ecdsaKey,
// 	)

// 	err = val.JSONRPCClient.SendTransaction(context.Background(), signedTx)
// 	require.NoError(t, err)

// 	require.NoError(t, n.WaitForNextBlock())

// 	_, err = n.WaitForHeight(11)
// 	require.NoError(t, err)

// 	blockNumber, _ = val.JSONRPCClient.BlockNumber(context.Background())
// 	expectedBlockNumber := uint64(11)

// 	require.Equal(t, expectedBlockNumber, blockNumber)

// 	balanceAfter, err := val.JSONRPCClient.BalanceAt(context.Background(), addr, big.NewInt(5))
// 	fmt.Printf("\nbalanceBefore: %s %b", balanceBefore, balanceBefore.Bytes())
// 	fmt.Printf("\nbalanceAfter: %s %b \n", balanceAfter, balanceAfter.Bytes())
// 	require.NoError(t, err)
// 	// // require.Equal(t, 1, balanceBefore.CmpAbs(balanceAfter))

// 	// receipt, err := val.JSONRPCClient.TransactionReceipt(context.Background(), common.HexToHash(signedTx.Hash().Hex()))
// 	// require.NoError(t, err)
// 	// fmt.Printf("\nTransactionReceipt: %v\n", receipt)

// 	time.Sleep(time.Second * 600)

// 	count, _ := val.JSONRPCClient.PendingTransactionCount(context.Background())
// 	fmt.Printf("\nPendingTransactionCount: %v\n", count)

// 	_, isPending, err := val.JSONRPCClient.TransactionByHash(context.Background(), common.HexToHash(signedTx.Hash().Hex()))
// 	require.NoError(t, err)
// 	require.False(t, isPending)
// }
