package v150

import (
	"strconv"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/types"
	"github.com/pkg/errors"
)

func (r *RevestingUpgradeHandler) prepareWhitelistFromHistoryState(accounts []authtypes.AccountI) error {
	r.ctx.Logger().Info("Prepare whiltelist from history state")

	for _, acc := range accounts {
		// Check if account is a ETH account
		ethAcc, ok := acc.(*types.EthAccount)
		if !ok {
			r.ctx.Logger().Info(acc.GetAddress().String() + " - not a ETH Account")
			continue
		}

		balance := r.getBondableBalanceFromHistory(acc)
		delegated, err := r.getDelegatedCoinsFromHistory(acc)
		if err != nil {
			return errors.Wrap(err, "failed to get delegated coins from history for account "+acc.GetAddress().String())
		}

		balance = balance.Add(delegated)

		if !balance.IsZero() && balance.Amount.GT(r.threshold) {
			r.ctx.Logger().Info("balance greater that threshold: " + acc.GetAddress().String() + " (" + balance.String() + "); WILL BE REVESTED")
			continue
		}

		r.ctx.Logger().Info("balance lower that threshold: " + acc.GetAddress().String() + " (" + balance.String() + "); WHITELISTED")
		r.wl[*ethAcc] = true
	}

	return nil
}

func (r *RevestingUpgradeHandler) validateWhitelist() error {
	r.ctx.Logger().Info("Validate whiltelist with latest state")

	sk := r.StakingKeeper
	bk := r.BankKeeper
	bondDenom := sk.BondDenom(r.ctx)

	changes := 0
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

		evmAddr := common.BytesToAddress(acc.GetAddress().Bytes())
		vestingBalance, err := r.getVestingContractBalance(evmAddr)
		if err != nil {
			return errors.Wrap(err, "failed to get vesting contract balance")
		}

		if !vestingBalance.IsZero() {
			balance = balance.Add(vestingBalance)
		}

		if !balance.IsZero() && balance.Amount.GT(r.threshold) && ok {
			r.ctx.Logger().Info("balance now greater than threshold: " + acc.GetAddress().String() + " (" + balance.String() + "); WILL BE REVESTED")
			delete(r.wl, acc)
			changes++
		}
	}

	switch {
	case changes > 0:
		r.ctx.Logger().Info("Accounts excluded from whitelist: " + strconv.Itoa(changes))
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
