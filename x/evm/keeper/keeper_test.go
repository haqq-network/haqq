package keeper_test

import (
	_ "embed"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	haqqtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/x/evm/keeper"
	"github.com/haqq-network/haqq/x/evm/statedb"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func (suite *KeeperTestSuite) TestWithChainID() {
	testCases := []struct {
		name       string
		chainID    string
		expChainID int64
		expPanic   bool
	}{
		{
			"fail - chainID is empty",
			"",
			0,
			true,
		},
		{
			"fail - invalid chainID",
			"chain12345",
			0,
			true,
		},
		{
			"success - Haqq mainnet chain ID",
			"haqq_11235-1",
			11235,
			false,
		},
		{
			"success - Haqq testnet chain ID",
			"haqq_54211-3",
			54211,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			kpr := keeper.Keeper{}
			ctx := suite.network.GetContext().WithChainID(tc.chainID)

			if tc.expPanic {
				suite.Require().Panics(func() {
					kpr.WithChainID(ctx)
				})
			} else {
				suite.Require().NotPanics(func() {
					kpr.WithChainID(ctx)
					suite.Require().Equal(tc.expChainID, kpr.ChainID().Int64())
				})
			}
		})
	}
}

func (suite *KeeperTestSuite) TestBaseFee() {
	testCases := []struct {
		name            string
		enableLondonHF  bool
		enableFeemarket bool
		expectBaseFee   *big.Int
	}{
		{"not enable london HF, not enable feemarket", false, false, nil},
		{"enable london HF, not enable feemarket", true, false, big.NewInt(0)},
		{"enable london HF, enable feemarket", true, true, big.NewInt(1000000000)},
		{"not enable london HF, enable feemarket", false, true, nil},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.enableFeemarket = tc.enableFeemarket
			suite.enableLondonHF = tc.enableLondonHF
			suite.SetupTest()
			err := suite.network.App.EvmKeeper.BeginBlock(suite.network.GetContext())
			suite.Require().NoError(err)
			params := suite.network.App.EvmKeeper.GetParams(suite.network.GetContext())
			ethCfg := params.ChainConfig.EthereumConfig(suite.network.App.EvmKeeper.ChainID())
			baseFee := suite.network.App.EvmKeeper.GetBaseFee(suite.network.GetContext(), ethCfg)
			suite.Require().Equal(tc.expectBaseFee, baseFee)
		})
	}
	suite.enableFeemarket = false
	suite.enableLondonHF = true
}

func (suite *KeeperTestSuite) TestGetAccountStorage() {
	var ctx sdk.Context
	testCases := []struct {
		name     string
		malleate func()
		expRes   []int
	}{
		{
			"Only one account that's not a contract (no storage)",
			func() {
				i := 0
				// NOTE: here we're removing all accounts except for one
				suite.network.App.AccountKeeper.IterateAccounts(ctx, func(account sdk.AccountI) bool {
					defer func() { i++ }()
					if i == 0 {
						return false
					}
					suite.network.App.AccountKeeper.RemoveAccount(ctx, account)
					return false
				})
			},
			[]int{0},
		},
		{
			"Two accounts - one contract (with storage), one wallet",
			func() {
				supply := big.NewInt(100)
				suite.DeployTestContract(suite.T(), ctx, suite.keyring.GetAddr(0), supply)
				i := 0
				suite.network.App.AccountKeeper.IterateAccounts(ctx, func(account sdk.AccountI) bool {
					defer func() { i++ }()
					var storage evmtypes.Storage
					ethAccount, ok := account.(haqqtypes.EthAccountI)
					if ok {
						storage = suite.network.App.EvmKeeper.GetAccountStorage(ctx, ethAccount.EthAddress())
					}
					if i == 0 || len(storage) > 0 {
						return false
					}
					suite.network.App.AccountKeeper.RemoveAccount(ctx, account)
					return false
				})
			},
			[]int{0, 2},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()
			tc.malleate()
			i := 0
			suite.network.App.AccountKeeper.IterateAccounts(ctx, func(account sdk.AccountI) bool {
				ethAccount, ok := account.(haqqtypes.EthAccountI)
				if !ok {
					// ignore non EthAccounts
					return false
				}

				addr := ethAccount.EthAddress()
				storage := suite.network.App.EvmKeeper.GetAccountStorage(ctx, addr)

				suite.Require().Equal(tc.expRes[i], len(storage))
				i++
				return false
			})
		})
	}
}

func (suite *KeeperTestSuite) TestGetAccountOrEmpty() {
	ctx := suite.network.GetContext()
	empty := statedb.Account{
		Balance:  new(big.Int),
		CodeHash: evmtypes.EmptyCodeHash,
	}

	supply := big.NewInt(100)
	contractAddr := suite.DeployTestContract(suite.T(), ctx, suite.keyring.GetAddr(0), supply)

	testCases := []struct {
		name     string
		addr     common.Address
		expEmpty bool
	}{
		{
			"unexisting account - get empty",
			common.Address{},
			true,
		},
		{
			"existing contract account",
			contractAddr,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			res := suite.network.App.EvmKeeper.GetAccountOrEmpty(ctx, tc.addr)
			if tc.expEmpty {
				suite.Require().Equal(empty, res)
			} else {
				suite.Require().NotEqual(empty, res)
			}
		})
	}
}
