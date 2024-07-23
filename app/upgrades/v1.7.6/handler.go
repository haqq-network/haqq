package v176

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	erc20keeper "github.com/haqq-network/haqq/x/erc20/keeper"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	liquidvestingkeeper "github.com/haqq-network/haqq/x/liquidvesting/keeper"
	liquidvestingtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
	daokeeepr "github.com/haqq-network/haqq/x/ucdao/keeper"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// TurnOnDAO turns on the DAO module
func TurnOnDAO(ctx sdk.Context, bk bankkeeper.Keeper, lk liquidvestingkeeper.Keeper, ak authkeeper.AccountKeeper, sk stakingkeeper.Keeper, dk daokeeepr.Keeper, erc20k erc20keeper.Keeper) error {
	logger := ctx.Logger()
	logger.Info("Start turning on DAO module")

	// Enable liquid vesting module
	lk.ResetParamsToDefault(ctx)

	totalUndelegated := sdk.ZeroInt()

	// Iterate over all accounts
	ak.IterateAccounts(ctx, func(acc authtypes.AccountI) (stop bool) {
		if !matchedAddressHash(acc.GetAddress()) {
			return false
		}

		hexAddr := common.BytesToAddress(acc.GetAddress().Bytes())
		accHexAddrStr := hexAddr.Hex()
		hashAddr := getAddrHash(acc.GetAddress())
		logger.Info(fmt.Sprintf("--- Processing account hashHex: %s ---", hashAddr))

		// Force complete staking
		forceStakingComplete(ctx, sk, acc.GetAddress())

		accAddr := acc.GetAddress()
		accAddrStr := accAddr.String()
		// Get account unlock_amount by schedule

		clawbackAccount, isClawback := acc.(*vestingtypes.ClawbackVestingAccount)
		if !isClawback {
			logger.Info(fmt.Sprintf("Account at address '%s' is not a vesting account, accHexAddrStr: '%s'", accAddrStr, accHexAddrStr))
			return false
		}

		// -- Locked only
		lockedAmounts := clawbackAccount.GetLockedOnly(ctx.BlockTime())
		lockedOnlyAmount := sdk.ZeroInt()
		for _, lockedAmount := range lockedAmounts {
			if lockedAmount.Denom == "aISLM" {
				lockedOnlyAmount = lockedOnlyAmount.Add(lockedAmount.Amount)
			}
		}

		// -- Total unvested
		vestingAmounts := clawbackAccount.GetUnvestedOnly(ctx.BlockTime())
		totalUnvested := sdk.ZeroInt()
		for _, unvestedAmount := range vestingAmounts {
			if unvestedAmount.Denom == "aISLM" {
				totalUnvested = totalUnvested.Add(unvestedAmount.Amount)
			}
		}

		// -- Total delegated
		delegations := sk.GetAllDelegatorDelegations(ctx, accAddr)
		totalDelegated := getTotalDelegated(ctx, sk, delegations)

		// -- Total locked
		totalLocked := lockedOnlyAmount.Sub(totalUnvested)

		// -- Undelegate
		if totalLocked.GTE(totalDelegated) {
			if err := undelegateAll(ctx, sk, bk, delegations); err != nil {
				logger.Error(fmt.Sprintf("undelegating all from account '%s': %s", accAddrStr, err.Error()))
				return false
			}

			clawbackAccount.DelegatedFree = sdk.NewCoins(sdk.NewCoin("aISLM", sdk.ZeroInt()))
			clawbackAccount.DelegatedVesting = sdk.NewCoins(sdk.NewCoin("aISLM", sdk.ZeroInt()))
		} else {
			// Undelegate with ratio
			if err := undelegateWithRatio(ctx, sk, delegations, bk, totalLocked, totalDelegated); err != nil {
				logger.Error(fmt.Sprintf("undelegating with ratio from account '%s': %s", accAddrStr, err.Error()))
				return false
			}

			delegatedFreeAmount := totalDelegated.Sub(totalLocked)
			clawbackAccount.DelegatedFree = sdk.NewCoins(sdk.NewCoin("aISLM", delegatedFreeAmount))
			clawbackAccount.DelegatedVesting = sdk.NewCoins(sdk.NewCoin("aISLM", sdk.ZeroInt()))
		}
		ak.SetAccount(ctx, clawbackAccount)

		// -- Debug
		delegations = sk.GetAllDelegatorDelegations(ctx, accAddr)

		totalDelegatedAfter := getTotalDelegated(ctx, sk, delegations)

		totalUndelegated = totalUndelegated.Add(totalDelegated.Sub(totalDelegatedAfter))

		// -- Make liquid coins
		liquidateProtoMsg := &liquidvestingtypes.MsgLiquidate{
			LiquidateFrom: accAddrStr,
			LiquidateTo:   accAddrStr,
			Amount:        sdk.NewCoin("aISLM", totalLocked),
		}

		liquidateResp, err := lk.Liquidate(ctx, liquidateProtoMsg)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed liquidate from account '%s': %s", accAddrStr, err.Error()))
			return false
		}

		convertProtoMsg := &erc20types.MsgConvertERC20{
			ContractAddress: liquidateResp.ContractAddr,
			Sender:          accHexAddrStr,
			Receiver:        accAddrStr,
			Amount:          liquidateResp.Minted.Amount,
		}

		_, err = erc20k.ConvertERC20(ctx, convertProtoMsg)
		if err != nil {
			logger.Error(fmt.Sprintf("converting token msg: '%s'", convertProtoMsg))
		}

		// -- Fund
		fundCoins := sdk.NewCoins(liquidateResp.Minted)

		if err := dk.Fund(ctx, fundCoins, accAddr); err != nil {
			logger.Error(fmt.Sprintf("Failed fund from account '%s': %s", accAddrStr, err.Error()))
			return false
		}

		// -- Update account
		return false
	})

	// DAO Total Balance

	balances := dk.GetAccountsBalances(ctx)
	totalDaoBalanceInaISLM := sdk.ZeroInt()

	for _, balance := range balances {
		totalDaoBalanceInaISLM = totalDaoBalanceInaISLM.Add(balance.Coins[0].Amount)
	}

	logger.Info(fmt.Sprintf("Total undelegated: %s", totalUndelegated.String()))
	logger.Info(fmt.Sprintf("DAO total balance in aISLM: %s", totalDaoBalanceInaISLM.String()))

	logger.Info("Finished turning on DAO module")

	return nil
}

