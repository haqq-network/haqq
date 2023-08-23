package v150

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/server/config"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/pkg/errors"

	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
)

type Deposit struct {
	Owner             common.Address `abi:"owner"`
	Timestamp         *big.Int       `abi:"timestamp"`
	SumInWeiDeposited *big.Int       `abi:"sumInWeiDeposited"`
	SumPaidAlready    *big.Int       `abi:"sumPaidAlready"`
	SumLeftToPay      *big.Int       `abi:"sumLeftToPay"`
}

type DepositTyped struct {
	StartTime time.Time
	Amount    sdk.Coin
	Paid      sdk.Coin
	Rest      sdk.Coin
}

func (r *RevestingUpgradeHandler) readVestingContractState() (map[string][]DepositTyped, error) {
	r.ctx.Logger().Info("Loading Vesting Contract state")

	bondDenom := r.StakingKeeper.BondDenom(r.ctx)
	caller := common.HexToAddress("0x2eA0342dBd00ee0cDDA68EAE9Ee06edDfe914dc1") // dummy caller
	contractAddress := r.getVestingContractAddress()
	contractABI, contractFunction := r.getVestingContractABI()

	// Parse the contract ABI
	cAbi, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return nil, errors.Wrap(err, "abi.JSON")
	}

	// Create a call data buffer for the function call
	callData, err := cAbi.Pack(contractFunction)
	if err != nil {
		return nil, errors.Wrap(err, "abi.Pack")
	}

	args, err := json.Marshal(evmtypes.TransactionArgs{
		From: &caller,
		To:   &contractAddress,
		Data: (*hexutil.Bytes)(&callData),
	})
	if err != nil {
		return nil, errors.Wrap(err, "json.Marshal")
	}

	calReq := &evmtypes.EthCallRequest{
		Args:   args,
		GasCap: config.DefaultGasCap,
	}

	// Call smart-contract
	resp, err := r.EvmKeeper.EthCall(r.ctx, calReq)
	if err != nil {
		return nil, errors.Wrap(err, "evm.EthCall")
	}

	// Parse contract response
	r.ctx.Logger().Info("Unpack response:")

	type Ret struct {
		Result []Deposit
	}
	var deposits Ret

	if err := cAbi.UnpackIntoInterface(&deposits, contractFunction, resp.Ret); err != nil {
		return nil, errors.Wrap(err, "abi.UnpackIntoInterface")
	}

	depositsTyped := make(map[string][]DepositTyped)
	for _, d := range deposits.Result {
		dt := DepositTyped{
			StartTime: time.Unix(d.Timestamp.Int64(), 0),
			Amount:    sdk.NewCoin(bondDenom, math.NewIntFromBigInt(d.SumInWeiDeposited)),
			Paid:      sdk.NewCoin(bondDenom, math.NewIntFromBigInt(d.SumPaidAlready)),
			Rest:      sdk.NewCoin(bondDenom, math.NewIntFromBigInt(d.SumLeftToPay)),
		}
		addr := sdk.AccAddress(d.Owner.Bytes())
		depositsTyped[addr.String()] = append(depositsTyped[addr.String()], dt)
		r.ctx.Logger().Info(fmt.Sprintf(" - deposit for %s [%s]: %s", addr.String(), d.Owner.Hex(), dt.Amount.String()))
	}

	return depositsTyped, nil
}

func (r *RevestingUpgradeHandler) getDefaultVestingPeriods(totalAmount sdk.Coin) (sdkvesting.Periods, sdkvesting.Periods) {
	periodLength := cliffPeriod
	unlockAmount := sdk.NewCoin(totalAmount.Denom, totalAmount.Amount.QuoRaw(numberOfPeriods))
	restAmount := totalAmount

	lockupPeriods := make(sdkvesting.Periods, 0, numberOfPeriods)
	for i := 0; i < numberOfPeriods; i++ {
		if i == numberOfPeriods-1 {
			unlockAmount = restAmount
		}

		period := sdkvesting.Period{Length: periodLength, Amount: sdk.NewCoins(unlockAmount)}
		lockupPeriods = append(lockupPeriods, period)
		restAmount = restAmount.Sub(unlockAmount)

		if i == 0 {
			periodLength = unlockPeriod
		}
	}

	vestingPeriods := sdkvesting.Periods{
		sdkvesting.Period{Length: 0, Amount: sdk.NewCoins(totalAmount)},
	}

	return lockupPeriods, vestingPeriods
}

