package types

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	denomBaseNamePrefix    = "aLIQUID"
	denomDisplayNamePrefix = "LIQUID"
)

// DenomBaseNameFromID compose denom name based on id for exponent 0
func DenomBaseNameFromID(id uint64) string {
	return fmt.Sprintf("%s%d", denomBaseNamePrefix, id)
}

// DenomDisplayNameFromID compose denom name based on id for exponent 18
func DenomDisplayNameFromID(id uint64) string {
	return fmt.Sprintf("%s%d", denomDisplayNamePrefix, id)
}

func DenomIDFromBaseName(baseName string) (uint64, error) {
	stringID := strings.TrimPrefix(baseName, denomBaseNamePrefix)
	result, err := strconv.Atoi(stringID)
	return uint64(result), err
}
