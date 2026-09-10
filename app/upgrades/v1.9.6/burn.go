package v196

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	ethtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	stakingkeeper "github.com/haqq-network/haqq/x/staking/keeper"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// BurnDenoms are the denominations removed from the frozen accounts, in the fixed
// order every step walks them.
var BurnDenoms = []string{ethiqtypes.BaseDenom, utils.BaseDenom}

const (
	EventTypeHackerFundsBurned    = "hacker_funds_burned"
	AttributeKeyAccount           = "account"
	AttributeKeyBurnedFromAccount = "burned_from_account"
	AttributeKeyBurnedFromEscrow  = "burned_from_ucdao_escrow"
)

// BurnHackerFunds burns every aISLM and aHAQQ the given accounts control: bank
// balances, delegations, unbondings and redelegations (settled immediately,
// without waiting for the unbonding period), delegation rewards, validator
// commission and the ucDAO escrow. The escrow goes through the same steps as the
// account itself, since a third party can put a vesting schedule and a delegation
// on it. It also revokes every authz grant and fee allowance the accounts are
// party to, then verifies nothing is left.
//
// Determinism: accounts are walked in sort.Strings order, every collection read
// from a store is sorted by a unique key before it is acted on, stores are never
// mutated while being iterated, and any failure is returned instead of skipped.
//
// Burning goes through the evm module account - the same send-then-burn the EVM
// keeper uses for balance decreases. It must not go through the staking pools or
// gov: the haqq bank keeper diverts their BurnCoins to the community pool.
func BurnHackerFunds(ctx sdk.Context, k Keepers, accounts []string) error {
	logger := ctx.Logger().With("upgrade", UpgradeName)

	addrs, err := parseAccounts(accounts)
	if err != nil {
		return err
	}

	// Membership lookups only; nothing below ranges over this map.
	targets := make(map[string]bool, len(addrs))
	for _, addr := range addrs {
		targets[addr.String()] = true
	}

	evmModule := k.AccountKeeper.GetModuleAddress(evmtypes.ModuleName)
	supplyBefore := supplyOf(ctx, k.BankKeeper)
	evmModuleBefore := balancesOf(ctx, k.BankKeeper, evmModule)
	trackedEscrowBefore := k.DaoKeeper.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount

	burned := sdk.NewCoins()
	burnedFromEscrow := sdk.NewCoins()
	for _, addr := range addrs {
		fromAccount, fromEscrow, err := burnAccount(ctx, k, addr)
		if err != nil {
			return errorsmod.Wrapf(err, "account %s", addr)
		}

		burned = burned.Add(fromAccount...).Add(fromEscrow...)
		burnedFromEscrow = burnedFromEscrow.Add(fromEscrow...)

		if !fromAccount.IsZero() || !fromEscrow.IsZero() {
			logger.Info("burned funds of frozen account",
				"account", addr.String(),
				"from_account", fromAccount.String(),
				"from_ucdao_escrow", fromEscrow.String(),
			)
		}
	}

	if err := revokeAuthzGrants(ctx, k.AuthzKeeper, targets); err != nil {
		return err
	}
	if err := revokeFeeAllowances(ctx, k.FeeGrantKeeper, targets); err != nil {
		return err
	}

	for _, addr := range addrs {
		if err := verifyAccountCleared(ctx, k, addr); err != nil {
			return errorsmod.Wrapf(err, "account %s", addr)
		}
	}
	if err := verifyNoGrants(ctx, k, targets); err != nil {
		return err
	}

	// Supply must drop by exactly what was burned. This also proves nothing was
	// diverted elsewhere (e.g. to the community pool) instead of being burned.
	supplyAfter := supplyOf(ctx, k.BankKeeper)
	for _, denom := range BurnDenoms {
		expected := supplyBefore.AmountOf(denom).Sub(burned.AmountOf(denom))
		if got := supplyAfter.AmountOf(denom); !got.Equal(expected) {
			return fmt.Errorf("%s supply is %s, expected %s after burning %s", denom, got, expected, burned.AmountOf(denom))
		}
	}

	if evmModuleAfter := balancesOf(ctx, k.BankKeeper, evmModule); !evmModuleAfter.Equal(evmModuleBefore) {
		return fmt.Errorf("evm module balance changed from %s to %s", evmModuleBefore, evmModuleAfter)
	}

	// TrackSubBalance saturates at zero, so sequential subtractions add up to a
	// single clamped one.
	expectedTracked := sdkmath.MaxInt(trackedEscrowBefore.Sub(burnedFromEscrow.AmountOf(utils.BaseDenom)), sdkmath.ZeroInt())
	if tracked := k.DaoKeeper.GetTotalBalanceOf(ctx, utils.BaseDenom).Amount; !tracked.Equal(expectedTracked) {
		return fmt.Errorf("ucdao total %s is %s, expected %s", utils.BaseDenom, tracked, expectedTracked)
	}

	logger.Info("burned funds of frozen accounts",
		"accounts", len(addrs),
		"total", burned.String(),
		"from_ucdao_escrow", burnedFromEscrow.String(),
	)

	return nil
}