func (r *RevestingUpgradeHandler) getVestingContractABI() (abi string, fn string) {
	abi = "[{\"inputs\":[],\"name\":\"getAllDeposits\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sumInWeiDeposited\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sumPaidAlready\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"sumLeftToPay\",\"type\":\"uint256\"}],\"internalType\":\"struct HaqqVestingV3.ExtendedDeposit[]\",\"name\":\"allDeposits\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"
	fn = "getAllDeposits"

	return
}

func (r *RevestingUpgradeHandler) getVestingContractAddress() common.Address {
	return common.HexToAddress(vestingContract)
}

func (r *RevestingUpgradeHandler) forceVestingContractWithdraw(accounts []authtypes.AccountI, deposits map[string][]DepositTyped) map[string]sdk.Coin {
	r.ctx.Logger().Info("Force Vesting Contract Withdraw:")
	withdrawn := make(map[string]sdk.Coin)
	fails := 0
	contractAddress := r.getVestingContractAddress()
	contractAccount := sdk.AccAddress(contractAddress.Bytes())

	for _, acc := range accounts {
		evmAddr := common.BytesToAddress(acc.GetAddress().Bytes())
		vestingDeposits, ok := deposits[acc.GetAddress().String()]
		if !ok || len(vestingDeposits) == 0 {
			continue
		}

		vestingBalance := sdk.NewCoin(r.StakingKeeper.BondDenom(r.ctx), math.ZeroInt())
		for _, d := range vestingDeposits {
			vestingBalance = vestingBalance.Add(d.Rest)
		}

		if !vestingBalance.IsZero() {
			if err := r.BankKeeper.SendCoins(r.ctx, contractAccount, acc.GetAddress(), sdk.NewCoins(vestingBalance)); err != nil {
				fails++
				r.ctx.Logger().Error(fmt.Sprintf(" - failed withdrawal to %s [%s]: %s", acc.GetAddress().String(), evmAddr.Hex(), vestingBalance.String()))
				continue
			}

			withdrawn[acc.GetAddress().String()] = vestingBalance
			r.ctx.Logger().Info(fmt.Sprintf(" - withdrawn to %s: %s", acc.GetAddress().String(), vestingBalance.String()))
		}
	}

	switch {
	case fails > 0:
		r.ctx.Logger().Error(fmt.Sprintf("HAS ERRORS: %d fails", fails))
	default:
		r.ctx.Logger().Info("SUCCESS!")
	}

	return withdrawn
}

func (r *RevestingUpgradeHandler) checkVestingContractBalance() sdk.Coin {
	contractHexAddr := r.getVestingContractAddress()
	contractAddr := sdk.AccAddress(contractHexAddr.Bytes())
	balance := r.BankKeeper.GetBalance(r.ctx, contractAddr, r.StakingKeeper.GetParams(r.ctx).BondDenom)

	return balance
}

