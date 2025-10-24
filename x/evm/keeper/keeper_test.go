package keeper_test

import (
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
			k := keeper.Keeper{}
			ctx := suite.network.GetContext().WithChainID(tc.chainID)

			if tc.expPanic {
				suite.Require().Panics(func() {
					k.WithChainID(ctx)
				})
			} else {
				suite.Require().NotPanics(func() {
					k.WithChainID(ctx)
					suite.Require().Equal(tc.expChainID, k.ChainID().Int64())
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
			suite.Require().NoError(suite.network.App.EvmKeeper.BeginBlock(suite.network.GetContext()))
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
		malleate func() common.Address
		expRes   []int
	}{
		{
			name:     "Only one account that's not a contract (no storage)",
			malleate: nil,
		},
		{
			name: "Two accounts - one contract (with storage), one wallet",
			malleate: func() common.Address {
				supply := big.NewInt(100)
				return suite.DeployTestContract(suite.T(), ctx, suite.keyring.GetAddr(0), supply)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()

			var contractAddr common.Address
			if tc.malleate != nil {
				contractAddr = tc.malleate()
			}

			i := 0
			suite.network.App.AccountKeeper.IterateAccounts(suite.network.GetContext(), func(account sdk.AccountI) bool {
				ethAccount, ok := account.(haqqtypes.EthAccountI)
				if !ok {
					// ignore non EthAccounts
					return false
				}

				addr := ethAccount.EthAddress()
				storage := suite.network.App.EvmKeeper.GetAccountStorage(suite.network.GetContext(), addr)

				if addr.Hex() == contractAddr.Hex() {
					suite.Require().NotEqual(0, len(storage),
						"expected account %d to have non-zero amount of storage slots, got %d",
						i, len(storage),
					)
				} else {
					suite.Require().Len(storage, 0,
						"expected account %d to have %d storage slots, got %d",
						i, 0, len(storage),
					)
				}
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
