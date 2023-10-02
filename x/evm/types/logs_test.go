package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
)

func TestTransactionLogsValidate(t *testing.T) {
	addr := utiltx.GenerateAddress().String()

	testCases := []struct {
		name    string
		txLogs  evmtypes.TransactionLogs
		expPass bool
	}{
		{
			"valid log",
			evmtypes.TransactionLogs{
				Hash: common.BytesToHash([]byte("tx_hash")).String(),
				Logs: []*evmtypes.Log{
					{
						Address:     addr,
						Topics:      []string{common.BytesToHash([]byte("topic")).String()},
						Data:        []byte("data"),
						BlockNumber: 1,
						TxHash:      common.BytesToHash([]byte("tx_hash")).String(),
						TxIndex:     1,
						BlockHash:   common.BytesToHash([]byte("block_hash")).String(),
						Index:       1,
						Removed:     false,
					},
				},
			},
			true,
		},
		{
			"empty hash",
			evmtypes.TransactionLogs{
				Hash: common.Hash{}.String(),
			},
			false,
		},
		{
			"nil log",
			evmtypes.TransactionLogs{
				Hash: common.BytesToHash([]byte("tx_hash")).String(),
				Logs: []*evmtypes.Log{nil},
			},
			false,
		},
		{
			"invalid log",
			evmtypes.TransactionLogs{
				Hash: common.BytesToHash([]byte("tx_hash")).String(),
				Logs: []*evmtypes.Log{{}},
			},
			false,
		},
		{
			"hash mismatch log",
			evmtypes.TransactionLogs{
				Hash: common.BytesToHash([]byte("tx_hash")).String(),
				Logs: []*evmtypes.Log{
					{
						Address:     addr,
						Topics:      []string{common.BytesToHash([]byte("topic")).String()},
						Data:        []byte("data"),
						BlockNumber: 1,
						TxHash:      common.BytesToHash([]byte("other_hash")).String(),
						TxIndex:     1,
						BlockHash:   common.BytesToHash([]byte("block_hash")).String(),
						Index:       1,
						Removed:     false,
					},
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.txLogs.Validate()
		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}

func TestValidateLog(t *testing.T) {
	addr := utiltx.GenerateAddress().String()

	testCases := []struct {
		name    string
		log     *evmtypes.Log
		expPass bool
	}{
		{
			"valid log",
			&evmtypes.Log{
				Address:     addr,
				Topics:      []string{common.BytesToHash([]byte("topic")).String()},
				Data:        []byte("data"),
				BlockNumber: 1,
				TxHash:      common.BytesToHash([]byte("tx_hash")).String(),
				TxIndex:     1,
				BlockHash:   common.BytesToHash([]byte("block_hash")).String(),
				Index:       1,
				Removed:     false,
			},
			true,
		},
		{
			"empty log", &evmtypes.Log{}, false,
		},
		{
			"zero address",
			&evmtypes.Log{
				Address: common.Address{}.String(),
			},
			false,
		},
		{
			"empty block hash",
			&evmtypes.Log{
				Address:   addr,
				BlockHash: common.Hash{}.String(),
			},
			false,
		},
		{
			"zero block number",
			&evmtypes.Log{
				Address:     addr,
				BlockHash:   common.BytesToHash([]byte("block_hash")).String(),
				BlockNumber: 0,
			},
			false,
		},
		{
			"empty tx hash",
			&evmtypes.Log{
				Address:     addr,
				BlockHash:   common.BytesToHash([]byte("block_hash")).String(),
				BlockNumber: 1,
				TxHash:      common.Hash{}.String(),
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.log.Validate()
		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}

func TestConversionFunctions(t *testing.T) {
	addr := utiltx.GenerateAddress().String()

	txLogs := evmtypes.TransactionLogs{
		Hash: common.BytesToHash([]byte("tx_hash")).String(),
		Logs: []*evmtypes.Log{
			{
				Address:     addr,
				Topics:      []string{common.BytesToHash([]byte("topic")).String()},
				Data:        []byte("data"),
				BlockNumber: 1,
				TxHash:      common.BytesToHash([]byte("tx_hash")).String(),
				TxIndex:     1,
				BlockHash:   common.BytesToHash([]byte("block_hash")).String(),
				Index:       1,
				Removed:     false,
			},
		},
	}

	// convert valid log to eth logs and back (and validate)
	conversionLogs := evmtypes.NewTransactionLogsFromEth(common.BytesToHash([]byte("tx_hash")), txLogs.EthLogs())
	conversionErr := conversionLogs.Validate()

	// create new transaction logs as copy of old valid one (and validate)
	copyLogs := evmtypes.NewTransactionLogs(common.BytesToHash([]byte("tx_hash")), txLogs.Logs)
	copyErr := copyLogs.Validate()

	require.Nil(t, conversionErr)
	require.Nil(t, copyErr)
}
