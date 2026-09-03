// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package staking

import (
	"errors"
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	stakingkeeper "github.com/haqq-network/haqq/x/staking/keeper"
)

const (
	// CreateValidatorMethod defines the ABI method name for the staking create validator transaction
	CreateValidatorMethod = "createValidator"
	// EditValidatorMethod defines the ABI method name for the staking edit validator transaction
	EditValidatorMethod = "editValidator"
	// DelegateMethod defines the ABI method name for the staking Delegate
	// transaction.
	DelegateMethod = "delegate"
	// UndelegateMethod defines the ABI method name for the staking Undelegate
	// transaction.
	UndelegateMethod = "undelegate"
	// RedelegateMethod defines the ABI method name for the staking Redelegate
	// transaction.
	RedelegateMethod = "redelegate"
	// CancelUnbondingDelegationMethod defines the ABI method name for the staking
	// CancelUnbondingDelegation transaction.
	CancelUnbondingDelegationMethod = "cancelUnbondingDelegation"
)

const (
	// DelegateAuthz defines the authorization type for the staking Delegate
	DelegateAuthz = stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE
	// UndelegateAuthz defines the authorization type for the staking Undelegate
	UndelegateAuthz = stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_UNDELEGATE
	// RedelegateAuthz defines the authorization type for the staking Redelegate
	RedelegateAuthz = stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_REDELEGATE
	// CancelUnbondingDelegationAuthz defines the authorization type for the staking
	CancelUnbondingDelegationAuthz = stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_CANCEL_UNBONDING_DELEGATION
)

// bankBaseSnapshot pairs an account with its sampled aISLM bank balance,
// captured immediately before a keeper call.
type bankBaseSnapshot struct {
	addr       sdk.AccAddress
	baseBefore sdkmath.Int
}

func (p *Precompile) snapshotBankBase(ctx sdk.Context, addr sdk.AccAddress) bankBaseSnapshot {
	return bankBaseSnapshot{
		addr:       addr,
		baseBefore: p.stakingKeeper.BaseDenomBankBalance(ctx, addr),
	}
}

// snapshotStakeAccounts samples aISLM of the delegator and of their distribution
// withdrawer. Cosmos SDK withdraws outstanding rewards onto the withdrawer
// (BeforeDelegationSharesModified) before Delegate / Undelegate / Redelegate /
// CancelUnbondingDelegation mutate shares. Duplicate addresses are kept; the
// mirror helper journals a single net delta per account.
func (p *Precompile) snapshotStakeAccounts(ctx sdk.Context, delegator sdk.AccAddress) ([]bankBaseSnapshot, error) {
	snaps := make([]bankBaseSnapshot, 0, 2)
	snaps = append(snaps, p.snapshotBankBase(ctx, delegator))
	withdrawer, err := p.distrKeeper.GetDelegatorWithdrawAddr(ctx, delegator)
	if err != nil {
		return nil, err
	}
	snaps = append(snaps, p.snapshotBankBase(ctx, withdrawer))
	return snaps, nil
}

// mirrorBankBaseDeltasIntoStateDB mirrors per-account bank deltas of the EVM
// gas denom (aISLM) into the EVM StateDB journal when the precompile is invoked
// from another contract (caller != origin).
//
// Staking messages update bank via the SDK (bonded tokens, auto-claimed
// rewards) without EVM journal entries. On Commit, SetBalance restores a stale
// EVM balance for any dirty account and mints or burns the difference. Journal
// the actual aISLM delta so commit stays conservative.
func (p *Precompile) mirrorBankBaseDeltasIntoStateDB(
	ctx sdk.Context,
	isCallerOrigin bool,
	snapshots ...bankBaseSnapshot,
) {
	if isCallerOrigin {
		return
	}
	seen := make(map[string]struct{}, len(snapshots))
	for _, s := range snapshots {
		key := s.addr.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		netBaseDelta := s.baseBefore.Sub(p.stakingKeeper.BaseDenomBankBalance(ctx, s.addr))
		if netBaseDelta.IsZero() {
			continue
		}
		// A withdraw address may be a non-EVM Cosmos account (e.g. a 32-byte
		// ADR-028 address). Those cannot be reconciled at Commit, so they must
		// not be journaled - see cmn.JournalableEVMAddress.
		hexAddr, ok := cmn.JournalableEVMAddress(s.addr)
		if !ok {
			continue
		}
		if netBaseDelta.IsNegative() {
			p.AddBalanceChangeEntries(cmn.NewBalanceChangeEntry(hexAddr, netBaseDelta.Neg().BigInt(), cmn.Add))
			continue
		}
		p.AddBalanceChangeEntries(cmn.NewBalanceChangeEntry(hexAddr, netBaseDelta.BigInt(), cmn.Sub))
	}
}

