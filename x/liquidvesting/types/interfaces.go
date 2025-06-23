package types

import (
	"context"
	"math/big"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
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

	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin

	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error

	GetDenomMetaData(ctx sdk.Context, denom string) (banktypes.Metadata, bool)
	SetDenomMetaData(ctx sdk.Context, denomMetaData banktypes.Metadata)
}

// ERC20Keeper defines the expected interface for the ERC20 module.
type ERC20Keeper interface {
	ToggleConversion(ctx sdk.Context, token string) (erc20types.TokenPair, error)
	GetTokenPairID(ctx sdk.Context, token string) []byte
	GetTokenPair(ctx sdk.Context, id []byte) (erc20types.TokenPair, bool)
	BalanceOf(ctx sdk.Context, abi abi.ABI, contract, account common.Address) *big.Int
	ConvertERC20(context.Context, *erc20types.MsgConvertERC20) (*erc20types.MsgConvertERC20Response, error)
	EnableDynamicPrecompiles(ctx sdk.Context, addresses ...common.Address) error
	SetTokenPair(ctx sdk.Context, tokenPair erc20types.TokenPair)
	SetDenomMap(ctx sdk.Context, denom string, id []byte)
	SetERC20Map(ctx sdk.Context, erc20 common.Address, id []byte)
}

// VestingKeeper defines the expected interface for the Vesting module.
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