// parseAccounts returns the accounts in sort.Strings order, rejecting
// duplicates and non-canonical bech32.
func parseAccounts(accounts []string) ([]sdk.AccAddress, error) {
	sorted := slices.Clone(accounts)
	sort.Strings(sorted)

	addrs := make([]sdk.AccAddress, 0, len(sorted))
	for i, bech := range sorted {
		if i > 0 && sorted[i-1] == bech {
			return nil, fmt.Errorf("duplicate account %s", bech)
		}

		addr, err := sdk.AccAddressFromBech32(bech)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "invalid account %s", bech)
		}
		if addr.String() != bech {
			return nil, fmt.Errorf("non-canonical account %s", bech)
		}

		addrs = append(addrs, addr)
	}

	return addrs, nil
}

// burnAccount releases everything the account and its ucDAO escrow hold in
// staking and distribution onto their balances, then burns both balances.
func burnAccount(ctx sdk.Context, k Keepers, addr sdk.AccAddress) (fromAccount, fromEscrow sdk.Coins, err error) {
	escrow := ucdaotypes.GetEscrowAddress(addr)
	for _, holder := range []sdk.AccAddress{addr, escrow} {
		if err := releaseFunds(ctx, k, holder); err != nil {
			return nil, nil, errorsmod.Wrapf(err, "release funds of %s", holder)
		}
	}

	fromEscrow, err = burnEscrow(ctx, k, addr, escrow)
	if err != nil {
		return nil, nil, errorsmod.Wrap(err, "burn ucdao escrow")
	}

	fromAccount = balancesOf(ctx, k.BankKeeper, addr)
	if err := burnFrom(ctx, k.BankKeeper, addr, fromAccount); err != nil {
		return nil, nil, errorsmod.Wrap(err, "burn balance")
	}

	if !fromAccount.IsZero() || !fromEscrow.IsZero() {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				EventTypeHackerFundsBurned,
				sdk.NewAttribute(AttributeKeyAccount, addr.String()),
				sdk.NewAttribute(AttributeKeyBurnedFromAccount, fromAccount.String()),
				sdk.NewAttribute(AttributeKeyBurnedFromEscrow, fromEscrow.String()),
			),
		)
	}

	return fromAccount, fromEscrow, nil
}

// releaseFunds moves everything the holder has in staking and distribution onto
// its balance and makes that balance spendable. The order matters: vesting is
// lifted first so nothing stays locked, and the withdraw address is reset before
// commission and rewards are paid out.
func releaseFunds(ctx sdk.Context, k Keepers, holder sdk.AccAddress) error {
	if err := unlockVestingAccount(ctx, k.AccountKeeper, holder); err != nil {
		return errorsmod.Wrap(err, "unlock vesting account")
	}
	if err := resetWithdrawAddress(ctx, k.DistrKeeper, holder); err != nil {
		return errorsmod.Wrap(err, "reset withdraw address")
	}
	if err := withdrawCommission(ctx, k.DistrKeeper, holder); err != nil {
		return errorsmod.Wrap(err, "withdraw validator commission")
	}
	if err := completeUnbondings(ctx, k.StakingKeeper, holder); err != nil {
		return errorsmod.Wrap(err, "complete unbondings")
	}
	if err := completeRedelegations(ctx, k.StakingKeeper, holder); err != nil {
		return errorsmod.Wrap(err, "complete redelegations")
	}
	// Unbond triggers the distribution hook that pays pending rewards out to the
	// (just reset) withdraw address.
	if err := undelegateAll(ctx, k.StakingKeeper, k.BankKeeper, holder); err != nil {
		return errorsmod.Wrap(err, "undelegate")
	}

	return nil
}