func (r *RevestingUpgradeHandler) implicitConvertIntoVestingAccount(acc authtypes.AccountI, totalVestingAmount sdk.Coin, deposits []DepositTyped) error {
	r.ctx.Logger().Info("Cpnvert into ClawbackVestingAccount:")

	moduleAcc := r.AccountKeeper.GetModuleAccount(r.ctx, vestingtypes.ModuleName)

	if len(deposits) == 0 {
		// should never happen
		return errors.New("empty deposits for account " + acc.GetAddress().String())
	}

	var (
		startTime                     time.Time
		lockupPeriods, vestingPeriods sdkvesting.Periods
		originalVesting               sdk.Coins
		totalAmount                   sdk.Coin
		totalVestingAmountCheck       sdk.Coin
	)
	for i, d := range deposits {
		if i == 0 {
			startTime = d.StartTime
			lockupPeriods, vestingPeriods = r.getContractVestingPeriods(d)
			originalVesting = sdk.NewCoins(d.Amount)
			totalAmount = d.Amount
			totalVestingAmountCheck = d.Rest
			continue
		}

		addLockupPeriods, addVestingPeriods := r.getContractVestingPeriods(d)
		newLockupStart, _, newLockupPeriods := vestingtypes.DisjunctPeriods(
			startTime.Unix(),
			d.StartTime.Unix(),
			lockupPeriods, addLockupPeriods)
		newVestingStart, _, newVestingPeriods := vestingtypes.DisjunctPeriods(
			startTime.Unix(),
			d.StartTime.Unix(),
			vestingPeriods, addVestingPeriods)

		if newLockupStart != newVestingStart {
			return fmt.Errorf(
				"vesting start time calculation should match lockup start (%d ≠ %d)",
				newVestingStart, newLockupStart,
			)
		}

		originalVesting = originalVesting.Add(d.Amount)
		lockupPeriods, vestingPeriods = newLockupPeriods, newVestingPeriods
		startTime = time.Unix(newLockupStart, 0)
		totalAmount = totalAmount.Add(d.Amount)
		totalVestingAmountCheck = totalVestingAmountCheck.Add(d.Rest)
	}

	if !totalVestingAmountCheck.IsGTE(totalVestingAmount) && !totalVestingAmount.IsGTE(totalVestingAmountCheck) {
		return fmt.Errorf(
			"vesting amounts don't match (%s ≠ %s)",
			totalVestingAmountCheck.String(), totalVestingAmount.String(),
		)
	}

	codeHash := common.BytesToHash(crypto.Keccak256(nil))
	baseAcc := authtypes.NewBaseAccountWithAddress(acc.GetAddress())
	vestingAcc := vestingtypes.NewClawbackVestingAccount(
		baseAcc,
		moduleAcc.GetAddress(),
		sdk.NewCoins(totalAmount),
		startTime,
		lockupPeriods,
		vestingPeriods,
		&codeHash,
	)

	// track delegation
	// how much is really delegated?
	bondedAmt := r.StakingKeeper.GetDelegatorBonded(r.ctx, vestingAcc.GetAddress())
	unbondingAmt := r.StakingKeeper.GetDelegatorUnbonding(r.ctx, vestingAcc.GetAddress())
	delegatedAmt := bondedAmt.Add(unbondingAmt)
	delegated := sdk.NewCoins(sdk.NewCoin(r.StakingKeeper.BondDenom(r.ctx), delegatedAmt))
	// cap DV at the current unvested amount, DF rounds out to current delegated
	unvested := vestingAcc.GetVestingCoins(r.ctx.BlockTime())
	vestingAcc.DelegatedVesting = delegated.Min(unvested)
	vestingAcc.DelegatedFree = delegated.Sub(vestingAcc.DelegatedVesting...)

	va := r.AccountKeeper.NewAccount(r.ctx, vestingAcc)
	r.AccountKeeper.SetAccount(r.ctx, va)

	locked := vestingAcc.GetLockedOnly(r.ctx.BlockTime())
	unvestedOnly := vestingAcc.GetUnvestedOnly(r.ctx.BlockTime())
	vested := vestingAcc.GetVestedOnly(r.ctx.BlockTime())

	r.ctx.Logger().Info(fmt.Sprintf(" - start time: %s", vestingAcc.StartTime.String()))
	r.ctx.Logger().Info(fmt.Sprintf(" - start time unix: %d", vestingAcc.StartTime.Unix()))
	r.ctx.Logger().Info(fmt.Sprintf(" - end time: %d", vestingAcc.EndTime))
	r.ctx.Logger().Info(fmt.Sprintf(" - lockup periods: %d", len(vestingAcc.LockupPeriods)))
	r.ctx.Logger().Info(fmt.Sprintf(" - vesting periods: %d", len(vestingAcc.VestingPeriods)))

	r.ctx.Logger().Info("Vesting account balances:")
	r.ctx.Logger().Info(fmt.Sprintf(" - locked: %s", locked.String()))
	r.ctx.Logger().Info(fmt.Sprintf(" - unvested: %s", unvestedOnly.String()))
	r.ctx.Logger().Info(fmt.Sprintf(" - vested: %s", vested.String()))

	return nil
}

func (r *RevestingUpgradeHandler) getContractVestingPeriods(depo DepositTyped) (sdkvesting.Periods, sdkvesting.Periods) {
	bondDenom := r.StakingKeeper.BondDenom(r.ctx)
	periodLength := int64(2592000) // 30 days in seconds
	periodsNumber := 24

	unlockAmount := sdk.NewCoin(bondDenom, depo.Amount.Amount.QuoRaw(int64(periodsNumber)))
	restAmount := depo.Amount

	lockupPeriods := make(sdkvesting.Periods, 0, periodsNumber)
	for i := 0; i < periodsNumber; i++ {
		if i == periodsNumber-1 {
			unlockAmount = restAmount
		}

		period := sdkvesting.Period{Length: periodLength, Amount: sdk.NewCoins(unlockAmount)}
		lockupPeriods = append(lockupPeriods, period)
		restAmount = restAmount.Sub(unlockAmount)

		if i == 0 {
			periodLength = unlockPeriod
		}
	}

	vestingPeriods := sdkvesting.Periods{
		sdkvesting.Period{Length: 0, Amount: sdk.NewCoins(depo.Amount)},
	}

	return lockupPeriods, vestingPeriods
}
