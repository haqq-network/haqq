package types

import "fmt"

const denomNamePrefix = "liquidDenom"

func DenomNameFromID(id uint64) string {
	return fmt.Sprintf("%s%d", denomNamePrefix, id)
}
