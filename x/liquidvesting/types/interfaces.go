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

// ERC20Keeper defines the expected interface for the ERC20 module.
type ERC20Keeper interface {
	ToggleConversion(ctx sdk.Context, token string) (erc20types.TokenPair, error)
	GetTokenPairID(ctx sdk.Context, token string) []byte
	GetTokenPair(ctx sdk.Context, id []byte) (erc20types.TokenPair, bool)
	BalanceOf(ctx sdk.Context, abi abi.ABI, contract, account common.Address) *big.Int
	ConvertCoin(goCtx context.Context, msg *erc20types.MsgConvertCoin) (*erc20types.MsgConvertCoinResponse, error)
	ConvertERC20(context.Context, *erc20types.MsgConvertERC20) (*erc20types.MsgConvertERC20Response, error)
	RegisterCoin(ctx sdk.Context, coinMetadata banktypes.Metadata) (*erc20types.TokenPair, error)
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
