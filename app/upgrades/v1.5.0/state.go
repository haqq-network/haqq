package v150

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
)

func (r *RevestingUpgradeHandler) loadHistoryBalancesState() error {
	r.ctx.Logger().Info("Loading history balances state")
	bondDenom := r.StakingKeeper.BondDenom(r.ctx)

	var bankState banktypes.GenesisState
	if err := r.cdc.UnmarshalJSON(bankStateJSON, &bankState); err != nil {
		return errors.Wrap(err, "failed to unmarshal bank state")
	}

	if len(bankState.Balances) == 0 {
		return errors.New("empty balances")
	}

	r.oldBalances = make(map[string]sdk.Coin, len(bankState.Balances))
	for _, balance := range bankState.Balances {
		zero := sdk.NewCoin(bondDenom, math.ZeroInt())
		r.oldBalances[balance.Address] = zero
		for _, coin := range balance.Coins {
			if coin.Denom == bondDenom {
				r.oldBalances[balance.Address] = coin
				break
			}
		}
	}

	return nil
}

func (r *RevestingUpgradeHandler) loadHistoryStakingState() error {
	r.ctx.Logger().Info("Loading history balances state")

	var stakingState stakingtypes.GenesisState
	if err := r.cdc.UnmarshalJSON(stakingStateJSON, &stakingState); err != nil {
		return errors.Wrap(err, "failed to unmarshal staking state")
	}

	if len(stakingState.Delegations) == 0 {
		return errors.Wrap(stakingtypes.ErrNoDelegation, "empty delegations")
	}

	r.oldDelegations = make(map[string][]stakingtypes.Delegation, len(stakingState.Delegations))
	for _, delegation := range stakingState.Delegations {
		r.oldDelegations[delegation.DelegatorAddress] = append(r.oldDelegations[delegation.DelegatorAddress], delegation)
	}

	if len(stakingState.Validators) == 0 {
		return errors.Wrap(stakingtypes.ErrNoValidatorFound, "empty validators")
	}

	r.oldValidators = make(map[string]stakingtypes.Validator, len(stakingState.Validators))
	for _, validator := range stakingState.Validators {
		r.oldValidators[validator.OperatorAddress] = validator
	}

	return nil
}

func (r *RevestingUpgradeHandler) getBondableBalanceFromHistory(acc authtypes.AccountI) sdk.Coin {
	balance, found := r.oldBalances[acc.GetAddress().String()]
	if !found {
		return sdk.NewCoin(r.StakingKeeper.BondDenom(r.ctx), math.ZeroInt())
	}

	return balance
}

func (r *RevestingUpgradeHandler) getDelegatedCoinsFromHistory(acc authtypes.AccountI) (sdk.Coin, error) {
	bondDenom := r.StakingKeeper.BondDenom(r.ctx)
	amount := sdk.NewCoin(bondDenom, math.ZeroInt())

	delegations := r.oldDelegations[acc.GetAddress().String()]
	for _, delegation := range delegations {
		val, found := r.oldValidators[delegation.GetValidatorAddr().String()]
		if !found {
			return amount, errors.Wrap(stakingtypes.ErrNoValidatorFound, "failed to get validator from history store")
		}

		amount = amount.Add(sdk.NewCoin(bondDenom, val.TokensFromShares(delegation.Shares).TruncateInt()))
	}

	return amount, nil
}