// unlockVestingAccount replaces a vesting account with a plain EthAccount so
// every coin becomes spendable. Anyone can turn an address - a frozen account or
// its ucDAO escrow - into a clawback vesting account through
// MsgConvertIntoVestingAccount, so without this a third party could lock coins
// there and fail the upgrade.
//
// The base account is copied, never recreated: account number, sequence and
// pubkey must survive (v1.5.0 recreated accounts and v1.6.0/v1.6.1 had to
// restore them).
func unlockVestingAccount(ctx sdk.Context, ak authkeeper.AccountKeeper, addr sdk.AccAddress) error {
	acc := ak.GetAccount(ctx, addr)
	if acc == nil {
		return nil
	}

	_, isClawback := acc.(*vestingtypes.ClawbackVestingAccount)
	_, isVesting := acc.(vestingexported.VestingAccount)
	if !isClawback && !isVesting {
		return nil
	}

	ethAcc, ok := ethtypes.ProtoAccount().(*ethtypes.EthAccount)
	if !ok {
		return fmt.Errorf("unexpected proto account type %T", ethtypes.ProtoAccount())
	}
	ethAcc.BaseAccount = authtypes.NewBaseAccount(acc.GetAddress(), acc.GetPubKey(), acc.GetAccountNumber(), acc.GetSequence())

	if withCode, ok := acc.(interface{ GetCodeHash() common.Hash }); ok {
		if codeHash := withCode.GetCodeHash(); codeHash != (common.Hash{}) {
			if err := ethAcc.SetCodeHash(codeHash); err != nil {
				return err
			}
		}
	}

	ak.SetAccount(ctx, ethAcc)
	return nil
}

// resetWithdrawAddress points the holder's rewards and commission back at
// itself, so nothing is paid out to an address outside the burn.
func resetWithdrawAddress(ctx sdk.Context, dk distrkeeper.Keeper, addr sdk.AccAddress) error {
	withdrawAddr, err := dk.GetDelegatorWithdrawAddr(ctx, addr)
	if err != nil {
		return err
	}
	if withdrawAddr.Equals(addr) {
		return nil
	}

	return dk.SetDelegatorWithdrawAddr(ctx, addr, addr)
}

// withdrawCommission pays accumulated validator commission out to the holder.
// It runs for every holder: one that does not operate a validator has no
// commission and gets ErrNoValidatorCommission, which is expected.
func withdrawCommission(ctx sdk.Context, dk distrkeeper.Keeper, addr sdk.AccAddress) error {
	_, err := dk.WithdrawValidatorCommission(ctx, sdk.ValAddress(addr))
	if err != nil && !errors.Is(err, distrtypes.ErrNoValidatorCommission) {
		return err
	}

	return nil
}

// completeUnbondings matures every unbonding entry and completes it, paying the
// tokens out of the not bonded pool immediately (the v1.7.6 approach). The UBD
// queue keeps the stale entries; the staking EndBlocker skips pairs that no
// longer exist.
func completeUnbondings(ctx sdk.Context, sk stakingkeeper.Keeper, addr sdk.AccAddress) error {
	ubds, err := sk.GetAllUnbondingDelegations(ctx, addr)
	if err != nil {
		return err
	}
	slices.SortFunc(ubds, func(a, b stakingtypes.UnbondingDelegation) int {
		return strings.Compare(a.ValidatorAddress, b.ValidatorAddress)
	})

	for _, ubd := range ubds {
		for i := range ubd.Entries {
			ubd.Entries[i].CompletionTime = ctx.BlockTime()
		}
		if err := sk.SetUnbondingDelegation(ctx, ubd); err != nil {
			return err
		}

		valAddr, err := sk.ValidatorAddressCodec().StringToBytes(ubd.ValidatorAddress)
		if err != nil {
			return err
		}
		if _, err := sk.CompleteUnbonding(ctx, addr, valAddr); err != nil {
			return errorsmod.Wrapf(err, "validator %s", ubd.ValidatorAddress)
		}
	}

	return nil
}