func (p *Precompile) snapshotAndRun(
	ctx sdk.Context,
	isCallerOrigin bool,
	delegatorBech32 string,
	run func() error,
) error {
	var snaps []bankBaseSnapshot
	if !isCallerOrigin {
		// Deliberately not MustAccAddressFromBech32: HandleGasError only recovers
		// out-of-gas panics and re-raises everything else, so a malformed address
		// here would take the node down instead of failing the call.
		delegator, err := sdk.AccAddressFromBech32(delegatorBech32)
		if err != nil {
			return err
		}
		snaps, err = p.snapshotStakeAccounts(ctx, delegator)
		if err != nil {
			return err
		}
	}
	if err := run(); err != nil {
		return err
	}
	p.mirrorBankBaseDeltasIntoStateDB(ctx, isCallerOrigin, snaps...)
	return nil
}

// CreateValidator performs create validator.
func (p Precompile) CreateValidator(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	bondDenom, err := p.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return nil, err
	}
	msg, validatorHexAddr, err := NewMsgCreateValidator(args, bondDenom)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"commission", msg.Commission.String(),
		"min_self_delegation", msg.MinSelfDelegation.String(),
		"validator_address", validatorHexAddr.String(),
		"pubkey", msg.Pubkey.String(),
		"value", msg.Value.Amount.String(),
	)

	// ATM there's no authorization type for the MsgCreateValidator
	// and MsgEditValidator (source: https://github.com/cosmos/cosmos-sdk/blob/4bd73b667f8aed50ad4602ddf862a4ed6e1450a8/x/staking/proto/cosmos/staking/v1beta1/authz.proto#L39-L50)
	// so, for the time being, we won't allow calls from smart contracts
	if contract.CallerAddress != origin {
		return nil, errors.New(ErrCannotCallFromContract)
	}

	// we only allow the tx signer "origin" to create their own validator.
	if origin != validatorHexAddr {
		return nil, fmt.Errorf(ErrDifferentOriginFromDelegator, origin.String(), validatorHexAddr.String())
	}

	// Execute the transaction using the message server
	msgSrv := stakingkeeper.NewMsgServerImpl(&p.stakingKeeper)
	if _, err = msgSrv.CreateValidator(ctx, msg); err != nil {
		return nil, err
	}

	// Here we don't add journal entries here because calls from
	// smart contracts are not supported at the moment for this method.

	// Emit the event for the create validator transaction
	if err = p.EmitCreateValidatorEvent(ctx, stateDB, msg, validatorHexAddr); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// EditValidator performs edit validator.
