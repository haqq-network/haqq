package v150

import (
	"encoding/json"
	"math/big"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	unlockAmount := sdk.NewCoin(totalAmount.Denom, totalAmount.Amount.QuoRaw(24))
	restAmount := totalAmount

	lockupPeriods := make(sdkvesting.Periods, 0, 24)
	for i := 0; i < 24; i++ {
		if i == 23 {
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

func (r *RevestingUpgradeHandler) getVestingContractABI() string {
	return `[
            {
              "anonymous": false,
              "inputs": [
                {
                  "indexed": true,
                  "internalType": "address",
                  "name": "beneficiaryAddress",
                  "type": "address"
                },
                {
                  "indexed": true,
                  "internalType": "uint256",
                  "name": "depositId",
                  "type": "uint256"
                },
                {
                  "indexed": true,
                  "internalType": "uint256",
                  "name": "timestamp",
                  "type": "uint256"
                },
                {
                  "indexed": false,
                  "internalType": "uint256",
                  "name": "sumInWeiDeposited",
                  "type": "uint256"
                },
                {
                  "indexed": false,
                  "internalType": "address",
                  "name": "depositedBy",
                  "type": "address"
                }
              ],
              "name": "DepositMade",
              "type": "event"
            },
            {
              "anonymous": false,
              "inputs": [
                {
                  "indexed": false,
                  "internalType": "uint8",
                  "name": "version",
                  "type": "uint8"
                }
              ],
              "name": "Initialized",
              "type": "event"
            },
            {
              "anonymous": false,
              "inputs": [
                {
                  "indexed": true,
                  "internalType": "address",
                  "name": "beneficiary",
                  "type": "address"
                },
                {
                  "indexed": false,
                  "internalType": "uint256",
                  "name": "sumInWei",
                  "type": "uint256"
                },
                {
                  "indexed": true,
                  "internalType": "address",
                  "name": "triggeredByAddress",
                  "type": "address"
                }
              ],
              "name": "WithdrawalMade",
              "type": "event"
            },
            {
              "inputs": [],
              "name": "MAX_DEPOSITS",
              "outputs": [
                {
                  "internalType": "uint256",
                  "name": "",
                  "type": "uint256"
                }
              ],
              "stateMutability": "view",
              "type": "function"
            },
            {
              "inputs": [],
              "name": "NUMBER_OF_PAYMENTS",
              "outputs": [
                {
                  "internalType": "uint256",
                  "name": "",
                  "type": "uint256"
                }
              ],
              "stateMutability": "view",
              "type": "function"
            },
            {
              "inputs": [],
              "name": "TIME_BETWEEN_PAYMENTS",
              "outputs": [
                {
                  "internalType": "uint256",
                  "name": "",
                  "type": "uint256"
                }
              ],
              "stateMutability": "view",
              "type": "function"
            },
            {
              "inputs": [
                {
                  "internalType": "address",
                  "name": "_beneficiaryAddress",
                  "type": "address"
                },
                {
                  "internalType": "uint256",
                  "name": "_depositId",
                  "type": "uint256"
                }
              ],
              "name": "amountForOneWithdrawal",
              "outputs": [
                {
                  "internalType": "uint256",
                  "name": "",
                  "type": "uint256"
                }
              ],
              "stateMutability": "view",
              "type": "function"
            },
            {
              "inputs": [
                {
                  "internalType": "address",
                  "name": "_beneficiaryAddress",
                  "type": "address"
                },
                {
                  "internalType": "uint256",
                  "name": "_depositId",
                  "type": "uint256"
                }
              ],
              "name": "amountToWithdrawNow",
              "outputs": [
                {
                  "internalType": "uint256",
                  "name": "",
                  "type": "uint256"
                }
              ],
              "stateMutability": "view",
              "type": "function"
            },
            {
              "inputs": [
                {
                  "internalType": "address",
                  "name": "_beneficiaryAddress",
                  "type": "address"
                }
              ],
              "name": "calculateAvailableSumForAllDeposits",
              "outputs": [
                {
                  "internalType": "uint256",
                  "name": "",
                  "type": "uint256"
                }
              ],
              "stateMutability": "view",
              "type": "function"
            },
            {
              "inputs": [
                {
                  "internalType": "address",
                  "name": "_beneficiaryAddress",
                  "type": "address"
                }
              ],
              "name": "deposit",
              "outputs": [
                {
                  "internalType": "bool",
                  "name": "success",
                  "type": "bool"
                }
              ],
              "stateMutability": "payable",
              "type": "function"
            },
            {
              "inputs": [
                {
                  "internalType": "address",
                  "name": "",
                  "type": "address"
                },
                {
                  "internalType": "uint256",
                  "name": "",
                  "type": "uint256"
                }
              ],
              "name": "deposits",
              "outputs": [
                {
                  "internalType": "uint256",
                  "name": "timestamp",
                  "type": "uint256"
                },
                {
                  "internalType": "uint256",
                  "name": "sumInWeiDeposited",
                  "type": "uint256"
                },
                {
                  "internalType": "uint256",
                  "name": "sumPaidAlready",
                  "type": "uint256"
                }
              ],
              "stateMutability": "view",
              "type": "function"
            },
            {
              "inputs": [
                {
                  "internalType": "address",
                  "name": "",
                  "type": "address"
                }
              ],
              "name": "depositsCounter",
              "outputs": [
                {
                  "internalType": "uint256",
                  "name": "",
                  "type": "uint256"
                }
              ],
              "stateMutability": "view",
              "type": "function"
            },
            {
              "inputs": [
                {
                  "internalType": "address",
                  "name": "_beneficiaryAddress",
                  "type": "address"
                },
                {
                  "internalType": "uint256",
                  "name": "_depositId",
                  "type": "uint256"
                }
              ],
              "name": "totalPayoutsUnblocked",
              "outputs": [
                {
                  "internalType": "uint256",
                  "name": "",
                  "type": "uint256"
                }
              ],
              "stateMutability": "view",
              "type": "function"
            },
            {
              "inputs": [
                {
                  "internalType": "address",
                  "name": "_newBeneficiaryAddress",
                  "type": "address"
                }
              ],
              "name": "transferDepositRights",
              "outputs": [],
              "stateMutability": "nonpayable",
              "type": "function"
            },
            {
              "inputs": [
                {
                  "internalType": "address",
                  "name": "_beneficiaryAddress",
                  "type": "address"
                }
              ],
              "name": "withdraw",
              "outputs": [
                {
                  "internalType": "bool",
                  "name": "success",
                  "type": "bool"
                }
              ],
              "stateMutability": "nonpayable",
              "type": "function"
            }
          ]`
}

func (r *RevestingUpgradeHandler) getVestingContractAddress() common.Address {
	return common.HexToAddress(vestingContract)
}

func (r *RevestingUpgradeHandler) getVestingContractBalance(addr common.Address) (sdk.Coin, error) {
	bondDenom := r.StakingKeeper.BondDenom(r.ctx)
	totalVestingAmount := sdk.NewCoin(bondDenom, sdk.ZeroInt())

	contractAddress := r.getVestingContractAddress()
	contractABI := r.getVestingContractABI()

	// Parse the contract ABI
	cAbi, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return totalVestingAmount, errors.Wrap(err, "abi.JSON")
	}

	// Create a call data buffer for the function call
	// Use "calculateTotalRemainingForAllDeposits" on release
	// callData, err := cAbi.Pack("calculateTotalRemainingForAllDeposits", addr)
	// TODO SET CORRECT FUNCTION NAME
	callData, err := cAbi.Pack("calculateAvailableSumForAllDeposits", addr)
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
	// Use "calculateTotalRemainingForAllDeposits" on release
	// if err := cAbi.UnpackIntoInterface(&amount, "calculateTotalRemainingForAllDeposits", resp.Ret); err != nil {
	// TODO SET CORRECT FUNCTION NAME
	if err := cAbi.UnpackIntoInterface(&amount, "calculateAvailableSumForAllDeposits", resp.Ret); err != nil {
		return totalVestingAmount, errors.Wrap(err, "abi.UnpackIntoInterface")
	}

	amt := math.NewIntFromBigInt(amount)

	if !amt.IsZero() {
		totalVestingAmount = sdk.NewCoin(bondDenom, amt)
	}

	return totalVestingAmount, nil
}
