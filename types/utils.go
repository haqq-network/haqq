package types

import "strings"

const (
	// MainnetChainID defines the Haqq EIP155 chain ID for main network
	MainNetChainID = "haqq_11235"
	// TestnetChainID defines the Haqq EIP155 chain ID for testedge network
	TestEdgeNetChainID = "haqq_53211"
	// LocalNetChainID defines the Haqq EIP155 chain ID for local network
	LocalNetChainID = "haqq_121799"
)

func IsMainNetwork(chainID string) bool {
	return strings.HasPrefix(chainID, MainNetChainID)
}

func IsTestEdgeNetwork(chainID string) bool {
	return strings.HasPrefix(chainID, TestEdgeNetChainID)
}

func IsLocalNetwork(chainID string) bool {
	return strings.HasPrefix(chainID, LocalNetChainID)
}