func (p Precompile) EditValidator(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, validatorHexAddr, err := NewMsgEditValidator(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"validator_address", msg.ValidatorAddress,
		"commission_rate", msg.CommissionRate,
		"min_self_delegation", msg.MinSelfDelegation,
	)

	// ATM there's no authorization type for the MsgCreateValidator
	// and MsgEditValidator (source: https://github.com/cosmos/cosmos-sdk/blob/4bd73b667f8aed50ad4602ddf862a4ed6e1450a8/x/staking/proto/cosmos/staking/v1beta1/authz.proto#L39-L50)
	// so, for the time being, we won't allow calls from smart contracts
	if contract.CallerAddress != origin {
		return nil, errors.New(ErrCannotCallFromContract)
	}

	// we only allow the tx signer "origin" to edit their own validator.
	if origin != validatorHexAddr {
		return nil, fmt.Errorf(ErrDifferentOriginFromValidator, origin.String(), validatorHexAddr.String())
	}

	// Execute the transaction using the message server
	msgSrv := stakingkeeper.NewMsgServerImpl(&p.stakingKeeper)
	if _, err = msgSrv.EditValidator(ctx, msg); err != nil {
		return nil, err
	}

	// Emit the event for the edit validator transaction
	if err = p.EmitEditValidatorEvent(ctx, stateDB, msg, validatorHexAddr); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// Delegate performs a delegation of coins from a delegator to a validator.
func (p *Precompile) Delegate(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	bondDenom, err := p.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return nil, err
	}
	msg, delegatorHexAddr, err := NewMsgDelegate(args, bondDenom)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ delegator_address: %s, validator_address: %s, amount: %s }",
			delegatorHexAddr,
			msg.ValidatorAddress,
			msg.Amount.Amount,
		),
	)

	var (
		// stakeAuthz is the authorization grant for the caller and the delegator address
		stakeAuthz *stakingtypes.StakeAuthorization
		// expiration is the expiration time of the authorization grant
		expiration *time.Time

		// isCallerOrigin is true when the contract caller is the same as the origin
		isCallerOrigin = contract.CallerAddress == origin
		// isCallerDelegator is true when the contract caller is the same as the delegator
		isCallerDelegator = contract.CallerAddress == delegatorHexAddr
	)

	// The provided delegator address should always be equal to the origin address.
	// In case the contract caller address is the same as the delegator address provided,
	// update the delegator address to be equal to the origin address.
	// Otherwise, if the provided delegator address is different from the origin address,
	// return an error because is a forbidden operation
	if isCallerDelegator {
		delegatorHexAddr = origin
	} else if origin != delegatorHexAddr {
		return nil, fmt.Errorf(ErrDifferentOriginFromDelegator, origin.String(), delegatorHexAddr.String())
	}

	// no need to have authorization when the contract caller is the same as origin (owner of funds)
	if !isCallerOrigin {
		// Check if the authorization grant exists for the caller and the origin
		stakeAuthz, expiration, err = authorization.CheckAuthzAndAllowanceForGranter(ctx, p.AuthzKeeper, contract.CallerAddress, delegatorHexAddr, &msg.Amount, DelegateMsg)
		if err != nil {
			return nil, err
		}
	}

	// Execute the transaction using the message server
	msgSrv := stakingkeeper.NewMsgServerImpl(&p.stakingKeeper)
	if err = p.snapshotAndRun(ctx, isCallerOrigin, msg.DelegatorAddress, func() error {
		_, err := msgSrv.Delegate(ctx, msg)
		return err
	}); err != nil {
		return nil, err
	}

	// Only update the authorization if the contract caller is different from the origin
	if !isCallerOrigin {
		if err := p.UpdateStakingAuthorization(ctx, contract.CallerAddress, delegatorHexAddr, stakeAuthz, expiration, DelegateMsg, msg); err != nil {
			return nil, err
		}
	}

	// Emit the event for the delegate transaction
	if err = p.EmitDelegateEvent(ctx, stateDB, msg, delegatorHexAddr); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// Undelegate performs the undelegation of coins from a validator for a delegate.
