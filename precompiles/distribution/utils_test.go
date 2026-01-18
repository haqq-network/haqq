package distribution_test

import (
	"cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/haqq-network/haqq/precompiles/staking"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	stakingkeeper "github.com/haqq-network/haqq/x/staking/keeper"
)

type stakingRewards struct {
	Delegator sdk.AccAddress
	Validator stakingtypes.Validator
	RewardAmt math.Int
}

var (
	testRewardsAmt, _       = math.NewIntFromString("1000000000000000000")
	validatorCommPercentage = math.LegacyNewDecWithPrec(5, 2) // 5% commission
	validatorCommAmt        = math.LegacyNewDecFromInt(testRewardsAmt).Mul(validatorCommPercentage).TruncateInt()
	expRewardsAmt           = testRewardsAmt.Sub(validatorCommAmt) // testRewardsAmt - commission
)

// prepareStakingRewards prepares the test suite for testing delegation rewards.
//
// Specified rewards amount are allocated to the specified validator using the distribution keeper,
// such that the given amount of tokens is outstanding as a staking reward for the account.
//
// The setup is done in the following way:
//   - Fund distribution module to pay for rewards.
//   - Allocate rewards to the validator.
func (s *PrecompileTestSuite) prepareStakingRewards(ctx sdk.Context, stkRs ...stakingRewards) (sdk.Context, error) {
	for _, r := range stkRs {
		// set distribution module account balance which pays out the rewards
		coins := sdk.NewCoins(sdk.NewCoin(s.bondDenom, r.RewardAmt))
		if err := s.mintCoinsForDistrMod(ctx, coins); err != nil {
			return ctx, err
		}

		// allocate rewards to validator
		allocatedRewards := sdk.NewDecCoins(sdk.NewDecCoin(s.bondDenom, r.RewardAmt))
		if err := s.network.App.DistrKeeper.AllocateTokensToValidator(ctx, r.Validator, allocatedRewards); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

// mintCoinsForDistrMod is a helper function to mint a specific amount of coins from the
// distribution module to pay for staking rewards.
func (s *PrecompileTestSuite) mintCoinsForDistrMod(ctx sdk.Context, amount sdk.Coins) error {
	// Minting tokens for the FeeCollector to simulate fee accrued.
	if err := s.network.App.BankKeeper.MintCoins(
		ctx,
		coinomicstypes.ModuleName,
		amount,
	); err != nil {
		return err
	}

	return s.network.App.BankKeeper.SendCoinsFromModuleToModule(
		ctx,
		coinomicstypes.ModuleName,
		distrtypes.ModuleName,
		amount,
	)
}

// fundAccountWithBaseDenom is a helper function to fund a given address with the chain's
// base denomination.
func (s *PrecompileTestSuite) fundAccountWithBaseDenom(ctx sdk.Context, addr sdk.AccAddress, amount math.Int) error {
	coins := sdk.NewCoins(sdk.NewCoin(s.bondDenom, amount))
	if err := s.network.App.BankKeeper.MintCoins(ctx, coinomicstypes.ModuleName, coins); err != nil {
		return err
	}
	return s.network.App.BankKeeper.SendCoinsFromModuleToAccount(ctx, coinomicstypes.ModuleName, addr, coins)
}

func (s *PrecompileTestSuite) getStakingPrecompile() (*staking.Precompile, error) {
	return staking.NewPrecompile(
		s.network.App.StakingKeeper,
		s.network.App.AuthzKeeper,
	)
}

func generateKeys(count int) []keyring.Key {
	accs := make([]keyring.Key, 0, count)
	for i := 0; i < count; i++ {
		acc := keyring.NewKey()
		accs = append(accs, acc)
	}
	return accs
}

// setValidatorCommission sets the minimum commission rate to the given value (if needed)
// and updates the validator's commission rate using Cosmos SDK message server.
// commissionRate should be a value between 0 and 1e18 (0% to 100% with 18 decimals precision).
func (s *PrecompileTestSuite) setValidatorCommission(validatorOperatorAddr string, validatorPrivKey cryptotypes.PrivKey, commissionRate math.Int) error {
	ctx := s.network.GetContext()

	// Set minimum commission rate to 0% if setting commission to 0% to allow it
	if commissionRate.IsZero() {
		stakingParams, err := s.network.App.StakingKeeper.GetParams(ctx)
		if err != nil {
			return err
		}
		stakingParams.MinCommissionRate = math.LegacyZeroDec()
		if err := s.network.App.StakingKeeper.SetParams(ctx, stakingParams); err != nil {
			return err
		}
		if err := s.network.NextBlock(); err != nil {
			return err
		}
		ctx = s.network.GetContext()
	}

	// Convert commission rate from math.Int to math.LegacyDec
	// commissionRate is in range 0-1e18 (representing 0% to 100% with 18 decimals precision)
	// Divide by 1e18 to get decimal value (0.0 to 1.0)
	oneE18 := math.NewInt(1e18)
	commissionRateDec := math.LegacyNewDecFromInt(commissionRate).QuoInt(oneE18)

	// Create MsgEditValidator and execute via Cosmos SDK message server
	// Use "[do-not-modify]" for all description fields to indicate we don't want to change them
	msg := stakingtypes.NewMsgEditValidator(
		validatorOperatorAddr,
		stakingtypes.Description{
			Moniker:         "[do-not-modify]",
			Identity:        "[do-not-modify]",
			Website:         "[do-not-modify]",
			SecurityContact: "[do-not-modify]",
			Details:         "[do-not-modify]",
		},
		&commissionRateDec,
		nil, // do not modify min self delegation
	)

	msgServer := stakingkeeper.NewMsgServerImpl(&s.network.App.StakingKeeper)
	_, err := msgServer.EditValidator(ctx, msg)
	if err != nil {
		return err
	}

	return s.network.NextBlock()
}
