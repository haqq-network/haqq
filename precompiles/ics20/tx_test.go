package ics20_test

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/ics20"
	evmosutil "github.com/haqq-network/haqq/testutil"
	testutiltx "github.com/haqq-network/haqq/testutil/tx"
	"github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/evm/core/vm"
)

var (
	differentAddress = testutiltx.GenerateAddress()
	// thirdPartyAddress is neither the calling contract nor the transaction signer, so a
	// transfer naming it as the sender exercises the guard that keeps a caller from acting
	// for an account that did not sign the transaction.
	thirdPartyAddress       = testutiltx.GenerateAddress()
	amt               int64 = 1000000000000000000
	expBal                  = "99997650000000000000000" // initial balance is 100000 ISLM (minus transfer, minus fees, etc.)
)

func (s *PrecompileTestSuite) TestTransfer() {
	var ctx sdk.Context

	callingContractAddr := differentAddress
	method := s.precompile.Methods[ics20.TransferMethod]
	testCases := []struct {
		name        string
		malleate    func(sender, receiver sdk.AccAddress) []interface{}
		postCheck   func(sender, receiver sdk.AccAddress, data []byte, inputArgs []interface{})
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty args",
			func(sdk.AccAddress, sdk.AccAddress) []interface{} {
				return []interface{}{}
			},
			func(sdk.AccAddress, sdk.AccAddress, []byte, []interface{}) {
			},
			200000,
			true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 9, 0),
		},
		{
			"fail - no transfer authorization",
			func(sdk.AccAddress, sdk.AccAddress) []interface{} {
				path := s.coordinator.Setup(s.chainA.ChainID, s.chainB.ChainID)
				return []interface{}{
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					utils.BaseDenom,
					big.NewInt(1e18),
					common.BytesToAddress(s.chainA.SenderAccount.GetAddress().Bytes()),
					s.chainB.SenderAccount.GetAddress().String(),
					s.chainB.GetTimeoutHeight(),
					uint64(0),
					"memo",
				}
			},
			func(sdk.AccAddress, sdk.AccAddress, []byte, []interface{}) {
			},
			200000,
			true,
			"does not exist",
		},
		{
			"fail - channel does not exist",
			func(sdk.AccAddress, sdk.AccAddress) []interface{} {
				return []interface{}{
					"port",
					"channel-01",
					utils.BaseDenom,
					big.NewInt(1e18),
					common.BytesToAddress(s.chainA.SenderAccount.GetAddress().Bytes()),
					s.chainB.SenderAccount.GetAddress().String(),
					s.chainB.GetTimeoutHeight(),
					uint64(0),
					"memo",
				}
			},
			func(sdk.AccAddress, sdk.AccAddress, []byte, []interface{}) {
			},
			200000,
			true,
			channeltypes.ErrChannelNotFound.Error(),
		},
		{
			"fail - non authorized denom",
			func(sender, _ sdk.AccAddress) []interface{} {
				path := s.coordinator.Setup(s.chainA.ChainID, s.chainB.ChainID)
				err := s.NewTransferAuthorization(ctx, s.network.App, callingContractAddr, common.BytesToAddress(sender), path, defaultCoins, nil, []string{"memo"})
				s.Require().NoError(err)
				return []interface{}{
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					"uatom",
					big.NewInt(1e18),
					common.BytesToAddress(s.chainA.SenderAccount.GetAddress().Bytes()),
					s.chainB.SenderAccount.GetAddress().String(),
					s.chainB.GetTimeoutHeight(),
					uint64(0),
					"memo",
				}
			},
			func(sdk.AccAddress, sdk.AccAddress, []byte, []interface{}) {
			},
			200000,
			true,
			"requested amount is more than spend limit",
		},
		{
			"fail - allowance is less than transfer amount",
			func(sender, _ sdk.AccAddress) []interface{} {
				path := s.coordinator.Setup(s.chainA.ChainID, s.chainB.ChainID)
				err := s.NewTransferAuthorization(ctx, s.network.App, callingContractAddr, common.BytesToAddress(sender), path, defaultCoins, nil, []string{"memo"})
				s.Require().NoError(err)
				return []interface{}{
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					utils.BaseDenom,
					big.NewInt(2e18),
					common.BytesToAddress(s.chainA.SenderAccount.GetAddress().Bytes()),
					s.chainB.SenderAccount.GetAddress().String(),
					s.chainB.GetTimeoutHeight(),
					uint64(0),
					"memo",
				}
			},
			func(sdk.AccAddress, sdk.AccAddress, []byte, []interface{}) {
			},
			200000,
			true,
			"requested amount is more than spend limit",
		},
		{
			// The name used to be wrong: the sender argument was the calling contract itself,
			// so the contract was moving its own coins. Here the sender really is a third
			// party -- neither the caller nor the transaction signer -- and it has even
			// granted the caller a transfer authorization. The call must still be rejected,
			// because the precompile requires the sender to have signed the transaction.
			// That guard is what keeps a grant from being spent in a transaction its granter
			// never authorized.
			"fail - transfer 1 ISLM from somebody else's account, even holding their grant",
			func(_, receiver sdk.AccAddress) []interface{} {
				path := s.coordinator.Setup(s.chainA.ChainID, s.chainB.ChainID)
				err := s.NewTransferAuthorization(ctx, s.network.App, callingContractAddr, thirdPartyAddress, path, defaultCoins, nil, []string{"memo"})
				s.Require().NoError(err)
				// fund the third party's account
				err = evmosutil.FundAccountWithBaseDenom(ctx, s.network.App.BankKeeper, thirdPartyAddress.Bytes(), amt)
				s.Require().NoError(err)

				return []interface{}{
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					utils.BaseDenom,
					big.NewInt(amt),
					thirdPartyAddress,
					receiver.String(),
					s.chainB.GetTimeoutHeight(),
					uint64(0),
					"memo",
				}
			},
			func(_, _ sdk.AccAddress, _ []byte, _ []interface{}) {
				// the third party's grant is neither consumed nor deleted
				auth, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, callingContractAddr.Bytes(), thirdPartyAddress.Bytes(), ics20.TransferMsgURL)
				s.Require().NotNil(auth, "the third party's grant must survive a rejected call")
				transferAuthz, ok := auth.(*transfertypes.TransferAuthorization)
				s.Require().True(ok)
				s.Require().Equal(defaultCoins, transferAuthz.Allocations[0].SpendLimit)

				// and their balance is untouched
				balance := s.network.App.BankKeeper.GetBalance(ctx, thirdPartyAddress.Bytes(), utils.BaseDenom)
				s.Require().Equal(math.NewInt(amt), balance.Amount)
				s.Require().Equal(utils.BaseDenom, balance.Denom)
			},
			200000,
			true,
			"is not the same as sender address",
		},
		{
			"fail - transfer with memo string, but authorization does not allows it",
			func(sender, receiver sdk.AccAddress) []interface{} {
				path := s.coordinator.Setup(s.chainA.ChainID, s.chainB.ChainID)
				err := s.NewTransferAuthorization(ctx, s.network.App, callingContractAddr, common.BytesToAddress(sender), path, defaultCoins, nil, nil)
				s.Require().NoError(err)
				return []interface{}{
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					utils.BaseDenom,
					big.NewInt(1e18),
					common.BytesToAddress(sender.Bytes()),
					receiver.String(),
					s.chainB.GetTimeoutHeight(),
					uint64(0),
					"memo",
				}
			},
			func(sender, _ sdk.AccAddress, _ []byte, _ []interface{}) {
				// Check allowance remains unchanged
				authz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, callingContractAddr.Bytes(), sender, ics20.TransferMsgURL)
				transferAuthz := authz.(*transfertypes.TransferAuthorization)
				s.Require().Equal(transferAuthz.Allocations[0].SpendLimit, defaultCoins)
			},
			200000,
			true,
			"memo must be empty because allowed packet data in allocation is empty",
		},
		{
			"pass - transfer 1 ISLM from chainA to chainB and spend the entire allowance",
			func(sender, receiver sdk.AccAddress) []interface{} {
				path := s.coordinator.Setup(s.chainA.ChainID, s.chainB.ChainID)
				err := s.NewTransferAuthorization(ctx, s.network.App, callingContractAddr, common.BytesToAddress(sender), path, defaultCoins, nil, []string{"memo"})
				s.Require().NoError(err)
				return []interface{}{
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					utils.BaseDenom,
					big.NewInt(1e18),
					common.BytesToAddress(sender.Bytes()),
					receiver.String(),
					s.chainB.GetTimeoutHeight(),
					uint64(0),
					"memo",
				}
			},
			func(sender, _ sdk.AccAddress, _ []byte, _ []interface{}) {
				// Check allowance was deleted
				authz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, callingContractAddr.Bytes(), sender, ics20.TransferMsgURL)
				s.Require().Nil(authz)

				balance := s.network.App.BankKeeper.GetBalance(ctx, s.chainA.SenderAccount.GetAddress(), utils.BaseDenom)
				s.Require().Equal(expBal, balance.Amount.String())
				s.Require().Equal(utils.BaseDenom, balance.Denom)
			},
			200000,
			false,
			"",
		},
		{
			"pass - transfer 1 ISLM from chainA to chainB and don't change the unlimited spending limit",
			func(sender, receiver sdk.AccAddress) []interface{} {
				path := s.coordinator.Setup(s.chainA.ChainID, s.chainB.ChainID)
				err := s.NewTransferAuthorization(ctx, s.network.App, callingContractAddr, common.BytesToAddress(sender), path, maxUint256Coins, nil, []string{"memo"})
				s.Require().NoError(err)
				return []interface{}{
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					utils.BaseDenom,
					big.NewInt(1e18),
					common.BytesToAddress(sender.Bytes()),
					receiver.String(),
					s.chainB.GetTimeoutHeight(),
					uint64(0),
					"memo",
				}
			},
			func(sender, _ sdk.AccAddress, _ []byte, _ []interface{}) {
				// The allowance is spent after the transfer thus the authorization is deleted
				authz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, callingContractAddr.Bytes(), sender, ics20.TransferMsgURL)
				transferAuthz := authz.(*transfertypes.TransferAuthorization)
				s.Require().Equal(transferAuthz.Allocations[0].SpendLimit, maxUint256Coins)

				balance := s.network.App.BankKeeper.GetBalance(ctx, s.chainA.SenderAccount.GetAddress(), utils.BaseDenom)
				s.Require().Equal(expBal, balance.Amount.String())
				s.Require().Equal(utils.BaseDenom, balance.Denom)
			},
			200000,
			false,
			"",
		},
		{
			"pass - transfer 1 ISLM from chainA to chainB and only change 1 spend limit",
			func(sender, receiver sdk.AccAddress) []interface{} {
				path := s.coordinator.Setup(s.chainA.ChainID, s.chainB.ChainID)
				err := s.NewTransferAuthorization(ctx, s.network.App, callingContractAddr, common.BytesToAddress(sender), path, mutliSpendLimit, nil, []string{"memo"})
				s.Require().NoError(err)
				return []interface{}{
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					utils.BaseDenom,
					big.NewInt(1e18),
					common.BytesToAddress(sender.Bytes()),
					receiver.String(),
					s.chainB.GetTimeoutHeight(),
					uint64(0),
					"memo",
				}
			},
			func(sender, _ sdk.AccAddress, _ []byte, _ []interface{}) {
				// The allowance is spent after the transfer thus the authorization is deleted
				authz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, callingContractAddr.Bytes(), sender, ics20.TransferMsgURL)
				transferAuthz := authz.(*transfertypes.TransferAuthorization)
				s.Require().Equal(transferAuthz.Allocations[0].SpendLimit, atomCoins)

				balance := s.network.App.BankKeeper.GetBalance(ctx, s.chainA.SenderAccount.GetAddress(), utils.BaseDenom)
				s.Require().Equal(expBal, balance.Amount.String())
				s.Require().Equal(utils.BaseDenom, balance.Denom)
			},
			200000,
			false,
			"",
		},
		{
			"pass - transfer 1 ISLM from chainA to chainB and only change 1 spend limit for the associated allocation",
			func(sender, receiver sdk.AccAddress) []interface{} {
				path := s.coordinator.Setup(s.chainA.ChainID, s.chainB.ChainID)
				allocations := []transfertypes.Allocation{
					{
						SourcePort:        "port-01",
						SourceChannel:     "channel-03",
						SpendLimit:        atomCoins,
						AllowList:         nil,
						AllowedPacketData: []string{"*"}, // allow any memo string

					},
					{
						SourcePort:        path.EndpointA.ChannelConfig.PortID,
						SourceChannel:     path.EndpointA.ChannelID,
						SpendLimit:        defaultCoins,
						AllowList:         nil,
						AllowedPacketData: []string{"*"}, // allow any memo string
					},
				}
				err := s.NewTransferAuthorizationWithAllocations(ctx, s.network.App, callingContractAddr, common.BytesToAddress(sender), allocations)
				s.Require().NoError(err)
				return []interface{}{
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					utils.BaseDenom,
					big.NewInt(1e18),
					common.BytesToAddress(sender.Bytes()),
					receiver.String(),
					s.chainB.GetTimeoutHeight(),
					uint64(0),
					"memo",
				}
			},
			func(sender, _ sdk.AccAddress, _ []byte, _ []interface{}) {
				// The allowance is spent after the transfer thus the authorization is deleted
				authz, _ := s.network.App.AuthzKeeper.GetAuthorization(ctx, callingContractAddr.Bytes(), sender, ics20.TransferMsgURL)
				transferAuthz := authz.(*transfertypes.TransferAuthorization)
				s.Require().Equal(transferAuthz.Allocations[0].SpendLimit, atomCoins)

				balance := s.network.App.BankKeeper.GetBalance(ctx, s.chainA.SenderAccount.GetAddress(), utils.BaseDenom)
				s.Require().Equal(expBal, balance.Amount.String())
				s.Require().Equal(utils.BaseDenom, balance.Denom)
			},
			200000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			sender := s.chainA.SenderAccount.GetAddress()
			receiver := s.chainB.SenderAccount.GetAddress()

			contract := vm.NewContract(vm.AccountRef(common.BytesToAddress(sender)), s.precompile, big.NewInt(0), tc.gas)

			ctx = s.network.GetContext().WithGasMeter(storetypes.NewInfiniteGasMeter())
			initialGas := ctx.GasMeter().GasConsumed()
			s.Require().Zero(initialGas)

			args := tc.malleate(sender, receiver)

			// set the caller address to be another address (so we can test the authorization logic)
			contract.CallerAddress = callingContractAddr
			bz, err := s.precompile.Transfer(ctx, common.BytesToAddress(sender), contract, s.network.GetStateDB(), &method, args)

			if tc.expError {
				s.Require().ErrorContains(err, tc.errContains)
				s.Require().Empty(bz)
				if tc.postCheck != nil {
					tc.postCheck(sender, receiver, bz, args)
				}
			} else {
				s.Require().NoError(err)
				s.Require().Equal(bz, cmn.TrueValue)
				tc.postCheck(sender, receiver, bz, args)
			}
		})
	}
}
