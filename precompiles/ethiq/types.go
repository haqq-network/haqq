package ethiq

import (
	"fmt"
	"math/big"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/utils"

	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

func NewMintHaqqMsg(args []interface{}) (*ethiqtypes.MsgMintHaqq, common.Address, common.Address, error) {
	if len(args) != 3 {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	sender, ok := args[0].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(ErrInvalidSender, args[0])
	}

	receiver, ok := args[1].(common.Address)
	if !ok {
		return nil, common.Address{}, common.Address{}, fmt.Errorf(ErrInvalidReceiver, args[1])
	}

	amount, ok := args[2].(*big.Int)
	if !ok || amount == nil {
		return nil, common.Address{}, common.Address{}, errorsmod.Wrapf(ethiqtypes.ErrInvalidAmount, cmn.ErrInvalidAmount, args[2])
	}

	msg := &ethiqtypes.MsgMintHaqq{
		FromAddress: sdk.AccAddress(sender.Bytes()).String(),
		ToAddress:   sdk.AccAddress(receiver.Bytes()).String(),
		IslmAmount:  math.NewIntFromBigInt(amount),
	}

	return msg, sender, receiver, nil
}

func NewMintHaqqByApplicationMsg(args []interface{}) (*ethiqtypes.MsgMintHaqqByApplication, common.Address, error) {
	if len(args) != 2 {
		return nil, common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	sender, ok := args[0].(common.Address)
	if !ok {
		return nil, common.Address{}, fmt.Errorf(ErrInvalidSender, args[0])
	}

	appID, ok := args[1].(*big.Int)
	if !ok || appID == nil {
		return nil, common.Address{}, errorsmod.Wrapf(ethiqtypes.ErrInvalidApplicationID, ErrInvalidApplicationID, args[1])
	}

	msg := &ethiqtypes.MsgMintHaqqByApplication{
		FromAddress:   sdk.AccAddress(sender.Bytes()).String(),
		ApplicationId: appID.Uint64(),
	}

	return msg, sender, nil
}

func NewCalculateRequest(args []interface{}) (*ethiqtypes.QueryCalculateRequest, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("invalid input arguments. Expected 1, got %d", len(args))
	}

	amount, ok := args[0].(*big.Int)
	if !ok || amount == nil {
		return nil, errorsmod.Wrapf(ethiqtypes.ErrInvalidAmount, cmn.ErrInvalidAmount, args[0])
	}

	req := &ethiqtypes.QueryCalculateRequest{
		IslmAmount: amount.String(),
	}

	return req, nil
}

func NewCalculateForApplicationRequest(args []interface{}) (*ethiqtypes.QueryCalculateForApplicationRequest, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("invalid input arguments. Expected 1, got %d", len(args))
	}

	appID, ok := args[0].(*big.Int)
	if !ok || appID == nil {
		return nil, errorsmod.Wrapf(ethiqtypes.ErrInvalidApplicationID, ErrInvalidApplicationID, args[0])
	}

	req := &ethiqtypes.QueryCalculateForApplicationRequest{
		ApplicationId: appID.Uint64(),
	}

	return req, nil
}

func NewMintHaqqAuthorization(args []interface{}) (common.Address, common.Address, *ethiqtypes.MintHaqqAuthorization, error) {
	grantee, granter, amount, err := checkMintHaqqAuthzArgs(args)
	if err != nil {
		return common.Address{}, common.Address{}, nil, err
	}

	coin := sdk.NewCoin(utils.BaseDenom, math.NewIntFromBigInt(amount))

	mintAuthz := &ethiqtypes.MintHaqqAuthorization{
		SpendLimit: &coin,
	}

	if err = mintAuthz.ValidateBasic(); err != nil {
		return common.Address{}, common.Address{}, nil, err
	}

	return grantee, granter, mintAuthz, nil
}

func checkMintHaqqAuthzArgs(args []interface{}) (common.Address, common.Address, *big.Int, error) {
	if len(args) != 3 {
		return common.Address{}, common.Address{}, nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	owner, ok := args[0].(common.Address)
	if !ok {
		return common.Address{}, common.Address{}, nil, fmt.Errorf("invalid owner address: %v", args[0])
	}

	spender, ok := args[1].(common.Address)
	if !ok {
		return common.Address{}, common.Address{}, nil, fmt.Errorf("invalid spender address: %v", args[1])
	}

	amount, ok := args[2].(*big.Int)
	if !ok {
		return common.Address{}, common.Address{}, nil, fmt.Errorf("invalid amount: %v", args[2])
	}

	return owner, spender, amount, nil
}

// CheckAndAcceptAuthorizationIfNeeded checks if authorization exists and accepts the grant.
// In case the origin is the caller of the address, no authorization is required.
func CheckAndAcceptAuthorizationIfNeeded(
	ctx sdk.Context,
	contract *vm.Contract,
	origin common.Address,
	authzKeeper authzkeeper.Keeper,
	msgURL string,
) (*authz.AcceptResponse, *time.Time, error) {
	if contract.CallerAddress == origin {
		return nil, nil, nil
	}

	_, expiration, err := authorization.CheckAuthzExists(ctx, authzKeeper, contract.CallerAddress, origin, msgURL)
	if err != nil {
		return nil, nil, fmt.Errorf(authorization.ErrAuthzDoesNotExistOrExpired, contract.CallerAddress, origin)
	}

	switch msgURL {
	case MintHaqqMsgURL:
		// We need the actual message to accept, but we don't have it here
		// This will be handled in the transaction method
		return nil, expiration, nil
	case MsgMintHaqqByApplicationMsgURL:
		// Same as above
		return nil, expiration, nil
	default:
		return nil, nil, fmt.Errorf("unknown message URL: %s", msgURL)
	}
}
