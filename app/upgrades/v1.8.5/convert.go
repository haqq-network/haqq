package v185

import (
	"fmt"
	"math/big"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	erc20keeper "github.com/haqq-network/haqq/x/erc20/keeper"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	evmkeeper "github.com/haqq-network/haqq/x/evm/keeper"
)

var storeKey = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}

// BalanceResult contains the data needed to perform the balance conversion
type BalanceResult struct {
	address      sdk.AccAddress
	balanceBytes []byte
	id           int
}

// executeConversion receives the whole set of address with erc20 balances
// it sends the equivalent coin from the escrow address into the holder address
// it doesnt need to burn the erc20 balance, because the evm storage will be deleted later
func executeConversion(
	ctx sdk.Context,
	results []BalanceResult,
	bankKeeper bankkeeper.Keeper,
	nativeTokenPairs []erc20types.TokenPair,
) error {
	// Go trough every address with an erc20 balance
	for _, result := range results {
		tokenPair := nativeTokenPairs[result.id]

		// Convert balance Bytes into Big Int
		balance := new(big.Int).SetBytes(result.balanceBytes)
		if balance.Sign() <= 0 {
			continue
		}
		// Create the coin
		coins := sdk.Coins{sdk.Coin{Denom: tokenPair.Denom, Amount: sdk.NewIntFromBigInt(balance)}}

		if bankKeeper.BlockedAddr(result.address) {
			// Skip blocked address on balance migration as some Precompiles
			// and Module accounts are prohibited from receiving coins.
			ctx.Logger().Info(fmt.Sprintf("skip blocked address - %s, stuck coins: %s", result.address.String(), coins.String()))
			continue
		}

		err := bankKeeper.SendCoinsFromModuleToAccount(ctx, erc20types.ModuleName, result.address, coins)
		if err != nil {
			return err
		}
	}

	return nil
}

// ConvertERC20Coins iterates trough all the authmodule accounts and all missing accounts from the auth module
// recovers the balance from erc20 contracts for the registered token pairs
// and for each entry it sends the balance from escrow into the account.
func ConvertERC20Coins(
	ctx sdk.Context,
	logger log.Logger,
	accountKeeper authkeeper.AccountKeeper,
	bankKeeper bankkeeper.Keeper,
	evmKeeper evmkeeper.Keeper,
	nativeTokenPairs []erc20types.TokenPair,
) error {
	timeBegin := time.Now() // control the time of the execution
	var finalizedResults []BalanceResult

	i := 0
	accountKeeper.IterateAccounts(ctx, func(account authtypes.AccountI) (stop bool) {
		i++
		if i%100_000 == 0 {
			logger.Info(fmt.Sprintf("Processing account: %d", i))
		}

		addBalances(ctx, account.GetAddress(), evmKeeper, nativeTokenPairs, &finalizedResults)

		return false
	})

	logger.Info(fmt.Sprint("Finalized results: ", len(finalizedResults)))

	// execute the actual conversion.
	err := executeConversion(ctx, finalizedResults, bankKeeper, nativeTokenPairs)
	if err != nil {
		// panic(err)
		return err
	}

	// NOTE: if there are tokens left in the ERC-20 module account
	// we return an error because this implies that the migration of native
	// coins to ERC-20 tokens was not fully completed.
	erc20ModuleAccountAddress := authtypes.NewModuleAddress(erc20types.ModuleName)
	balances := bankKeeper.GetAllBalances(ctx, erc20ModuleAccountAddress)
	if !balances.IsZero() {
		logger.Info(fmt.Sprintf("there are still tokens in the erc-20 module account: %s", balances.String()))
		// we dont return an error here. Since we want the migration to pass
		// if any balance is left on escrow, we can recover it later.
	}
	duration := time.Since(timeBegin)
	logger.Info(fmt.Sprintf("Migration length %s", duration.String()))
	return nil
}

func addBalances(
	ctx sdk.Context,
	account sdk.AccAddress,
	evmKeeper evmkeeper.Keeper,
	nativeTokenPairs []erc20types.TokenPair,
	balances *[]BalanceResult,
) {
	concatBytes := append(common.LeftPadBytes(account.Bytes(), 32), storeKey...)
	key := crypto.Keccak256Hash(concatBytes)

	var value []byte
	for tokenID, tokenPair := range nativeTokenPairs {
		value = evmKeeper.GetFastState(ctx, tokenPair.GetERC20Contract(), key)
		if len(value) == 0 {
			continue
		}
		*balances = append(*balances, BalanceResult{address: account, balanceBytes: value, id: tokenID})
	}
}

// getNativeTokenPairs returns the token pairs that are registered for native Cosmos coins.
func getNativeTokenPairs(
	ctx sdk.Context,
	erc20Keeper erc20keeper.Keeper,
) []erc20types.TokenPair {
	var nativeTokenPairs []erc20types.TokenPair

	erc20Keeper.IterateTokenPairs(ctx, func(tokenPair erc20types.TokenPair) bool {
		// NOTE: here we check if the token pair contains an IBC coin. For now, we only want to convert those.
		if !tokenPair.IsNativeCoin() {
			return false
		}

		nativeTokenPairs = append(nativeTokenPairs, tokenPair)
		return false
	})

	return nativeTokenPairs
}
