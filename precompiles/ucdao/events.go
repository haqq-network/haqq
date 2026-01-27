// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package ucdao

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

const (
	// EventTypeApproval defines the event type for the Approval event.
	EventTypeApproval = "Approval"
	// EventTypeRevocation defines the event type for the Revocation event.
	EventTypeRevocation = "Revocation"
	// EventTypeFund defines the event type for the Fund event.
	EventTypeFund = "Fund"
	// EventTypeTransferOwnership defines the event type for the TransferOwnership event.
	EventTypeTransferOwnership = "TransferOwnership"
)

// EmitApprovalEvent emits an Approval event.
func (p Precompile) EmitApprovalEvent(ctx sdk.Context, stateDB vm.StateDB, owner, spender common.Address, coins sdk.Coins) error {
	return p.emitEventWithTwoAddressesAndCoins(ctx, stateDB, EventTypeApproval, owner, spender, coins)
}

// EmitRevocationEvent emits a Revocation event.
func (p Precompile) EmitRevocationEvent(ctx sdk.Context, stateDB vm.StateDB, owner, spender common.Address) error {
	topics, err := p.makeTopicsWithTwoAddresses(EventTypeRevocation, owner, spender)
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        nil,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint: gosec // G115
	})

	return nil
}

// EmitFundEvent emits a Fund event.
func (p Precompile) EmitFundEvent(ctx sdk.Context, stateDB vm.StateDB, depositor common.Address, coins sdk.Coins) error {
	event := p.ABI.Events[EventTypeFund]
	topics := make([]common.Hash, 2)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(depositor)
	if err != nil {
		return err
	}

	// Pack the arguments to be used as the Data field
	arguments := event.Inputs.NonIndexed()
	packed, err := arguments.Pack(cmn.NewCoinsResponse(coins))
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint: gosec // G115
	})

	return nil
}

// EmitTransferOwnershipEvent emits a TransferOwnership event.
func (p Precompile) EmitTransferOwnershipEvent(ctx sdk.Context, stateDB vm.StateDB, from, to common.Address, coins sdk.Coins) error {
	return p.emitEventWithTwoAddressesAndCoins(ctx, stateDB, EventTypeTransferOwnership, from, to, coins)
}

// makeTopicsWithTwoAddresses creates event topics with two indexed addresses.
func (p Precompile) makeTopicsWithTwoAddresses(eventType string, addr1, addr2 common.Address) ([]common.Hash, error) {
	event := p.ABI.Events[eventType]
	topics := make([]common.Hash, 3)

	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(addr1)
	if err != nil {
		return nil, err
	}

	topics[2], err = cmn.MakeTopic(addr2)
	if err != nil {
		return nil, err
	}

	return topics, nil
}

// emitEventWithTwoAddressesAndCoins is a helper function to emit events with two indexed addresses and coins data.
func (p Precompile) emitEventWithTwoAddressesAndCoins(
	ctx sdk.Context,
	stateDB vm.StateDB,
	eventType string,
	addr1, addr2 common.Address,
	coins sdk.Coins,
) error {
	topics, err := p.makeTopicsWithTwoAddresses(eventType, addr1, addr2)
	if err != nil {
		return err
	}

	event := p.ABI.Events[eventType]
	arguments := event.Inputs.NonIndexed()
	packed, err := arguments.Pack(cmn.NewCoinsResponse(coins))
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint: gosec // G115
	})

	return nil
}
