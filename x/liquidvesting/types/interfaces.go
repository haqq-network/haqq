package types

import (
	"context"
	"math/big"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
	NewAccount(context.Context, sdk.AccountI) sdk.AccountI
	GetModuleAddress(moduleName string) sdk.AccAddress
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	BlockedAddr(addr sdk.AccAddress) bool

	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error

	HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin

	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error

	GetDenomMetaData(ctx context.Context, denom string) (banktypes.Metadata, bool)
	SetDenomMetaData(ctx context.Context, denomMetaData banktypes.Metadata)
}

// ERC20Keeper defines the expected interface for the ERC20 module.
type ERC20Keeper interface {
	ToggleConversion(context.Context, *erc20types.MsgToggleConversion) (*erc20types.MsgToggleConversionResponse, error)
	ConvertERC20(ctx context.Context, msg *erc20types.MsgConvertERC20) (*erc20types.MsgConvertERC20Response, error)

	GetTokenPairID(ctx sdk.Context, token string) []byte
	GetTokenPair(ctx sdk.Context, id []byte) (erc20types.TokenPair, bool)
	BalanceOf(ctx sdk.Context, abi abi.ABI, contract, account common.Address) *big.Int
	EnableDynamicPrecompiles(ctx sdk.Context, addresses ...common.Address) error
	SetTokenPair(ctx sdk.Context, tokenPair erc20types.TokenPair)
	SetDenomMap(ctx sdk.Context, denom string, id []byte)
	SetERC20Map(ctx sdk.Context, erc20 common.Address, id []byte)
}

// VestingKeeper defines the expected interface for the Vesting module.
type VestingKeeper interface {
	ApplyVestingSchedule(
		ctx context.Context,
		funder, funded sdk.AccAddress,
		coins sdk.Coins,
		startTime time.Time,
		lockupPeriods, vestingPeriods sdkvesting.Periods,
		merge bool,
	) (vestingAcc *vestingtypes.ClawbackVestingAccount, newAccountCreated, wasMerged bool, err error)
}