// completeRedelegations matures every redelegation entry and completes it. A
// redelegation holds no funds - the tokens already sit in the destination
// delegation, which undelegateAll removes - but the record itself must go too:
// otherwise the final check fails. As with unbondings, the redelegation queue
// keeps stale entries and the staking EndBlocker skips triplets that no longer
// exist.
func completeRedelegations(ctx sdk.Context, sk stakingkeeper.Keeper, addr sdk.AccAddress) error {
	reds, err := redelegationsOf(ctx, sk, addr)
	if err != nil {
		return err
	}

	for _, red := range reds {
		for i := range red.Entries {
			red.Entries[i].CompletionTime = ctx.BlockTime()
		}
		if err := sk.SetRedelegation(ctx, red); err != nil {
			return err
		}

		valSrcAddr, err := sk.ValidatorAddressCodec().StringToBytes(red.ValidatorSrcAddress)
		if err != nil {
			return err
		}
		valDstAddr, err := sk.ValidatorAddressCodec().StringToBytes(red.ValidatorDstAddress)
		if err != nil {
			return err
		}
		if _, err := sk.CompleteRedelegation(ctx, addr, valSrcAddr, valDstAddr); err != nil {
			return errorsmod.Wrapf(err, "redelegation %s -> %s", red.ValidatorSrcAddress, red.ValidatorDstAddress)
		}
	}

	return nil
}

