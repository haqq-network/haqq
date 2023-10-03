package types

import (
	gethaccounts "github.com/ethereum/go-ethereum/accounts"
)

var (
	// Bip44CoinType satisfies EIP84. See https://github.com/ethereum/EIPs/issues/84 for more info.
	Bip44CoinType uint32 = 60

	// BIP44HDPath is the default BIP44 HD path used on Ethereum.
	BIP44HDPath = gethaccounts.DefaultBaseDerivationPath.String()
)

type (
	HDPathIterator func() gethaccounts.DerivationPath
)

// NewHDPathIterator receives a base path as a string and a boolean for the desired iterator type and
// returns a function that iterates over the base HD path, returning the string.
func NewHDPathIterator(basePath string, ledgerIter bool) (HDPathIterator, error) {
	hdPath, err := gethaccounts.ParseDerivationPath(basePath)
	if err != nil {
		return nil, err
	}

	if ledgerIter {
		return gethaccounts.LedgerLiveIterator(hdPath), nil
	}

	return gethaccounts.DefaultIterator(hdPath), nil
}
