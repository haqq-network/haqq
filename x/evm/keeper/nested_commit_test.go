package keeper_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/x/evm/statedb"
	"github.com/haqq-network/haqq/x/evm/types"
)

// TestNestedEVMCommitRefusedInPrecompileContext checks the guard that keeps a
// stateful precompile from committing a nested EVM call. Such a call writes
// storage through its own StateDB into the precompile's cache context, where
// the calling transaction's StateDB never looks, so the caller could spend the
// same ERC20 balance again after the precompile returned.
func (suite *KeeperTestSuite) TestNestedEVMCommitRefusedInPrecompileContext() {
	suite.SetupTest()
	k := suite.network.App.EvmKeeper
	from := suite.keyring.GetAddr(0)
	to := utiltx.GenerateAddress()

	plain := suite.network.GetContext()
	marked := types.WithPrecompileContext(plain, nil)
	suite.Require().False(types.IsPrecompileContext(plain))
	suite.Require().True(types.IsPrecompileContext(marked))
	suite.Require().True(types.IsPrecompileContext(marked.WithGasMeter(plain.GasMeter())),
		"the mark must survive deriving a new context")

	_, err := k.CallEVMWithData(marked, from, &to, nil, true)
	suite.Require().ErrorIs(err, types.ErrNestedEVMCommit)

	_, err = k.CallEVMWithData(marked, from, &to, nil, false)
	suite.Require().NoError(err, "a read-only nested call writes nothing and stays allowed")

	_, err = k.CallEVMWithData(plain, from, &to, nil, true)
	suite.Require().NoError(err, "outside a precompile committing calls are unaffected")
}

// TestKeeperRefusesEVMStateWritesInPrecompileContext checks that code reached
// from a stateful precompile cannot write EVM state through the keeper. Written
// to the cache context, the calling transaction's StateDB would re-commit its
// own copy of the account over it; such code has to write through the StateDB
// attached to its context instead.
func (suite *KeeperTestSuite) TestKeeperRefusesEVMStateWritesInPrecompileContext() {
	addr := utiltx.GenerateAddress()
	code := []byte{0x60, 0x00}
	codeHash := crypto.Keccak256(code)

	testCases := []struct {
		name  string
		write func(ctx sdk.Context) error
	}{
		{"set account", func(ctx sdk.Context) error {
			return suite.network.App.EvmKeeper.SetAccount(ctx, addr, *statedb.NewEmptyAccount())
		}},
		{"set balance", func(ctx sdk.Context) error {
			return suite.network.App.EvmKeeper.SetBalance(ctx, addr, big.NewInt(0))
		}},
		{"set state", func(ctx sdk.Context) error {
			return suite.network.App.EvmKeeper.SetState(ctx, addr, common.Hash{1}, []byte{1})
		}},
		{"set code", func(ctx sdk.Context) error {
			return suite.network.App.EvmKeeper.SetCode(ctx, codeHash, code)
		}},
		{"delete account", func(ctx sdk.Context) error {
			return suite.network.App.EvmKeeper.DeleteAccount(ctx, addr)
		}},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := suite.network.GetContext()

			suite.Require().ErrorIs(tc.write(types.WithPrecompileContext(ctx, nil)), types.ErrPrecompileStateWrite)
			suite.Require().NoError(tc.write(ctx), "outside a precompile the write is allowed")
		})
	}
}
