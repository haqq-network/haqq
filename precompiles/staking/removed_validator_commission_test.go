package staking_test

import (
	"math/big"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/staking"
	commonfactory "github.com/haqq-network/haqq/testutil/integration/common/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/utils"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// removableValidator is an unbonded validator whose last shares the chaos fixture
// holds and which has commission accumulated, so that taking those shares removes
// it and pays the commission out to the operator's withdraw address.
type removableValidator struct {
	env        *chaosEnv
	operator   testkeyring.Key
	valAddr    sdk.ValAddress
	validator  string
	stake      *big.Int
	commission math.Int
}

func (s *PrecompileTestSuite) setupRemovableValidator() removableValidator {
	env := s.newChaosEnv(math.NewInt(5e18))
	bank := s.network.App.BankKeeper

	operator := s.keyring.GetKey(1)
	valAddr := sdk.ValAddress(operator.AccAddr)

	// Keep the new validator out of the active set: with less stake than one unit
	// of consensus power it is never bonded, and an unbonded validator is what
	// x/staking removes once its last delegation is gone.
	selfDelegation := math.NewInt(1e17)
	s.Require().NoError(s.factory.CreateValidator(
		operator.Priv,
		ed25519.GenPrivKey().PubKey(),
		sdk.NewCoin(s.bondDenom, selfDelegation),
		stakingtypes.NewDescription("removed", "", "", "", ""),
		stakingtypes.NewCommissionRates(
			math.LegacyNewDecWithPrec(1, 1),
			math.LegacyNewDecWithPrec(2, 1),
			math.LegacyNewDecWithPrec(1, 2),
		),
		math.OneInt(),
	))
	s.Require().NoError(s.network.NextBlock())

	validator, err := s.network.App.StakingKeeper.GetValidator(s.network.GetContext(), valAddr)
	s.Require().NoError(err)
	s.Require().True(validator.IsUnbonded(), "the validator must stay out of the active set")

	// The fixture delegates its own coins, so it needs no grant.
	stake := big.NewInt(5e17)
	res, err := s.factory.ExecuteContractCall(
		s.keyring.GetPrivKey(0),
		evmTxArgs(env.chaos),
		chaosRun(env, []chaosStep{call(env.staking, common.Big0, env.pack(env.staking, staking.DelegateMethod,
			env.chaos, validator.OperatorAddress, stake), 0)}),
	)
	s.Require().NoError(err)
	s.Require().True(res.IsOK(), "delegation from the fixture failed: %s", res.Log)
	s.Require().NoError(s.network.NextBlock())

	// The operator leaves, so the fixture holds the validator's last shares.
	undelegateRes, err := s.factory.CommitCosmosTx(operator.Priv, commonfactory.CosmosTxArgs{
		Msgs: []sdk.Msg{stakingtypes.NewMsgUndelegate(
			operator.AccAddr.String(), validator.OperatorAddress, sdk.NewCoin(s.bondDenom, selfDelegation),
		)},
	})
	s.Require().NoError(err)
	s.Require().True(undelegateRes.IsOK(), "operator undelegation failed: %s", undelegateRes.Log)
	s.Require().NoError(s.network.NextBlock())

	// Accrue commission the removal will pay out, backed by the distribution module.
	ctx := s.network.GetContext()
	commission := math.NewInt(3e17)
	decCommission := sdk.NewDecCoinsFromCoins(sdk.NewCoin(utils.BaseDenom, commission))
	outstanding, err := s.network.App.DistrKeeper.GetValidatorOutstandingRewards(ctx, valAddr)
	s.Require().NoError(err)
	s.Require().NoError(s.network.App.DistrKeeper.SetValidatorOutstandingRewards(
		ctx, valAddr, distrtypes.ValidatorOutstandingRewards{Rewards: outstanding.Rewards.Add(decCommission...)},
	))
	s.Require().NoError(s.network.App.DistrKeeper.SetValidatorAccumulatedCommission(
		ctx, valAddr, distrtypes.ValidatorAccumulatedCommission{Commission: decCommission},
	))
	coins := sdk.NewCoins(sdk.NewCoin(utils.BaseDenom, commission))
	s.Require().NoError(bank.MintCoins(ctx, coinomicstypes.ModuleName, coins))
	s.Require().NoError(bank.SendCoinsFromModuleToModule(ctx, coinomicstypes.ModuleName, distrtypes.ModuleName, coins))
	s.Require().NoError(s.network.NextBlock())

	return removableValidator{
		env:        env,
		operator:   operator,
		valAddr:    valAddr,
		validator:  validator.OperatorAddress,
		stake:      stake,
		commission: commission,
	}
}

// takeLastSharesAndCheckCommission dirties the operator's account with a 1 wei
// transfer, runs the call that takes the validator's last shares in the same
// transaction, and checks that the commission the removal pays the operator
// survives the commit.
func (s *PrecompileTestSuite) takeLastSharesAndCheckCommission(rv removableValidator, takeShares []byte) {
	bank := s.network.App.BankKeeper
	ctx := s.network.GetContext()
	operatorBefore := bank.GetBalance(ctx, rv.operator.AccAddr, utils.BaseDenom).Amount
	supplyBefore := s.bankSupply()

	res, err := s.factory.ExecuteContractCall(
		s.keyring.GetPrivKey(0),
		evmTxArgs(rv.env.chaos),
		chaosRun(rv.env, []chaosStep{
			send(rv.operator.Addr, big.NewInt(1)),
			call(rv.env.staking, common.Big0, takeShares, 0),
		}),
	)
	s.Require().NoError(err)
	s.Require().True(res.IsOK(), "taking the last shares from the fixture failed: %s", res.Log)
	s.Require().NoError(s.network.NextBlock())

	ctx = s.network.GetContext()
	_, err = s.network.App.StakingKeeper.GetValidator(ctx, rv.valAddr)
	s.Require().ErrorIs(err, stakingtypes.ErrNoValidatorFound,
		"the call must remove the validator, otherwise no commission is paid out and this test proves nothing")

	operatorAfter := bank.GetBalance(ctx, rv.operator.AccAddr, utils.BaseDenom).Amount
	s.Require().Equal(
		operatorBefore.Add(math.OneInt()).Add(rv.commission).String(),
		operatorAfter.String(),
		"the operator's commission payout was not mirrored into the StateDB journal",
	)
	s.Require().Equal(supplyBefore.String(), s.bankSupply().String(), "aISLM supply moved")
	s.Require().NoError(s.network.CheckAccountingInvariants())
}

// TestUndelegateRemovingValidatorMirrorsOperatorCommission guards the balance
// mirror against a credit the staking precompile does not make itself.
//
// When an undelegation takes the last shares of an unbonded validator, x/staking
// removes the validator, and the distribution hook AfterValidatorRemoved pays the
// validator's accumulated commission to the operator's withdraw address. That
// account is neither the delegator nor the delegator's withdrawer. Unless the
// mirror snapshots it too, the payout stays out of the StateDB journal, and if the
// operator's account is dirty in the same transaction - a 1 wei transfer to it is
// enough - Commit writes back the pre-payout balance and burns the commission.
func (s *PrecompileTestSuite) TestUndelegateRemovingValidatorMirrorsOperatorCommission() {
	s.SetupTest()
	rv := s.setupRemovableValidator()
	s.takeLastSharesAndCheckCommission(rv, rv.env.pack(rv.env.staking, staking.UndelegateMethod,
		rv.env.chaos, rv.validator, rv.stake))
}

func evmTxArgs(to common.Address) evmtypes.EvmTxArgs {
	return evmtypes.EvmTxArgs{To: &to, GasLimit: 8_000_000}
}

func chaosRun(env *chaosEnv, steps []chaosStep) factory.CallArgs {
	return factory.CallArgs{ContractABI: env.fixture.ABI, MethodName: "run", Args: []interface{}{steps, false}}
}
