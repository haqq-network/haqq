package ethiq

import (
	"fmt"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/utils"
	ethiqtypes "github.com/haqq-network/haqq/x/ethiq/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

func (p Precompile) Approve(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	grantee, coin, typeURLs, err := authorization.CheckApprovalArgs(args, utils.BaseDenom)
	if err != nil {
		return nil, err
	}

	for _, typeURL := range typeURLs {
		switch typeURL {
		case MintHaqqMsgURL:
			if err = p.grantOrDeleteMintHaqqAuthz(ctx, grantee, origin, coin, typeURL); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf(cmn.ErrInvalidMsgType, "ethiq", typeURL)
		}
	}

	if err := p.EmitApprovalEvent(ctx, stateDB, grantee, origin, coin, typeURLs); err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (p Precompile) ApproveApplicationID(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	grantee, appID, methods, err := checkApplicationIDArgs(args)
	if err != nil {
		return nil, err
	}

	for _, methodURL := range methods {
		if methodURL != MsgMintHaqqByApplicationMsgURL {
			return nil, fmt.Errorf(cmn.ErrInvalidMsgType, "ethiq", methodURL)
		}
	}

	appIDs := []uint64{appID}
	// Application-based authz is keyed by message type URL, so we merge with an existing grant
	// instead of overwriting it when approveApplicationID is called multiple times.
	if existingAuthz, _ := p.AuthzKeeper.GetAuthorization(ctx, grantee.Bytes(), origin.Bytes(), MsgMintHaqqByApplicationMsgURL); existingAuthz != nil {
		existingMintByAppAuthz, ok := existingAuthz.(*ethiqtypes.MintHaqqByApplicationIDAuthorization)
		if !ok {
			return nil, errorsmod.Wrapf(authz.ErrUnknownAuthorizationType, "expected: *ethiqtypes.MintHaqqByApplicationIDAuthorization, received: %T", existingAuthz)
		}

		appIDs = make([]uint64, 0, len(existingMintByAppAuthz.ApplicationsList)+1)
		appIDs = append(appIDs, existingMintByAppAuthz.ApplicationsList...)

		found := false
		for _, existingAppID := range existingMintByAppAuthz.ApplicationsList {
			if existingAppID == appID {
				found = true
				break
			}
		}
		if !found {
			appIDs = append(appIDs, appID)
		}
	}

	authz := &ethiqtypes.MintHaqqByApplicationIDAuthorization{
		ApplicationsList: appIDs,
	}

	if err = authz.ValidateBasic(); err != nil {
		return nil, err
	}

	expiration := ctx.BlockTime().Add(p.ApprovalExpiration).UTC()
	if err = p.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), origin.Bytes(), authz, &expiration); err != nil {
		return nil, err
	}

	if err := p.EmitApprovalEvent(ctx, stateDB, grantee, origin, nil, methods); err != nil {
		return nil, err
	}

	if err := p.EmitApplicationIDApprovalEvent(ctx, stateDB, grantee, origin, appID, appIDs); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// Revoke deletes whole grants by message type URL.
//
// It accepts MsgMintHaqqByApplication as well, even though the matching grant can only be
// created through approveApplicationID: approving needs type-specific data (which application),
// revoking does not. Without it there would be no single call that drops an application grant -
// only revokeApplicationID one ID at a time, which is useless when a grantee has to be cut off
// at once. The staking precompile treats all four of its message types the same way.
//
// DeleteGrant is not type-aware, so this is also the only way to clear a grant of a foreign
// authorization type (a GenericAuthorization placed under the same URL by a Cosmos MsgGrant),
// which approveApplicationID and revokeApplicationID both refuse to touch.
func (p Precompile) Revoke(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	grantee, typeURLs, err := authorization.CheckRevokeArgs(args)
	if err != nil {
		return nil, err
	}

	for _, typeURL := range typeURLs {
		switch typeURL {
		case MintHaqqMsgURL, MsgMintHaqqByApplicationMsgURL:
			// DeleteGrant fails when there is no such grant, and the precompile call is atomic,
			// so a list naming one missing type URL reverts the whole revocation.
			if err = p.AuthzKeeper.DeleteGrant(ctx, grantee.Bytes(), origin.Bytes(), typeURL); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf(cmn.ErrInvalidMsgType, "ethiq", typeURL)
		}
	}

	if err = authorization.EmitRevocationEvent(cmn.EmitEventArgs{
		Ctx:            ctx,
		StateDB:        stateDB,
		ContractAddr:   p.Address(),
		ContractEvents: p.ABI.Events,
		EventData: authorization.EventRevocation{
			Granter:  origin,
			Grantee:  grantee,
			TypeUrls: typeURLs,
		},
	}); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p Precompile) RevokeApplicationID(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	grantee, appID, methods, err := checkApplicationIDArgs(args)
	if err != nil {
		return nil, err
	}

	for _, methodURL := range methods {
		if methodURL != MsgMintHaqqByApplicationMsgURL {
			return nil, fmt.Errorf(cmn.ErrInvalidMsgType, "ethiq", methodURL)
		}
	}

	existingAuthz, expiration, err := authorization.CheckAuthzExists(ctx, p.AuthzKeeper, grantee, origin, MsgMintHaqqByApplicationMsgURL)
	if err != nil {
		return nil, err
	}

	existingMintByAppAuthz, ok := existingAuthz.(*ethiqtypes.MintHaqqByApplicationIDAuthorization)
	if !ok {
		return nil, errorsmod.Wrapf(authz.ErrUnknownAuthorizationType, "expected: *ethiqtypes.MintHaqqByApplicationIDAuthorization, received: %T", existingAuthz)
	}

	remaining := make([]uint64, 0, len(existingMintByAppAuthz.ApplicationsList))
	found := false
	for _, existingAppID := range existingMintByAppAuthz.ApplicationsList {
		if existingAppID == appID {
			found = true
			continue
		}
		remaining = append(remaining, existingAppID)
	}
	if !found {
		return nil, fmt.Errorf("application ID %d is not in allow list", appID)
	}

	if len(remaining) == 0 {
		if err = p.AuthzKeeper.DeleteGrant(ctx, grantee.Bytes(), origin.Bytes(), MsgMintHaqqByApplicationMsgURL); err != nil {
			return nil, err
		}

		// Revocation states that the grant is gone, so it is emitted only when it actually is.
		// A partial revoke leaves the grant in place and is reported by ApplicationIDRevocation
		// alone - see the event semantics note in events.go.
		if err = authorization.EmitRevocationEvent(cmn.EmitEventArgs{
			Ctx:            ctx,
			StateDB:        stateDB,
			ContractAddr:   p.Address(),
			ContractEvents: p.ABI.Events,
			EventData: authorization.EventRevocation{
				Granter:  origin,
				Grantee:  grantee,
				TypeUrls: methods,
			},
		}); err != nil {
			return nil, err
		}
	} else {
		updated := &ethiqtypes.MintHaqqByApplicationIDAuthorization{ApplicationsList: remaining}
		if err = updated.ValidateBasic(); err != nil {
			return nil, err
		}
		if err = p.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), origin.Bytes(), updated, expiration); err != nil {
			return nil, err
		}
	}

	if err = p.EmitApplicationIDRevocationEvent(ctx, stateDB, grantee, origin, appID, remaining); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// IncreaseAllowance implements the ethiq increase allowance transactions.
