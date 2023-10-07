package utils

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
)

const (
	MainNetChainID   = "haqq_11235"
	TestEdge1ChainID = "haqq_53211"
	TestEdge2ChainID = "haqq_54211"
	LocalNetChainID  = "haqq_121799"
	// BaseDenom defines the Haqq Network mainnet denomination
	BaseDenom = "aISLM"
)

func IsMainNetwork(chainID string) bool {
	return strings.HasPrefix(chainID, MainNetChainID)
}

func IsTestEdge1Network(chainID string) bool {
	return strings.HasPrefix(chainID, TestEdge1ChainID)
}

func IsTestEdge2Network(chainID string) bool {
	return strings.HasPrefix(chainID, TestEdge2ChainID)
}

func IsLocalNetwork(chainID string) bool {
	return strings.HasPrefix(chainID, LocalNetChainID)
}

func IsAllowedVestingFunderAccount(funder string) bool {
	// allowed accounts for vesting funder
	funders := map[string]bool{
		"haqq1uu7epkq75j2qzqvlyzfkljc8h277gz7kxqah0v": true, // mainnet
		"haqq185tcnd67yh9jngx090cggck0yrjsft9sj3lkht": true,
		"haqq1527hg2arxkk0jd53pq80l0l9gjjlclsuxlwmq8": true,
		"haqq1e666058j3ya392rspuxrt69tw6qhrxtxx8z9ha": true,
	}

	// check if funder account is allowed
	_, ok := funders[funder]

	return ok
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
// while also changing bech32 human readable prefix (HRP) to the value set on
// the global sdk.Config (eg: `haqq`).
// The function fails if the provided bech32 address is invalid.
func GetHaqqAddressFromBech32(address string) (sdk.AccAddress, error) {
	bech32Prefix := strings.SplitN(address, "1", 2)[0]
	if bech32Prefix == address {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid bech32 address: %s", address)
	}

	addressBz, err := sdk.GetFromBech32(address, bech32Prefix)
	if err != nil {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid address %s, %s", address, err.Error())
	}

	// safety check: shouldn't happen
	if err := sdk.VerifyAddressFormat(addressBz); err != nil {
		return nil, err
	}

	return sdk.AccAddress(addressBz), nil
}
