package types

const (
	MainNetChainID     = "haqq_11235-1"
	TestEdgeNetChainID = "haqq_53211-1"
	LocalNetChainID    = "haqq_121799-1"
)

func IsMainNetwork(chainID string) bool {
	return chainID == MainNetChainID
}

func IsTestEdgeNetwork(chainID string) bool {
	return chainID == TestEdgeNetChainID
}

func IsLocalNetwork(chainID string) bool {
	return chainID == LocalNetChainID
}
