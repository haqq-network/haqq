package types

import "fmt"

const (
	denomNamePrefix0  = "aLIQUIDDENOM"
	denomNamePrefix18 = "LIQUIDDENOM"
)

// DenomName0FromID compose denom name based on id for exponent 0
func DenomName0FromID(id uint64) string {
	return fmt.Sprintf("%s%d", denomNamePrefix0, id)
}

// DenomName0FromID compose denom name based on id for exponent 18
func DenomName18FromID(id uint64) string {
	return fmt.Sprintf("%s%d", denomNamePrefix18, id)
}
