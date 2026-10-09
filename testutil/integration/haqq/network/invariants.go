package network

import (
	"fmt"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
)

// CheckAccountingInvariants runs the module invariants that a broken
// reconciliation between EVM and Cosmos state would violate - a coin minted or
// burned outside the bank's bookkeeping, a staking pool drained or topped up by
// an EVM commit, a distribution module account that no longer covers what it
// owes - and returns the first one found broken.
//
// It is opt-in rather than run on every block: some tests deliberately move
// coins between module accounts to set up a scenario, which these invariants
// would rightly flag.
func (n *IntegrationNetwork) CheckAccountingInvariants() error {
	invariants := []struct {
		name  string
		check sdktypes.Invariant
	}{
		{"bank total supply", bankkeeper.TotalSupply(n.app.BankKeeper)},
		{"staking module accounts", stakingkeeper.ModuleAccountInvariants(n.app.StakingKeeper.Keeper)},
		{"distribution module account", distrkeeper.ModuleAccountInvariant(n.app.DistrKeeper)},
		{"distribution non-negative outstanding", distrkeeper.NonNegativeOutstandingInvariant(n.app.DistrKeeper)},
	}

	for _, inv := range invariants {
		if msg, broken := inv.check(n.GetContext()); broken {
			return fmt.Errorf("%s invariant broken: %s", inv.name, msg)
		}
	}
	return nil
}
