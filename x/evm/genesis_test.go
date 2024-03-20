package evm_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	haqqtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/x/evm"
	"github.com/haqq-network/haqq/x/evm/statedb"
	"github.com/haqq-network/haqq/x/evm/types"
)

func (suite *EvmTestSuite) TestInitGenesis() {
	privkey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)

	address := common.HexToAddress(privkey.PubKey().Address().String())

	var (
		vmdb *statedb.StateDB
		ctx  sdk.Context
	)

	testCases := []struct {
		name     string
		malleate func()
		genState *types.GenesisState
		expPanic bool
	}{
		{
			"default",
			func() {},
			types.DefaultGenesisState(),
			false,
		},
		{
			"valid account",
			func() {
				vmdb.AddBalance(address, big.NewInt(1))
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Storage: types.Storage{
							{Key: common.BytesToHash([]byte("key")).String(), Value: common.BytesToHash([]byte("value")).String()},
						},
					},
				},
			},
			false,
		},
		{
			"account not found",
			func() {},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
					},
				},
			},
			true,
		},
		{
			"invalid account type",
			func() {
				acc := authtypes.NewBaseAccountWithAddress(address.Bytes())
				accNum := suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				acc.AccountNumber = accNum
				suite.network.App.AccountKeeper.SetAccount(ctx, acc)
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
					},
				},
			},
			true,
		},
		{
			"invalid code hash",
			func() {
				acc := suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, address.Bytes())
				suite.network.App.AccountKeeper.SetAccount(ctx, acc)
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Code:    "ffffffff",
					},
				},
			},
			true,
		},
		{
			"ignore empty account code checking",
			func() {
				acc := suite.network.App.AccountKeeper.NewAccountWithAddress(ctx, address.Bytes())

				suite.network.App.AccountKeeper.SetAccount(ctx, acc)
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Code:    "",
					},
				},
			},
			false,
		},
		{
			"ignore empty account code checking with non-empty codehash",
			func() {
				ethAcc := &haqqtypes.EthAccount{
					BaseAccount: authtypes.NewBaseAccount(address.Bytes(), nil, 0, 0),
					CodeHash:    common.BytesToHash([]byte{1, 2, 3}).Hex(),
				}
				accNum := suite.network.App.AccountKeeper.NextAccountNumber(ctx)
				ethAcc.AccountNumber = accNum
				suite.network.App.AccountKeeper.SetAccount(ctx, ethAcc)
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Code:    "",
					},
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset values
			vmdb = suite.StateDB()
			ctx = suite.network.GetContext()

			tc.malleate()
			err := vmdb.Commit()
			suite.Require().NoError(err)

			if tc.expPanic {
				suite.Require().Panics(
					func() {
						_ = evm.InitGenesis(ctx, suite.network.App.EvmKeeper, suite.network.App.AccountKeeper, *tc.genState)
					},
				)
			} else {
				suite.Require().NotPanics(
					func() {
						_ = evm.InitGenesis(ctx, suite.network.App.EvmKeeper, suite.network.App.AccountKeeper, *tc.genState)
					},
				)
			}
		})
	}
}
