package ethiq

import (
	"fmt"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/utils"

	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

// ParseApplicationID converts an ABI uint256 argument into an ethiq application ID.
//
// ApplicationId is uint64 while the ABI argument is 256 bits wide, and big.Int.Uint64 keeps
// only the low 64 bits: 2^64+5 would silently address application 5. Every entry point that
// takes an application ID from calldata must go through here.
func ParseApplicationID(arg interface{}) (uint64, error) {
	appID, ok := arg.(*big.Int)
	if !ok || appID == nil {
		return 0, errorsmod.Wrapf(ethiqtypes.ErrInvalidApplicationID, ErrInvalidApplicationID, arg)
	}

	// Sign is unreachable for a uint256 argument; it guards the helper against a future
	// caller that hands it a signed value.
	if appID.Sign() < 0 || appID.BitLen() > 64 {
		return 0, errorsmod.Wrapf(
			ethiqtypes.ErrInvalidApplicationID,
			"application ID %s does not fit in uint64",
			appID.String(),
		)
	}

	return appID.Uint64(), nil
}

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

	appID, err := ParseApplicationID(args[1])
	if err != nil {
		return nil, common.Address{}, err
	}

	msg := &ethiqtypes.MsgMintHaqqByApplication{
		FromAddress:   sdk.AccAddress(sender.Bytes()).String(),
		ApplicationId: appID,
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

	appID, err := ParseApplicationID(args[0])
	if err != nil {
		return nil, err
	}

	req := &ethiqtypes.QueryCalculateForApplicationRequest{
		ApplicationId: appID,
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
	msg sdk.Msg,
) error {
	if contract.CallerAddress == origin {
		return nil
	}

	auth, expiration, err := authorization.CheckAuthzExists(ctx, authzKeeper, contract.CallerAddress, origin, sdk.MsgTypeURL(msg))
	if err != nil {
		return fmt.Errorf(authorization.ErrAuthzDoesNotExistOrExpired, contract.CallerAddress, origin)
	}

	// Accept the grant with the actual message
	_, isMintAuth := auth.(*ethiqtypes.MintHaqqAuthorization)
	_, isMintByAppAuth := auth.(*ethiqtypes.MintHaqqByApplicationIDAuthorization)
	if !isMintAuth && !isMintByAppAuth {
		return fmt.Errorf("expected MintHaqqAuthorization or MintHaqqByApplicationIDAuthorization, got %T", auth)
	}

	resp, err := auth.Accept(ctx, msg)
	if err != nil {
		return err
	}

	if !resp.Accept {
		return fmt.Errorf("authorization not accepted")
	}

	// Update grant if needed (application-based authz is always deleted after use)
	if resp.Delete {
		if err = authzKeeper.DeleteGrant(ctx, contract.CallerAddress.Bytes(), origin.Bytes(), sdk.MsgTypeURL(msg)); err != nil {
			return err
		}
	} else if resp.Updated != nil {
		if err = authzKeeper.SaveGrant(ctx, contract.CallerAddress.Bytes(), origin.Bytes(), resp.Updated, expiration); err != nil {
			return err
		}
	}

	return nil
}