func (p Precompile) IncreaseAllowance(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	grantee, coin, typeURLs, err := authorization.CheckApprovalArgs(args, utils.BaseDenom)
	if err != nil {
		return nil, err
	}

	// Reject the unlimited sentinel up front: CheckApprovalArgs returns a nil coin
	// for MaxUint256, and increasing or decreasing a finite limit by "unlimited"
	// has no meaning. Doing it here makes the error independent of whether the
	// grant exists.
	if err := authorization.RequireLimitedAmount(coin, authorization.IncreaseAllowanceMethod); err != nil {
		return nil, err
	}

	for _, typeURL := range typeURLs {
		switch typeURL {
		case MintHaqqMsgURL:
			if err = p.increaseMintHaqqAllowance(ctx, grantee, origin, coin, typeURL); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf(cmn.ErrInvalidMsgType, "ethiq", typeURL)
		}
	}

	if err := p.EmitAllowanceChangeEvent(ctx, stateDB, grantee, origin, typeURLs); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// DecreaseAllowance implements the ethiq decrease allowance transactions.
func (p Precompile) DecreaseAllowance(
	ctx sdk.Context,
	origin common.Address,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	grantee, coin, typeURLs, err := authorization.CheckApprovalArgs(args, utils.BaseDenom)
	if err != nil {
		return nil, err
	}

	// Reject the unlimited sentinel up front: CheckApprovalArgs returns a nil coin
	// for MaxUint256, and increasing or decreasing a finite limit by "unlimited"
	// has no meaning. Doing it here makes the error independent of whether the
	// grant exists.
	if err := authorization.RequireLimitedAmount(coin, authorization.DecreaseAllowanceMethod); err != nil {
		return nil, err
	}

	for _, typeURL := range typeURLs {
		switch typeURL {
		case MintHaqqMsgURL:
			if err = p.decreaseMintHaqqAllowance(ctx, grantee, origin, coin, typeURL); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf(cmn.ErrInvalidMsgType, "ethiq", typeURL)
		}
	}

	if err := p.EmitAllowanceChangeEvent(ctx, stateDB, grantee, origin, typeURLs); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// grantOrDeleteMintHaqqAuthz grants or deletes mint haqq authorization.
func (p Precompile) grantOrDeleteMintHaqqAuthz(
	ctx sdk.Context,
	grantee, granter common.Address,
	coin *sdk.Coin,
	msgURL string,
) error {
	// Case 1: coin is nil -> set authorization with no limit
	if coin == nil || coin.IsNil() {
		p.Logger(ctx).Debug(
			"setting authorization without limit",
			"grantee", grantee.String(),
			"granter", granter.String(),
		)
		return p.createMintHaqqAuthz(ctx, grantee, granter, nil, msgURL)
	}

	// Case 2: coin amount is zero or negative -> delete the authorization
	if !coin.Amount.IsPositive() {
		p.Logger(ctx).Debug(
			"deleting authorization",
			"grantee", grantee.String(),
			"granter", granter.String(),
		)
		return p.AuthzKeeper.DeleteGrant(ctx, grantee.Bytes(), granter.Bytes(), msgURL)
	}

	// Case 3: coin amount is non zero -> set with custom amount
	return p.createMintHaqqAuthz(ctx, grantee, granter, coin, msgURL)
}

// createMintHaqqAuthz creates a mint haqq authorization.
func (p Precompile) createMintHaqqAuthz(
	ctx sdk.Context,
	grantee, granter common.Address,
	coin *sdk.Coin,
	_ string,
) error {
	mintAuthz := &ethiqtypes.MintHaqqAuthorization{
		SpendLimit: coin,
	}

	if err := mintAuthz.ValidateBasic(); err != nil {
		return err
	}

	expiration := ctx.BlockTime().Add(p.ApprovalExpiration).UTC()
	return p.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), mintAuthz, &expiration)
}

