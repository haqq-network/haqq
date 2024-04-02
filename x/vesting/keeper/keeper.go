package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/vesting/types"
)

// Keeper of this module maintains collections of vesting.
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper
}

// NewKeeper creates new instances of the vesting Keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	sk types.StakingKeeper,
) Keeper {
	return Keeper{
		storeKey:      storeKey,
		cdc:           cdc,
		accountKeeper: ak,
		bankKeeper:    bk,
		stakingKeeper: sk,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
