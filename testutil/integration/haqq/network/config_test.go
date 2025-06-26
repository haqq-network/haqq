package network_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	grpchandler "github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	"github.com/haqq-network/haqq/utils"
)

func TestWithBalances(t *testing.T) {
	key1Balance := sdk.NewCoins(sdk.NewInt64Coin(utils.BaseDenom, 1e18))
	key2Balance := sdk.NewCoins(
		sdk.NewInt64Coin(utils.BaseDenom, 2e18),
		sdk.NewInt64Coin("other", 3e18),
	)

	// Create a new network with 2 pre-funded accounts
	keyring := testkeyring.New(2)
	balances := []banktypes.Balance{
		{
			Address: keyring.GetAccAddr(0).String(),
			Coins:   key1Balance,
		},
		{
			Address: keyring.GetAccAddr(1).String(),
			Coins:   key2Balance,
		},
	}
	nw := network.New(
		network.WithBalances(balances...),
	)
	handler := grpchandler.NewIntegrationHandler(nw)

	req, err := handler.GetAllBalances(keyring.GetAccAddr(0))
	require.NoError(t, err, "error getting balances")
	require.Len(t, req.Balances, 1, "wrong number of balances")
	require.Equal(t, balances[0].Coins, req.Balances, "wrong balances")

	req, err = handler.GetAllBalances(keyring.GetAccAddr(1))
	require.NoError(t, err, "error getting balances")
	require.Len(t, req.Balances, 2, "wrong number of balances")
	require.Equal(t, balances[1].Coins, req.Balances, "wrong balances")
}
