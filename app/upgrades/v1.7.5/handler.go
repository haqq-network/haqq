package v175

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"sort"
	"strings"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	liquidvestingkeeper "github.com/haqq-network/haqq/x/liquidvesting/keeper"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	evmkeeper "github.com/haqq-network/haqq/x/evm/keeper"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"

	erc20keeper "github.com/haqq-network/haqq/x/erc20/keeper"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"

	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/haqq-network/haqq/utils"
)

// TurnOffLiquidVesting turns off the liquid vesting module
func TurnOffLiquidVesting(ctx sdk.Context, bk bankkeeper.Keeper, lk liquidvestingkeeper.Keeper, erc20 erc20keeper.Keeper, ek evmkeeper.Keeper, ak authkeeper.AccountKeeper) error {
	logger := ctx.Logger()
	logger.Info("Start turning off liquid vesting module")

	// Enable liquid vesting module if not enabled
	if !lk.IsLiquidVestingEnabled(ctx) {
		lk.SetLiquidVestingEnabled(ctx, true)
	}

	// Collect all storage entries for aLIQUID tokens
	storageMap := collectStorageEntries(ctx, erc20, ek)
	// Redeem's vector
	redeemsVector := make([]liquidvestingtypes.MsgRedeem, 0)

	// Collect all reedem messages
	var wg sync.WaitGroup
	accChan := make(chan authtypes.AccountI, 100)
	worker := func() {
		defer wg.Done()
		for acc := range accChan {
			processAccount(ctx, acc, storageMap, lk, &redeemsVector)
		}
	}

	numWorkers := runtime.NumCPU() // Use all available CPUs for parallel processing
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker()
	}

	ak.IterateAccounts(ctx, func(acc authtypes.AccountI) (stop bool) {
		accChan <- acc
		return false
	})
	close(accChan)

	wg.Wait()

	// Sort redeems vector to ensure determinism
	sort.Slice(redeemsVector, func(i, j int) bool {
		return bytes.Compare(redeemsVector[i].GetSignBytes(), redeemsVector[j].GetSignBytes()) < 0
	})

	// Redeem all aLIQUID tokens
	for _, redeemMsg := range redeemsVector {
		if _, err := lk.Redeem(ctx, &redeemMsg); err != nil {
			log.Fatalf("Failed to redeem: %v", err)
		}
	}

	// Disable liquid vesting module
	lk.SetLiquidVestingEnabled(ctx, false)

	logger.Info("Finished turning off liquid vesting module")

	return nil
}

// collectStorageEntries collects all storage entries for aLIQUID tokens
func collectStorageEntries(ctx sdk.Context, erc20 erc20keeper.Keeper, ek evmkeeper.Keeper) map[string]map[erc20types.TokenPair]evmtypes.State {
	storageMap := make(map[string]map[erc20types.TokenPair]evmtypes.State)

	// Iterate over all aLIQUID token pairs
	erc20.IterateTokenPairs(ctx, func(tokenPair erc20types.TokenPair) (stop bool) {
		if strings.Contains(tokenPair.Denom, "aLIQUID") {
			commonAddr := common.HexToAddress(tokenPair.Erc20Address)
			storage := ek.GetAccountStorage(ctx, commonAddr)

			// Collect all evm storage entries for aLIQUID tokens
			for _, state := range storage {
				if _, exists := storageMap[state.Key]; !exists {
					storageMap[state.Key] = make(map[erc20types.TokenPair]evmtypes.State)
				}
				storageMap[state.Key][tokenPair] = state
			}
		}
		return false
	})
	return storageMap
}

// processAccount processes an account and creates a redeem message if the account has aLIQUID tokens
func processAccount(ctx sdk.Context, acc authtypes.AccountI, storageMap map[string]map[erc20types.TokenPair]evmtypes.State, lk liquidvestingkeeper.Keeper, redeemsVector *[]liquidvestingtypes.MsgRedeem) {
	addrStr := common.BytesToAddress(acc.GetAddress().Bytes()).String()
	if addrStr == "0x0000000000000000000000000000000000000000" {
		return
	}

	calculatedKey := utils.CalculateStorageKey(addrStr, 2)

	if storage, exists := storageMap[calculatedKey]; exists {
		for tokenPair, state := range storage {
			if state.Value != "0x0000000000000000000000000000000000000000000000000000000000000000" {
				logTokenInfo(ctx, addrStr, tokenPair)

				// Parse the hex value to get the tokens balance of the account from the ERC20 EVM contract
				value := utils.ParseHexValue(state.Value)
				evmBalance := sdk.NewCoin(tokenPair.Denom, sdk.NewIntFromBigInt(value))

				logBalanceInfo(ctx, evmBalance)

				ownerAddr := sdk.AccAddress(common.HexToAddress(addrStr).Bytes())

				// Get the liquid denom from liquid vesting keeper
				liquidDenom, found := lk.GetDenom(ctx, tokenPair.Denom)
				if !found {
					log.Fatalf("Failed to get denom: %s", tokenPair.Denom)
				}

				// Adjust the balance to the maximum available amount that can be redeemed
				evmBalance.Amount = adjustBalance(evmBalance.Amount, liquidDenom)

				// Create a redeem message
				redeemMsg := liquidvestingtypes.NewMsgRedeem(ownerAddr, ownerAddr, evmBalance)
				// Append the redeem message to the vector
				*redeemsVector = append(*redeemsVector, *redeemMsg)
			}
		}
	}
}

// adjustBalance adjusts the balance to the maximum available amount that can be redeemed
func adjustBalance(evmAmount sdk.Int, liquidDenom liquidvestingtypes.Denom) sdk.Int {
	periodsAmount := liquidDenom.LockupPeriods.TotalAmount().AmountOf("aISLM")
	if evmAmount.GT(periodsAmount) {
		evmAmount = periodsAmount
	}
	return evmAmount
}

func logTokenInfo(ctx sdk.Context, addrStr string, tokenPair erc20types.TokenPair) {
	logger := ctx.Logger()

	logger.Info(fmt.Sprintf("ERC20 owner address: %s", addrStr))
	logger.Info(fmt.Sprintf("ERC20 token address: %s", tokenPair.Erc20Address))
	logger.Info(fmt.Sprintf("ERC20 denom: %s", tokenPair.Denom))
}

func logBalanceInfo(ctx sdk.Context, evmBalance sdk.Coin) {
	logger := ctx.Logger()

	logger.Info(fmt.Sprintf("ERC20 value: %+v", evmBalance))
}
