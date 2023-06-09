package v1_5_0

import (
	"cosmossdk.io/math"
	"encoding/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/evmos/ethermint/server/config"
	"github.com/evmos/ethermint/types"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	vestingkeeper "github.com/haqq-network/haqq/x/vesting/keeper"
	vestingtypes "github.com/haqq-network/haqq/x/vesting/types"
	"github.com/pkg/errors"
	"math/big"
	"strconv"
	"strings"
)

type RevestingUpgradeHandler struct {
	ctx           sdk.Context
	AccountKeeper authkeeper.AccountKeeper
	BankKeeper    bankkeeper.Keeper
	StakingKeeper stakingkeeper.Keeper
	EvmKeeper     *evmkeeper.Keeper
	VestingKeeper vestingkeeper.Keeper
	vals          map[*sdk.ValAddress]math.Int
}

func NewRevestingUpgradeHandler(
	ctx sdk.Context,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	sk stakingkeeper.Keeper,
	evm *evmkeeper.Keeper,
	vk vestingkeeper.Keeper,
) *RevestingUpgradeHandler {
	return &RevestingUpgradeHandler{
		ctx:           ctx,
		AccountKeeper: ak,
		BankKeeper:    bk,
		StakingKeeper: sk,
		EvmKeeper:     evm,
		VestingKeeper: vk,
		vals:          make(map[*sdk.ValAddress]math.Int),
	}
}

func (r *RevestingUpgradeHandler) Run() error {
	accounts := r.AccountKeeper.GetAllAccounts(r.ctx)
	if len(accounts) == 0 {
		// Short circuit if there are no accounts
		r.ctx.Logger().Info("No accounts found")
		return nil
	}

	// Store validators and their bonded tokens for further processing
	validators := r.StakingKeeper.GetBondedValidatorsByPower(r.ctx)
	if len(validators) > 50 {
		validators = validators[:50]
	}

	if len(validators) == 0 {
		return errors.New("no bonded validators found")
	}

	for _, validator := range validators {
		op := validator.GetOperator()
		r.vals[&op] = validator.GetTokens()
	}

	r.ctx.Logger().Info("Total validators before upgrade: " + strconv.Itoa(len(r.vals)))

	for _, acc := range accounts {
		r.ctx.Logger().Info("---")
		r.ctx.Logger().Info("Account: " + acc.GetAddress().String())

		evmAddr := common.BytesToAddress(acc.GetAddress().Bytes())
		r.ctx.Logger().Info("EVM Account: " + evmAddr.Hex())

		if r.isAccountWhitelisted(acc.GetAddress().String()) {
			r.ctx.Logger().Info("WHITELISTED — skip")
			continue
		}

		// Check if account is a ETH account
		if _, ok := acc.(*types.EthAccount); !ok {
			r.ctx.Logger().Info("Not a ETH Account — skip")
			continue
		}

		evmAcc := r.EvmKeeper.GetAccountWithoutBalance(r.ctx, evmAddr)
		if evmAcc != nil && evmAcc.IsContract() {
			r.ctx.Logger().Info("CONTRACT — skip")
			continue
		}

		// TODO Remove before release
		// Log balance before Undelegate
		balanceBeforeUND := r.BankKeeper.GetBalance(r.ctx, acc.GetAddress(), r.StakingKeeper.BondDenom(r.ctx))
		r.ctx.Logger().Info("Balance before undelegation: " + balanceBeforeUND.String())

		// Undelegate all coins for account
		oldDelegations, totalUndelegatedAmount, err := r.UndelegateAllTokens(acc.GetAddress())
		if err != nil {
			return errors.Wrap(err, "error undelegating tokens")
		}
		r.ctx.Logger().Info("Total undelegated amount: " + totalUndelegatedAmount.String())

		// TODO Remove before release
		// Log balance before upgrade
		balanceAfterUND := r.BankKeeper.GetBalance(r.ctx, acc.GetAddress(), r.StakingKeeper.BondDenom(r.ctx))
		r.ctx.Logger().Info("Balance after undelegation: " + balanceAfterUND.String())

		vestedAmount, err := r.WithdrawCoinsFromVestingContract(evmAddr)
		if err != nil {
			return errors.Wrap(err, "error withdrawing coins from vesting contract")
		}
		r.ctx.Logger().Info("Total vested amount: " + vestedAmount.String())

		// Get account balance
		balance := r.BankKeeper.GetBalance(r.ctx, acc.GetAddress(), r.StakingKeeper.BondDenom(r.ctx))
		r.ctx.Logger().Info("Balance before revesting: " + balance.String())

		// Send all coins to the vesting module account and process revesting with staking
		if err := r.Revesting(acc, balance); err != nil {
			return errors.Wrap(err, "error revesting")
		}

		// TODO Remove before release
		// Log balance after revesting
		// --------------------------

		shares, err := r.Restaking(acc, balance, oldDelegations)
		if err != nil {
			return errors.Wrap(err, "error restaking")
		}

		r.ctx.Logger().Info("New staking shares:")
		for valAddr, share := range shares {
			r.ctx.Logger().Info(share.String() + " to " + valAddr.String())
		}

		// TODO Remove before release
		// Log balance before upgrade
		balanceAfter := r.BankKeeper.GetBalance(r.ctx, acc.GetAddress(), r.StakingKeeper.BondDenom(r.ctx))
		r.ctx.Logger().Info("Balance after (re)staking: " + balanceAfter.String())
	}

	return nil
}

