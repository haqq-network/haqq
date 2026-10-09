package distribution_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/haqq-network/haqq/cmd/config"
	"github.com/haqq-network/haqq/precompiles/distribution"
)

// TestNewMsgWithdrawValidatorCommissionRejectsWideAddress covers the last Cosmos-to-EVM
// boundary in the precompiles that used to truncate.
//
// NewMsgWithdrawValidatorCommission returns the EVM address that
// WithdrawValidatorCommission compares the caller and the origin against:
//
//	if contract.CallerAddress != validatorHexAddr && origin != validatorHexAddr { reject }
//
// Cosmos accepts operator addresses of up to 255 bytes, and common.BytesToAddress keeps
// only the trailing 20 bytes of a longer one. Since the whole string comes from
// calldata, so do those trailing bytes: a caller could hand in a 32-byte operator
// address whose tail is its own EVM address and satisfy the check against an address it
// does not control. The message server rejects such an operator afterwards, because no
// validator on this chain has one, so this was never a live theft - but an
// authorization check must not be decided by a truncation in the first place.
func TestNewMsgWithdrawValidatorCommissionRejectsWideAddress(t *testing.T) {
	config.SetBech32Prefixes(sdk.GetConfig())

	attacker := common.HexToAddress("0x1122334455667788990011223344556677889900")

	// 32 bytes, ADR-028 shaped, whose trailing 20 bytes are the attacker's EVM address.
	wide := append(bytes.Repeat([]byte{0xAB}, 12), attacker.Bytes()...)
	require.Len(t, wide, 32)
	wideBech32, err := sdk.Bech32ifyAddressBytes(config.Bech32PrefixValAddr, wide)
	require.NoError(t, err)

	_, _, err = distribution.NewMsgWithdrawValidatorCommission([]interface{}{wideBech32})
	require.Error(t, err, "a validator address that does not decode to 20 bytes must be refused")
	require.Contains(t, err.Error(), "must decode to a 20-byte address")

	// The same input under the old helper resolved to the attacker's own address, which
	// is exactly what the authorization check would then have accepted.
	require.Equal(t, attacker, common.BytesToAddress(wide), "precondition of the regression")
}

// TestNewMsgWithdrawValidatorCommissionAcceptsOperatorAddress pins the normal path: a
// 20-byte operator address still resolves to its own EVM address, unchanged.
func TestNewMsgWithdrawValidatorCommissionAcceptsOperatorAddress(t *testing.T) {
	config.SetBech32Prefixes(sdk.GetConfig())

	operator := common.HexToAddress("0x00112233445566778899AABBCCDDEEFF00112233")
	operatorBech32 := sdk.ValAddress(operator.Bytes()).String()

	msg, validatorHexAddr, err := distribution.NewMsgWithdrawValidatorCommission([]interface{}{operatorBech32})
	require.NoError(t, err)
	require.Equal(t, operatorBech32, msg.ValidatorAddress, "the message must carry the address as given")
	require.Equal(t, operator, validatorHexAddr)
}

// TestNewMsgWithdrawValidatorCommissionRejectsNonValidatorAddress pins the parsing
// change that came with it.
//
// The address is decoded as a validator operator address, which is what the message
// field holds. The previous helper instead sniffed the string for "val" and fell back
// to parsing it as an account address - and "val" is spellable in the bech32 data part,
// so which branch ran depended on the address's own characters.
func TestNewMsgWithdrawValidatorCommissionRejectsNonValidatorAddress(t *testing.T) {
	config.SetBech32Prefixes(sdk.GetConfig())

	account := sdk.AccAddress(common.HexToAddress("0x00112233445566778899AABBCCDDEEFF00112233").Bytes()).String()

	_, _, err := distribution.NewMsgWithdrawValidatorCommission([]interface{}{account})
	require.Error(t, err, "an account address is not a validator operator address")
}

// TestNewMsgWithdrawValidatorCommissionRejectsEmptyAddress keeps the empty-string case
// an error rather than the zero address, which would compare equal to an uninitialised
// caller or origin.
func TestNewMsgWithdrawValidatorCommissionRejectsEmptyAddress(t *testing.T) {
	config.SetBech32Prefixes(sdk.GetConfig())

	_, _, err := distribution.NewMsgWithdrawValidatorCommission([]interface{}{""})
	require.Error(t, err)
}
