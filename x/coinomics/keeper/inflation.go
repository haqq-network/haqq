package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	"github.com/haqq-network/haqq/x/coinomics/types"
)

func (k Keeper) MintAndAllocate(ctx sdk.Context) error {
	// Convert current block timestamp to Dec type for calculations
	currentBlockTS, _ := sdk.NewDecFromStr(math.NewInt(ctx.BlockTime().UnixMilli()).String())

	// Skip minting for the first block after activation, waiting for previous block timestamp to be set
	if k.GetPrevBlockTS(ctx) == sdk.ZeroInt() {
		k.SetPrevBlockTS(ctx, currentBlockTS.RoundInt())
		return nil
	}

	// Determine if the current year is a leap year
	currentYear := ctx.BlockTime().Year()
	isLeapYear := (currentYear%4 == 0 && currentYear%100 != 0) || currentYear%400 == 0

	// Define milliseconds in a year, considering leap year
	var yearInMillis sdk.Dec
	if isLeapYear {
		yearInMillis, _ = sdk.NewDecFromStr("31622400000") // 366 days in milliseconds
	} else {
		yearInMillis, _ = sdk.NewDecFromStr("31536000000") // 365 days in milliseconds
	}

	// Calculate the mint amount based on total bonded tokens and time elapsed since the last block.
	params := k.GetParams(ctx)
	rewardCoefficient := params.RewardCoefficient.Quo(sdk.NewDec(100))
	prevBlockTS, _ := sdk.NewDecFromStr(k.GetPrevBlockTS(ctx).String())
	totalBonded, _ := sdk.NewDecFromStr(k.stakingKeeper.TotalBondedTokens(ctx).String())

	// totalBonded * rewardCoefficient * ((currentBlockTS - prevBlockTS) / yearInMillis)
	blockMint := totalBonded.Mul(rewardCoefficient).Mul((currentBlockTS.Sub(prevBlockTS)).Quo(yearInMillis))

	bankTotalSupply, _ := sdk.NewDecFromStr(k.bankKeeper.GetSupply(ctx, params.MintDenom).Amount.String())
	maxSupply, _ := sdk.NewDecFromStr(k.GetMaxSupply(ctx).Amount.String())

	// Ensure minting does not exceed the maximum supply
	if bankTotalSupply.Add(blockMint).GT(maxSupply) {
		blockMint = maxSupply.Sub(bankTotalSupply)
		params.EnableCoinomics = false
		k.SetParams(ctx, params)
	}

	if blockMint.IsNegative() {
		// state is corrupted
		errStr := fmt.Sprintf("MintAndAllocate # blockMint is negative # blockMint: %s, totalBonded: %s, rewardCoefficient: %s, currentBlockTS: %s, prevBlockTS: %s, yearInMillis: %s", blockMint.String(), totalBonded.String(), rewardCoefficient.String(), currentBlockTS.String(), prevBlockTS.String(), yearInMillis.String())

		ctx.Logger().Error(errStr)

		return nil
	}

	// Mint and allocate the calculated coin amount
	totalMintOnBlockCoin := sdk.NewCoin(params.MintDenom, blockMint.RoundInt())
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

	// Update the previous block timestamp for the next cycle.
	k.SetPrevBlockTS(ctx, currentBlockTS.RoundInt())

	return nil
}

func (k Keeper) MintCoins(ctx sdk.Context, coin sdk.Coin) error {
	coins := sdk.NewCoins(coin)

	// Skip minting if no coins are specified
	if coins.Empty() {
		return nil
	}

	// Perform the minting action
	return k.bankKeeper.MintCoins(ctx, types.ModuleName, coins)
}
