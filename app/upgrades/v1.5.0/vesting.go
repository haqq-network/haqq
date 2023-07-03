package v150

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/common"
)

func (r *RevestingUpgradeHandler) getVestingPeriods(totalAmount sdk.Coin) (sdkvesting.Periods, sdkvesting.Periods) {
	periodLength := int64(2592000) // 30 days in seconds
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
	}

	vestingPeriods := sdkvesting.Periods{
		sdkvesting.Period{Length: 0, Amount: sdk.NewCoins(totalAmount)},
	}

	return lockupPeriods, vestingPeriods
}

func (r *RevestingUpgradeHandler) getZeroVestingPeriods() (sdkvesting.Periods, sdkvesting.Periods) {
	lockupPeriods := sdkvesting.Periods{
		sdkvesting.Period{Length: 0, Amount: sdk.NewCoins(sdk.NewCoin(r.StakingKeeper.BondDenom(r.ctx), sdk.ZeroInt()))},
	}
	vestingPeriods := sdkvesting.Periods{
		sdkvesting.Period{Length: 0, Amount: sdk.NewCoins(sdk.NewCoin(r.StakingKeeper.BondDenom(r.ctx), sdk.ZeroInt()))},
	}

	return lockupPeriods, vestingPeriods
}

func (r *RevestingUpgradeHandler) getVestingSmartContract() (common.Address, string) {
	contract := common.HexToAddress("0x40a3e24b85D32f3f68Ee9e126B8dD9dBC2D301Eb")
	contractAbi := `[
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

	return contract, contractAbi
}
