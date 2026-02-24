package ethiq

import (
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
)

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

// EmitMintHaqqEventWithApplicationID creates a new mint haqq event with the actual haqq amount and application ID.
func EmitMintHaqqEventWithApplicationID(
	ctx sdk.Context,
	stateDB vm.StateDB,
	event abi.Event,
	precompileAddr, sender, receiver common.Address,
	applicationID uint64,
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
	packed, err := arguments.Pack(sdkmath.NewIntFromUint64(applicationID).BigInt(), haqqAmount.BigInt())
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
