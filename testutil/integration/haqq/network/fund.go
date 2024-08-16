package network

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
)

// FundAccount funds the given account with the given amount of coins.
func (n *IntegrationNetwork) FundAccount(addr sdk.AccAddress, coins sdk.Coins) error {
	ctx := n.GetContext()

	if err := n.app.BankKeeper.MintCoins(ctx, coinomicstypes.ModuleName, coins); err != nil {
		return err
	}

	return n.app.BankKeeper.SendCoinsFromModuleToAccount(ctx, coinomicstypes.ModuleName, addr, coins)
}

// FundAccountWithBaseDenom funds the given account with the given amount of the network's
// base denomination.
func (n *IntegrationNetwork) FundAccountWithBaseDenom(addr sdk.AccAddress, amount sdkmath.Int) error {
	return n.FundAccount(addr, sdk.NewCoins(sdk.NewCoin(n.GetDenom(), amount)))
}
