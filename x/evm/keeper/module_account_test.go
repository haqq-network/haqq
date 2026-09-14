package keeper_test

import (
	"math/big"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/x/evm/statedb"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// TestSetAccountRefusesModuleAccountBalanceChange pins the guard in SetAccount at its
// own level, so it survives changes that do not go through a precompile.
//
// A module account's balance is owned by its module and moves through bank calls that
// leave no EVM journal entry. StateDB also reads balances from the transaction context
// while precompiles write through the cache context, so the EVM's view of such an
// account can be stale by exactly the amount a precompile just moved. Reconciling that
// view would mint or burn module funds; SetAccount must refuse instead.
func (suite *KeeperTestSuite) TestSetAccountRefusesModuleAccountBalanceChange() {
	suite.SetupTest()

	k := suite.network.App.EvmKeeper
	ctx := suite.network.GetContext()

	bondedPool := common.BytesToAddress(authtypes.NewModuleAddress(stakingtypes.BondedPoolName).Bytes())
	bankBalance := k.GetBalance(ctx, bondedPool)
	suite.Require().Positive(bankBalance.Sign(), "the bonded pool must hold the validators' stake")

	suite.Run("a matching balance is a no-op", func() {
		err := k.SetAccount(ctx, bondedPool, statedb.Account{
			Balance:  new(big.Int).Set(bankBalance),
			CodeHash: evmtypes.EmptyCodeHash,
		})
		suite.Require().NoError(err)
		suite.Require().Zero(k.GetBalance(ctx, bondedPool).Cmp(bankBalance))
	})

	suite.Run("a stale low balance is refused instead of burning the pool", func() {
		stale := new(big.Int).Sub(bankBalance, big.NewInt(1))
		err := k.SetAccount(ctx, bondedPool, statedb.Account{
			Balance:  stale,
			CodeHash: evmtypes.EmptyCodeHash,
		})
		suite.Require().ErrorIs(err, evmtypes.ErrInvalidAccount)
		suite.Require().Zero(k.GetBalance(ctx, bondedPool).Cmp(bankBalance), "nothing may be burned")
	})

	suite.Run("a higher balance is refused instead of minting into the pool", func() {
		inflated := new(big.Int).Add(bankBalance, big.NewInt(1))
		err := k.SetAccount(ctx, bondedPool, statedb.Account{
			Balance:  inflated,
			CodeHash: evmtypes.EmptyCodeHash,
		})
		suite.Require().ErrorIs(err, evmtypes.ErrInvalidAccount)
		suite.Require().Zero(k.GetBalance(ctx, bondedPool).Cmp(bankBalance), "nothing may be minted")
	})

	suite.Run("a regular account is still reconciled", func() {
		eoa := common.BytesToAddress(suite.keyring.GetAddr(0).Bytes())
		target := new(big.Int).Add(k.GetBalance(ctx, eoa), big.NewInt(1))
		err := k.SetAccount(ctx, eoa, statedb.Account{
			Balance:  target,
			CodeHash: evmtypes.EmptyCodeHash,
		})
		suite.Require().NoError(err)
		suite.Require().Zero(k.GetBalance(ctx, eoa).Cmp(target))
	})
}
