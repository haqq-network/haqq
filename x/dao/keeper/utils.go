package keeper

import "regexp"

var aLiquidDenom = regexp.MustCompile(`^aLIQUID[0-9]+$`)

func IsLiquidToken(denom string) bool {
	return aLiquidDenom.MatchString(denom)
}
