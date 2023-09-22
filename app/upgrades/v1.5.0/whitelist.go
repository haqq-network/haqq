package v150

import (
	"fmt"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/evmos/ethermint/types"
	"github.com/pkg/errors"
)

func (r *RevestingUpgradeHandler) prepareWhitelistFromHistoryState(accounts []authtypes.AccountI) error {
	r.ctx.Logger().Info("Prepare whiltelist from historical state")
	r.ctx.Logger().Info("Accounts to be revested:")

	toBeRevested := 0
	for _, acc := range accounts {
		ethAcc, ok := acc.(*types.EthAccount)
		if !ok {
			continue
		}

		balance := r.getBondableBalanceFromHistory(acc)
		delegated, err := r.getDelegatedCoinsFromHistory(acc)
		if err != nil {
			return errors.Wrap(err, "failed to get delegated coins from history for account "+acc.GetAddress().String())
		}

		balance = balance.Add(delegated)

		if !balance.IsZero() && balance.Amount.GTE(r.threshold) {
			r.ctx.Logger().Info(fmt.Sprintf(" - %s: %s", acc.GetAddress().String(), balance.String()))
			toBeRevested++
			continue
		}

		r.wl[*ethAcc] = true
	}

	r.ctx.Logger().Info(fmt.Sprintf("Number of accounts to be revested: %d", toBeRevested))

	return nil
}

func (r *RevestingUpgradeHandler) validateWhitelist() error {
	r.ctx.Logger().Info("Validate whiltelist with latest state")

	sk := r.StakingKeeper
	bk := r.BankKeeper
	bondDenom := sk.BondDenom(r.ctx)
	changes := 0

	r.ctx.Logger().Info("Accounts excluded from whitelist and will be revested:")
	for acc, ok := range r.wl {
		balance := bk.GetBalance(r.ctx, acc.GetAddress(), bondDenom)
		delegations := sk.GetAllDelegatorDelegations(r.ctx, acc.GetAddress())
		delegationResps, err := stakingkeeper.DelegationsToDelegationResponses(r.ctx, sk, delegations)
		if err != nil {
			return errors.Wrap(err, "failed to get delegation responses for account "+acc.GetAddress().String())
		}

		for _, delegation := range delegationResps {
			balance = balance.Add(delegation.Balance)
		}

		if !balance.IsZero() && balance.Amount.GTE(r.threshold) && ok {
			r.ctx.Logger().Info(fmt.Sprintf(" - %s: %s", acc.GetAddress().String(), balance.String()))
			delete(r.wl, acc)
			changes++
		}
	}

	switch {
	case changes > 0:
		r.ctx.Logger().Info(fmt.Sprintf("Number of excluded accounts: %d", changes))
	default:
		r.ctx.Logger().Info("Nothing changed.")
	}

	return nil
}

func (r *RevestingUpgradeHandler) isAccountWhitelisted(acc types.EthAccount) bool {
	allowed, ok := r.wl[acc]
	if !ok || !allowed {
		return r.isAccountIgnored(acc.GetAddress().String())
	}

	return allowed
}

func (r *RevestingUpgradeHandler) isAccountIgnored(acc string) bool {
	allowed, ok := r.ignore[acc]
	if !ok {
		return false
	}

	return allowed
}
