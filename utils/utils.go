package utils

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
	"golang.org/x/exp/constraints"

	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	haqqtypes "github.com/haqq-network/haqq/types"
)

const (
	// MainNetChainID defines the Haqq Network EIP155 chain ID for mainnet
	MainNetChainID = "haqq_11235"
	// TestEdge2ChainID defines the Haqq Network EIP155 chain ID for public testnet
	TestEdge2ChainID = "haqq_54211"
	// LocalNetChainID defines the Haqq Network EIP155 chain ID for local devnet
	LocalNetChainID = "haqq_121799"
	// BaseDenom defines the Haqq Network basic denomination
	BaseDenom = "aISLM"
)

func IsMainNetwork(chainID string) bool {
	return strings.HasPrefix(chainID, MainNetChainID)
}

func IsTestEdge2Network(chainID string) bool {
	return strings.HasPrefix(chainID, TestEdge2ChainID)
}

func IsLocalNetwork(chainID string) bool {
	return strings.HasPrefix(chainID, LocalNetChainID)
}

// IsSupportedKey returns true if the pubkey type is supported by the chain
// (i.e eth_secp256k1, amino multisig, ed25519).
// NOTE: Nested multisigs are not supported.
func IsSupportedKey(pubkey cryptotypes.PubKey) bool {
	switch pubkey := pubkey.(type) {
	case *ethsecp256k1.PubKey, *ed25519.PubKey:
		return true
	case multisig.PubKey:
		if len(pubkey.GetPubKeys()) == 0 {
			return false
		}

		for _, pk := range pubkey.GetPubKeys() {
			switch pk.(type) {
			case *ethsecp256k1.PubKey, *ed25519.PubKey:
				continue
			default:
				// Nested multisigs are unsupported
				return false
			}
		}

		return true
	default:
		return false
	}
}

// GetHaqqAddressFromBech32 returns the sdk.Account address of given address,
// while also changing bech32 human-readable prefix (HRP) to the value set on
// the global sdk.Config (eg: `haqq`).
// The function fails if the provided bech32 address is invalid.
func GetHaqqAddressFromBech32(address string) (sdk.AccAddress, error) {
	bech32Prefix := strings.SplitN(address, "1", 2)[0]
	if bech32Prefix == address {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid bech32 address: %s", address)
	}

	return CreateAccAddressFromBech32(address, bech32Prefix)
}

// CreateAccAddressFromBech32 creates an AccAddress from a Bech32 string.
func CreateAccAddressFromBech32(address, bech32prefix string) (sdk.AccAddress, error) {
	if len(strings.TrimSpace(address)) == 0 {
		return sdk.AccAddress{}, fmt.Errorf("empty address string is not allowed")
	}

	if len(strings.TrimSpace(bech32prefix)) == 0 {
		return sdk.AccAddress{}, fmt.Errorf("empty bech32 prefix string is not allowed")
	}

	addressBz, err := sdk.GetFromBech32(address, bech32prefix)
	if err != nil {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid address %s, %s", address, err.Error())
	}

	// safety check: shouldn't happen
	if err := sdk.VerifyAddressFormat(addressBz); err != nil {
		return nil, err
	}

	return sdk.AccAddress(addressBz), nil
}

func GetAccAddressFromEthAddress(addrString string) sdk.AccAddress {
	addr := common.HexToAddress(addrString).Bytes()
	return sdk.AccAddress(addr)
}

// GetIBCDenomAddress returns the address from the hash of the ICS20's DenomTrace Path.
func GetIBCDenomAddress(denom string) (common.Address, error) {
	if !strings.HasPrefix(denom, "ibc/") {
		return common.Address{}, ibctransfertypes.ErrInvalidDenomForTransfer.Wrapf("coin %s does not have 'ibc/' prefix", denom)
	}

	if len(denom) < 5 || strings.TrimSpace(denom[4:]) == "" {
		return common.Address{}, ibctransfertypes.ErrInvalidDenomForTransfer.Wrapf("coin %s is not a valid IBC voucher hash", denom)
	}

	// Get the address from the hash of the ICS20's DenomTrace Path
	bz, err := ibctransfertypes.ParseHexHash(denom[4:])
	if err != nil {
		return common.Address{}, ibctransfertypes.ErrInvalidDenomForTransfer.Wrap(err.Error())
	}

	return common.BytesToAddress(bz), nil
}

// ComputeIBCDenomTrace compute the ibc voucher denom trace associated with
// the portID, channelID, and the given a token denomination.
// For ibc-go v10, use types.Denom and Hop instead of legacy DenomTrace.
func ComputeIBCDenomTrace(portID, channelID, denom string) ibctransfertypes.Denom {
	return ibctransfertypes.NewDenom(denom, ibctransfertypes.Hop{PortId: portID, ChannelId: channelID})
}

// ComputeIBCDenom compute the ibc voucher denom associated to
// the portID, channelID, and the given a token denomination.
func ComputeIBCDenom(portID, channelID, denom string) string {
	return ComputeIBCDenomTrace(portID, channelID, denom).IBCDenom()
}

// ParseHexValue parses a hex string into a big.Int
func ParseHexValue(hexStr string) *big.Int {
	hexStr = Remove0xPrefix(hexStr)

	value := new(big.Int)
	if _, ok := value.SetString(hexStr, 16); !ok {
		log.Fatalf("Failed to parse hex string: %s", hexStr)
	}

	return value
}

// Remove0xPrefix removes the 0x prefix from a hex string
func Remove0xPrefix(s string) string {
	if len(s) > 1 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		return s[2:]
	}
	return s
}

// Keccak256 calculates the keccak256 hash of a byte slice
func Keccak256(data []byte) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write(data)
	return hash.Sum(nil)
}

// CalculateStorageKey calculates the storage key for a given address and index
func CalculateStorageKey(addr string, i int) string {
	pos := fmt.Sprintf("%064x", i)
	key := strings.ToLower(Remove0xPrefix(addr))
	keyPadded := fmt.Sprintf("%064s", key)
	combined := keyPadded + pos

	combinedBytes, err := hex.DecodeString(combined)
	if err != nil {
		log.Fatalf("Failed to decode hex string: %v", err)
	}

	storageKey := Keccak256(combinedBytes)
	return "0x" + hex.EncodeToString(storageKey)
}

// IsContractAccount checks if the given account is a contract account
func IsContractAccount(acc authtypes.AccountI) error {
	contractETHAccount, ok := acc.(haqqtypes.EthAccountI)
	if !ok {
		return fmt.Errorf("account is not an eth account")
	}

	if contractETHAccount.Type() != haqqtypes.AccountTypeContract {
		return fmt.Errorf("account is not a contract account")
	}
	return nil
}

// SortSlice sorts a slice of any ordered type.
func SortSlice[T constraints.Ordered](slice []T) {
	sort.Slice(slice, func(i, j int) bool {
		return slice[i] < slice[j]
	})
}
