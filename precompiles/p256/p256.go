package p256

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/haqq-network/haqq/crypto/secp256r1"
)

var _ vm.PrecompiledContract = &Precompile{}

const (
	// VerifyGas is the secp256r1 elliptic curve signature verifier gas price.
	VerifyGas uint64 = 3450
	// VerifyInputLength defines the required input length (160 bytes).
	VerifyInputLength = 160
)

// PrecompileAddress defines the hex address of the p256 precompiled contract.
const PrecompileAddress = "0x0000000000000000000000000000000000000100"

// Precompile secp256r1 (P256) signature verification
// implemented as a native contract as per EIP-7212.
// See https://eips.ethereum.org/EIPS/eip-7212 for details
type Precompile struct{}

// Address defines the address of the p256 precompiled contract.
func (Precompile) Address() common.Address {
	return common.HexToAddress(PrecompileAddress)
}

// RequiredGas returns the static gas required to execute the precompiled contract.
func (p Precompile) RequiredGas(_ []byte) uint64 {
	return VerifyGas
}

// Run executes the p256 signature verification using ECDSA.
//
// Input data: 160 bytes of data including:
//   - 32 bytes of the signed data hash
//   - 32 bytes of the r component of the signature
//   - 32 bytes of the s component of the signature
//   - 32 bytes of the x coordinate of the public key
//   - 32 bytes of the y coordinate of the public key
//
// Output data: 32 bytes of result data and error
//   - If the signature verification process succeeds, it returns 1 in 32 bytes format
func (p *Precompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) (bz []byte, err error) {
	input := contract.Input
	// Check the input length
	if len(input) != VerifyInputLength {
		// Input length is invalid
		return nil, nil
	}

	// Extract the hash, r, s, x, y from the input
	hash := input[0:32]
	r, s := new(big.Int).SetBytes(input[32:64]), new(big.Int).SetBytes(input[64:96])
	x, y := new(big.Int).SetBytes(input[96:128]), new(big.Int).SetBytes(input[128:160])

	// Verify the secp256r1 signature
	if secp256r1.Verify(hash, r, s, x, y) {
		// Signature is valid
		return common.LeftPadBytes(common.Big1.Bytes(), 32), nil
	}

	// Signature is invalid
	return nil, nil
}
