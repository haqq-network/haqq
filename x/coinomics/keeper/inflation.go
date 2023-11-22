package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	"github.com/haqq-network/haqq/x/coinomics/types"
)

func (k Keeper) MintAndAllocate(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	rewardCoefficient := params.RewardCoefficient.Quo(sdk.NewDec(100))
	prevBlockTS, _ := sdk.NewDecFromStr(k.GetPrevBlockTS(ctx).String())
	totalBonded, _ := sdk.NewDecFromStr(k.stakingKeeper.TotalBondedTokens(ctx).String())

	currentBlockTS, _ := sdk.NewDecFromStr(math.NewInt(ctx.BlockTime().UnixMilli()).String())
	yearInMillis, _ := sdk.NewDecFromStr("31536000000")

	blockMint := totalBonded.Mul(rewardCoefficient).Mul(currentBlockTS.Sub(prevBlockTS).Quo(yearInMillis))
	totalMintOnBlockCoin := sdk.NewCoin(params.MintDenom, blockMint.RoundInt())

	// Mint coins to coinomics module
	if err := k.MintCoins(ctx, totalMintOnBlockCoin); err != nil {
		return errors.Wrap(err, "failed mint coins")
	}

	// Allocate remaining coinomics module balance to destribution
	err := k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.ModuleName,
		k.feeCollectorName,
		sdk.NewCoins(totalMintOnBlockCoin),
	)
	if err != nil {
		return errors.Wrap(err, "failed send coins from coinomics to distribution")
	}

	k.SetPrevBlockTS(ctx, currentBlockTS.RoundInt())

	return nil
}

func (k Keeper) MintCoins(ctx sdk.Context, coin sdk.Coin) error {
	coins := sdk.NewCoins(coin)

	// skip as no coins need to be minted
	if coins.Empty() {
		return nil
	}

	return k.bankKeeper.MintCoins(ctx, types.ModuleName, coins)
}
