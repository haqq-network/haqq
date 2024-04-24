package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// BurnCoins burns the coins from the module account.
func (k BaseKeeper) BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error {
	switch moduleName {
	case govtypes.ModuleName, stakingtypes.BondedPoolName, stakingtypes.NotBondedPoolName:
		// Update fee pool via distrKeeper.FundCommunityPool method.
		//
		// Add sender module account check as the FundCommunityPool method executes bankKeeper.SendCoinsFromAccountToModule
		// instead of bankKeeper.SendCoinsFromModuleToModule.
		// Lack of such sender account check is the only difference them.
		senderAddr := k.ak.GetModuleAddress(moduleName)
		if senderAddr == nil {
			panic(errorsmod.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s does not exist", moduleName))
		}

		return k.dk.FundCommunityPool(ctx, amounts, senderAddr)
	}

	return k.BaseKeeper.BurnCoins(ctx, moduleName, amounts)
}
