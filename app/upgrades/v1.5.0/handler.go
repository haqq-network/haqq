package v150

import (
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/types"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	"github.com/pkg/errors"

	vestingkeeper "github.com/haqq-network/haqq/x/vesting/keeper"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

type RevestingUpgradeHandler struct {
	ctx            sdk.Context
	AccountKeeper  authkeeper.AccountKeeper
	BankKeeper     bankkeeper.Keeper
	StakingKeeper  stakingkeeper.Keeper
	EvmKeeper      *evmkeeper.Keeper
	VestingKeeper  vestingkeeper.Keeper
	cdc            codec.Codec
	vals           map[string]math.Int
	threshold      math.Int
	wl             map[types.EthAccount]bool
	ignore         map[string]bool
	oldBalances    map[string]sdk.Coin
	oldDelegations map[string][]stakingtypes.Delegation
	oldValidators  map[string]stakingtypes.Validator
}

func NewRevestingUpgradeHandler(
	ctx sdk.Context,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	sk stakingkeeper.Keeper,
	evm *evmkeeper.Keeper,
	vk vestingkeeper.Keeper,
	threshold math.Int,
	cdc codec.Codec,
) *RevestingUpgradeHandler {
	return &RevestingUpgradeHandler{
		ctx:           ctx,
		AccountKeeper: ak,
		BankKeeper:    bk,
		StakingKeeper: sk,
		EvmKeeper:     evm,
		VestingKeeper: vk,
		cdc:           cdc,
		vals:          make(map[string]math.Int),
		threshold:     threshold,
		wl:            make(map[types.EthAccount]bool),
		ignore:        make(map[string]bool),
	}
}

func (r *RevestingUpgradeHandler) SetIgnoreList(list map[string]bool) {
	r.ignore = list
}

func (r *RevestingUpgradeHandler) GetIgnoreList() map[string]bool {
	return r.ignore
}

func (r *RevestingUpgradeHandler) Run() error {
	r.ctx.Logger().Info("Run revesting upgrade.")

	if err := r.loadHistoryBalancesState(); err != nil {
		return errors.Wrap(err, "error loading history balances state")
	}

	if err := r.loadHistoryStakingState(); err != nil {
		return errors.Wrap(err, "error loading history staking state")
	}

	r.ctx.Logger().Info(fmt.Sprintf("Found %d validators", len(r.oldValidators)))
	r.ctx.Logger().Info(fmt.Sprintf("Found %d delegations", len(r.oldDelegations)))
	r.ctx.Logger().Info(fmt.Sprintf("Found %d balances", len(r.oldBalances)))

	ubdPoolBalanceBefore := r.checkUnbondingPoolBalance()
	r.ctx.Logger().Info("Unbonding pool balance before: " + ubdPoolBalanceBefore.String())
	r.ctx.Logger().Info("Force unbonding and redelegation.")

	if err := r.forceDequeueUnbondingAndRedelegation(); err != nil {
		return errors.Wrap(err, "error force dequeue unbonding and redelegation")
	}

	ubdPoolBalanceAfter := r.checkUnbondingPoolBalance()
	r.ctx.Logger().Info("Unbonding pool balance after: " + ubdPoolBalanceAfter.String())

	accounts := r.AccountKeeper.GetAllAccounts(r.ctx)
	if len(accounts) == 0 {
		// Short circuit if there are no accounts
		r.ctx.Logger().Info("No accounts found")
		return nil
	}
	r.ctx.Logger().Info("Found accounts to process: " + strconv.Itoa(len(accounts)))

	r.ctx.Logger().Info("Withdraw funds from Vesting Contract...")
	withdrawnVestingAmounts := r.forceContractVestingWithdraw(accounts)

	if err := r.prepareWhitelistFromHistoryState(accounts); err != nil {
		return errors.Wrap(err, "error preparing whitelist from history state")
	}

	if err := r.validateWhitelist(); err != nil {
		return errors.Wrap(err, "error	validating whitelist")
	}

	for _, acc := range accounts {
		r.ctx.Logger().Info("---")
		r.ctx.Logger().Info("Account: " + acc.GetAddress().String())

		evmAddr := common.BytesToAddress(acc.GetAddress().Bytes())
		r.ctx.Logger().Info("EVM Account: " + evmAddr.Hex())

		// Check if account is a ETH account
		ethAcc, ok := acc.(*types.EthAccount)
		if !ok {
			r.ctx.Logger().Info("Not a ETH Account — skip")
			continue
		}

		if r.isAccountWhitelisted(*ethAcc) {
			r.ctx.Logger().Info("WHITELISTED — skip")
			continue
		}

		// Restake coins by default
		isSmartContract := false
		evmAcc := r.EvmKeeper.GetAccountWithoutBalance(r.ctx, evmAddr)
		if evmAcc != nil && evmAcc.IsContract() {
			r.ctx.Logger().Info("CONTRACT — do not delegate its coins")
			isSmartContract = true
		}

		// TODO Remove before release
		// Log balance before Undelegate
		balanceBeforeUND := r.BankKeeper.GetBalance(r.ctx, acc.GetAddress(), r.StakingKeeper.BondDenom(r.ctx))
		r.ctx.Logger().Info("Balance before undelegation: " + balanceBeforeUND.String())

		var (
			totalUndelegatedAmount sdk.Coin
			err                    error
		)
		restoreDelegations := make(map[*sdk.ValAddress]sdk.Coin)
		if !isSmartContract {
			// Undelegate all coins for account
			restoreDelegations, totalUndelegatedAmount, err = r.UndelegateAllTokens(acc.GetAddress())
			if err != nil {
				return errors.Wrap(err, "error undelegating tokens")
			}
			r.ctx.Logger().Info("Total undelegated amount: " + totalUndelegatedAmount.String())
		}

		// Get account balance
		balance := r.BankKeeper.GetBalance(r.ctx, acc.GetAddress(), r.StakingKeeper.BondDenom(r.ctx))
		r.ctx.Logger().Info("Balance before revesting: " + balance.String())

		// Send all coins to the vesting module account and process revesting with staking
		if err := r.Revesting(acc, balance); err != nil {
			return errors.Wrap(err, "error revesting")
		}

		if !isSmartContract {
			shares, err := r.Restaking(acc, balance, restoreDelegations)
			if err != nil {
				return errors.Wrap(err, "error restaking")
			}

			r.ctx.Logger().Info("New staking shares:")
			for valAddr, share := range shares {
				r.ctx.Logger().Info(share.String() + " to " + valAddr.String())
			}
		}

		// Delete entry from map to prevent double revesting
		delete(withdrawnVestingAmounts, acc.GetAddress().String())

		// TODO Remove before release
		// Log balance after revesting
		balanceAfter := r.BankKeeper.GetBalance(r.ctx, acc.GetAddress(), r.StakingKeeper.BondDenom(r.ctx))
		r.ctx.Logger().Info("Balance after (re)vesting: " + balanceAfter.String())
	}

	if err := r.FinalizeContractRevesting(withdrawnVestingAmounts); err != nil {
		return errors.Wrap(err, "error finalize contract revesting")
	}

	return nil
}

// UndelegateAllTokens undelegates all tokens from the all validators and returns the undelegated amount per validator
// and the total undelegated amount
func (r *RevestingUpgradeHandler) UndelegateAllTokens(delAddr sdk.AccAddress) (map[*sdk.ValAddress]sdk.Coin, sdk.Coin, error) {
	bondDenom := r.StakingKeeper.BondDenom(r.ctx)
	totalUndelegatedAmount := sdk.NewCoin(bondDenom, sdk.ZeroInt())

	delegations := r.StakingKeeper.GetAllDelegatorDelegations(r.ctx, delAddr)
	if len(delegations) == 0 {
		return nil, totalUndelegatedAmount, nil
	}

	// unbond from all validators
	undelegatedAmounts := make(map[*sdk.ValAddress]sdk.Coin, len(delegations))
	for _, delegation := range delegations {
		valAddr, _ := sdk.ValAddressFromBech32(delegation.GetValidatorAddr().String())
		validator, found := r.StakingKeeper.GetValidator(r.ctx, valAddr)
		if !found {
			// Impossible as we are iterating over active delegations
			return nil, totalUndelegatedAmount, errors.New("validator not found")
		}

		delegationShares := delegation.GetShares()
		isValidatorOperator := delAddr.Equals(validator.GetOperator())
		delegatedAmount := validator.TokensFromShares(delegationShares).TruncateInt()

		// Reduce unbonding amount if validator's min self delegation is less than threshold
		// and delegation is from validator's operator.
		// This is needed to avoid auto-jail validator on unbonding.
		// If validator's min self delegation is greater than threshold, then jail validator.
		if isValidatorOperator && validator.MinSelfDelegation.LT(r.threshold) {
			r.ctx.Logger().Info("Validator operator self delegation!")
			// Skip if delegated amount is less than validator's min self delegation
			if delegatedAmount.LT(validator.MinSelfDelegation) {
				continue
			}

			amountToUnbond := delegatedAmount.Sub(validator.MinSelfDelegation)
			sharesToUnbond, err := validator.SharesFromTokens(amountToUnbond)
			if err != nil {
				return nil, totalUndelegatedAmount, errors.Wrap(err, "failed to reduce unbonding amount")
			}

			r.ctx.Logger().Info(fmt.Sprintf(
				"Unbonding validator's self delegation: %s out of %s shares",
				sharesToUnbond.String(),
				delegationShares.String(),
			))

			delegationShares = sharesToUnbond
		}

		ubdAmount, err := r.StakingKeeper.Unbond(r.ctx, delAddr, valAddr, delegationShares)
		if err != nil {
			return nil, totalUndelegatedAmount, errors.Wrap(err, "failed to unbond tokens")
		}

		// transfer the validator tokens to the not bonded pool
		// TODO Cover unbonding and unbonded validators
		coins := sdk.NewCoins(sdk.NewCoin(bondDenom, ubdAmount))
		if validator.IsBonded() {
			if err := r.BankKeeper.SendCoinsFromModuleToModule(r.ctx, stakingtypes.BondedPoolName, stakingtypes.NotBondedPoolName, coins); err != nil {
				return nil, totalUndelegatedAmount, errors.Wrap(err, "failed to transfer tokens from bonded to not bonded pool")
			}

			if err := r.BankKeeper.UndelegateCoinsFromModuleToAccount(r.ctx, stakingtypes.NotBondedPoolName, delAddr, coins); err != nil {
				return nil, totalUndelegatedAmount, errors.Wrap(err, "failed to transfer tokens from not bonded pool to delegator's address")
			}

			r.reduceValidatorPower(valAddr.String(), ubdAmount)
		} else {
			// Should not happen
			r.ctx.Logger().Error("Validator is not bonded!")
			r.ctx.Logger().Error("Validator: " + valAddr.String())
			r.ctx.Logger().Error("Delegator: " + delAddr.String())
			r.ctx.Logger().Error("Amount: " + delegatedAmount.String())
			continue
		}

		undelegatedCoin := sdk.NewCoin(bondDenom, ubdAmount)
		undelegatedAmounts[&valAddr] = undelegatedCoin
		totalUndelegatedAmount = totalUndelegatedAmount.Add(undelegatedCoin)
	}

	return undelegatedAmounts, totalUndelegatedAmount, nil
}

func (r *RevestingUpgradeHandler) Revesting(acc authtypes.AccountI, coin sdk.Coin) error {
	moduleAcc := r.AccountKeeper.GetModuleAccount(r.ctx, vestingtypes.ModuleName)
	if err := r.BankKeeper.SendCoinsFromAccountToModule(
		r.ctx,
		acc.GetAddress(),
		vestingtypes.ModuleName,
		sdk.NewCoins(coin),
	); err != nil {
		return errors.Wrap(err, "failed to send coins to vesting module")
	}

	// Convert to a vesting account
	lockupPeriods, vestingPeriods := r.getVestingPeriods(coin)
	msg := vestingtypes.NewMsgConvertIntoVestingAccount(
		moduleAcc.GetAddress(),
		acc.GetAddress(),
		r.ctx.BlockTime(),
		lockupPeriods,
		vestingPeriods,
		true,
		false,
		nil,
	)

	_, err := r.VestingKeeper.ConvertIntoVestingAccount(r.ctx, msg)
	if err != nil {
		return errors.Wrap(err, "failed to convert into clawback vesting account")
	}

	return nil
}

func (r *RevestingUpgradeHandler) Restaking(acc authtypes.AccountI, totalAmount sdk.Coin, oldDelegations map[*sdk.ValAddress]sdk.Coin) (map[*sdk.ValAddress]sdk.Dec, error) {
	shares := make(map[*sdk.ValAddress]sdk.Dec)
	restAmount := totalAmount
	if len(oldDelegations) > 0 {
		r.ctx.Logger().Info("found old delegations, restaking...")
		for valAddr, amt := range oldDelegations {
			val, found := r.StakingKeeper.GetValidator(r.ctx, valAddr.Bytes())
			if !found {
				// Should never happen, but just in case
				return map[*sdk.ValAddress]sdk.Dec{}, errors.Wrapf(stakingtypes.ErrNoValidatorFound, "validator %s does not exist", valAddr)
			}

			newShares, err := r.StakingKeeper.Delegate(r.ctx, acc.GetAddress(), amt.Amount, stakingtypes.Unbonded, val, true)
			if err != nil {
				return map[*sdk.ValAddress]sdk.Dec{}, errors.Wrap(err, "failed to delegate")
			}

			r.increaseValidatorPower(valAddr.String(), amt.Amount)
			r.ctx.Logger().Info("restaked " + amt.String() + " to " + valAddr.String())

			restAmount = restAmount.Sub(amt)
			shares[valAddr] = newShares
		}
	}

	if restAmount.IsZero() {
		return shares, nil
	}

	val, valAddr, err := r.getWeakestValidator()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get weakest validator")
	}

	restShares, err := r.StakingKeeper.Delegate(r.ctx, acc.GetAddress(), restAmount.Amount, stakingtypes.Unbonded, *val, true)
	if err != nil {
		return map[*sdk.ValAddress]sdk.Dec{}, errors.Wrap(err, "failed to delegate")
	}
	shares[valAddr] = restShares

	// Add power to validator
	r.increaseValidatorPower(valAddr.String(), restAmount.Amount)

	return shares, nil
}

