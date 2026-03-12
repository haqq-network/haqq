package liquid

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/utils"
	liquidtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
)

// NewLiquidateMsg builds a MsgLiquidate from ABI arguments.
// Expected args: [from common.Address, to common.Address, amount *big.Int].
func NewLiquidateMsg(args []interface{}) (*liquidtypes.MsgLiquidate, common.Address, common.Address, error) {
	if len(args) != 3 {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	from, ok := args[0].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(ErrInvalidSender, args[0])
	}

	to, ok := args[1].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(ErrInvalidReceiver, args[1])
	}

	amount, ok := args[2].(*big.Int)
	if !ok || amount == nil {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(ErrInvalidAmount, args[2])
	}

	coin := sdk.NewCoin(utils.BaseDenom, math.NewIntFromBigInt(amount))

	msg := &liquidtypes.MsgLiquidate{
		LiquidateFrom: sdk.AccAddress(from.Bytes()).String(),
		LiquidateTo:   sdk.AccAddress(to.Bytes()).String(),
		Amount:        coin,
	}

	return msg, from, to, nil
}

// NewRedeemMsg builds a MsgRedeem from ABI arguments.
// Expected args: [from common.Address, to common.Address, denom string, amount *big.Int].
func NewRedeemMsg(args []interface{}) (*liquidtypes.MsgRedeem, common.Address, common.Address, error) {
	if len(args) != 4 {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 4, len(args))
	}

	from, ok := args[0].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(ErrInvalidSender, args[0])
	}

	to, ok := args[1].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(ErrInvalidReceiver, args[1])
	}

	denom, ok := args[2].(string)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(ErrInvalidDenom, args[2])
	}

	amount, ok := args[3].(*big.Int)
	if !ok || amount == nil {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(ErrInvalidAmount, args[3])
	}

	coin := sdk.NewCoin(denom, math.NewIntFromBigInt(amount))

	msg := &liquidtypes.MsgRedeem{
		RedeemFrom: sdk.AccAddress(from.Bytes()).String(),
		RedeemTo:   sdk.AccAddress(to.Bytes()).String(),
		Amount:     coin,
	}

	return msg, from, to, nil
}
