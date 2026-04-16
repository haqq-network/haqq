package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	erc20types "github.com/haqq-network/haqq/x/erc20/types"
)

// AccountKeeper defines the expected account keeper interface
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
	SetAccount(context.Context, sdk.AccountI)
	GetModuleAddress(string) sdk.AccAddress
	GetModuleAccount(context.Context, string) sdk.ModuleAccountI
}

// BankKeeper defines the expected bank keeper interface
type BankKeeper interface {
	GetAllBalances(context.Context, sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(context.Context, sdk.AccAddress, string, sdk.Coins) error
	SendCoinsFromModuleToAccount(context.Context, string, sdk.AccAddress, sdk.Coins) error
	MintCoins(context.Context, string, sdk.Coins) error
	BurnCoins(context.Context, string, sdk.Coins) error
	GetSupply(context.Context, string) sdk.Coin
	GetDenomMetaData(context.Context, string) (banktypes.Metadata, bool)
	SetDenomMetaData(context.Context, banktypes.Metadata)
}

type ERC20Keeper interface {
	IsDenomRegistered(sdk.Context, string) bool
	SetToken(sdk.Context, erc20types.TokenPair)
	EnableDynamicPrecompiles(sdk.Context, ...common.Address) error
	RegisterERC20CodeHash(sdk.Context, common.Address) error
}

type LiquidVestingKeeper interface {
	Redeem(sdk.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coin) error
}

type UCDAOKeeper interface {
	TrackAddBalance(sdk.Context, sdk.Coin)
	TrackSubBalance(sdk.Context, sdk.Coin)
	SetHoldersIndex(sdk.Context, sdk.AccAddress)
}
