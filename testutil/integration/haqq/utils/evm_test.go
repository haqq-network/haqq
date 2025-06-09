package utils_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/contracts"
	testfactory "github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	testhandler "github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	testnetwork "github.com/haqq-network/haqq/testutil/integration/haqq/network"
	"github.com/haqq-network/haqq/testutil/integration/haqq/utils"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func TestGetERC20Balance(t *testing.T) {
	keyring := testkeyring.New(1)
	network := testnetwork.NewUnitTestNetwork(
		testnetwork.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	)
	handler := testhandler.NewIntegrationHandler(network)
	factory := testfactory.New(network, handler)

	sender := keyring.GetKey(0)
	mintAmount := big.NewInt(100)

	// Deploy an ERC-20 contract
	erc20Addr, err := factory.DeployContract(
		sender.Priv,
		evmtypes.EvmTxArgs{},
		testfactory.ContractDeploymentData{
			Contract:        contracts.ERC20MinterBurnerDecimalsContract,
			ConstructorArgs: []interface{}{"TestToken", "TT", uint8(18)},
		},
	)
	require.NoError(t, err, "failed to deploy contract")
	require.NoError(t, network.NextBlock(), "failed to advance block")

	balance, err := utils.GetERC20Balance(network, erc20Addr, sender.Addr)
	require.NoError(t, err, "failed to get ERC20 balance")
	require.Equal(t, common.Big0.Int64(), balance.Int64(), "expected no balance before minting")

	// Mint some tokens
	_, err = factory.ExecuteContractCall(
		sender.Priv,
		evmtypes.EvmTxArgs{
			To: &erc20Addr,
		},
		testfactory.CallArgs{
			ContractABI: contracts.ERC20MinterBurnerDecimalsContract.ABI,
			MethodName:  "mint",
			Args:        []interface{}{sender.Addr, mintAmount},
		},
	)
	require.NoError(t, err, "failed to mint tokens")

	require.NoError(t, network.NextBlock(), "failed to advance block")

	balance, err = utils.GetERC20Balance(network, erc20Addr, sender.Addr)
	require.NoError(t, err, "failed to get ERC20 balance")
	require.Equal(t, mintAmount.Int64(), balance.Int64(), "expected different balance after minting")
}