// increaseMintHaqqAllowance increases the allowance for mint haqq authorization.
func (p Precompile) increaseMintHaqqAllowance(
	ctx sdk.Context,
	grantee, granter common.Address,
	coin *sdk.Coin,
	msgURL string,
) error {
	// Reject the unlimited sentinel before touching coin: CheckApprovalArgs returns
	// a nil coin for MaxUint256 and the addition below dereferences it.
	if err := authorization.RequireLimitedAmount(coin, authorization.IncreaseAllowanceMethod); err != nil {
		return err
	}

	existingAuthz, expiration, err := authorization.CheckAuthzExists(ctx, p.AuthzKeeper, grantee, granter, msgURL)
	if err != nil {
		return err
	}

	mintAuthz, ok := existingAuthz.(*ethiqtypes.MintHaqqAuthorization)
	if !ok {
		return errorsmod.Wrapf(authz.ErrUnknownAuthorizationType, "expected: *ethiqtypes.MintHaqqAuthorization, received: %T", existingAuthz)
	}

	// If the authorization has no limit, no operation is performed
	if mintAuthz.SpendLimit == nil {
		p.Logger(ctx).Debug("increaseAllowance called with no limit (mintAuthz.SpendLimit == nil): no-op")
		return nil
	}

	// Add the amount to the limit, rejecting a sum that does not fit in sdkmath.Int
	// instead of panicking inside Int.Add.
	newLimit, err := authorization.AddAllowance(mintAuthz.SpendLimit.Amount, coin.Amount)
	if err != nil {
		return err
	}
	mintAuthz.SpendLimit.Amount = newLimit

	return p.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), mintAuthz, expiration)
}

