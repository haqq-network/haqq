package keeper

import (
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"

	"github.com/haqq-network/haqq/x/ibc/transfer/types"
)

// Keeper defines the modified IBC transfer keeper that embeds the original one.
// It also contains the bank keeper and the erc20 keeper to support ERC20 tokens
// to be sent via IBC.
type Keeper struct {
	*keeper.Keeper
	bankKeeper    types.BankKeeper
	erc20Keeper   types.ERC20Keeper
	accountKeeper types.AccountKeeper
}

// NewKeeper creates a new IBC transfer Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService corestore.KVStoreService,
	paramSubspace transfertypes.ParamSubspace,

	ics4Wrapper porttypes.ICS4Wrapper,
	channelKeeper transfertypes.ChannelKeeper,
	msgRouter transfertypes.MessageRouter,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	erc20Keeper types.ERC20Keeper,
	authority string,
) Keeper {
	// create the original IBC transfer keeper for embedding
	transferKeeper := keeper.NewKeeper(
		cdc, storeService, paramSubspace,
		ics4Wrapper, channelKeeper, msgRouter,
		accountKeeper, bankKeeper,
		authority,
	)

	return Keeper{
		Keeper:        &transferKeeper,
		bankKeeper:    bankKeeper,
		erc20Keeper:   erc20Keeper,
		accountKeeper: accountKeeper,
	}
}
