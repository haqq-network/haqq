package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/haqq-network/haqq/utils"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	"github.com/haqq-network/haqq/x/liquidvesting/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

var _ Keeper = (*BaseKeeper)(nil)

// Keeper defines a module interface that facilitates liquid vesting operations.
type Keeper interface {
	Liquidate(ctx sdk.Context, fromAddress, toAddress sdk.AccAddress, amount sdk.Coin) (sdk.Coin, string, error)
	Redeem(ctx sdk.Context, fromAddress, toAddress sdk.AccAddress, amount sdk.Coin) error

	// Bank balance helpers used by the precompile to mirror SDK bank deltas into
	// the EVM StateDB journal when the precompile is invoked from a contract.
	BaseDenomBankBalance(ctx sdk.Context, addr sdk.AccAddress) sdkmath.Int

	// Params methods
	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error
	IsLiquidVestingEnabled(ctx sdk.Context) bool
	SetLiquidVestingEnabled(ctx sdk.Context, enable bool)
	ResetParamsToDefault(ctx sdk.Context)

	// Denom methods
	CreateDenom(ctx sdk.Context, originalDenom string, startTime int64, periods sdkvesting.Periods) (types.Denom, error)
	UpdateDenomPeriods(ctx sdk.Context, baseDenom string, newPeriods sdkvesting.Periods) error
	DeleteDenom(ctx sdk.Context, baseDenom string)
	GetDenom(ctx sdk.Context, baseDenom string) (val types.Denom, found bool)
	SetDenom(ctx sdk.Context, denom types.Denom)
	GetAllDenoms(ctx sdk.Context) []types.Denom
	GetDenomCounter(ctx sdk.Context) uint64
	SetDenomCounter(ctx sdk.Context, counter uint64)
	IterateDenoms(ctx sdk.Context, cb func(account types.Denom) (stop bool))

	// grpc query endpoints
	Denom(goCtx context.Context, req *types.QueryDenomRequest) (*types.QueryDenomResponse, error)
	Denoms(goCtx context.Context, req *types.QueryDenomsRequest) (*types.QueryDenomsResponse, error)
}

type BaseKeeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	paramstore paramtypes.Subspace

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	erc20Keeper   types.ERC20Keeper
	vestingKeeper types.VestingKeeper
}

// NewKeeper creates new Keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	ps paramtypes.Subspace,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	erc20 types.ERC20Keeper,
	vk types.VestingKeeper,
) Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return BaseKeeper{
		cdc:           cdc,
		storeKey:      storeKey,
		paramstore:    ps,
		accountKeeper: ak,
		bankKeeper:    bk,
		erc20Keeper:   erc20,
		vestingKeeper: vk,
	}
}

// BaseDenomBankBalance returns the bank balance of aISLM for addr (utils.BaseDenom).
// On Haqq, EvmDenom defaults to utils.BaseDenom; this is the balance the EVM keeper
// reconciles for native EVM accounts on tx commit, so the liquid precompile uses it
// to compute the bank delta produced by Liquidate/Redeem keeper calls and mirror
// that delta into the EVM StateDB journal when the caller is a contract.
func (k BaseKeeper) BaseDenomBankBalance(ctx sdk.Context, addr sdk.AccAddress) sdkmath.Int {
	return k.bankKeeper.GetBalance(ctx, addr, utils.BaseDenom).Amount
}

