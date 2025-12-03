package types

import (
	"crypto/sha256"
	"regexp"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var aLiquidDenom = regexp.MustCompile(`^aLIQUID[0-9]+$`)

func IsLiquidToken(denom string) bool {
	return aLiquidDenom.MatchString(denom)
}

// GetEscrowAddress returns the escrow address for the specified share owner.
// This function is based on native GetEscrowAddress of IBC Transfer module and
// follows the format as outlined in ADR 028 with minimal changes:
// https://github.com/cosmos/cosmos-sdk/blob/master/docs/architecture/adr-028-public-key-addresses.md
func GetEscrowAddress(owner sdk.AccAddress) sdk.AccAddress {
	preImage := []byte(ModuleName)
	preImage = append(preImage, 0)
	preImage = append(preImage, owner.Bytes()...)
	hash := sha256.Sum256(preImage)
	return hash[:20]
}
