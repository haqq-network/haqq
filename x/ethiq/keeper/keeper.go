package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/haqq-network/haqq/utils"
	erc20keeper "github.com/haqq-network/haqq/x/erc20/keeper"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	"github.com/haqq-network/haqq/x/ethiq/types"
)

// Keeper of the ethiq store
type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	paramstore paramtypes.Subspace

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	erc20Keeper   erc20keeper.Keeper
}

// NewKeeper creates a new ethiq Keeper instance
func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	ps paramtypes.Subspace,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	ek erc20keeper.Keeper,
) Keeper {
	// ensure ethiq module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the ethiq module account has not been set")
	}

	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		storeKey:      storeKey,
		cdc:           cdc,
		paramstore:    ps,
		accountKeeper: ak,
		bankKeeper:    bk,
		erc20Keeper:   ek,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetEthiqSupply returns the current supply of aethiq coins
func (k Keeper) GetEthiqSupply(ctx sdk.Context) sdkmath.Int {
	return k.bankKeeper.GetSupply(ctx, types.BaseDenom).Amount
}

// CalculateRequiredISLM calculates the amount of aISLM required to mint the given amount of aethiq
// Formula: requiredIslm = StartRate * ethiqAmount + CurveCoefficient * ((currentEthiqTotalSupply + ethiqAmount)^(PowerCoefficient) - (currentEthiqTotalSupply)^(PowerCoefficient)) / (PowerCoefficient)
func (k Keeper) CalculateRequiredISLM(ctx sdk.Context, ethiqAmount sdkmath.Int) (sdkmath.Int, sdkmath.LegacyDec, error) {
	params := k.GetParams(ctx)

	if !params.Enabled {
		return sdkmath.Int{}, sdkmath.LegacyDec{}, types.ErrModuleDisabled
	}

	currentEthiqTotalSupply := sdkmath.LegacyNewDecFromInt(k.GetEthiqSupply(ctx))
	supplyAfter := currentEthiqTotalSupply.Add(sdkmath.LegacyNewDecFromInt(ethiqAmount))

	// Calculate (currentEthiqTotalSupply + ethiqAmount)^(PowerCoefficient)
	powerAfter, err := powerDec(supplyAfter, params.PowerCoefficient)
	if err != nil {
		return sdkmath.Int{}, sdkmath.LegacyDec{}, errorsmod.Wrapf(types.ErrCalculationFailed, "failure during power calculation: %v", err)
	}

	// Calculate (currentEthiqTotalSupply)^(PowerCoefficient)
	powerBefore, err := powerDec(currentEthiqTotalSupply, params.PowerCoefficient)
	if err != nil {
		return sdkmath.Int{}, sdkmath.LegacyDec{}, errorsmod.Wrapf(types.ErrCalculationFailed, "failure during power calculation: %v", err)
	}

	// Calculate the difference: (supplyAfter^PowerCoefficient - supplyBefore^PowerCoefficient)
	diff := powerAfter.Sub(powerBefore)

	// Calculate: CurveCoefficient * diff / PowerCoefficient
	curvePart := params.CurveCoefficient.Mul(diff).Quo(params.PowerCoefficient)

	// Calculate: StartRate * ethiqAmount
	startRatePart := params.StartRate.Mul(sdkmath.LegacyNewDecFromInt(ethiqAmount))

	// Total required: StartRate * ethiqAmount + CurveCoefficient * diff / PowerCoefficient
	result := startRatePart.Add(curvePart)

	// Convert to Int (truncate)
	requiredISLM := result.TruncateInt()

	// Ensure at least 1
	if requiredISLM.LT(sdkmath.OneInt()) {
		requiredISLM = sdkmath.OneInt()
		result = sdkmath.LegacyNewDecFromInt(requiredISLM)
	}

	// Calculate average price per unit
	pricePerUnit := result.Quo(sdkmath.LegacyNewDecFromInt(ethiqAmount))

	return requiredISLM, pricePerUnit, nil
}

