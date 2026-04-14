package ucdao

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	ucdaotypes "github.com/haqq-network/haqq/x/ucdao/types"
)

const (
	// EventTypeMintHaqq defines the event type for the ucdao ConvertToHaqq transaction.
	EventTypeMintHaqq = "MintHaqq"
)

// EmitApprovalEvent creates a new approval event emitted on an Approve, IncreaseAllowance and DecreaseAllowance transactions.
func (p Precompile) EmitApprovalEvent(ctx sdk.Context, stateDB vm.StateDB, grantee, granter common.Address, coin *sdk.Coin, typeUrls []string) error {
	// Prepare the event topics
	event := p.ABI.Events[authorization.EventTypeApproval]
	topics := make([]common.Hash, 3)

	// The first topic is always the signature of the event.
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

	// Check if the coin is set to infinite
	value := abi.MaxUint256
	if coin != nil {
		value = coin.Amount.BigInt()
	}

	// Pack the arguments to be used as the Data field
	arguments := abi.Arguments{event.Inputs[2], event.Inputs[3]}
	packed, err := arguments.Pack(typeUrls, value)
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

// EmitAllowanceChangeEvent creates a new allowance change event emitted on an IncreaseAllowance and DecreaseAllowance transactions.
func (p Precompile) EmitAllowanceChangeEvent(ctx sdk.Context, stateDB vm.StateDB, grantee, granter common.Address, typeUrls []string) error {
	// Prepare the event topics
	event := p.ABI.Events[authorization.EventTypeAllowanceChange]
	topics := make([]common.Hash, 3)

	// The first topic is always the signature of the event.
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

	newValues := make([]*big.Int, len(typeUrls))
	for i, msgURL := range typeUrls {
		// Not including expiration and convert check because we have already checked it in the previous call
		msgAuthz, _ := p.AuthzKeeper.GetAuthorization(ctx, grantee.Bytes(), granter.Bytes(), msgURL)
		convAuthz, isConvertAuthz := msgAuthz.(*ucdaotypes.ConvertToHaqqAuthorization)
		_, isTransferAuthz := msgAuthz.(*ucdaotypes.TransferOwnershipAuthorization)
		if !isConvertAuthz && !isTransferAuthz {
			// should never happen in normal flow
			continue
		}

		if convAuthz.SpendLimit == nil {
			newValues[i] = abi.MaxUint256
		} else {
			newValues[i] = convAuthz.SpendLimit.Amount.BigInt()
		}
	}

	// Pack the arguments to be used as the Data field
	arguments := abi.Arguments{event.Inputs[2], event.Inputs[3]}
	packed, err := arguments.Pack(typeUrls, newValues)
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

// EmitMintHaqqEventWithAmount creates a new mint haqq event with the actual haqq amount.
func EmitMintHaqqEventWithAmount(
	ctx sdk.Context,
	stateDB vm.StateDB,
	event abi.Event,
	precompileAddr, sender, receiver common.Address,
	islmAmount sdkmath.Int,
	haqqAmount sdkmath.Int,
) error {
	// Prepare the event topics
	topics := make([]common.Hash, 3)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	// sender and receiver are indexed
	topics[1], err = cmn.MakeTopic(sender)
	if err != nil {
		return err
	}
	topics[2], err = cmn.MakeTopic(receiver)
	if err != nil {
		return err
	}

	// Pack the arguments to be used as the Data field
	arguments := abi.Arguments{event.Inputs[2], event.Inputs[3]}
	packed, err := arguments.Pack(islmAmount.BigInt(), haqqAmount.BigInt())
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