func (r *RevestingUpgradeHandler) isAccountWhitelisted(addr string) bool {
	whitelist := map[string]bool{
		//"haqq196srgtdaqrhqehdx36hfacrwmhlfznwpt78rct": true, // random address
	}

	_, ok := whitelist[addr]
	return ok
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

// UndelegateAllTokens undelegates all tokens from the all validators and returns the undelegated amount per validator
// and the total undelegated amount
func (r *RevestingUpgradeHandler) UndelegateAllTokens(delAddr sdk.AccAddress) (map[*sdk.ValAddress]sdk.Coin, sdk.Coin, error) {
	bondDenom := r.StakingKeeper.BondDenom(r.ctx)
	totalUndelegatedAmount := sdk.NewCoin(bondDenom, sdk.NewInt(0))

	delegations := r.StakingKeeper.GetAllDelegatorDelegations(r.ctx, delAddr)
	if len(delegations) == 0 {
		return nil, totalUndelegatedAmount, nil
	}

	// unbond from all validators
	undelegatedAmounts := make(map[*sdk.ValAddress]sdk.Coin, len(delegations))
	for _, delegation := range delegations {
		valAddr, _ := sdk.ValAddressFromBech32(delegation.GetValidatorAddr().String())
		ubdAmount, err := r.StakingKeeper.Unbond(r.ctx, delAddr, valAddr, delegation.GetShares())
		if err != nil {
			return nil, totalUndelegatedAmount, errors.Wrap(err, "failed to unbond tokens")
		}

		validator, found := r.StakingKeeper.GetValidator(r.ctx, valAddr)
		if !found {
			// Impossible as we are iterating over active delegations
			return nil, totalUndelegatedAmount, errors.New("validator not found")
		}

		// transfer the validator tokens to the not bonded pool
		coins := sdk.NewCoins(sdk.NewCoin(bondDenom, ubdAmount))
		if validator.IsBonded() {
			if err := r.BankKeeper.SendCoinsFromModuleToModule(r.ctx, stakingtypes.BondedPoolName, stakingtypes.NotBondedPoolName, coins); err != nil {
				return nil, totalUndelegatedAmount, errors.Wrap(err, "failed to transfer tokens from bonded to not bonded pool")
			}

			if err := r.BankKeeper.UndelegateCoinsFromModuleToAccount(r.ctx, stakingtypes.NotBondedPoolName, delAddr, coins); err != nil {
				return nil, totalUndelegatedAmount, errors.Wrap(err, "failed to transfer tokens from not bonded pool to delegator's address")
			}
		} else {
			// Should not happen
			r.ctx.Logger().Info("Validator is not bonded...")
		}

		undelegatedCoin := sdk.NewCoin(bondDenom, ubdAmount)
		undelegatedAmounts[&valAddr] = sdk.NewCoin(bondDenom, ubdAmount)
		totalUndelegatedAmount = totalUndelegatedAmount.Add(undelegatedCoin)
	}

	return undelegatedAmounts, totalUndelegatedAmount, nil
}

func (r *RevestingUpgradeHandler) WithdrawCoinsFromVestingContract(addr common.Address) (sdk.Coin, error) {
	bondDenom := r.StakingKeeper.BondDenom(r.ctx)
	totalVestingAmount := sdk.NewCoin(bondDenom, sdk.NewInt(0))

	contractAddress, contractABI := r.getVestingSmartContract()

	// Parse the contract ABI
	cAbi, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return totalVestingAmount, errors.Wrap(err, "abi.JSON")
	}

	// Create a call data buffer for the function call
	// Use "calculateTotalRemainingForAllDeposits" on release
	//callData, err := cAbi.Pack("calculateTotalRemainingForAllDeposits", addr)
	callData, err := cAbi.Pack("calculateAvailableSumForAllDeposits", addr)
	if err != nil {
		return totalVestingAmount, errors.Wrap(err, "abi.Pack")
	}

	args, err := json.Marshal(evmtypes.TransactionArgs{
		From: &addr,
		To:   &contractAddress,
		Data: (*hexutil.Bytes)(&callData),
	})

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
	//if err := cAbi.UnpackIntoInterface(&amount, "calculateTotalRemainingForAllDeposits", resp.Ret); err != nil {
	if err := cAbi.UnpackIntoInterface(&amount, "calculateAvailableSumForAllDeposits", resp.Ret); err != nil {
		return totalVestingAmount, errors.Wrap(err, "abi.UnpackIntoInterface")
	}

	amt := math.NewIntFromBigInt(amount)
	// Transfer the tokens from the vesting contract to the account
	if !amt.IsZero() {
		totalVestingAmount.Amount = amt
		contractAccount := sdk.AccAddress(contractAddress.Bytes())
		acc := sdk.AccAddress(addr.Bytes())
		if err := r.BankKeeper.SendCoins(r.ctx, contractAccount, acc, sdk.NewCoins(totalVestingAmount)); err != nil {
			return totalVestingAmount, errors.Wrap(err, "failed to transfer tokens from vesting contract to account")
		}
	}

	return totalVestingAmount, nil
}

