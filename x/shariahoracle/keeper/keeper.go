package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
)

// Keeper of the shariahoracle store
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	paramstore paramtypes.Subspace

	erc20Keeper   types.ERC20Keeper
	accountKeeper types.AccountKeeper
}

// NewKeeper creates new Keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	ps paramtypes.Subspace,
	ek types.ERC20Keeper,
	ak types.AccountKeeper,
) Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		paramstore:    ps,
		erc20Keeper:   ek,
		accountKeeper: ak,
	}
}
