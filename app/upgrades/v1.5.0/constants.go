package v150

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
)

const (
	// UpgradeName is the shared upgrade plan name for mainnet and testnet
	UpgradeName = "v1.5.0"

	// internal constants
	numberOfPeriods      = 720
	cliffSixMonthsPeriod = int64(15552000) // 6 months in seconds
	cliffOneYearPeriod   = int64(31104000) // 1 year in seconds
	unlockPeriod         = int64(86400)    // 24 hours in seconds
	exp                  = uint64(10e17)   // 1 ISLM base
	threshold            = 1000000         // 1000000 ISLM for prod
	vestingContract      = "0x1ba8624B418BFfaA93c369B0841204a9d50fA4D5"
	longCliffAddress1    = "haqq1497ds93u23varcq32gg635c6tkvslqllgxfy5q"
	longCliffAddress2    = "haqq1uzleh48vrx26z5mpxdjzzxfp3gv3wwlfzvdkhn"
	longCliffAddress3    = "haqq1kawnyp8w7ydk9fgtvp7m8t8kqf0vypykr3rj7v"
)

var (
	//go:embed bank.gz
	bankStateGZ   []byte // nolint: golint
	bankStateJSON []byte // nolint: golint

	//go:embed staking.gz
	stakingStateGZ   []byte // nolint: golint
	stakingStateJSON []byte // nolint: golint
)

func init() {
	bankGzipReader, err := gzip.NewReader(bytes.NewBuffer(bankStateGZ))
	if err != nil {
		panic(err)
	}
	defer bankGzipReader.Close()

	bankStateJSON, err = io.ReadAll(bankGzipReader)
	if err != nil {
		panic(err)
	}

	stakingGzipReader, err := gzip.NewReader(bytes.NewBuffer(stakingStateGZ))
	if err != nil {
		panic(err)
	}
	defer stakingGzipReader.Close()

	stakingStateJSON, err = io.ReadAll(stakingGzipReader)
	if err != nil {
		panic(err)
	}
}

// getIgnoreList returns a static predefined list of addresses that should be ignored by the revesting upgrade
func getIgnoreList() map[string]bool {
	return map[string]bool{
		// -- Vesting contract
		// Contract Mainnet: 0x1ba8624B418BFfaA93c369B0841204a9d50fA4D5
		"haqq1rw5xyj6p30l64y7rdxcggysy482slfx4tzkapq": true,
		// Contract ProxyAdmin Mainnet: 0x29876c4A2D095A9eBBE8fD1D8432C5c7f6f9DE35
		"haqq19xrkcj3dp9dfawlgl5wcgvk9clm0nh3458hqhk": true,

		// --- Static whitelist
		// biz-non-vested: 0x133EF7227BA0f57578c695f7057bF65F88cf4Ce0
		"haqq1zvl0wgnm5r6h27xxjhms27lkt7yv7n8qna6cp2": true,
		// priv sale: 0x665b5b43e3327259E8e4756E6c5FEEeA5929E40D
		"haqq1ved4kslrxfe9n68yw4hxchlwafvjneqdaguwe2": true,
		// partners: 0xC7F57Cb872fd843bC8a9410A9B26e1C4807Dd159
		"haqq1cl6hewrjlkzrhj9fgy9fkfhpcjq8m52ev53zg0": true,
		// Evergreen DAO (distribution): 0x93354845030274cD4bf1686Abd60AB28EC52e1a7
		"haqq1jv65s3grqf6v6jl3dp4t6c9t9rk99cd89c30hf": true,
	}
}