// Mint mints aethiq coins in exchange for aISLM coins
// It validates the mint request, calculates required ISLM, burns ISLM, and mints ethiq
// Returns the actual ISLM amount used and any error
func (k Keeper) Mint(ctx sdk.Context, ethiqAmount sdkmath.Int, maxIslmAmount sdkmath.Int, fromAddress sdk.AccAddress, toAddress sdk.AccAddress) (sdkmath.Int, error) {
	params := k.GetParams(ctx)

	// Short no-op circuit if module is disabled
	if !params.Enabled {
		return sdkmath.ZeroInt(), nil
	}

	// Validate toAddress
	if toAddress.Empty() {
		return sdkmath.ZeroInt(), errorsmod.Wrap(types.ErrInvalidAddress, "to_address cannot be empty")
	}

	// Return error if ethiqAmount is outbound the limits
	if ethiqAmount.LT(params.MinMintPerTx) {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrInvalidAmount, "ethiq_amount %s is less than min_mint_per_tx %s", ethiqAmount, params.MinMintPerTx)
	}

	if ethiqAmount.GT(params.MaxMintPerTx) {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrInvalidAmount, "ethiq_amount %s is greater than max_mint_per_tx %s", ethiqAmount, params.MaxMintPerTx)
	}

	// Return error if ethiqAmount less than 1
	if ethiqAmount.LT(sdkmath.OneInt()) {
		return sdkmath.ZeroInt(), errorsmod.Wrap(types.ErrInvalidAmount, "ethiq_amount must be at least 1")
	}

	// Calculate required ISLM amount
	requiredISLM, _, err := k.CalculateRequiredISLM(ctx, ethiqAmount)
	if err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrap(err, "failed to calculate required ISLM")
	}

	// Return error if RequiredISLM less than 1
	if requiredISLM.LT(sdkmath.OneInt()) {
		return sdkmath.ZeroInt(), errorsmod.Wrap(types.ErrInvalidAmount, "calculated required ISLM is less than 1")
	}

	// Return error if RequiredISLM greater than maxIslmAmount
	if requiredISLM.GT(maxIslmAmount) {
		return sdkmath.ZeroInt(), errorsmod.Wrapf(types.ErrInsufficientFunds, "required ISLM %s is greater than max_islm_amount %s", requiredISLM, maxIslmAmount)
	}

	// Send RequiredISLM coins to module account
	islmCoin := sdk.NewCoin(utils.BaseDenom, requiredISLM)
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, fromAddress, types.ModuleName, sdk.NewCoins(islmCoin))
	if err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrap(err, "failed to send ISLM to module account")
	}

	// Burn RequiredISLM coins from module account
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(islmCoin))
	if err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrap(err, "failed to burn ISLM coins")
	}

	// Update TotalBurnedAmount
	k.AddToTotalBurnedAmount(ctx, requiredISLM)

	// Mint ethiqAmount coins to module account
	ethiqCoin := sdk.NewCoin(types.BaseDenom, ethiqAmount)
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(ethiqCoin))
	if err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrap(err, "failed to mint ethiq coins")
	}

	// Send minted ethiqAmount from module account to toAddress
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, toAddress, sdk.NewCoins(ethiqCoin))
	if err != nil {
		return sdkmath.ZeroInt(), errorsmod.Wrap(err, "failed to send ethiq to recipient")
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeMintExecuted,
			sdk.NewAttribute(types.AttributeKeyEthiqAmount, ethiqAmount.String()),
			sdk.NewAttribute(types.AttributeKeyISLMAmount, requiredISLM.String()),
			sdk.NewAttribute(types.AttributeKeyToAddress, toAddress.String()),
			sdk.NewAttribute(types.AttributeKeyFromAddress, fromAddress.String()),
		),
	)

	return requiredISLM, nil
}

// EnsureEthiqMetadata ensures that the aethiq denom metadata is set up correctly
func (k Keeper) EnsureEthiqMetadata(ctx sdk.Context) error {
	// Check if metadata already exists
	_, found := k.bankKeeper.GetDenomMetaData(ctx, types.BaseDenom)
	if found {
		return nil
	}

	// Create metadata with base denom (exponent 0) and display denom (exponent 18)
	metadata := banktypes.Metadata{
		Description: "Ethiq token",
		Base:        types.BaseDenom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    types.BaseDenom,
				Exponent: 0,
			},
			{
				Denom:    types.DisplayDenom,
				Exponent: 18,
			},
		},
		Name:    "Ethiq",
		Symbol:  "ETHIQ",
		Display: types.DisplayDenom,
	}

	if err := metadata.Validate(); err != nil {
		return err
	}

	k.bankKeeper.SetDenomMetaData(ctx, metadata)
	return nil
}

// EnsureEthiqERC20Registration ensures that aethiq is registered as a dynamic precompile
func (k Keeper) EnsureEthiqERC20Registration(ctx sdk.Context) error {
	// Check if already registered
	if k.erc20Keeper.IsDenomRegistered(ctx, types.BaseDenom) {
		return nil
	}

	// Derive ERC20 address from denom (similar to how IBC denoms work)
	// We'll use a deterministic address based on the module address and denom
	moduleAddr := k.accountKeeper.GetModuleAddress(types.ModuleName)
	denomBytes := []byte(types.BaseDenom)
	hash := crypto.Keccak256(moduleAddr.Bytes(), denomBytes)
	erc20Address := common.BytesToAddress(hash[:20])

	// Create token pair
	pair := erc20types.NewTokenPair(erc20Address, types.BaseDenom, erc20types.OWNER_MODULE)
	if err := pair.Validate(); err != nil {
		return err
	}

	// Set token pair in erc20 keeper
	k.erc20Keeper.SetToken(ctx, pair)

	// Register as dynamic precompile
	err := k.erc20Keeper.EnableDynamicPrecompiles(ctx, erc20Address)
	if err != nil {
		return err
	}

	// Register code hash
	k.erc20Keeper.RegisterERC20CodeHash(ctx, erc20Address)

	return nil
}
