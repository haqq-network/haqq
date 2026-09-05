package bech32_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/cmd/config"
	"github.com/haqq-network/haqq/precompiles/bech32"
)

// TestBech32ToHexRejectsNonEVMAddresses pins the fix for the truncation in bech32ToHex.
//
// sdk.VerifyAddressFormat only rejects empty addresses and anything above
// address.MaxAddrLen (255), so a 32-byte ADR-028 / interchain account passes it.
// common.BytesToAddress then keeps the trailing 20 bytes, which the supplier of the
// string chooses freely: bech32ToHex("<12 arbitrary bytes || attacker EVM address>")
// used to return the attacker's address. A contract deriving a payout target from a
// user-supplied Cosmos address would pay the attacker.
//
// Same class as the withdraw-address truncation in the journal mirrors, at a different
// boundary - see docs/security/haqq-precompile-withdraw-address-truncation-2026-09.md.
func (s *PrecompileTestSuite) TestBech32ToHexRejectsNonEVMAddresses() {
	s.SetupTest()
	method := s.precompile.Methods[bech32.Bech32ToHexMethod]

	attacker := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	// A 32-byte address whose trailing 20 bytes are the attacker's EVM address, exactly
	// the shape common.BytesToAddress would collapse onto.
	wide := make([]byte, 32)
	copy(wide[12:], attacker.Bytes())
	wideBech32, err := sdk.Bech32ifyAddressBytes(config.Bech32Prefix, wide)
	s.Require().NoError(err)

	// Guard the guard: without the fix this input would truncate exactly onto the
	// attacker, so the test would prove nothing if the shape were wrong.
	s.Require().Equal(attacker, common.BytesToAddress(wide))

	s.Run("32-byte address is refused, not truncated", func() {
		bz, err := s.precompile.Bech32ToHex(&method, []interface{}{wideBech32})
		s.Require().Error(err)
		s.Require().ErrorContains(err, "has no EVM representation")
		s.Require().ErrorContains(err, "decodes to 32 bytes")
		s.Require().Empty(bz)
	})

	s.Run("other non-20-byte lengths are refused too", func() {
		for _, n := range []int{1, 19, 21, 64} {
			addr, err := sdk.Bech32ifyAddressBytes(config.Bech32Prefix, make([]byte, n))
			s.Require().NoError(err)

			_, err = s.precompile.Bech32ToHex(&method, []interface{}{addr})
			s.Require().Errorf(err, "a %d-byte address must be refused", n)
			s.Require().ErrorContains(err, "has no EVM representation")
		}
	})

	s.Run("20-byte addresses still convert", func() {
		expected := s.keyring.GetAddr(0)
		addr, err := sdk.Bech32ifyAddressBytes(config.Bech32Prefix, expected.Bytes())
		s.Require().NoError(err)

		bz, err := s.precompile.Bech32ToHex(&method, []interface{}{addr})
		s.Require().NoError(err)

		args, err := s.precompile.Unpack(bech32.Bech32ToHexMethod, bz)
		s.Require().NoError(err)
		s.Require().Equal(expected, args[0].(common.Address))
	})
}