func (r *RevestingUpgradeHandler) Revesting(acc authtypes.AccountI, coin sdk.Coin) error {
	moduleAcc := r.AccountKeeper.GetModuleAccount(r.ctx, vestingtypes.ModuleName)
	if err := r.BankKeeper.SendCoinsFromAccountToModule(
		r.ctx,
		acc.GetAddress(),
		vestingtypes.ModuleName,
		sdk.NewCoins(coin),
	); err != nil {
		return errors.Wrap(err, "failed to send coins to vesting module")
	}

	// Convert to empty vesting account
	zeroLockingPeriods, zeroVestingPeriods := r.getZeroVestingPeriods()
	baseAcc := authtypes.NewBaseAccountWithAddress(acc.GetAddress())
	vestingAcc := vestingtypes.NewClawbackVestingAccount(
		baseAcc,
		moduleAcc.GetAddress(),
		sdk.NewCoins(sdk.NewCoin(coin.Denom, sdk.ZeroInt())),
		r.ctx.BlockTime(),
		zeroLockingPeriods,
		zeroVestingPeriods,
	)
	newAcc := r.AccountKeeper.NewAccount(r.ctx, vestingAcc)
	r.AccountKeeper.SetAccount(r.ctx, newAcc)

	// Create a new vesting by merge
	lockupPeriods, vestingPeriods := r.getVestingPeriods(coin)
	msg := vestingtypes.NewMsgCreateClawbackVestingAccount(
		moduleAcc.GetAddress(),
		acc.GetAddress(),
		r.ctx.BlockTime(),
		lockupPeriods,
		vestingPeriods,
		true,
	)
	if err := msg.ValidateBasic(); err != nil {
		return errors.Wrap(err, "failed to validate msg")
	}

	_, err := r.VestingKeeper.CreateClawbackVestingAccount(r.ctx, msg)
	if err != nil {
		return errors.Wrap(err, "failed to create clawback vesting account")
	}

	return nil
}

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
		sdkvesting.Period{Length: 1, Amount: sdk.NewCoins(totalAmount)},
	}

	return lockupPeriods, vestingPeriods
}

