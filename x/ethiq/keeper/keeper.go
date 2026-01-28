package keeper

import (
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// GetHaqqSupply returns the current supply of aHAQQ coins
func (k Keeper) GetHaqqSupply(ctx sdk.Context) sdkmath.Int {
	return k.bankKeeper.GetSupply(ctx, types.BaseDenom).Amount
}

// EnsureHaqqMetadata ensures that the aethiq denom metadata is set up correctly
func (k Keeper) EnsureHaqqMetadata(ctx sdk.Context) error {
	// Check if metadata already exists
	_, found := k.bankKeeper.GetDenomMetaData(ctx, types.BaseDenom)
	if found {
		return nil
	}

	if err := types.HaqqDenomMetaData.Validate(); err != nil {
		return err
	}

	k.bankKeeper.SetDenomMetaData(ctx, types.HaqqDenomMetaData)

	return nil
}

// EnsureHaqqERC20Registration ensures that aHAQQ is registered as a dynamic precompile
func (k Keeper) EnsureHaqqERC20Registration(ctx sdk.Context) error {
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
	if err := k.erc20Keeper.EnableDynamicPrecompiles(ctx, erc20Address); err != nil {
		return err
	}

	// Register code hash
	if err := k.erc20Keeper.RegisterERC20CodeHash(ctx, erc20Address); err != nil {
		return err
	}

	return nil
}
