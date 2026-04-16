package liquid

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

const (
	// EventTypeLiquidate defines the event type for the Liquidate transaction.
	EventTypeLiquidate = "Liquidate"
	// EventTypeRedeem defines the event type for the Redeem transaction.
	EventTypeRedeem = "Redeem"
)

// EmitLiquidateEvent creates a new Liquidate event with indexed sender and receiver.
func (p Precompile) EmitLiquidateEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	sender, receiver, erc20Contract common.Address,
	amount *big.Int,
) error {
	event := p.ABI.Events[EventTypeLiquidate]

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
	packed, err := arguments.Pack(amount, erc20Contract)
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint: gosec // blockHeight can't overflow uint64
	})

	return nil
}

// EmitRedeemEvent creates a new Redeem event with indexed sender and receiver.
func (p Precompile) EmitRedeemEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	sender, receiver common.Address,
	denom string,
	amount *big.Int,
) error {
	event := p.ABI.Events[EventTypeRedeem]

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
	packed, err := arguments.Pack(denom, amount)
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint: gosec // blockHeight can't overflow uint64
	})

	return nil
}
