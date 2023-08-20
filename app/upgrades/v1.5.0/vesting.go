package v150

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/evmos/ethermint/server/config"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/pkg/errors"
)

func (r *RevestingUpgradeHandler) getVestingPeriods(totalAmount sdk.Coin) (sdkvesting.Periods, sdkvesting.Periods) {
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

func (r *RevestingUpgradeHandler) getVestingContractABI() (string, string) {
	return "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_beneficiaryAddress\",\"type\":\"address\"}],\"name\":\"calculateTotalRemainingForAllDeposits\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]", "calculateTotalRemainingForAllDeposits"
}

func (r *RevestingUpgradeHandler) getVestingContractAddress() common.Address {
	return common.HexToAddress(vestingContract)
}

func (r *RevestingUpgradeHandler) getVestingContractBalance(addr common.Address) (sdk.Coin, error) {
	bondDenom := r.StakingKeeper.BondDenom(r.ctx)
	totalVestingAmount := sdk.NewCoin(bondDenom, sdk.ZeroInt())

	contractAddress := r.getVestingContractAddress()
	contractABI, contractFunction := r.getVestingContractABI()

	if addr.Hex() == contractAddress.Hex() || addr.Hex() == vestingContractProxy {
		return totalVestingAmount, nil
	}

	// Parse the contract ABI
	cAbi, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return totalVestingAmount, errors.Wrap(err, "abi.JSON")
	}

	// Create a call data buffer for the function call
	callData, err := cAbi.Pack(contractFunction, addr)
	if err != nil {
		return totalVestingAmount, errors.Wrap(err, "abi.Pack")
	}

	args, err := json.Marshal(evmtypes.TransactionArgs{
		From: &addr,
		To:   &contractAddress,
		Data: (*hexutil.Bytes)(&callData),
	})
	if err != nil {
		return totalVestingAmount, errors.Wrap(err, "json.Marshal")
	}

	calReq := &evmtypes.EthCallRequest{
		Args:   args,
		GasCap: config.DefaultGasCap,
	}

	// Call smart-contract
	resp, err := r.EvmKeeper.EthCall(r.ctx, calReq)
	if err != nil {
		return totalVestingAmount, errors.Wrap(err, "evm.EthCall")
	}

	// Parse contract response
	var amount *big.Int
	if err := cAbi.UnpackIntoInterface(&amount, contractFunction, resp.Ret); err != nil {
		return totalVestingAmount, errors.Wrap(err, "abi.UnpackIntoInterface")
	}

	amt := math.NewIntFromBigInt(amount)

	if !amt.IsZero() {
		totalVestingAmount = sdk.NewCoin(bondDenom, amt)
	}

	return totalVestingAmount, nil
}

func (r *RevestingUpgradeHandler) forceContractVestingWithdraw(accounts []authtypes.AccountI) map[string]sdk.Coin {
	withdrawn := make(map[string]sdk.Coin)
	fails := 0

	for _, acc := range accounts {
		evmAddr := common.BytesToAddress(acc.GetAddress().Bytes())
		vestingBalance, err := r.getVestingContractBalance(evmAddr)
		if err != nil {
			fails++
			r.ctx.Logger().Error(
				"forceContractVestingWithdraw -> getVestingContractBalance: " + err.Error() + "! " +
					"Account: " + acc.GetAddress().String() + "[" + evmAddr.Hex() + "]",
			)
			continue
		}

		if !vestingBalance.IsZero() {
			contractAddress := r.getVestingContractAddress()
			contractAccount := sdk.AccAddress(contractAddress.Bytes())
			if err := r.BankKeeper.SendCoins(r.ctx, contractAccount, acc.GetAddress(), sdk.NewCoins(vestingBalance)); err != nil {
				fails++
				r.ctx.Logger().Error(
					"forceContractVestingWithdraw: failed to transfer tokens from vesting contract to account: " + err.Error() + "! " +
						"Account: " + acc.GetAddress().String() + "[" + evmAddr.Hex() + "], Amount: " + vestingBalance.String(),
				)
				continue
			}
			withdrawn[acc.GetAddress().String()] = vestingBalance
		}
	}

	switch {
	case fails > 0:
		r.ctx.Logger().Error(fmt.Sprintf("forceContractVestingWithdraw: %d fails", fails))
	default:
		r.ctx.Logger().Info("forceContractVestingWithdraw: SUCCESS!")
	}

	return withdrawn
}