// The provided amount cannot be negative. This is validated in the msg.ValidateBasic() function.
func (p *Precompile) Undelegate(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	bondDenom, err := p.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return nil, err
	}
	msg, delegatorHexAddr, err := NewMsgUndelegate(args, bondDenom)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ delegator_address: %s, validator_address: %s, amount: %s }",
			delegatorHexAddr,
			msg.ValidatorAddress,
			msg.Amount.Amount,
		),
	)

	var (
		// stakeAuthz is the authorization grant for the caller and the delegator address
		stakeAuthz *stakingtypes.StakeAuthorization
		// expiration is the expiration time of the authorization grant
		expiration *time.Time

		// isCallerOrigin is true when the contract caller is the same as the origin
		isCallerOrigin = contract.CallerAddress == origin
		// isCallerDelegator is true when the contract caller is the same as the delegator
		isCallerDelegator = contract.CallerAddress == delegatorHexAddr
	)

	// The provided delegator address should always be equal to the origin address.
	// In case the contract caller address is the same as the delegator address provided,
	// update the delegator address to be equal to the origin address.
	// Otherwise, if the provided delegator address is different from the origin address,
	// return an error because is a forbidden operation
	if isCallerDelegator {
		delegatorHexAddr = origin
	} else if origin != delegatorHexAddr {
		return nil, fmt.Errorf(ErrDifferentOriginFromDelegator, origin.String(), delegatorHexAddr.String())
	}

	// no need to have authorization when the contract caller is the same as origin (owner of funds)
	if !isCallerOrigin {
		// Check if the authorization grant exists for the caller and the origin
		stakeAuthz, expiration, err = authorization.CheckAuthzAndAllowanceForGranter(ctx, p.AuthzKeeper, contract.CallerAddress, delegatorHexAddr, &msg.Amount, UndelegateMsg)
		if err != nil {
			return nil, err
		}
	}

	// Execute the transaction using the message server
	msgSrv := stakingkeeper.NewMsgServerImpl(&p.stakingKeeper)
	var res *stakingtypes.MsgUndelegateResponse
	if err = p.snapshotAndRun(ctx, isCallerOrigin, msg.DelegatorAddress, func() error {
		var runErr error
		res, runErr = msgSrv.Undelegate(ctx, msg)
		return runErr
	}); err != nil {
		return nil, err
	}

	// Only update the authorization if the contract caller is different from the origin
	if !isCallerOrigin {
		if err := p.UpdateStakingAuthorization(ctx, contract.CallerAddress, delegatorHexAddr, stakeAuthz, expiration, UndelegateMsg, msg); err != nil {
			return nil, err
		}
	}

	// Emit the event for the undelegate transaction
	if err = p.EmitUnbondEvent(ctx, stateDB, msg, delegatorHexAddr, res.CompletionTime.UTC().Unix()); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.CompletionTime.UTC().Unix())
}

// Redelegate performs a redelegation of coins for a delegate from a source validator
// to a destination validator.
// The provided amount cannot be negative. This is validated in the msg.ValidateBasic() function.
func (p *Precompile) Redelegate(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	bondDenom, err := p.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return nil, err
	}
	msg, delegatorHexAddr, err := NewMsgRedelegate(args, bondDenom)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ delegator_address: %s, validator_src_address: %s, validator_dst_address: %s, amount: %s }",
			delegatorHexAddr,
			msg.ValidatorSrcAddress,
			msg.ValidatorDstAddress,
			msg.Amount.Amount,
		),
	)

	var (
		// stakeAuthz is the authorization grant for the caller and the delegator address
		stakeAuthz *stakingtypes.StakeAuthorization
		// expiration is the expiration time of the authorization grant
		expiration *time.Time

		// isCallerOrigin is true when the contract caller is the same as the origin
		isCallerOrigin = contract.CallerAddress == origin
		// isCallerDelegator is true when the contract caller is the same as the delegator
		isCallerDelegator = contract.CallerAddress == delegatorHexAddr
	)

	// The provided delegator address should always be equal to the origin address.
	// In case the contract caller address is the same as the delegator address provided,
	// update the delegator address to be equal to the origin address.
	// Otherwise, if the provided delegator address is different from the origin address,
	// return an error because is a forbidden operation
	if isCallerDelegator {
		delegatorHexAddr = origin
	} else if origin != delegatorHexAddr {
		return nil, fmt.Errorf(ErrDifferentOriginFromDelegator, origin.String(), delegatorHexAddr.String())
	}

	// no need to have authorization when the contract caller is the same as origin (owner of funds)
	if !isCallerOrigin {
		// Check if the authorization grant exists for the caller and the origin
		stakeAuthz, expiration, err = authorization.CheckAuthzAndAllowanceForGranter(ctx, p.AuthzKeeper, contract.CallerAddress, delegatorHexAddr, &msg.Amount, RedelegateMsg)
		if err != nil {
			return nil, err
		}
	}

	msgSrv := stakingkeeper.NewMsgServerImpl(&p.stakingKeeper)
	var res *stakingtypes.MsgBeginRedelegateResponse
	if err = p.snapshotAndRun(ctx, isCallerOrigin, msg.DelegatorAddress, func() error {
		var runErr error
		res, runErr = msgSrv.BeginRedelegate(ctx, msg)
		return runErr
	}); err != nil {
		return nil, err
	}

	// Only update the authorization if the contract caller is different from the origin
	if !isCallerOrigin {
		if err := p.UpdateStakingAuthorization(ctx, contract.CallerAddress, delegatorHexAddr, stakeAuthz, expiration, RedelegateMsg, msg); err != nil {
			return nil, err
		}
	}

	if err = p.EmitRedelegateEvent(ctx, stateDB, msg, delegatorHexAddr, res.CompletionTime.UTC().Unix()); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.CompletionTime.UTC().Unix())
}

