package ucdao

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	ucdaokeeper "github.com/haqq-network/haqq/x/ucdao/keeper"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

const (
	// ConvertToHaqqMethod defines the ABI method name for ConvertToHaqq transaction.
	ConvertToHaqqMethod = "convertToHaqq"
	// TransferOwnershipMethod defines the ABI method name for TransferOwnership transaction.
	TransferOwnershipMethod = "transferOwnership"
	// TransferOwnershipWithAmountMethod defines the ABI method name for TransferOwnershipWithAmount transaction.
	TransferOwnershipWithAmountMethod = "transferOwnershipWithAmount"
)

// ConvertToHaqqMsgURL defines the authorization type for MsgConvertToHaqq
var ConvertToHaqqMsgURL = sdk.MsgTypeURL(&ucdaotypes.MsgConvertToHaqq{})

// TransferOwnershipWithAmountMsgURL is the authorization type URL for ucDAO ownership
// grants.
//
// It names MsgTransferOwnershipWithAmount, not MsgTransferOwnership, and that is
// deliberate. ucdaotypes.TransferOwnershipAuthorization reports
// MsgTransferOwnershipWithAmount from MsgTypeURL() and its Accept casts the message to
// *MsgTransferOwnershipWithAmount, so that is the key authzkeeper.SaveGrant writes under
// and the key cosmos authz DispatchActions looks up. The precompile previously used
// MsgTransferOwnership here, which meant approve() wrote a grant nothing else could
// find: revoke, allowance and the allowance deltas all missed it, leaving a grant that
// could not be revoked from the EVM at all.
//
// The full-balance transferOwnership message carries no amount, so a spend limit cannot
// be expressed for it; it has no authorization type and cannot be authorized through the
// precompile. See TransferOwnership below.
var TransferOwnershipWithAmountMsgURL = sdk.MsgTypeURL(&ucdaotypes.MsgTransferOwnershipWithAmount{})

// escrowBaseSnapshot pairs a UC DAO holder with the aISLM bank balance of their
// escrow, captured immediately before a keeper call.
type escrowBaseSnapshot struct {
	holder     sdk.AccAddress
	baseBefore sdkmath.Int
}

func (p *Precompile) snapshotEscrowBase(ctx sdk.Context, holder sdk.AccAddress) escrowBaseSnapshot {
	return escrowBaseSnapshot{
		holder:     holder,
		baseBefore: p.daoKeeper.GetBalance(ctx, holder, utils.BaseDenom).Amount,
	}
}

// mirrorEscrowBaseDeltasIntoStateDB mirrors per-escrow bank deltas of the EVM
// gas denom (aISLM) into the EVM StateDB journal when the precompile is invoked
// from another contract (caller != origin).
//
// UC DAO funds live on derived escrow accounts. SendCoins/burn update those
// escrows in cacheCtx but do not journal SubBalance/AddBalance. If an escrow was
// already dirty in StateDB (e.g. a 1 wei touch), Commit SetBalance restores the
// stale EVM balance and mints the transferred amount back.
//
// Snapshots must be taken immediately before the keeper/msg work. Journal
// entries target the escrow EVM address (not the holder). Duplicate holders are
// mirrored once so a single net delta is applied.
func (p *Precompile) mirrorEscrowBaseDeltasIntoStateDB(
	ctx sdk.Context,
	isCallerOrigin bool,
	snapshots ...escrowBaseSnapshot,
) {
	if isCallerOrigin {
		return
	}
	seen := make(map[string]struct{}, len(snapshots))
	for _, s := range snapshots {
		key := s.holder.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		netBaseDelta := s.baseBefore.Sub(p.daoKeeper.GetBalance(ctx, s.holder, utils.BaseDenom).Amount)
		if netBaseDelta.IsZero() {
			continue
		}
		// GetEscrowAddress returns sha256(...)[:20] today, so this conversion is exact.
		// It still goes through the shared helper rather than common.BytesToAddress: that
		// is the rule for every Cosmos-to-EVM boundary, and it keeps the mirror correct if
		// the derivation ever widens.
		escrowHex, ok := cmn.EVMAddressFromCosmos(ucdaotypes.GetEscrowAddress(s.holder))
		if !ok {
			continue
		}
		if netBaseDelta.IsNegative() {
			p.AddBalanceChangeEntries(cmn.NewBalanceChangeEntry(escrowHex, netBaseDelta.Neg().BigInt(), cmn.Add))
			continue
		}
		p.AddBalanceChangeEntries(cmn.NewBalanceChangeEntry(escrowHex, netBaseDelta.BigInt(), cmn.Sub))
	}
}