func getAddrHash(acc sdk.AccAddress) string {
	hexAddr := common.BytesToAddress(acc.Bytes())

	hasher := sha256.New()
	hexWalletLower := strings.ToLower(hexAddr.Hex())
	hasher.Write([]byte(hexWalletLower))
	hash := hasher.Sum(nil)

	hashHex := strings.ToUpper(hex.EncodeToString(hash))
	return hashHex
}

func matchedAddressHash(acc sdk.AccAddress) bool {
	hashHex := getAddrHash(acc)
	return WhiteListedAccounts[hashHex]
}

func runUndelegation(ctx sdk.Context, sk stakingkeeper.Keeper, delAddr sdk.AccAddress, bk bankkeeper.Keeper, delegation stakingtypes.Delegation, shares math.LegacyDec) error {
	bondDenom := sk.BondDenom(ctx)

	valAddr, _ := sdk.ValAddressFromBech32(delegation.GetValidatorAddr().String())
	validator, found := sk.GetValidator(ctx, valAddr)
	if !found {
		// Impossible as we are iterating over active delegations
		return fmt.Errorf("validator not found: %s", valAddr.String())
	}

	ubdAmount, err := sk.Unbond(ctx, delAddr, valAddr, shares)
	if err != nil {
		return fmt.Errorf("error unbonding from validator '%s': %s", valAddr.String(), err.Error())
	}

	// transfer the validator tokens to the not bonded pool
	coins := sdk.NewCoins(sdk.NewCoin(bondDenom, ubdAmount))
	if validator.IsBonded() {
		if err := bk.SendCoinsFromModuleToModule(ctx, stakingtypes.BondedPoolName, stakingtypes.NotBondedPoolName, coins); err != nil {
			return fmt.Errorf("error transferring tokens from bonded to not bonded pool: %s", err.Error())
		}
	}

	// transfer the validator tokens to the delegator
	if err := bk.UndelegateCoinsFromModuleToAccount(ctx, stakingtypes.NotBondedPoolName, delAddr, coins); err != nil {
		return fmt.Errorf("error transferring tokens from not bonded pool to delegator's address: %s", err.Error())
	}

	return nil
}

