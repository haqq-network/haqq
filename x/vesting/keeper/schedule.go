package keeper

import (
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	ethtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/x/vesting/types"
)

// ApplyVestingSchedule takes funder and funded addresses
func (k Keeper) ApplyVestingSchedule(
	ctx sdk.Context,
	funder, funded sdk.AccAddress,
	coins sdk.Coins,
	startTime time.Time,
	lockupPeriods, vestingPeriods sdkvesting.Periods,
	merge bool,
) (vestingAcc *types.ClawbackVestingAccount, newAccountCreated, wasMerged bool, err error) {
	targetAccount := k.accountKeeper.GetAccount(ctx, funded)
	createNewAcc := targetAccount == nil

	var isClawback bool

	ethAcc, isEthAccount := targetAccount.(*ethtypes.EthAccount)
	vestingAcc, isClawback = targetAccount.(*types.ClawbackVestingAccount)

	if isClawback && !merge {
		return nil, false, false, errorsmod.Wrapf(types.ErrApplyShedule, "account %s already exists; consider using --merge", funded)
	}

	codeHash := common.BytesToHash(crypto.Keccak256(nil))
	if isEthAccount {
		codeHash = ethAcc.GetCodeHash()
	}

	switch {
	case createNewAcc:
		baseAcc := authtypes.NewBaseAccountWithAddress(funded)
		vestingAcc = types.NewClawbackVestingAccount(
			baseAcc,
			funder,
			coins,
			startTime,
			lockupPeriods,
			vestingPeriods,
			&codeHash,
		)
		acc := k.accountKeeper.NewAccount(ctx, vestingAcc)
		k.accountKeeper.SetAccount(ctx, acc)

		return vestingAcc, true, false, nil
	case !isClawback && !isEthAccount:
		return nil, false, false, errorsmod.Wrapf(types.ErrApplyShedule, "account %s already exists but can't be converted into vesting account", funded)
	case !isClawback && isEthAccount:
		baseAcc := ethAcc.GetBaseAccount()
		vestingAcc = types.NewClawbackVestingAccount(
			baseAcc,
			funder,
			coins,
			startTime,
			lockupPeriods,
			vestingPeriods,
			&codeHash,
		)
		acc := k.accountKeeper.NewAccount(ctx, vestingAcc)
		k.accountKeeper.SetAccount(ctx, acc)
		return vestingAcc, false, false, nil
	case isClawback && merge:
		if funder.String() != vestingAcc.FunderAddress {
			return nil, false, false, errorsmod.Wrapf(types.ErrApplyShedule, "account %s can only accept grants from account %s", funded, vestingAcc.FunderAddress)
		}

		err := k.addGrant(
			ctx,
			vestingAcc,
			types.Min64(startTime.Unix(), vestingAcc.StartTime.Unix()),
			lockupPeriods,
			vestingPeriods,
			coins,
		)
		if err != nil {
			return nil, false, false, err
		}
		k.accountKeeper.SetAccount(ctx, vestingAcc)
		return vestingAcc, false, true, nil
	default:
		return nil, false, true, errorsmod.Wrapf(types.ErrApplyShedule, "failed to initiate vesting for account %s", funded)
	}
}
