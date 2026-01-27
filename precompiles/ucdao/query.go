// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ucdao

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"

	cmn "github.com/haqq-network/haqq/precompiles/common"
)

// Balance returns the balance of a specific denom for an account in the UCDAO.
func (p Precompile) Balance(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	account, denom, err := ParseBalanceArgs(args)
	if err != nil {
		return nil, err
	}

	accAddr := sdk.AccAddress(account.Bytes())
	balance := p.ucdaoKeeper.GetBalance(ctx, accAddr, denom)

	return method.Outputs.Pack(balance.Amount.BigInt())
}

// AllBalances returns all balances for an account in the UCDAO.
func (p Precompile) AllBalances(
	ctx sdk.Context,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	account, err := ParseAllBalancesArgs(args)
	if err != nil {
		return nil, err
	}

	accAddr := sdk.AccAddress(account.Bytes())
	balances := p.ucdaoKeeper.GetAccountBalances(ctx, accAddr)

	return method.Outputs.Pack(cmn.NewCoinsResponse(balances))
}

// TotalBalance returns the total balance of the UCDAO.
func (p Precompile) TotalBalance(
	ctx sdk.Context,
	method *abi.Method,
	_ []interface{},
) ([]byte, error) {
	totalBalance := p.ucdaoKeeper.GetTotalBalance(ctx)

	return method.Outputs.Pack(cmn.NewCoinsResponse(totalBalance))
}

// Enabled returns whether the UCDAO module is enabled.
func (p Precompile) Enabled(
	ctx sdk.Context,
	method *abi.Method,
	_ []interface{},
) ([]byte, error) {
	enabled := p.ucdaoKeeper.IsModuleEnabled(ctx)

	return method.Outputs.Pack(enabled)
}