func undelegateWithRatio(ctx sdk.Context, sk stakingkeeper.Keeper, delegationsVector []stakingtypes.Delegation, bk bankkeeper.Keeper, totalLocked math.Int, totalDelegated math.Int) error {
	// stort delegations
	sort.Slice(delegationsVector, func(i, j int) bool {
		return bytes.Compare(delegationsVector[i].GetValidatorAddr().Bytes(), delegationsVector[j].GetValidatorAddr().Bytes()) < 0
	})

	// ---------

	delta := totalLocked

	for index, delegation := range delegationsVector {
		validator := sk.Validator(ctx, delegation.GetValidatorAddr())
		tokens := validator.TokensFromShares(delegation.GetShares()).TruncateInt()
		unstakingAmount := tokens.Mul(totalLocked).Quo(totalDelegated)

		delta = delta.Sub(unstakingAmount)

		if index == len(delegationsVector)-1 && !delta.IsZero() {
			unstakingAmount = unstakingAmount.Add(delta)
		}

		unstakingShares, err := validator.SharesFromTokens(unstakingAmount)
		if err != nil {
			return fmt.Errorf("error calculating shares from tokens: %s", err.Error())
		}

		if err := runUndelegation(ctx, sk, delegation.GetDelegatorAddr(), bk, delegation, unstakingShares); err != nil {
			return fmt.Errorf("error undelegating from validator '%s': %s", delegation.GetValidatorAddr().String(), err.Error())
		}
	}

	return nil
}

func undelegateAll(ctx sdk.Context, sk stakingkeeper.Keeper, bk bankkeeper.Keeper, delegationsVector []stakingtypes.Delegation) error {
	// stort delegations
	sort.Slice(delegationsVector, func(i, j int) bool {
		return bytes.Compare(delegationsVector[i].GetValidatorAddr().Bytes(), delegationsVector[j].GetValidatorAddr().Bytes()) < 0
	})

	for _, delegation := range delegationsVector {
		if err := runUndelegation(ctx, sk, delegation.GetDelegatorAddr(), bk, delegation, delegation.GetShares()); err != nil {
			return fmt.Errorf("error undelegating from validator '%s': %s", delegation.GetValidatorAddr().String(), err.Error())
		}
	}

	return nil
}

func getTotalDelegated(ctx sdk.Context, sk stakingkeeper.Keeper, delegations []stakingtypes.Delegation) math.Int {
	totalDelegated := math.NewInt(0)
	for _, delegation := range delegations {
		validator := sk.Validator(ctx, delegation.GetValidatorAddr())
		tokens := validator.TokensFromShares(delegation.GetShares()).TruncateInt()
		totalDelegated = totalDelegated.Add(tokens)
	}

	return totalDelegated
}

func forceStakingComplete(ctx sdk.Context, sk stakingkeeper.Keeper, accAddr sdk.AccAddress) {
	unbondingVec := sk.GetAllUnbondingDelegations(ctx, accAddr)

	ctx.Logger().Info(fmt.Sprintf("Dequeue all mature unbonding: %d for addr: %s", len(unbondingVec), accAddr.String()))

	for _, unbonding := range unbondingVec {
		for entry := range unbonding.Entries {
			unbonding.Entries[entry].CompletionTime = time.Unix(ctx.BlockTime().Unix()-100, 0)
		}

		sk.SetUnbondingDelegation(ctx, unbonding)

		_, err := sk.CompleteUnbonding(ctx, accAddr, sdk.ValAddress(unbonding.ValidatorAddress))
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("Error completing unbonding from validator '%s': %s", unbonding.ValidatorAddress, err.Error()))
		}
	}
}