func (r *RevestingUpgradeHandler) getZeroVestingPeriods() (sdkvesting.Periods, sdkvesting.Periods) {
	lockupPeriods := sdkvesting.Periods{
		sdkvesting.Period{Length: 1, Amount: sdk.NewCoins(sdk.NewCoin(r.StakingKeeper.BondDenom(r.ctx), sdk.ZeroInt()))},
	}
	vestingPeriods := sdkvesting.Periods{
		sdkvesting.Period{Length: 1, Amount: sdk.NewCoins(sdk.NewCoin(r.StakingKeeper.BondDenom(r.ctx), sdk.ZeroInt()))},
	}

	return lockupPeriods, vestingPeriods
}

func (r *RevestingUpgradeHandler) Restaking(acc authtypes.AccountI, totalAmount sdk.Coin, oldDelegations map[*sdk.ValAddress]sdk.Coin) (map[*sdk.ValAddress]sdk.Dec, error) {
	shares := make(map[*sdk.ValAddress]sdk.Dec)
	restAmount := totalAmount
	if len(oldDelegations) > 0 {
		r.ctx.Logger().Info("found old delegations, restaking...")
		for valAddr, amt := range oldDelegations {
			val, found := r.StakingKeeper.GetValidator(r.ctx, valAddr.Bytes())
			if !found {
				// Should never happen, but just in case
				return map[*sdk.ValAddress]sdk.Dec{}, errors.Wrapf(stakingtypes.ErrNoValidatorFound, "validator %s does not exist", valAddr)
			}

			newShares, err := r.StakingKeeper.Delegate(r.ctx, acc.GetAddress(), amt.Amount, stakingtypes.Unbonded, val, true)
			if err != nil {
				return map[*sdk.ValAddress]sdk.Dec{}, errors.Wrap(err, "failed to delegate")
			}

			r.ctx.Logger().Info("restaked " + amt.String() + " to " + valAddr.String())

			restAmount = restAmount.Sub(amt)
			shares[valAddr] = newShares
		}
	}

	// 1ISL deductable, leave it for future fees
	oneISLM := sdk.NewCoin(r.StakingKeeper.BondDenom(r.ctx), sdk.NewInt(1000000000000000000))
	if oneISLM.IsGTE(restAmount) {
		r.ctx.Logger().Info("too small balance to stake: " + restAmount.String())
		return shares, nil
	}
	restAmount = restAmount.Sub(oneISLM)

	val, valAddr := r.getWeakestValidator()
	restShares, err := r.StakingKeeper.Delegate(r.ctx, acc.GetAddress(), restAmount.Amount, stakingtypes.Unbonded, val, true)
	if err != nil {
		return map[*sdk.ValAddress]sdk.Dec{}, errors.Wrap(err, "failed to delegate")
	}
	shares[valAddr] = restShares

	// Add power to validator
	r.vals[valAddr] = r.vals[valAddr].Add(restAmount.Amount)

	return shares, nil
}

func (r *RevestingUpgradeHandler) getWeakestValidator() (stakingtypes.Validator, *sdk.ValAddress) {
	var weakestAddr *sdk.ValAddress
	for valAddr, _ := range r.vals {
		if weakestAddr == nil {
			weakestAddr = valAddr
			continue
		}

		if r.vals[valAddr].LT(r.vals[weakestAddr]) {
			weakestAddr = valAddr
		}
	}

	val, found := r.StakingKeeper.GetValidator(r.ctx, weakestAddr.Bytes())
	if !found {
		// Should never happen, but just in case
		panic("validator not found")
	}

	return val, weakestAddr
}