// decreaseMintHaqqAllowance decreases the allowance for mint haqq authorization.
func (p Precompile) decreaseMintHaqqAllowance(
	ctx sdk.Context,
	grantee, granter common.Address,
	coin *sdk.Coin,
	msgURL string,
) error {
	// Reject the unlimited sentinel before touching coin: CheckApprovalArgs returns
	// a nil coin for MaxUint256 and every branch below dereferences it.
	if err := authorization.RequireLimitedAmount(coin, authorization.DecreaseAllowanceMethod); err != nil {
		return err
	}

	existingAuthz, expiration, err := authorization.CheckAuthzExists(ctx, p.AuthzKeeper, grantee, granter, msgURL)
	if err != nil {
		return err
	}

	mintAuthz, ok := existingAuthz.(*ethiqtypes.MintHaqqAuthorization)
	if !ok {
		return errorsmod.Wrapf(authz.ErrUnknownAuthorizationType, "expected: *ethiqtypes.MintHaqqAuthorization, received: %T", existingAuthz)
	}

	// If the authorization has no limit, no operation is performed
	if mintAuthz.SpendLimit == nil {
		p.Logger(ctx).Debug("decreaseAllowance called with no limit (mintAuthz.SpendLimit == nil): no-op")
		return nil
	}

	// If the authorization limit is less than the subtraction amount, return error
	if mintAuthz.SpendLimit.Amount.LT(coin.Amount) {
		return fmt.Errorf("decrease amount %s exceeds spend limit %s", coin.Amount, mintAuthz.SpendLimit.Amount)
	}

	// Subtract the amount from the limit
	mintAuthz.SpendLimit.Amount = mintAuthz.SpendLimit.Amount.Sub(coin.Amount)

	// If limit becomes zero or negative, delete the authorization
	if !mintAuthz.SpendLimit.Amount.IsPositive() {
		return p.AuthzKeeper.DeleteGrant(ctx, grantee.Bytes(), granter.Bytes(), msgURL)
	}

	return p.AuthzKeeper.SaveGrant(ctx, grantee.Bytes(), granter.Bytes(), mintAuthz, expiration)
}

// EmitApprovalEvent creates a new approval event.
func (p Precompile) EmitApprovalEvent(ctx sdk.Context, stateDB vm.StateDB, grantee, granter common.Address, coin *sdk.Coin, typeURLs []string) error {
	event := p.ABI.Events[authorization.EventTypeApproval]
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

	value := abi.MaxUint256
	if coin != nil {
		value = coin.Amount.BigInt()
	}

	arguments := abi.Arguments{event.Inputs[2], event.Inputs[3]}
	packed, err := arguments.Pack(typeURLs, value)
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

// EmitAllowanceChangeEvent creates a new allowance change event.
func (p Precompile) EmitAllowanceChangeEvent(ctx sdk.Context, stateDB vm.StateDB, grantee, granter common.Address, typeURLs []string) error {
	event := p.ABI.Events[authorization.EventTypeAllowanceChange]
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

	newValues := make([]*big.Int, len(typeURLs))
	for i, msgURL := range typeURLs {
		msgAuthz, _ := p.AuthzKeeper.GetAuthorization(ctx, grantee.Bytes(), granter.Bytes(), msgURL)
		if mintAuthz, ok := msgAuthz.(*ethiqtypes.MintHaqqAuthorization); ok {
			if mintAuthz.SpendLimit == nil {
				newValues[i] = abi.MaxUint256
			} else {
				newValues[i] = mintAuthz.SpendLimit.Amount.BigInt()
			}
		} else {
			newValues[i] = abi.MaxUint256
		}
	}

	arguments := abi.Arguments{event.Inputs[2], event.Inputs[3]}
	packed, err := arguments.Pack(typeURLs, newValues)
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

// checkApplicationIDArgs checks and parses the arguments of approveApplicationID and
// revokeApplicationID, which share the (address grantee, uint256 applicationId, string[] methods)
// signature declared in EthiqI.sol and abi.json.
func checkApplicationIDArgs(args []interface{}) (common.Address, uint64, []string, error) {
	if len(args) != 3 {
		return common.Address{}, 0, nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	grantee, ok := args[0].(common.Address)
	if !ok {
		return common.Address{}, 0, nil, fmt.Errorf("invalid grantee address: %v", args[0])
	}

	appID, err := ParseApplicationID(args[1])
	if err != nil {
		return common.Address{}, 0, nil, err
	}

	methods, ok := args[2].([]string)
	if !ok {
		return common.Address{}, 0, nil, fmt.Errorf("invalid methods: %v", args[2])
	}
	if len(methods) == 0 {
		return common.Address{}, 0, nil, fmt.Errorf("methods array cannot be empty")
	}

	return grantee, appID, methods, nil
}
