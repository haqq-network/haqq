package ucdao_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/ucdao"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	"github.com/haqq-network/haqq/x/evm/statedb"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// runUCDAO executes the ucDAO precompile with the given calldata and readOnly flag.
// The StateDB is returned so a caller can Commit it: the precompile writes into the
// StateDB cache context, and those writes only reach the store on commit.
func (s *PrecompileTestSuite) runUCDAO(input []byte, readOnly bool) ([]byte, *statedb.StateDB, error) {
	ctx := s.network.GetContext()
	baseFee := s.network.App.FeeMarketKeeper.GetBaseFee(ctx)

	caller := s.keyring.GetAddr(0)
	contract := vm.NewPrecompile(vm.AccountRef(caller), s.precompile, big.NewInt(0), uint64(1e6))
	contract.Input = input

	contractAddr := contract.Address()
	txArgs := evmtypes.EvmTxArgs{
		ChainID:   s.network.App.EvmKeeper.ChainID(),
		Nonce:     0,
		To:        &contractAddr,
		GasLimit:  100000,
		GasPrice:  app.MinGasPrices.BigInt(),
		GasFeeCap: baseFee,
		GasTipCap: big.NewInt(1),
		Accesses:  &gethtypes.AccessList{},
	}
	msgEthereumTx, err := s.factory.GenerateMsgEthereumTx(s.keyring.GetPrivKey(0), txArgs)
	s.Require().NoError(err)
	signedMsg, err := s.factory.SignMsgEthereumTx(s.keyring.GetPrivKey(0), msgEthereumTx)
	s.Require().NoError(err)

	cfg, err := s.network.App.EvmKeeper.EVMConfig(ctx, ctx.BlockHeader().ProposerAddress, s.network.App.EvmKeeper.ChainID())
	s.Require().NoError(err)

	signer := gethtypes.LatestSignerForChainID(s.network.GetEIP155ChainID())
	msg, err := signedMsg.AsMessage(signer, baseFee)
	s.Require().NoError(err)

	stDB := s.network.GetStateDB()
	evm := s.network.App.EvmKeeper.NewEVM(ctx, msg, cfg, nil, stDB)
	precompiles, found, err := s.network.App.EvmKeeper.GetPrecompileInstance(ctx, contractAddr)
	s.Require().NoError(err)
	s.Require().True(found)
	evm.WithPrecompiles(precompiles.Map, precompiles.Addresses)

	bz, err := s.precompile.Run(evm, contract, readOnly)
	return bz, stDB, err
}

// TestAuthzMethodsAreWriteProtected pins the fix for the ucDAO IsTransaction gap.
//
// The four authorization methods write Cosmos authz grants but were classified as
// queries, so RunSetup's `readOnly && isTransaction(name)` gate never fired and they
// could be executed from a read-only frame. On HAQQ that is not only STATICCALL:
// CALLCODE and DELEGATECALL also hand the precompile readOnly = true
// (x/evm/core/vm/evm.go, lines 298/343/397 vs 231 for CALL), so a mutating precompile
// is meant to be reachable through CALL alone.
func (s *PrecompileTestSuite) TestAuthzMethodsAreWriteProtected() {
	// NOTE: SetupTest() rebuilds the keyring, so the grantee must be read inside each
	// subtest, after the reset - not captured from the enclosing scope.
	testCases := []struct {
		name  string
		input func(grantee common.Address) []byte
	}{
		{
			authorization.ApproveMethod,
			func(grantee common.Address) []byte {
				input, err := s.precompile.Pack(authorization.ApproveMethod,
					grantee, big.NewInt(1e18), []string{ucdao.ConvertToHaqqMsgURL})
				s.Require().NoError(err)
				return input
			},
		},
		{
			authorization.RevokeMethod,
			func(grantee common.Address) []byte {
				input, err := s.precompile.Pack(authorization.RevokeMethod,
					grantee, []string{ucdao.ConvertToHaqqMsgURL})
				s.Require().NoError(err)
				return input
			},
		},
		{
			authorization.IncreaseAllowanceMethod,
			func(grantee common.Address) []byte {
				input, err := s.precompile.Pack(authorization.IncreaseAllowanceMethod,
					grantee, big.NewInt(1), []string{ucdao.ConvertToHaqqMsgURL})
				s.Require().NoError(err)
				return input
			},
		},
		{
			authorization.DecreaseAllowanceMethod,
			func(grantee common.Address) []byte {
				input, err := s.precompile.Pack(authorization.DecreaseAllowanceMethod,
					grantee, big.NewInt(1), []string{ucdao.ConvertToHaqqMsgURL})
				s.Require().NoError(err)
				return input
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name+" is rejected in a read-only frame", func() {
			s.SetupTest()

			bz, stDB, err := s.runUCDAO(tc.input(s.keyring.GetAddr(1)), true)
			s.Require().ErrorIs(err, vm.ErrWriteProtection)
			s.Require().Nil(bz)

			// Nothing was written - not even into the StateDB cache context, which is
			// where a precompile writes before commit.
			cacheCtx, cacheErr := stDB.GetCacheContext()
			s.Require().NoError(cacheErr)
			grant, _ := s.network.App.AuthzKeeper.GetAuthorization(
				cacheCtx, s.keyring.GetAccAddr(1), s.keyring.GetAccAddr(0), ucdao.ConvertToHaqqMsgURL,
			)
			s.Require().Nil(grant, "a read-only frame must not create a grant")
		})
	}

	// The same call through CALL (readOnly = false) still works, so the fix does not
	// close the legitimate path.
	s.Run("approve still works through CALL", func() {
		s.SetupTest()

		bz, stDB, err := s.runUCDAO(testCases[0].input(s.keyring.GetAddr(1)), false)
		s.Require().NoError(err)
		s.Require().NotEmpty(bz)

		// The precompile writes into the StateDB cache context; commit to observe it.
		s.Require().NoError(stDB.Commit())

		grant, _ := s.network.App.AuthzKeeper.GetAuthorization(
			s.network.GetContext(),
			s.keyring.GetAccAddr(1), s.keyring.GetAccAddr(0), ucdao.ConvertToHaqqMsgURL,
		)
		s.Require().NotNil(grant)
	})
}

// TestIsTransactionMatchesABIMutability guards the class rather than the instance:
// IsTransaction is a hand-written switch that duplicates stateMutability from abi.json,
// and nothing else cross-checks the two. This is what would have caught the gap.
func (s *PrecompileTestSuite) TestIsTransactionMatchesABIMutability() {
	for name, method := range s.precompile.Methods {
		mutates := method.StateMutability != "view" && method.StateMutability != "pure"
		s.Require().Equalf(
			mutates, s.precompile.IsTransaction(name),
			"IsTransaction(%q) disagrees with abi.json stateMutability %q", name, method.StateMutability,
		)
	}
}