func (p *Precompile) ConvertToHaqq(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, sender, receiver, err := NewConvertToHaqqMsg(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"origin", origin.String(),
		"caller", contract.CallerAddress.String(),
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ sender: %s, receiver: %s, islm_amount: %s }",
			msg.Sender,
			msg.Receiver,
			msg.IslmAmount.String(),
		),
	)

	// isCallerSender is true when the contract caller is the same as the sender
	isCallerSender := contract.CallerAddress == sender
	isCallerOrigin := contract.CallerAddress == origin

	// If the contract caller is not the same as the sender, the sender must be the origin
	if isCallerSender {
		sender = origin
	} else if origin != sender {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, origin.String(), sender.String())
	}

	// Check and accept authorization if needed
	if err := CheckAndAcceptAuthorizationIfNeeded(ctx, contract, sender, p.AuthzKeeper, msg); err != nil {
		return nil, err
	}

	senderAcc := sdk.MustAccAddressFromBech32(msg.Sender)
	senderEscrowBefore := p.snapshotEscrowBase(ctx, senderAcc)

	msgSrv := ucdaokeeper.NewMsgServerImpl(p.daoKeeper)
	res, err := msgSrv.ConvertToHaqq(ctx, msg)
	if err != nil {
		return nil, err
	}

	p.mirrorEscrowBaseDeltasIntoStateDB(ctx, isCallerOrigin, senderEscrowBefore)

	if err = EmitMintHaqqEventWithAmount(
		ctx,
		stateDB,
		p.ABI.Events[EventTypeMintHaqq],
		p.Address(),
		sender,
		receiver,
		msg.IslmAmount,
		res.MintedCoin.Amount,
	); err != nil {
		return nil, err
	}

	// Return minted amount as uint256
	return method.Outputs.Pack(res.MintedCoin.Amount.BigInt())
}

func (p *Precompile) TransferOwnership(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, owner, _, err := NewTransferOwnershipMsg(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"origin", origin.String(),
		"caller", contract.CallerAddress.String(),
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ owner: %s, newOwner: %s, amount: %s }",
			msg.Owner,
			msg.NewOwner,
			"all balance",
		),
	)

	isCallerOrigin := contract.CallerAddress == origin

	// A contract caller can never be authorized for this message: SaveGrant keys a grant
	// by authorization.MsgTypeURL(), TransferOwnershipAuthorization reports
	// MsgTransferOwnershipWithAmount, and CheckAndAcceptAuthorizationIfNeeded rejects any
	// authorization that is not one of the two ucDAO types - so a lookup under
	// MsgTransferOwnership can never succeed. Reject it here with a message that says why,
	// instead of a generic "grant does not exist" that suggests issuing one would help.
	if !isCallerOrigin {
		return nil, fmt.Errorf(ErrTransferOwnershipNotDelegatable, TransferOwnershipWithAmountMsgURL)
	}

	// The caller is the origin, so the owner must be the origin too.
	if origin != owner {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, origin.String(), owner.String())
	}

	// NOTE: these two snapshots and the mirror call below are a no-op as the method stands.
	// The guard above returns unless isCallerOrigin, and the mirror returns immediately when
	// it is - journaling is only ever needed for a contract caller. They are kept, and kept
	// next to the keeper call, so that the pair does not have to be reconstructed if
	// transferOwnership ever becomes delegatable: an authorization type for it would make
	// contract callers reachable here, and a mirror added back later, away from the code that
	// moves the coins, is exactly how the double-credit bug in TransferOwnershipWithAmount
	// happened. The cost is two bank reads per call.
	ownerAcc := sdk.MustAccAddressFromBech32(msg.Owner)
	newOwnerAcc := sdk.MustAccAddressFromBech32(msg.NewOwner)
	escrowBefore := []escrowBaseSnapshot{
		p.snapshotEscrowBase(ctx, ownerAcc),
		p.snapshotEscrowBase(ctx, newOwnerAcc),
	}

	msgSrv := ucdaokeeper.NewMsgServerImpl(p.daoKeeper)
	_, err = msgSrv.TransferOwnership(ctx, msg)
	if err != nil {
		return nil, err
	}

	p.mirrorEscrowBaseDeltasIntoStateDB(ctx, isCallerOrigin, escrowBefore...)

	return []byte{}, nil
}

func (p *Precompile) TransferOwnershipWithAmount(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	_ vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, owner, _, err := NewTransferOwnershipWithAmountMsg(args)
	if err != nil {
		return nil, err
	}

	p.Logger(ctx).Debug(
		"tx called",
		"origin", origin.String(),
		"caller", contract.CallerAddress.String(),
		"method", method.Name,
		"args", fmt.Sprintf(
			"{ owner: %s, newOwner: %s, amount: %s }",
			msg.Owner,
			msg.NewOwner,
			msg.Amount.String(),
		),
	)

	// isCallerSender is true when the contract caller is the same as the sender
	isCallerSender := contract.CallerAddress == owner
	isCallerOrigin := contract.CallerAddress == origin

	// If the contract caller is not the same as the sender, the sender must be the origin
	if isCallerSender {
		owner = origin
	} else if origin != owner {
		return nil, fmt.Errorf(ErrDifferentOriginFromSender, origin.String(), owner.String())
	}

	// Check and accept authorization if needed
	if err := CheckAndAcceptAuthorizationIfNeeded(ctx, contract, owner, p.AuthzKeeper, msg); err != nil {
		return nil, err
	}

	// Ensure origin is the owner
	if origin != owner {
		return nil, fmt.Errorf("origin (%s) must be the owner (%s)", origin.String(), owner.String())
	}

	ownerAcc := sdk.MustAccAddressFromBech32(msg.Owner)
	newOwnerAcc := sdk.MustAccAddressFromBech32(msg.NewOwner)
	escrowBefore := []escrowBaseSnapshot{
		p.snapshotEscrowBase(ctx, ownerAcc),
		p.snapshotEscrowBase(ctx, newOwnerAcc),
	}

	msgSrv := ucdaokeeper.NewMsgServerImpl(p.daoKeeper)
	_, err = msgSrv.TransferOwnershipWithAmount(ctx, msg)
	if err != nil {
		return nil, err
	}

	p.mirrorEscrowBaseDeltasIntoStateDB(ctx, isCallerOrigin, escrowBefore...)

	return []byte{}, nil
}
