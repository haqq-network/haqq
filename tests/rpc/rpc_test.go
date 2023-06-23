// This is a test utility for Ethermint's Web3 JSON-RPC services.
//
// To run these tests please first ensure you have the ethermintd running
// and have started the RPC service with `ethermintd rest-server`.
//
// You can configure the desired HOST and MODE as well
package rpctesting

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

var (
	MODE    = os.Getenv("MODE")
	privKey = os.Getenv("PRIV_KEY")
	HOST    = os.Getenv("HOST")
)

func TestMain(m *testing.M) {
	if MODE != "rpc" {
		_, _ = fmt.Fprintln(os.Stdout, "Skipping RPC test")
		return
	}

	if HOST == "" {
		HOST = "http://localhost:8545"
	}

	// Start all tests
	code := m.Run()
	os.Exit(code)
}

func TestEth_SendTransaction(_ *testing.T) {
	// txHash := sendTestTransaction(t)

	fmt.Printf("\n\n\n#########################\n\n\n")

	client, _ := ethclient.Dial(HOST)

	blockNumber, _ := client.BlockNumber(context.Background())

	fmt.Printf("Host: %v\n", HOST)
	fmt.Printf("privKey: %v\n", privKey)
	fmt.Printf("blockNumber: %v\n", blockNumber)

	ecdsaKey, _ := ethcrypto.ToECDSA(common.Hex2Bytes(privKey))
	fmt.Printf("ecdsaKey: %v\n", ecdsaKey)
	fmt.Printf("ecdsaKey Pubkey: %v\n", &ecdsaKey.PublicKey)

	ethAddress := ethcrypto.PubkeyToAddress(ecdsaKey.PublicKey)
	fmt.Printf("ethAddress from key: %v\n", ethAddress)

	balance, _ := client.BalanceAt(context.Background(), ethAddress, big.NewInt(2))
	fmt.Printf("balance: %v\n", balance)

	gasPrice, _ := client.SuggestGasPrice(context.Background())
	fmt.Printf("gasPrice: %v\n", gasPrice)

	toEthAddress := common.HexToAddress("0xD77E96273fe7DB671a35BBB395CDe6E5aA310A34")
	fmt.Printf("toEthAddress: %v\n", toEthAddress)

	chainID, _ := client.ChainID(context.Background())
	nonce, _ := client.PendingNonceAt(context.Background(), toEthAddress)

	fmt.Printf("chainID: %v \nnonce: %v\n", chainID, nonce)

	fmt.Printf("\n\n#########################\n\n\n")

	// signer := types.LatestSignerForChainID(chainID)

	// fmt.Printf("\nethAddress: %v\n", ethPrivKey.PubKey().Address())

	// rpcRes := call(t, "eth_newBlockFilter", []string{})

	// var ID string
	// err := json.Unmarshal(rpcRes.Result, &ID)
	// require.NoError(t, err)

	// txHash := sendTestTransaction(t)
	// receipt := waitForReceipt(t, txHash)
	// require.NotNil(t, receipt, "transaction failed")
	// require.Equal(t, "0x1", receipt["status"].(string))

	// changesRes := call(t, "eth_getFilterChanges", []string{ID})
	// var hashes []common.Hash
	// err = json.Unmarshal(changesRes.Result, &hashes)
	// require.NoError(t, err)
	// require.GreaterOrEqual(t, len(hashes), 1)
}