func (r *RevestingUpgradeHandler) FinalizeContractRevesting(withdrawnVestingAmounts map[string]sdk.Coin) error {
	r.ctx.Logger().Info("Finalize Contract Revesting")

	if len(withdrawnVestingAmounts) == 0 {
		r.ctx.Logger().Info("Nothing to do")
		return nil
	}

	for addr, amount := range withdrawnVestingAmounts {
		r.ctx.Logger().Info("---")
		r.ctx.Logger().Info("Account: " + addr)

		var (
			totalUndelegatedAmount sdk.Coin
			err                    error
		)
		accAddr := sdk.MustAccAddressFromBech32(addr)
		acc := types.ProtoAccount()
		if err := acc.SetAddress(accAddr); err != nil {
			return errors.Wrap(err, "get account from address")
		}
		restoreDelegations, totalUndelegatedAmount, err := r.UndelegateAllTokens(accAddr)
		if err != nil {
			return errors.Wrap(err, "error undelegating tokens")
		}
		r.ctx.Logger().Info("Total undelegated amount: " + totalUndelegatedAmount.String())

		balance := amount.Add(totalUndelegatedAmount)

		// Send all coins to the vesting module account and process revesting with staking
		if err := r.Revesting(acc, balance); err != nil {
			return errors.Wrap(err, "error revesting")
		}

		shares, err := r.Restaking(acc, balance, restoreDelegations)
		if err != nil {
			return errors.Wrap(err, "error restaking")
		}

		r.ctx.Logger().Info("New staking shares:")
		for valAddr, share := range shares {
			r.ctx.Logger().Info(share.String() + " to " + valAddr.String())
		}

		// TODO Remove before release
		// Log balance after revesting
		balanceAfter := r.BankKeeper.GetBalance(r.ctx, accAddr, r.StakingKeeper.BondDenom(r.ctx))
		r.ctx.Logger().Info("Balance after (re)vesting: " + balanceAfter.String())
	}

	return nil
}