// Liquidate liquidates specified amount of token locked in vesting into liquid token
func (k BaseKeeper) Liquidate(ctx sdk.Context, liquidateFromAddress sdk.AccAddress, liquidateToAddress sdk.AccAddress, amount sdk.Coin) (sdk.Coin, string, error) {
	if !k.IsLiquidVestingEnabled(ctx) {
		return sdk.Coin{}, "", errorsmod.Wrapf(types.ErrModuleIsDisabled, "liquid vesting module is disabled")
	}

	// check amount denom
	if amount.Denom != utils.BaseDenom {
		return sdk.Coin{}, "", errorsmod.Wrapf(errortypes.ErrInvalidRequest, "unable to liquidate any other coin except aISLM")
	}

	// check amount
	minLiquidation := k.GetParams(ctx).MinimumLiquidationAmount
	if amount.IsLT(sdk.NewCoin(utils.BaseDenom, minLiquidation)) {
		return sdk.Coin{}, "", errorsmod.Wrapf(errortypes.ErrInvalidRequest, "unable to liquidate amount lesser than %d", minLiquidation)
	}

	liquidateFrom := liquidateFromAddress.String()
	liquidateTo := liquidateToAddress.String()

	// get account
	liquidateFromAccount := k.accountKeeper.GetAccount(ctx, liquidateFromAddress)
	if liquidateFromAccount == nil {
		return sdk.Coin{}, "", errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", liquidateFrom)
	}

	// check from account is vesting account
	va, isClawback := liquidateFromAccount.(*vestingtypes.ClawbackVestingAccount)
	if !isClawback {
		return sdk.Coin{}, "", errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s is regular nothing to liquidate", liquidateFrom)
	}

	// check there is not vesting periods on the schedule
	if !va.GetVestingCoins(ctx.BlockTime()).IsZero() {
		return sdk.Coin{}, "", errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s has vesting ongoing periods, unable to liquidate unvested coins", liquidateFrom)
	}

	// check account has liquidation target denom locked in vesting
	hasTargetDenom, lockedBalance := va.GetLockedUpCoins(ctx.BlockTime()).Find(amount.Denom)
	if !(hasTargetDenom) {
		return sdk.Coin{}, "", errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s doesn't contain coin specified as liquidation target", liquidateFrom)
	}

	// validate current locked periods have sufficient amount to be liquidated
	if lockedBalance.IsLT(amount) {
		return sdk.Coin{}, "", errorsmod.Wrapf(errortypes.ErrInvalidRequest, "account %s doesn't have sufficient amount of target coin for liquidation", liquidateFrom)
	}

	// calculate new schedule
	upcomingPeriods := types.ExtractUpcomingPeriods(va.GetStartTime(), va.GetEndTime(), va.LockupPeriods, ctx.BlockTime().Unix())
	decreasedPeriods, diffPeriods, err := types.SubtractAmountFromPeriods(upcomingPeriods, amount)
	if err != nil {
		return sdk.Coin{}, "", errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to calculate new schedule: %s", err.Error())
	}
	va.LockupPeriods = types.ReplacePeriodsTail(va.LockupPeriods, decreasedPeriods)
	va.OriginalVesting = va.OriginalVesting.Sub(amount)

	// all vesting periods are completed at this point, so we can reduce amounts without additional extracting logic
	decreasedVestingPeriods, _, err := types.SubtractAmountFromPeriods(va.VestingPeriods, amount)
	if err != nil {
		return sdk.Coin{}, "", errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to calculate new schedule: %s", err.Error())
	}

	va.VestingPeriods = types.ReplacePeriodsTail(va.VestingPeriods, decreasedVestingPeriods)

	k.accountKeeper.SetAccount(ctx, va)

	// transfer liquidated amount to liquid vesting module account
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, liquidateFromAddress, types.ModuleName, sdk.NewCoins(amount))
	if err != nil {
		return sdk.Coin{}, "", errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to transfer liquidated locked coins from account to module: %s", err.Error())
	}

	diffPeriods[0].Length -= types.CurrentPeriodShift(va.StartTime.Unix(), ctx.BlockTime().Unix(), va.LockupPeriods)
	liquidDenom, err := k.CreateDenom(ctx, amount.Denom, ctx.BlockTime().Unix(), diffPeriods)
	if err != nil {
		return sdk.Coin{}, "", errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to create denom for liquid token: %s", err.Error())
	}

	// create new sdk denom for liquidated locked coins
	liquidTokenMetadata := banktypes.Metadata{
		Description: "Liquid vesting token",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    liquidDenom.GetBaseDenom(),
				Exponent: 0,
			},
			{
				Denom:    liquidDenom.GetDisplayDenom(),
				Exponent: 18,
			},
		},
		Base:    liquidDenom.GetBaseDenom(),
		Display: liquidDenom.GetDisplayDenom(),
		Name:    liquidDenom.GetDisplayDenom(),
		Symbol:  liquidDenom.GetDisplayDenom(),
	}

	liquidTokenCoin := sdk.NewCoin(liquidDenom.GetBaseDenom(), amount.Amount)
	liquidTokenCoins := sdk.NewCoins(liquidTokenCoin)
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, liquidTokenCoins)
	if err != nil {
		return sdk.Coin{}, "", errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to mint liquid token: %s", err.Error())
	}

	k.bankKeeper.SetDenomMetaData(ctx, liquidTokenMetadata)

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, liquidateToAddress, liquidTokenCoins)
	if err != nil {
		return sdk.Coin{}, "", errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to transfer liquid tokens to account %s", err.Error())
	}

	// bind newly created denom to erc20 token
	// Create dummy IBC denom, just to bind ERC20 Precompile with newly created aLiquid denom
	fakeIBCDenom := utils.ComputeIBCDenom(types.ModuleName, liquidTokenMetadata.Base, amount.Denom)
	tokenPair, err := erc20types.NewTokenPairSTRv2(fakeIBCDenom)
	if err != nil {
		return sdk.Coin{}, "", errorsmod.Wrapf(types.ErrLiquidationFailed, "failed to create erc20 token pair: %s", err.Error())
	}
	// Set real denom to token pair, so precompile could handle transfers properly
	tokenPair.Denom = liquidTokenMetadata.Base
	// k.erc20Keeper.SetToken(ctx, tokenPair) unwrap it below due to pointer receiver in original method.
	k.erc20Keeper.SetTokenPair(ctx, tokenPair)
	k.erc20Keeper.SetDenomMap(ctx, tokenPair.Denom, tokenPair.GetID())
	k.erc20Keeper.SetERC20Map(ctx, tokenPair.GetERC20Contract(), tokenPair.GetID())

	err = k.erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract())
	if err != nil {
		return sdk.Coin{}, "", err
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeLiquidate,
				sdk.NewAttribute(sdk.AttributeKeySender, liquidateFrom),
				sdk.NewAttribute(types.AttributeKeyDestination, liquidateTo),
				sdk.NewAttribute(types.AttributeKeyAmount, liquidTokenCoin.String()),
			),
		},
	)

	return liquidTokenCoin, tokenPair.Erc20Address, nil
}

