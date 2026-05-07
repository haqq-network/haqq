// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package p256_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"math/big"
	"testing"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/suite"

	"github.com/cometbft/cometbft/crypto"
	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/crypto/cryptobyte/asn1"

	"github.com/haqq-network/haqq/precompiles/p256"
)

var s *PrecompileTestSuite

type PrecompileTestSuite struct {
	suite.Suite
	p256Priv   *ecdsa.PrivateKey
	precompile *p256.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(PrecompileTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Precompile Test Suite")
}

func (s *PrecompileTestSuite) SetupTest() {
	p256Priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s.Require().NoError(err)
	s.p256Priv = p256Priv
	s.precompile = &p256.Precompile{}
}

func signMsg(msg []byte, priv *ecdsa.PrivateKey) []byte {
	hash := crypto.Sha256(msg)

	rInt, sInt, err := ecdsa.Sign(rand.Reader, priv, hash)
	s.Require().NoError(err)

	input := make([]byte, p256.VerifyInputLength)
	copy(input[0:32], hash)
	copy(input[32:64], rInt.Bytes())
	copy(input[64:96], sInt.Bytes())
	copy(input[96:128], priv.PublicKey.X.Bytes())
	copy(input[128:160], priv.PublicKey.Y.Bytes())

	return input
}

func parseSignature(sig []byte) (r, s []byte, err error) {
	var inner cryptobyte.String
	input := cryptobyte.String(sig)
	var rRaw []byte
	var sRaw []byte
	if !input.ReadASN1(&inner, asn1.SEQUENCE) ||
		!input.Empty() ||
		!inner.ReadASN1Integer(&rRaw) ||
		!inner.ReadASN1Integer(&sRaw) ||
		!inner.Empty() {
		return nil, nil, errors.New("invalid ASN.1")
	}

	r, err = normalizeScalar32(rRaw)
	if err != nil {
		return nil, nil, err
	}

	s, err = normalizeScalar32(sRaw)
	if err != nil {
		return nil, nil, err
	}

	return r, s, nil
}

func normalizeScalar32(raw []byte) ([]byte, error) {
	scalar := new(big.Int).SetBytes(raw).Bytes()
	if len(scalar) > 32 {
		return nil, errors.New("invalid scalar length")
	}

	out := make([]byte, 32)
	copy(out[32-len(scalar):], scalar)
	return out, nil
}