// redelegationsOf reads all redelegations of the delegator, sorted by source and
// destination validator.
func redelegationsOf(ctx sdk.Context, sk stakingkeeper.Keeper, addr sdk.AccAddress) ([]stakingtypes.Redelegation, error) {
	var reds []stakingtypes.Redelegation
	err := sk.IterateDelegatorRedelegations(ctx, addr, func(red stakingtypes.Redelegation) bool {
		reds = append(reds, red)
		return false
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(reds, func(a, b stakingtypes.Redelegation) int {
		if c := strings.Compare(a.ValidatorSrcAddress, b.ValidatorSrcAddress); c != 0 {
			return c
		}
		return strings.Compare(a.ValidatorDstAddress, b.ValidatorDstAddress)
	})

	return reds, nil
}

// undelegateAll removes every delegation of the holder, own validator
// included, and pays the tokens out immediately instead of starting an
// unbonding period (the v1.7.6 approach).
func undelegateAll(ctx sdk.Context, sk stakingkeeper.Keeper, bk bankkeeper.Keeper, addr sdk.AccAddress) error {
	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return err
	}

	delegations, err := sk.GetAllDelegatorDelegations(ctx, addr)
	if err != nil {
		return err
	}
	slices.SortFunc(delegations, func(a, b stakingtypes.Delegation) int {
		return strings.Compare(a.ValidatorAddress, b.ValidatorAddress)
	})

	for _, delegation := range delegations {
		valAddr, err := sk.ValidatorAddressCodec().StringToBytes(delegation.ValidatorAddress)
		if err != nil {
			return err
		}

		// Read the status before Unbond, as staking's own Undelegate does.
		validator, err := sk.GetValidator(ctx, valAddr)
		if err != nil {
			return errorsmod.Wrapf(err, "validator %s", delegation.ValidatorAddress)
		}

		amount, err := sk.Unbond(ctx, addr, valAddr, delegation.Shares)
		if err != nil {
			return errorsmod.Wrapf(err, "unbond from %s", delegation.ValidatorAddress)
		}
		if amount.IsZero() {
			continue
		}

		coins := sdk.NewCoins(sdk.NewCoin(bondDenom, amount))
		if validator.IsBonded() {
			if err := bk.SendCoinsFromModuleToModule(ctx, stakingtypes.BondedPoolName, stakingtypes.NotBondedPoolName, coins); err != nil {
				return errorsmod.Wrapf(err, "move %s to not bonded pool", coins)
			}
		}
		if err := bk.UndelegateCoinsFromModuleToAccount(ctx, stakingtypes.NotBondedPoolName, addr, coins); err != nil {
			return errorsmod.Wrapf(err, "pay out %s from %s", coins, delegation.ValidatorAddress)
		}
	}

	return nil
}

// burnEscrow burns the account's ucDAO escrow and keeps ucDAO's bookkeeping in
// line: the per-account balance is the escrow's bank balance, so only the global
// aISLM counter and the holders index need updating. aHAQQ can only reach an
// escrow by a plain transfer and is never counted, so it is not subtracted.
func burnEscrow(ctx sdk.Context, k Keepers, owner, escrow sdk.AccAddress) (sdk.Coins, error) {
	coins := balancesOf(ctx, k.BankKeeper, escrow)
	if err := burnFrom(ctx, k.BankKeeper, escrow, coins); err != nil {
		return nil, err
	}

	if islm := coins.AmountOf(utils.BaseDenom); islm.IsPositive() {
		k.DaoKeeper.TrackSubBalance(ctx, sdk.NewCoin(utils.BaseDenom, islm))
	}

	// Drops the holder once the escrow is empty; never registers a new one.
	if k.DaoKeeper.IsHolder(ctx, owner) {
		k.DaoKeeper.SetHoldersIndex(ctx, owner)
	}

	return coins, nil
}

// burnFrom burns coins held by an arbitrary account through the evm module
// account, the same path x/evm SetBalance uses.
func burnFrom(ctx sdk.Context, bk bankkeeper.Keeper, from sdk.AccAddress, coins sdk.Coins) error {
	if coins.IsZero() {
		return nil
	}
	if err := bk.SendCoinsFromAccountToModule(ctx, from, evmtypes.ModuleName, coins); err != nil {
		return err
	}

	return bk.BurnCoins(ctx, evmtypes.ModuleName, coins)
}

// revokeAuthzGrants deletes every grant the accounts are granter or grantee of.
// A grantee can execute on the granter's behalf through MsgExec, so a grant is a
// way to reach a frozen account's funds.
func revokeAuthzGrants(ctx sdk.Context, ak authzkeeper.Keeper, targets map[string]bool) error {
	grants, err := collectAuthzGrants(ctx, ak, targets)
	if err != nil {
		return err
	}

	for _, g := range grants {
		if err := ak.DeleteGrant(ctx, g.grantee, g.granter, g.msgType); err != nil {
			return errorsmod.Wrapf(err, "revoke %s grant %s -> %s", g.msgType, g.granter, g.grantee)
		}
	}

	return nil
}

// revokeFeeAllowances deletes every fee allowance the accounts are granter or
// grantee of.
func revokeFeeAllowances(ctx sdk.Context, fk feegrantkeeper.Keeper, targets map[string]bool) error {
	allowances, err := collectFeeAllowances(ctx, fk, targets)
	if err != nil {
		return err
	}

	msgServer := feegrantkeeper.NewMsgServerImpl(fk)
	for _, a := range allowances {
		if _, err := msgServer.RevokeAllowance(ctx, &feegrant.MsgRevokeAllowance{Granter: a.granter, Grantee: a.grantee}); err != nil {
			return errorsmod.Wrapf(err, "revoke fee allowance %s -> %s", a.granter, a.grantee)
		}
	}

	return nil
}

type grantRef struct {
	granter, grantee sdk.AccAddress
	msgType          string
}

// collectAuthzGrants reads the grants first and returns them sorted; they are
// deleted afterwards, never while the store is being iterated.
func collectAuthzGrants(ctx sdk.Context, ak authzkeeper.Keeper, targets map[string]bool) ([]grantRef, error) {
	var (
		grants  []grantRef
		iterErr error
	)
	ak.IterateGrants(ctx, func(granter, grantee sdk.AccAddress, grant authz.Grant) bool {
		if !targets[granter.String()] && !targets[grantee.String()] {
			return false
		}

		authorization, err := grant.GetAuthorization()
		if err != nil {
			iterErr = errorsmod.Wrapf(err, "grant %s -> %s", granter, grantee)
			return true
		}

		grants = append(grants, grantRef{granter: granter, grantee: grantee, msgType: authorization.MsgTypeURL()})
		return false
	})
	if iterErr != nil {
		return nil, iterErr
	}

	slices.SortFunc(grants, func(a, b grantRef) int {
		if c := bytes.Compare(a.granter, b.granter); c != 0 {
			return c
		}
		if c := bytes.Compare(a.grantee, b.grantee); c != 0 {
			return c
		}
		return strings.Compare(a.msgType, b.msgType)
	})

	return grants, nil
}

type allowanceRef struct {
	granter, grantee string
}

// collectFeeAllowances reads the allowances first and returns them sorted.
func collectFeeAllowances(ctx sdk.Context, fk feegrantkeeper.Keeper, targets map[string]bool) ([]allowanceRef, error) {
	var allowances []allowanceRef
	err := fk.IterateAllFeeAllowances(ctx, func(grant feegrant.Grant) bool {
		if targets[grant.Granter] || targets[grant.Grantee] {
			allowances = append(allowances, allowanceRef{granter: grant.Granter, grantee: grant.Grantee})
		}
		return false
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(allowances, func(a, b allowanceRef) int {
		if c := strings.Compare(a.granter, b.granter); c != 0 {
			return c
		}
		return strings.Compare(a.grantee, b.grantee)
	})

	return allowances, nil
}

// verifyAccountCleared fails unless neither the account nor its ucDAO escrow
// holds anything burnable anymore.
func verifyAccountCleared(ctx sdk.Context, k Keepers, addr sdk.AccAddress) error {
	for _, holder := range []sdk.AccAddress{addr, ucdaotypes.GetEscrowAddress(addr)} {
		if err := verifyHolderCleared(ctx, k, holder); err != nil {
			return errorsmod.Wrapf(err, "holder %s", holder)
		}
	}

	return nil
}

// verifyHolderCleared fails unless the holder has no BurnDenoms balance, no
// staking positions, no commission and a withdraw address pointing at itself.
func verifyHolderCleared(ctx sdk.Context, k Keepers, holder sdk.AccAddress) error {
	for _, denom := range BurnDenoms {
		if bal := k.BankKeeper.GetBalance(ctx, holder, denom); !bal.IsZero() {
			return fmt.Errorf("balance left: %s", bal)
		}
	}

	delegations, err := k.StakingKeeper.GetAllDelegatorDelegations(ctx, holder)
	if err != nil {
		return err
	}
	if len(delegations) > 0 {
		return fmt.Errorf("%d delegations left", len(delegations))
	}

	ubds, err := k.StakingKeeper.GetAllUnbondingDelegations(ctx, holder)
	if err != nil {
		return err
	}
	if len(ubds) > 0 {
		return fmt.Errorf("%d unbonding delegations left", len(ubds))
	}

	reds, err := redelegationsOf(ctx, k.StakingKeeper, holder)
	if err != nil {
		return err
	}
	if len(reds) > 0 {
		return fmt.Errorf("%d redelegations left", len(reds))
	}

	withdrawAddr, err := k.DistrKeeper.GetDelegatorWithdrawAddr(ctx, holder)
	if err != nil {
		return err
	}
	if !withdrawAddr.Equals(holder) {
		return fmt.Errorf("withdraw address is %s", withdrawAddr)
	}

	commission, err := k.DistrKeeper.GetValidatorAccumulatedCommission(ctx, sdk.ValAddress(holder))
	if err != nil {
		return err
	}
	// Only the sub-unit decimal remainder may stay behind.
	truncated, _ := commission.Commission.TruncateDecimal()
	for _, denom := range BurnDenoms {
		if amt := truncated.AmountOf(denom); !amt.IsZero() {
			return fmt.Errorf("validator commission left: %s%s", amt, denom)
		}
	}

	return nil
}

// verifyNoGrants fails if any grant or fee allowance still involves the accounts.
func verifyNoGrants(ctx sdk.Context, k Keepers, targets map[string]bool) error {
	grants, err := collectAuthzGrants(ctx, k.AuthzKeeper, targets)
	if err != nil {
		return err
	}
	if len(grants) > 0 {
		return fmt.Errorf("%d authz grants left", len(grants))
	}

	allowances, err := collectFeeAllowances(ctx, k.FeeGrantKeeper, targets)
	if err != nil {
		return err
	}
	if len(allowances) > 0 {
		return fmt.Errorf("%d fee allowances left", len(allowances))
	}

	return nil
}

// balancesOf returns the account's BurnDenoms balances.
func balancesOf(ctx sdk.Context, bk bankkeeper.Keeper, addr sdk.AccAddress) sdk.Coins {
	coins := make([]sdk.Coin, 0, len(BurnDenoms))
	for _, denom := range BurnDenoms {
		coins = append(coins, bk.GetBalance(ctx, addr, denom))
	}

	return sdk.NewCoins(coins...)
}

// supplyOf returns the total supply of BurnDenoms.
func supplyOf(ctx sdk.Context, bk bankkeeper.Keeper) sdk.Coins {
	coins := make([]sdk.Coin, 0, len(BurnDenoms))
	for _, denom := range BurnDenoms {
		coins = append(coins, bk.GetSupply(ctx, denom))
	}

	return sdk.NewCoins(coins...)
}
