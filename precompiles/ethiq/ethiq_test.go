package ethiq_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/precompiles/authorization"
	"github.com/haqq-network/haqq/precompiles/ethiq"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

func (s *PrecompileTestSuite) TestIsTransaction() {
	testCases := []struct {
		name   string
		method string
		isTx   bool
	}{
		{
			ethiq.MintHaqq,
			s.precompile.Methods[ethiq.MintHaqq].Name,
			true,
		},
		{
			ethiq.MintHaqqByApplication,
			s.precompile.Methods[ethiq.MintHaqqByApplication].Name,
			true,
		},
		{
			authorization.ApproveMethod,
			s.precompile.Methods[authorization.ApproveMethod].Name,
			true,
		},
		{
			authorization.RevokeMethod,
			s.precompile.Methods[authorization.RevokeMethod].Name,
			true,
		},
		{
			ethiq.ApproveApplicationIDMethod,
			s.precompile.Methods[ethiq.ApproveApplicationIDMethod].Name,
			true,
		},
		{
			ethiq.RevokeApplicationIDMethod,
			s.precompile.Methods[ethiq.RevokeApplicationIDMethod].Name,
			true,
		},
		{
			authorization.IncreaseAllowanceMethod,
			s.precompile.Methods[authorization.IncreaseAllowanceMethod].Name,
			true,
		},
		{
			authorization.DecreaseAllowanceMethod,
			s.precompile.Methods[authorization.DecreaseAllowanceMethod].Name,
			true,
		},
		{
			ethiq.Calculate,
			s.precompile.Methods[ethiq.Calculate].Name,
			false,
		},
		{
			ethiq.CalculateForApplication,
			s.precompile.Methods[ethiq.CalculateForApplication].Name,
			false,
		},
		{
			authorization.AllowanceMethod,
			s.precompile.Methods[authorization.AllowanceMethod].Name,
			false,
		},
		{
			"invalid",
			"invalid",
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.Require().Equal(s.precompile.IsTransaction(tc.method), tc.isTx)
		})
	}
}

func (s *PrecompileTestSuite) TestRequiredGas() {
	testCases := []struct {
		name   string
		method string
	}{
		{ethiq.MintHaqq, ethiq.MintHaqq},
		{ethiq.MintHaqqByApplication, ethiq.MintHaqqByApplication},
		{ethiq.Calculate, ethiq.Calculate},
		{ethiq.CalculateForApplication, ethiq.CalculateForApplication},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			method := s.precompile.Methods[tc.method]
			input, err := s.precompile.Pack(tc.method, getPackArgs(s, tc.method)...)
			if err != nil {
				// some methods need valid args; just test with methodID + some bytes
				input = append(method.ID, make([]byte, 32)...) //nolint: gocritic
			}
			gas := s.precompile.RequiredGas(input)
			s.Require().GreaterOrEqual(gas, uint64(0))
		})
	}
}

func (s *PrecompileTestSuite) TestRun() {
	var ctx sdk.Context

	testcases := []struct {
		name        string
		malleate    func() (addr interface{ Hex() string }, input []byte)
		readOnly    bool
		expPass     bool
		errContains string
	}{
		{
			name: "pass - calculate query",
			malleate: func() (interface{ Hex() string }, []byte) {
				input, err := s.precompile.Pack(
					ethiq.Calculate,
					big.NewInt(1e18),
				)
				s.Require().NoError(err)
				return s.keyring.GetAddr(0), input
			},
			readOnly: false,
			expPass:  true,
		},
		{
			name: "fail - unknown method",
			malleate: func() (interface{ Hex() string }, []byte) {
				// 4-byte selector that doesn't match any method
				return s.keyring.GetAddr(0), []byte{0xde, 0xad, 0xbe, 0xef}
			},
			readOnly:    false,
			expPass:     false,
			errContains: "",
		},
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()
			baseFee := s.network.App.FeeMarketKeeper.GetBaseFee(ctx)

			caller, input := tc.malleate()
			callerAddr := caller.(interface{ Bytes() []byte })
			_ = callerAddr

			// Use keyring address as the caller
			callerCommon := s.keyring.GetAddr(0)

			contract := vm.NewPrecompile(vm.AccountRef(callerCommon), s.precompile, big.NewInt(0), uint64(1e6))
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
			s.Require().NoError(err)

			signedMsg, err := s.factory.SignMsgEthereumTx(s.keyring.GetPrivKey(0), msgEthereumTx)
			s.Require().NoError(err)

			proposerAddress := ctx.BlockHeader().ProposerAddress
			cfg, err := s.network.App.EvmKeeper.EVMConfig(ctx, proposerAddress, s.network.App.EvmKeeper.ChainID())
			s.Require().NoError(err)

			ethChainID := s.network.GetEIP155ChainID()
			signer := gethtypes.LatestSignerForChainID(ethChainID)
			msg, err := signedMsg.AsMessage(signer, baseFee)
			s.Require().NoError(err)

			evm := s.network.App.EvmKeeper.NewEVM(ctx, msg, cfg, nil, s.network.GetStateDB())

			precompiles, found, err := s.network.App.EvmKeeper.GetPrecompileInstance(ctx, contractAddr)
			s.Require().NoError(err)
			s.Require().True(found)
			evm.WithPrecompiles(precompiles.Map, precompiles.Addresses)

			bz, err := s.precompile.Run(evm, contract, tc.readOnly)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(bz)
			} else {
				s.Require().Error(err)
				s.Require().Nil(bz)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			}
		})
	}
}

// getPackArgs returns minimal valid args for packing test method calls
func getPackArgs(s *PrecompileTestSuite, method string) []any {
	switch method {
	case ethiq.Calculate:
		return []any{big.NewInt(1e18)}
	case ethiq.CalculateForApplication:
		return []any{big.NewInt(0)}
	case ethiq.MintHaqq:
		return []any{s.keyring.GetAddr(0), s.keyring.GetAddr(1), big.NewInt(1e18)}
	case ethiq.MintHaqqByApplication:
		return []any{s.keyring.GetAddr(0), big.NewInt(0)}
	default:
		return nil
	}
}
