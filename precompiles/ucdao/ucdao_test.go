package ucdao_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/precompiles/ucdao"
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
			ucdao.ConvertToHaqqMethod,
			s.precompile.Methods[ucdao.ConvertToHaqqMethod].Name,
			true,
		},
		{
			ucdao.TransferOwnershipMethod,
			s.precompile.Methods[ucdao.TransferOwnershipMethod].Name,
			true,
		},
		{
			ucdao.TransferOwnershipWithAmountMethod,
			s.precompile.Methods[ucdao.TransferOwnershipWithAmountMethod].Name,
			true,
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
		{ucdao.ConvertToHaqqMethod, ucdao.ConvertToHaqqMethod},
		{ucdao.TransferOwnershipMethod, ucdao.TransferOwnershipMethod},
		{ucdao.TransferOwnershipWithAmountMethod, ucdao.TransferOwnershipWithAmountMethod},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			method := s.precompile.Methods[tc.method]
			// Use methodID + padding to test RequiredGas
			input := append(method.ID, make([]byte, 32)...) //nolint: gocritic
			gas := s.precompile.RequiredGas(input)
			s.Require().GreaterOrEqual(gas, uint64(0))
		})
	}
}

func (s *PrecompileTestSuite) TestRequiredGas_ShortInput() {
	s.Require().Equal(uint64(0), s.precompile.RequiredGas([]byte{0x01}))
}

func (s *PrecompileTestSuite) TestRun() {
	var ctx sdk.Context

	testcases := []struct {
		name        string
		malleate    func() []byte
		readOnly    bool
		expPass     bool
		errContains string
	}{
		{
			name: "fail - unknown method selector",
			malleate: func() []byte {
				return []byte{0xde, 0xad, 0xbe, 0xef}
			},
			readOnly:    false,
			expPass:     false,
			errContains: "",
		},
		{
			name: "fail - convertToHaqq with wrong origin",
			malleate: func() []byte {
				// pack with a different sender than origin
				differentAddr := s.keyring.GetAddr(1)
				input, err := s.precompile.Pack(
					ucdao.ConvertToHaqqMethod,
					differentAddr,
					s.keyring.GetAddr(1),
					big.NewInt(1e18),
				)
				s.Require().NoError(err)
				return input
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

			input := tc.malleate()
			caller := s.keyring.GetAddr(0)

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
