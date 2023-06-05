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
