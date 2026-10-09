package ethiq

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

const (
	// EventTypeMintHaqq defines the event type for the ethiq MintHaqq transaction.
	EventTypeMintHaqq = "MintHaqq"
	// EventTypeMintHaqqByApplication defines the event type for the ethiq MintHaqqByApplication transaction.
	EventTypeMintHaqqByApplication = "MintHaqqByApplication"
	// EventTypeApplicationIDApproval defines the event type emitted when an application ID is approved.
	EventTypeApplicationIDApproval = "ApplicationIDApproval"
	// EventTypeApplicationIDRevocation defines the event type emitted when an application ID is revoked.
	EventTypeApplicationIDRevocation = "ApplicationIDRevocation"
)

// Authorization event semantics for application-based grants:
//
//   - Revocation means the grant is gone. revoke emits it on its own.
//   - ApplicationIDRevocation means one ID was removed; the list the grant still holds is
//     attached, and an empty list means the grant was deleted along with the last ID.
//   - revokeApplicationID removing the last ID emits both: one ID was removed AND the grant
//     is gone. The two statements do not contradict each other.
//   - Approval carries neither the ID nor the resulting list, so ApplicationIDApproval is the
//     authoritative one for approveApplicationID.
//
// An indexer that tracks application grants therefore has to follow ApplicationIDApproval,
// ApplicationIDRevocation, and Revocation carrying the MsgMintHaqqByApplication type URL.

// EmitMintHaqqEventWithAmount creates a new mint haqq event with the actual haqq amount.
func EmitMintHaqqEventWithAmount(
	ctx sdk.Context,
	stateDB vm.StateDB,
	event abi.Event,
	precompileAddr, sender, receiver common.Address,
	islmAmount sdkmath.Int,
	haqqAmount sdkmath.Int,
) error {
	return emitMintHaqqEvent(ctx, stateDB, event, precompileAddr, sender, receiver, islmAmount.BigInt(), haqqAmount.BigInt())
}

// EmitMintHaqqEventWithApplicationID creates a new mint haqq event with the actual haqq amount and application ID.
func EmitMintHaqqEventWithApplicationID(
	ctx sdk.Context,
	stateDB vm.StateDB,
	event abi.Event,
	precompileAddr, sender, receiver common.Address,
	applicationID uint64,
	haqqAmount sdkmath.Int,
) error {
	return emitMintHaqqEvent(ctx, stateDB, event, precompileAddr, sender, receiver, sdkmath.NewIntFromUint64(applicationID).BigInt(), haqqAmount.BigInt())
}

// emitMintHaqqEvent emits an EVM log with indexed sender/receiver topics and two non-indexed data arguments.
func emitMintHaqqEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	event abi.Event,
	precompileAddr, sender, receiver common.Address,
	dataArg0, dataArg1 interface{},
) error {
	topics := make([]common.Hash, 3)
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(sender)
	if err != nil {
		return err
	}
	topics[2], err = cmn.MakeTopic(receiver)
	if err != nil {
		return err
	}

	arguments := abi.Arguments{event.Inputs[2], event.Inputs[3]}
	packed, err := arguments.Pack(dataArg0, dataArg1)
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     precompileAddr,
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint: gosec // G115 blockHeight is positive int64 and can't overflow uint64
	})

	return nil
}

// EmitApplicationIDApprovalEvent emits the resulting allow list after approveApplicationID.
//
// The shared Approval event carries neither the application ID nor the resulting list, so on its
// own it cannot be told apart from an unlimited MintHaqq approval. This event is the authoritative
// one for application-based grants.
func (p Precompile) EmitApplicationIDApprovalEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	grantee, granter common.Address,
	applicationID uint64,
	applicationIDs []uint64,
) error {
	return p.emitApplicationIDEvent(ctx, stateDB, EventTypeApplicationIDApproval, grantee, granter, applicationID, applicationIDs)
}

// EmitApplicationIDRevocationEvent emits the allow list left after revokeApplicationID.
//
// revokeApplicationID removes a single ID, so the shared Revocation event overstates what
// happened whenever the grant survives. An empty remaining list means the grant was deleted.
func (p Precompile) EmitApplicationIDRevocationEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	grantee, granter common.Address,
	applicationID uint64,
	remainingIDs []uint64,
) error {
	return p.emitApplicationIDEvent(ctx, stateDB, EventTypeApplicationIDRevocation, grantee, granter, applicationID, remainingIDs)
}

// emitApplicationIDEvent emits an EVM log with indexed grantee/granter topics, the application ID
// the call acted on, and the allow list the grant holds afterwards.
func (p Precompile) emitApplicationIDEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	eventType string,
	grantee, granter common.Address,
	applicationID uint64,
	applicationIDs []uint64,
) error {
	event := p.ABI.Events[eventType]

	topics := make([]common.Hash, 3)
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(grantee)
	if err != nil {
		return err
	}
	topics[2], err = cmn.MakeTopic(granter)
	if err != nil {
		return err
	}

	// The ABI declares uint256[]; go-ethereum packs it from []*big.Int.
	packedIDs := make([]*big.Int, len(applicationIDs))
	for i, id := range applicationIDs {
		packedIDs[i] = new(big.Int).SetUint64(id)
	}

	arguments := abi.Arguments{event.Inputs[2], event.Inputs[3]}
	packed, err := arguments.Pack(new(big.Int).SetUint64(applicationID), packedIDs)
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint: gosec // G115 blockHeight is positive int64 and can't overflow uint64
	})

	return nil
}
