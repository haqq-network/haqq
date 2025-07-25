// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package p256_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"

	"github.com/cometbft/cometbft/crypto"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/p256"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

var trueValue = common.LeftPadBytes(common.Big1.Bytes(), 32)

func (s *PrecompileTestSuite) TestAddress() {
	s.Require().Equal(p256.PrecompileAddress, s.precompile.Address().String())
}

func (s *PrecompileTestSuite) TestRequiredGas() {
	s.Require().Equal(p256.VerifyGas, s.precompile.RequiredGas(nil))
}

func (s *PrecompileTestSuite) TestRun() {
	testCases := []struct {
		name    string
		sign    func() []byte
		expPass bool
	}{
		{
			"pass - Sign",
			func() []byte {
				msg := []byte("hello world")
				hash := crypto.Sha256(msg)

				rInt, sInt, err := ecdsa.Sign(rand.Reader, s.p256Priv, hash)
				s.Require().NoError(err)

				input := make([]byte, p256.VerifyInputLength)
				copy(input[0:32], hash)
				copy(input[32:64], rInt.Bytes())
				copy(input[64:96], sInt.Bytes())
				copy(input[96:128], s.p256Priv.PublicKey.X.Bytes())
				copy(input[128:160], s.p256Priv.PublicKey.Y.Bytes())

				return input
			},
			true,
		},
		{
			"pass - sign ASN.1 encoded signature",
			func() []byte {
				msg := []byte("hello world")
				hash := crypto.Sha256(msg)

				sig, err := ecdsa.SignASN1(rand.Reader, s.p256Priv, hash)
				s.Require().NoError(err)

				rBz, sBz, err := parseSignature(sig)
				s.Require().NoError(err)

				input := make([]byte, p256.VerifyInputLength)
				copy(input[0:32], hash)
				copy(input[32:64], rBz)
				copy(input[64:96], sBz)
				copy(input[96:128], s.p256Priv.PublicKey.X.Bytes())
				copy(input[128:160], s.p256Priv.PublicKey.Y.Bytes())

				return input
			},
			true,
		},
		{
			"fail - invalid signature",
			func() []byte {
				privB, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				s.Require().NoError(err)

				bz := elliptic.MarshalCompressed(elliptic.P256(), s.p256Priv.X, s.p256Priv.Y)
				s.Require().NotEmpty(bz)

				msg := []byte("hello world")
				hash := crypto.Sha256(msg)

				rInt, sInt, err := ecdsa.Sign(rand.Reader, s.p256Priv, hash)
				s.Require().NoError(err)

				input := make([]byte, p256.VerifyInputLength)
				copy(input[0:32], hash)
				copy(input[32:64], rInt.Bytes())
				copy(input[64:96], sInt.Bytes())
				copy(input[96:128], privB.PublicKey.X.Bytes())
				copy(input[128:160], privB.PublicKey.Y.Bytes())

				return input
			},
			false,
		},
		{
			"fail - invalid length",
			func() []byte {
				msg := []byte("hello world")
				hash := crypto.Sha256(msg)

				input := make([]byte, 32)
				copy(input[0:32], hash)

				return input
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			input := tc.sign()
			bz, err := s.precompile.Run(nil, &vm.Contract{Input: input}, false)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(trueValue, bz)
			} else {
				s.Require().NoError(err)
				s.Require().Empty(bz)
			}
		})
	}
}
