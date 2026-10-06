package liquid_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/precompiles/liquid"
	"github.com/haqq-network/haqq/utils"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	"github.com/haqq-network/haqq/x/evm/statedb"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	liquidtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
)

// runPrecompileOn runs the liquid precompile against the given StateDB, so the
// caller can prepare it beforehand and commit it afterwards like a transaction.
func (s *PrecompileTestSuite) runPrecompileOn(ctx sdk.Context, stateDB *statedb.StateDB, input []byte) error {
	precompileAddr := s.precompile.Address()
	msg, err := s.factory.GenerateGethCoreMsg(s.keyring.GetPrivKey(0), evmtypes.EvmTxArgs{
		ChainID:   s.network.App.EvmKeeper.ChainID(),
		To:        &precompileAddr,
		GasLimit:  1_000_000,
		GasPrice:  app.MinGasPrices.BigInt(),
		GasFeeCap: s.network.App.FeeMarketKeeper.GetBaseFee(ctx),
		Accesses:  &gethtypes.AccessList{},
	})
	if err != nil {
		return err
	}

	cfg, err := s.network.App.EvmKeeper.EVMConfig(ctx, ctx.BlockHeader().ProposerAddress, s.network.App.EvmKeeper.ChainID())
	if err != nil {
		return err
	}
	evm := s.network.App.EvmKeeper.NewEVM(ctx, msg, cfg, nil, stateDB)
	precompiles, found, err := s.network.App.EvmKeeper.GetPrecompileInstance(ctx, precompileAddr)
	if err != nil {
		return err
	}
	s.Require().True(found)
	evm.WithPrecompiles(precompiles.Map, precompiles.Addresses)

	contract := vm.NewPrecompile(vm.AccountRef(s.keyring.GetAddr(0)), s.precompile, big.NewInt(0), 1_000_000)
	contract.Input = input
	_, err = s.precompile.Run(evm, contract, false)
	return err
}

// TestLiquidateRegistersERC20CodeThroughStateDB checks that the ERC20 precompile
// a liquidation creates ends up with its code once the transaction commits.
//
// Liquidating enables a new ERC20 precompile, whose account gets the ERC20 code
// hash. That registration used to be written through the x/evm keeper into the
// precompile's cache context, behind the back of the calling StateDB: if the
// transaction had touched the address before - it is predictable from the
// liquid denom - the StateDB held its own, code-less copy of the account and
// re-committed it over the registration. The registration now goes through the
// StateDB itself, which also reverts it together with the precompile call.
func (s *PrecompileTestSuite) TestLiquidateRegistersERC20CodeThroughStateDB() {
	erc20CodeHash := crypto.Keccak256Hash(common.FromHex(erc20types.Erc20Bytecode))
	liquidDenom := liquidtypes.DenomBaseNameFromID(0)
	predicted, err := utils.GetIBCDenomAddress(utils.ComputeIBCDenom(liquidtypes.ModuleName, liquidDenom, utils.BaseDenom))
	s.Require().NoError(err)

	testCases := []struct {
		name     string
		preTouch bool
	}{
		{"address untouched before the liquidation", false},
		{"address touched before the liquidation", true},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx := s.network.GetContext()
			s.createClawbackVestingAccount(ctx, sdk.AccAddress(s.keyring.GetAddr(0).Bytes()))

			stateDB := statedb.New(ctx, s.network.App.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash())))
			if tc.preTouch {
				// e.g. the calling contract sent value to the address first: the
				// StateDB now holds a dirty account for it without code
				stateDB.CreateAccount(predicted)
			}

			input, err := s.precompile.Pack(liquid.LiquidateMethod, s.keyring.GetAddr(0), s.keyring.GetAddr(1), big.NewInt(1_000_000))
			s.Require().NoError(err)
			s.Require().NoError(s.runPrecompileOn(ctx, stateDB, input))
			s.Require().NoError(stateDB.Commit())

			pairID := s.network.App.Erc20Keeper.GetTokenPairID(ctx, liquidDenom)
			pair, found := s.network.App.Erc20Keeper.GetTokenPair(ctx, pairID)
			s.Require().True(found, "the liquidation registers a token pair for %s", liquidDenom)
			s.Require().Equal(predicted, pair.GetERC20Contract(), "the test must target the precompile the liquidation enabled")

			account := s.network.App.EvmKeeper.GetAccount(ctx, predicted)
			s.Require().NotNil(account)
			s.Require().Equal(erc20CodeHash, common.BytesToHash(account.CodeHash), "the ERC20 precompile account must keep its code hash")
			s.Require().NotEmpty(s.network.App.EvmKeeper.GetCode(ctx, erc20CodeHash), "the ERC20 bytecode must be stored")
		})
	}
}
