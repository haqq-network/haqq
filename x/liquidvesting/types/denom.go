package types

import "fmt"

const denomNamePrefix = "liquidDenom"

func DenomNameFromID(ID uint64) string {
	return fmt.Sprintf("%s%d", denomNamePrefix, ID)
}
