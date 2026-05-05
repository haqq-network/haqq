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
			// Symmetric kill-switch for the Redeem dispatcher: a static call
			// (readOnly = true) must be rejected upfront by RunSetup before
			// any keeper or mirror logic is reached. Without this row, only
			// Liquidate's read-only protection is exercised.
			name: "fail - redeem read-only",
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
		{
			// Tier 3 defence: input shorter than the 4-byte ABI selector
			// must be rejected by the dispatcher without invoking any
			// method. This is the smallest "garbage input" attack surface.
			name: "fail - input shorter than 4-byte selector",
			malleate: func(_ sdk.Context) []byte {
				return []byte{0x12, 0x34, 0x56}
			},
			readOnly: false,
			expPass:  false,
		},
		{
			// Tier 3 defence: a valid Liquidate selector followed by a
			// truncated args payload must be rejected by ABI unpack
			// inside Run, not bubble through to the keeper. The selector
			// is computed below from the method ID so the test cannot
			// drift if the ABI changes.
			name: "fail - liquidate selector with truncated args",
			malleate: func(_ sdk.Context) []byte {
				selector := s.precompile.Methods[liquid.LiquidateMethod].ID
				input := make([]byte, 0, len(selector)+8)
				input = append(input, selector...)
				input = append(input, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08)
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