// Redeem redeems specified amount of liquid token into original locked token and adds them to account
func (k BaseKeeper) Redeem(ctx sdk.Context, fromAddress sdk.AccAddress, toAddress sdk.AccAddress, amount sdk.Coin) error {
	if !k.IsLiquidVestingEnabled(ctx) {
		return errorsmod.Wrapf(types.ErrModuleIsDisabled, "liquid vesting module is disabled")
	}

	fromAccount := k.accountKeeper.GetAccount(ctx, fromAddress)
	if fromAccount == nil {
		return errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", fromAddress.String())
	}

	// query liquid token info
	liquidDenom, found := k.GetDenom(ctx, amount.Denom)
	if !found {
		return errorsmod.Wrapf(errortypes.ErrNotFound, "liquidDenom %s does not exist", amount.Denom)
	}

	// Look up the ERC20 pair, but do not gate the redemption on it.
	//
	// Redeeming is burning the liquid denom and paying out the principal the module holds;
	// none of that needs x/erc20. The pair exists only to give the liquid denom an EVM
	// representation. Gating on it meant governance disabling the pair - or the global
	// EnableErc20 flag, see below - stranded the principal while the denom stayed
	// transferable through the bank and the ERC20 precompile, which does not consult
	// pair.Enabled either. The pair is used here for cleanup only.
	var (
		tokenPair      erc20types.TokenPair
		tokenPairFound bool
	)
	if tokenPairID := k.erc20Keeper.GetTokenPairID(ctx, amount.Denom); len(tokenPairID) > 0 {
		tokenPair, tokenPairFound = k.erc20Keeper.GetTokenPair(ctx, tokenPairID)
	}

	// check fromAccount has enough liquid token in balance
	if balance := k.bankKeeper.GetBalance(ctx, fromAddress, amount.Denom); balance.IsLT(amount) {
		return errorsmod.Wrapf(types.ErrRedeemFailed, "from account has insufficient balance")
	}

	// transfer liquid denom to liquidvesting module
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, fromAddress, types.ModuleName, sdk.NewCoins(amount))
	if err != nil {
		return errorsmod.Wrapf(types.ErrRedeemFailed, "failed to transfer liquid token to module: %s", err.Error())
	}

	// burn liquid token specified amount
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(amount))
	if err != nil {
		return errorsmod.Wrapf(types.ErrRedeemFailed, "failed to burn liquid tokens: %s", err.Error())
	}

	// subtract burned amount from token schedule
	originalDenomCoin := sdk.NewCoin(liquidDenom.GetOriginalDenom(), amount.Amount)
	decreasedPeriods, diffPeriods, err := types.SubtractAmountFromPeriods(liquidDenom.LockupPeriods, originalDenomCoin)
	if err != nil {
		return errorsmod.Wrapf(types.ErrRedeemFailed, "failed to calculate new liquid denom schedule: %s", err.Error())
	}
	// save modified token schedule
	if decreasedPeriods.TotalAmount().IsZero() {
		k.DeleteDenom(ctx, liquidDenom.GetBaseDenom())

		// Disabling the pair is cleanup for a denom that no longer exists, so write it
		// straight to the erc20 store. The msg server used before impersonated governance by
		// passing the gov module address as Authority, and it refuses to run at all while
		// EnableErc20 is off - which aborted the final redemption of every denom, the one
		// that zeroes the schedule, and left the last holders unable to exit.
		if tokenPairFound && tokenPair.Enabled {
			tokenPair.Enabled = false
			k.erc20Keeper.SetTokenPair(ctx, tokenPair)
		}
	} else {
		err = k.UpdateDenomPeriods(ctx, liquidDenom.GetBaseDenom(), decreasedPeriods)
		if err != nil {
			return errorsmod.Wrapf(types.ErrRedeemFailed, "failed to update liquid denom schedule: %s", err.Error())
		}
	}

	// transfer original token to account
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, toAddress, sdk.NewCoins(originalDenomCoin))
	if err != nil {
		return errorsmod.Wrapf(types.ErrRedeemFailed, "failed to transfer original denom to target account: %s", err.Error())
	}

	upcomingPeriods := types.ExtractUpcomingPeriods(
		liquidDenom.GetStartTime().Unix(),
		liquidDenom.GetEndTime().Unix(),
		diffPeriods,
		ctx.BlockTime().Unix(),
	)

	// if there are upcoming periods, apply vesting schedule on target account
	if len(upcomingPeriods) > 0 {
		funder := k.accountKeeper.GetModuleAddress(types.ModuleName)
		if funder == nil {
			return errorsmod.Wrapf(types.ErrRedeemFailed, "failed to get funder address")
		}

		// check if toAddress already a vesting account to apply current funder
		toAccount := k.accountKeeper.GetAccount(ctx, toAddress)
		if toAccount == nil {
			return errorsmod.Wrapf(errortypes.ErrNotFound, "account %s does not exist", toAddress.String())
		}
		toVestingAcc, isClawback := toAccount.(*vestingtypes.ClawbackVestingAccount)
		if isClawback {
			funder = sdk.MustAccAddressFromBech32(toVestingAcc.FunderAddress)
		}

		// Applying a schedule converts the recipient into a clawback vesting account, and for a
		// contract that conversion is a one-way door: MsgConvertVestingAccount is signed by the
		// account itself, which a contract cannot do, and the vesting precompile is not
		// registered, so there is no EVM path to it either. ConvertIntoVestingAccount refuses
		// contracts outright for that reason.
		//
		// Redeem cannot refuse them outright: x/ethiq redeems a Safe's own liquid balance back
		// into it (redeemAllLiquidVestingCoins), and there redeemFrom == redeemTo, so the
		// contract is acting on itself through an authorized call. What must not be possible is
		// a third party pushing that state onto someone else's contract, which redeemTo being
		// unconstrained would otherwise allow for the price of a dust redeem.
		if !fromAddress.Equals(toAddress) && !isClawback {
			if err := utils.IsContractAccount(toAccount); err == nil {
				return errorsmod.Wrapf(
					types.ErrRedeemFailed,
					"account %s is a contract account and cannot be converted into a clawback vesting account by another account",
					toAddress,
				)
			}
		}

		_, _, _, err = k.vestingKeeper.ApplyVestingSchedule(
			ctx,
			funder,
			toAddress,
			sdk.NewCoins(originalDenomCoin),
			liquidDenom.GetStartTime(),
			diffPeriods,
			sdkvesting.Periods{{Length: 0, Amount: sdk.NewCoins(originalDenomCoin)}},
			true,
		)
		if err != nil {
			return errorsmod.Wrapf(types.ErrRedeemFailed, "failed to apply vesting schedule to account %s: %s", toAddress, err.Error())
		}
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeRedeem,
				sdk.NewAttribute(sdk.AttributeKeySender, fromAddress.String()),
				sdk.NewAttribute(types.AttributeKeyDestination, toAddress.String()),
				sdk.NewAttribute(types.AttributeKeyAmount, amount.String()),
			),
		},
	)

	return nil
}
