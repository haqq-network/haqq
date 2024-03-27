package grpc

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/encoding"
)

// GetAccount returns the account for the given address.
func (gqh *IntegrationHandler) GetAccount(address string) (sdk.AccountI, error) {
	authClient := gqh.network.GetAuthClient()
	res, err := authClient.Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: address,
	})
	if err != nil {
		return nil, err
	}

	encodingCgf := encoding.MakeConfig(app.ModuleBasics)
	var acc sdk.AccountI
	if err = encodingCgf.InterfaceRegistry.UnpackAny(res.Account, &acc); err != nil {
		return nil, err
	}
	return acc, nil
}
