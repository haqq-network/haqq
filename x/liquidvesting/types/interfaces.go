package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/haqq-network/haqq/x/erc20/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
	"time"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(sdk.Context, sdk.AccAddress) authtypes.AccountI // only used for simulation
	SetAccount(sdk.Context, authtypes.AccountI)
	NewAccount(ctx sdk.Context, acc authtypes.AccountI) authtypes.AccountI
	GetModuleAddress(moduleName string) sdk.AccAddress

	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	BlockedAddr(addr sdk.AccAddress) bool

	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error

	HasBalance(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coin) bool

	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error

	GetDenomMetaData(ctx sdk.Context, denom string) (banktypes.Metadata, bool)
	SetDenomMetaData(ctx sdk.Context, denomMetaData banktypes.Metadata)
}

type ERC20Keeper interface {
	RegisterCoin(ctx sdk.Context, coinMetadata banktypes.Metadata) (*types.TokenPair, error)
}

type VestingKeeper interface {
	ApplyVestingSchedule(
		ctx sdk.Context,
		funder, funded sdk.AccAddress,
		coins sdk.Coins,
		startTime time.Time,
		lockupPeriods, vestingPeriods sdkvesting.Periods,
		merge bool,
	) (vestingAcc *vestingtypes.ClawbackVestingAccount, newAccountCreated, wasMerged bool, err error)
}
