package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	paramstore paramtypes.Subspace

	accountKeeper authkeeper.AccountKeeper
	bankKeeper    bankkeeper.Keeper
	erc20Keeper   types.ERC20Keeper
	vestingKeeper types.VestingKeeper
}

// NewKeeper creates new Keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	ps paramtypes.Subspace,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	erc20 types.ERC20Keeper,
	vk types.VestingKeeper,
) Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		paramstore:    ps,
		accountKeeper: ak,
		bankKeeper:    bk,
		erc20Keeper:   erc20,
		vestingKeeper: vk,
	}
}