// CancelUnbondingDelegation will cancel the unbonding of a delegation and delegate
// back to the validator being unbonded from.
// The provided amount cannot be negative. This is validated in the msg.ValidateBasic() function.
func (p *Precompile) CancelUnbondingDelegation(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	bondDenom, err := p.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return nil, err
	}
	msg, delegatorHexAddr, err := NewMsgCancelUnbondingDelegation(args, bondDenom)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ delegator_address: %s, validator_address: %s, amount: %s, creation_height: %d }",
			delegatorHexAddr,
			msg.ValidatorAddress,
			msg.Amount.Amount,
			msg.CreationHeight,
		),
	)

	var (
		// stakeAuthz is the authorization grant for the caller and the delegator address
		stakeAuthz *stakingtypes.StakeAuthorization
		// expiration is the expiration time of the authorization grant
		expiration *time.Time

		// isCallerOrigin is true when the contract caller is the same as the origin
		isCallerOrigin = contract.CallerAddress == origin
		// isCallerDelegator is true when the contract caller is the same as the delegator
		isCallerDelegator = contract.CallerAddress == delegatorHexAddr
	)

	// The provided delegator address should always be equal to the origin address.
	// In case the contract caller address is the same as the delegator address provided,
	// update the delegator address to be equal to the origin address.
	// Otherwise, if the provided delegator address is different from the origin address,
	// return an error because is a forbidden operation
	if isCallerDelegator {
		delegatorHexAddr = origin
	} else if origin != delegatorHexAddr {
		return nil, fmt.Errorf(ErrDifferentOriginFromDelegator, origin.String(), delegatorHexAddr.String())
	}

	// no need to have authorization when the contract caller is the same as origin (owner of funds)
	if !isCallerOrigin {
		// Check if the authorization grant exists for the caller and the origin
		stakeAuthz, expiration, err = authorization.CheckAuthzAndAllowanceForGranter(ctx, p.AuthzKeeper, contract.CallerAddress, delegatorHexAddr, &msg.Amount, CancelUnbondingDelegationMsg)
		if err != nil {
			return nil, err
		}
	}

	msgSrv := stakingkeeper.NewMsgServerImpl(&p.stakingKeeper)
	if err = p.snapshotAndRun(ctx, isCallerOrigin, msg.DelegatorAddress, func() error {
		_, err := msgSrv.CancelUnbondingDelegation(ctx, msg)
		return err
	}); err != nil {
		return nil, err
	}

	// Only update the authorization if the contract caller is different from the origin
	if !isCallerOrigin {
		if err := p.UpdateStakingAuthorization(ctx, contract.CallerAddress, delegatorHexAddr, stakeAuthz, expiration, CancelUnbondingDelegationMsg, msg); err != nil {
			return nil, err
		}
	}

	if err = p.EmitCancelUnbondingDelegationEvent(ctx, stateDB, msg, delegatorHexAddr); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}
