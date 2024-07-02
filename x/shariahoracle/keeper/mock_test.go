package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	evm "github.com/haqq-network/haqq/x/evm/types"
	"github.com/haqq-network/haqq/x/shariahoracle/types"
	"github.com/stretchr/testify/mock"
)

var _ types.ERC20Keeper = &MockERC20Keeper{}

type MockERC20Keeper struct {
	mock.Mock
}

func (m *MockERC20Keeper) CallEVM(_ sdk.Context, _ abi.ABI, _ common.Address, _ common.Address, _ bool, _ string, _ ...interface{}) (*evm.MsgEthereumTxResponse, error) {
	args := m.Called(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*evm.MsgEthereumTxResponse), args.Error(1)
}
