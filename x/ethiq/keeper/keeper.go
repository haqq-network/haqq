package keeper

import (
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

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

// GetParams returns the total set of ethiq parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var params types.Params
	k.paramstore.GetParamSet(ctx, &params)
	return params
}

// SetParams sets the ethiq parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// GetTotalBurnedAmount returns the total amount of burned coins
func (k Keeper) GetTotalBurnedAmount(ctx sdk.Context) sdk.Coin {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TotalBurnedAmountKey)
	if bz == nil {
		return sdk.NewCoin(types.ISLMBaseDenom, sdkmath.ZeroInt())
	}

	var coin sdk.Coin
	k.cdc.MustUnmarshal(bz, &coin)
	return coin
}

// SetTotalBurnedAmount sets the total amount of burned coins
func (k Keeper) SetTotalBurnedAmount(ctx sdk.Context, coin sdk.Coin) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&coin)
	store.Set(types.TotalBurnedAmountKey, bz)
}

// AddToTotalBurnedAmount adds to the total burned amount
func (k Keeper) AddToTotalBurnedAmount(ctx sdk.Context, amount sdkmath.Int) {
	current := k.GetTotalBurnedAmount(ctx)
	newAmount := current.Amount.Add(amount)
	k.SetTotalBurnedAmount(ctx, sdk.NewCoin(types.ISLMBaseDenom, newAmount))
}

// GetEthiqSupply returns the current supply of aethiq coins
func (k Keeper) GetEthiqSupply(ctx sdk.Context) sdkmath.Int {
	return k.bankKeeper.GetSupply(ctx, types.BaseDenom).Amount
}

// CalculateRequiredISLM calculates the amount of aISLM required to mint the given amount of aethiq
// Uses a bonding curve formula: price = curveCoefficient * (supply + startRate) ^ powerCoefficient
// The required ISLM is the integral of the price function from supplyBefore to supplyAfter
// We use numerical integration (trapezoidal rule) to calculate the integral
func (k Keeper) CalculateRequiredISLM(ctx sdk.Context, ethiqAmount sdkmath.Int) (sdkmath.Int, sdkmath.LegacyDec, error) {
	params := k.GetParams(ctx)

	supplyBefore := sdkmath.LegacyNewDecFromInt(k.GetEthiqSupply(ctx))
	supplyAfter := supplyBefore.Add(sdkmath.LegacyNewDecFromInt(ethiqAmount))

	// Use numerical integration with trapezoidal rule
	// Divide the interval into steps for accuracy
	// For small amounts, use fewer steps; for large amounts, use more steps
	numSteps := int64(100)
	if ethiqAmount.LT(sdkmath.NewInt(1_000_000_000_000_000)) { // Less than 0.001 ethiq
		numSteps = 10
	}

	stepSize := sdkmath.LegacyNewDecFromInt(ethiqAmount).QuoInt64(numSteps)

	// Price function: price(x) = curveCoefficient * (x + startRate) ^ powerCoefficient
	priceFunc := func(x sdkmath.LegacyDec) sdkmath.LegacyDec {
		term := x.Add(params.StartRate)
		// Calculate term^powerCoefficient using iterative multiplication
		// Convert powerCoefficient to a ratio for approximation
		// For powerCoefficient = 1.1, we approximate as term^1 * term^0.1
		// term^0.1 ≈ term^(1/10) which we can approximate
		powerInt := params.PowerCoefficient.TruncateInt().Int64()
		powerFrac := params.PowerCoefficient.Sub(sdkmath.LegacyNewDecFromInt64(powerInt))

		result := sdkmath.LegacyOneDec()
		// Calculate integer power part
		for i := int64(0); i < powerInt; i++ {
			result = result.Mul(term)
		}

		// Approximate fractional power: term^fraction ≈ 1 + fraction * (term - 1) for small fractions
		// This is a linear approximation around 1
		if powerFrac.GT(sdkmath.LegacyZeroDec()) {
			fractionalPart := sdkmath.LegacyOneDec().Add(powerFrac.Mul(term.Sub(sdkmath.LegacyOneDec())))
			result = result.Mul(fractionalPart)
		}

		return params.CurveCoefficient.Mul(result)
	}

	// Trapezoidal rule: ∫f(x)dx ≈ (h/2) * [f(x0) + 2*f(x1) + 2*f(x2) + ... + 2*f(xn-1) + f(xn)]
	sum := priceFunc(supplyBefore).Add(priceFunc(supplyAfter))

	current := supplyBefore.Add(stepSize)
	for i := int64(1); i < numSteps; i++ {
		sum = sum.Add(priceFunc(current).MulInt64(2))
		current = current.Add(stepSize)
	}

	result := sum.Mul(stepSize).QuoInt64(2)

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
