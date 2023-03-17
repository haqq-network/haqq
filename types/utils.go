package types

import "strings"

const (
	MainNetChainID   = "haqq_11235"
	TestEdge1ChainID = "haqq_53211"
	TestEdge2ChainID = "haqq_54211"
	LocalNetChainID  = "haqq_121799"
	// BaseDenom defines the Evmos mainnet denomination
	BaseDenom = "aislm"
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
