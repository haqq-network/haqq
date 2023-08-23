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
	numberOfPeriods = 720
	cliffPeriod     = int64(120)    // 2 minutes in seconds TODO set to 15552000 (6 months in seconds)
	unlockPeriod    = int64(15)     // 15 seconds TODO set to 86400 (24 hours in seconds)
	exp             = uint64(10e17) // 1 ISLM base
	threshold       = 1000000       // 1000000 ISLM for prod
	vestingContract = "0x1ba8624B418BFfaA93c369B0841204a9d50fA4D5"
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
		"haqq196srgtdaqrhqehdx36hfacrwmhlfznwpt78rct": true, // Team account
		"haqq1rw5xyj6p30l64y7rdxcggysy482slfx4tzkapq": true, // Vesting Contract Mainnet: 0x1ba8624B418BFfaA93c369B0841204a9d50fA4D5
		"haqq19xrkcj3dp9dfawlgl5wcgvk9clm0nh3458hqhk": true, // Vesting Contract ProxyAdmin Mainnet: 0x29876c4A2D095A9eBBE8fD1D8432C5c7f6f9DE35
		// Static whitelist
		"haqq1yljf766n5n7j9dljwyh2duunjkw3jhdzpg68kw": true, // biz-msig: 0x27e49f6B53A4fD22B7F2712EA6F393959D195Da2
		"haqq1xw804zanweujx2fxc40mhdl65ku0ksplnwrmv2": true, // biz-msig-gnosis: 0x338efA8BB37679232926c55FbbB7Faa5b8FB403f
		"haqq1jy4rhr8kqr9a6u0lcs3yaqzszgs5sg6x38xqds": true, // biz-msig-staking: 0x912A3b8cF600CbDD71ffC4224e80501221482346
		"haqq1ved4kslrxfe9n68yw4hxchlwafvjneqdaguwe2": true, // priv sale: 0x665b5b43e3327259E8e4756E6c5FEEeA5929E40D
		"haqq1cl6hewrjlkzrhj9fgy9fkfhpcjq8m52ev53zg0": true, // partners: 0xC7F57Cb872fd843bC8a9410A9B26e1C4807Dd159
		// Valop
		// "haqq1jh375g33t6l3kd5wjhmscju2kyfezfkjyj5n4p": true, // Main Validator
	}
}
