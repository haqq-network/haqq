package liquid_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/precompiles/liquid"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	liquidtypes "github.com/haqq-network/haqq/x/liquidvesting/types"
)

// runPrecompile is a helper that builds a full EVM and calls s.precompile.Run.
func (s *PrecompileTestSuite) runPrecompile(ctx sdk.Context, caller sdk.AccAddress, input []byte, readOnly bool) ([]byte, error) {
	baseFee := s.network.App.FeeMarketKeeper.GetBaseFee(ctx)

	contract := vm.NewPrecompile(vm.AccountRef(caller), s.precompile, big.NewInt(0), uint64(1e6))
	contract.Input = input

	contractAddr := contract.Address()
	txArgs := evmtypes.EvmTxArgs{
		ChainID:   s.network.App.EvmKeeper.ChainID(),
		Nonce:     0,
		To:        &contractAddr,
		Amount:    nil,
		GasLimit:  100000,
		GasPrice:  app.MinGasPrices.BigInt(),
		GasFeeCap: baseFee,
		GasTipCap: big.NewInt(1),
		Accesses:  &gethtypes.AccessList{},
	}

	msgEthereumTx, err := s.factory.GenerateMsgEthereumTx(s.keyring.GetPrivKey(0), txArgs)
	if err != nil {
		return nil, err
	}

	signedMsg, err := s.factory.SignMsgEthereumTx(s.keyring.GetPrivKey(0), msgEthereumTx)
	if err != nil {
		return nil, err
	}

	proposerAddress := ctx.BlockHeader().ProposerAddress
	cfg, err := s.network.App.EvmKeeper.EVMConfig(ctx, proposerAddress, s.network.App.EvmKeeper.ChainID())
	if err != nil {
		return nil, err
	}

	ethChainID := s.network.GetEIP155ChainID()
	signer := gethtypes.LatestSignerForChainID(ethChainID)
	msg, err := signedMsg.AsMessage(signer, baseFee)
	if err != nil {
		return nil, err
	}

	evm := s.network.App.EvmKeeper.NewEVM(ctx, msg, cfg, nil, s.network.GetStateDB())

	precompiles, found, evmErr := s.network.App.EvmKeeper.GetPrecompileInstance(ctx, contractAddr)
	if evmErr != nil {
		return nil, evmErr
	}
	if !found {
		return nil, nil
	}
	evm.WithPrecompiles(precompiles.Map, precompiles.Addresses)

	return s.precompile.Run(evm, contract, readOnly)
}

// ---------------------------------------------------------------------------
// TestRun covers the Run dispatcher.
// ---------------------------------------------------------------------------

func (s *PrecompileTestSuite) TestRun() {
	testCases := []struct {
		name        string
		malleate    func(ctx sdk.Context) []byte
		readOnly    bool
		expPass     bool
		errContains string
	}{
		{
			name: "fail - unknown method selector",
			malleate: func(_ sdk.Context) []byte {
				return []byte{0xde, 0xad, 0xbe, 0xef}
			},
			readOnly: false,
			expPass:  false,
		},
		{
			name: "fail - liquidate read-only",
			malleate: func(_ sdk.Context) []byte {
				input, err := s.precompile.Pack(
					liquid.LiquidateMethod,
					s.keyring.GetAddr(0),
					s.keyring.GetAddr(1),
					big.NewInt(1_000_000),
				)
				s.Require().NoError(err)
				return input
			},
			readOnly: true,
			expPass:  false,
		},
		{
			name: "fail - liquidate with wrong origin",
			malleate: func(_ sdk.Context) []byte {
				// Use a different address from the keyring sender (index 0) as the "from"
				// so it fails the origin check.
				differentAddr := utiltx.GenerateAddress()
				input, err := s.precompile.Pack(
					liquid.LiquidateMethod,
					differentAddr,
					s.keyring.GetAddr(1),
					big.NewInt(1_000_000),
				)
				s.Require().NoError(err)
				return input
			},
			readOnly: false,
			expPass:  false,
		},
		{
			name: "success - liquidate",
			malleate: func(ctx sdk.Context) []byte {
				// Create a vesting account for keyring[0]
				fromAddr := sdk.AccAddress(s.keyring.GetAddr(0).Bytes())
				s.createClawbackVestingAccount(ctx, fromAddr)

				input, err := s.precompile.Pack(
					liquid.LiquidateMethod,
					s.keyring.GetAddr(0),
					s.keyring.GetAddr(1),
					big.NewInt(1_000_000),
				)
				s.Require().NoError(err)
				return input
			},
			readOnly: false,
			expPass:  true,
		},
		{
			name: "fail - redeem with no liquid tokens",
			malleate: func(_ sdk.Context) []byte {
				input, err := s.precompile.Pack(
					liquid.RedeemMethod,
					s.keyring.GetAddr(0),
					s.keyring.GetAddr(0),
					liquidtypes.DenomBaseNameFromID(0),
					big.NewInt(1_000_000),
				)
				s.Require().NoError(err)
				return input
			},
			readOnly: false,
			expPass:  false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx := s.network.GetContext()

			input := tc.malleate(ctx)
			bz, err := s.runPrecompile(ctx, sdk.AccAddress(s.keyring.GetAddr(0).Bytes()), input, tc.readOnly)
			if tc.expPass {
				s.Require().NoError(err)
				_ = bz
			} else {
				s.Require().Error(err)
			}
		})
	}
}
