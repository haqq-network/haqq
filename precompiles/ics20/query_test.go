package ics20_test

import (
	"fmt"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	"github.com/cosmos/cosmos-sdk/types/query"
	haqqibctesting "github.com/haqq-network/haqq/ibc/testing"
	"github.com/haqq-network/haqq/precompiles/authorization"
	cmn "github.com/haqq-network/haqq/precompiles/common"
	"github.com/haqq-network/haqq/precompiles/ics20"
	"github.com/haqq-network/haqq/utils"
)

func (s *PrecompileTestSuite) TestDenom() {
	var expTrace types.Denom
	method := s.precompile.Methods[ics20.DenomMethod]
	testCases := []struct {
		name        string
		malleate    func() []interface{}
		postCheck   func(data []byte, inputArgs []interface{})
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty args",
			func() []interface{} { return []interface{}{} },
			func([]byte, []interface{}) {},
			200000,
			true,
			"invalid input arguments",
		},
		{
			"fail - invalid denom trace",
			func() []interface{} {
				return []interface{}{"invalid denom trace"}
			},
			func([]byte, []interface{}) {},
			200000,
			true,
			"invalid denom trace",
		},
		{
			"success - denom trace not found, return empty struct",
			func() []interface{} {
				expTrace = types.ExtractDenomFromPath("transfer/channelToA/transfer/channelToB/" + utils.BaseDenom)
				return []interface{}{expTrace.Hash().String()}
			},
			func(data []byte, _ []interface{}) {
				var out ics20.DenomResponse
				err := s.precompile.UnpackIntoInterface(&out, ics20.DenomMethod, data)
				s.Require().NoError(err, "failed to unpack output", err)
				s.Require().Equal("", out.Denom.Base)
			},
			200000,
			false,
			"",
		},
		{
			"success - denom trace",
			func() []interface{} {
				expTrace = types.ExtractDenomFromPath("transfer/channelToA/transfer/channelToB/" + utils.BaseDenom)
				s.network.App.TransferKeeper.SetDenom(s.network.GetContext(), expTrace)
				return []interface{}{expTrace.Hash().String()}
			},
			func(data []byte, _ []interface{}) {
				var out ics20.DenomResponse
				err := s.precompile.UnpackIntoInterface(&out, ics20.DenomMethod, data)
				s.Require().NoError(err, "failed to unpack output", err)
				s.Require().Equal(expTrace.Base, out.Denom.Base)
			},
			200000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			contract := s.NewPrecompileContract(tc.gas)
			args := tc.malleate()
			bz, err := s.precompile.Denom(s.network.GetContext(), contract, &method, args)

			if tc.expError {
				s.Require().ErrorContains(err, tc.errContains)
				s.Require().Empty(bz)
			} else {
				s.Require().NoError(err)
				tc.postCheck(bz, args)
			}
		})
	}
}

func (s *PrecompileTestSuite) TestDenoms() {
	var expTraces []types.Denom
	method := s.precompile.Methods[ics20.DenomsMethod]
	testCases := []struct {
		name        string
		malleate    func() []interface{}
		postCheck   func(data []byte, inputArgs []interface{})
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty args",
			func() []interface{} { return []interface{}{} },
			func([]byte, []interface{}) {},
			200000,
			true,
			"invalid number of arguments",
		},
		{
			"success - gets denom traces",
			func() []interface{} {
				expTraces = []types.Denom{
					types.NewDenom(utils.BaseDenom),
					types.ExtractDenomFromPath("transfer/channelToA/transfer/channelToB/" + utils.BaseDenom),
					types.ExtractDenomFromPath("transfer/channelToB/" + utils.BaseDenom),
				}

				for _, trace := range expTraces {
					s.network.App.TransferKeeper.SetDenom(s.network.GetContext(), trace)
				}
				return []interface{}{query.PageRequest{
					Key:        nil,
					Offset:     0,
					Limit:      3,
					CountTotal: true,
					Reverse:    false,
				}}
			},
			func(data []byte, _ []interface{}) {
				var out ics20.DenomsResponse
				err := s.precompile.UnpackIntoInterface(&out, ics20.DenomsMethod, data)
				s.Require().NoError(err, "failed to unpack output", err)
				s.Require().Equal(uint64(3), out.PageResponse.Total)
				s.Require().Equal(3, len(out.Denoms))
				for i, denom := range out.Denoms {
					s.Require().Equal(expTraces[i].Base, denom.Base)
				}
			},
			200000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			contract := s.NewPrecompileContract(tc.gas)
			args := tc.malleate()
			bz, err := s.precompile.Denoms(s.network.GetContext(), contract, &method, args)

			if tc.expError {
				s.Require().ErrorContains(err, tc.errContains)
				s.Require().Empty(bz)
			} else {
				s.Require().NoError(err)
				tc.postCheck(bz, args)
			}
		})
	}
}

func (s *PrecompileTestSuite) TestDenomHash() {
	reqTrace := types.ExtractDenomFromPath("transfer/channelToA/transfer/channelToB/" + utils.BaseDenom)
	method := s.precompile.Methods[ics20.DenomHashMethod]
	testCases := []struct {
		name        string
		malleate    func() []interface{}
		postCheck   func(data []byte, inputArgs []interface{})
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"success - trace not found, returns empty string",
			func() []interface{} { return []interface{}{"transfer/channelToB/transfer/channelToA"} },
			func(data []byte, _ []interface{}) {
				var hash string
				err := s.precompile.UnpackIntoInterface(&hash, ics20.DenomHashMethod, data)
				s.Require().NoError(err, "failed to unpack output", err)
				s.Require().Equal("", hash)
			},
			200000,
			false,
			"",
		},
		{
			"success - get the hash of a denom trace",
			func() []interface{} {
				s.network.App.TransferKeeper.SetDenom(s.network.GetContext(), reqTrace)
				// Use base denom for hash lookup
				return []interface{}{
					reqTrace.Base,
				}
			},
			func(data []byte, _ []interface{}) {
				var hash string
				err := s.precompile.UnpackIntoInterface(&hash, ics20.DenomHashMethod, data)
				s.Require().NoError(err, "failed to unpack output", err)
				s.Require().Equal(reqTrace.Hash().String(), hash)
			},
			200000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			contract := s.NewPrecompileContract(tc.gas)
			args := tc.malleate()

			bz, err := s.precompile.DenomHash(s.network.GetContext(), contract, &method, args)

			if tc.expError {
				s.Require().ErrorContains(err, tc.errContains)
				s.Require().Empty(bz)
			} else {
				s.Require().NoError(err)
				tc.postCheck(bz, args)
			}
		})
	}
}

func (s *PrecompileTestSuite) TestAllowance() {
	var (
		path   = haqqibctesting.NewTransferPath(s.chainA, s.chainB)
		path2  = haqqibctesting.NewTransferPath(s.chainA, s.chainB)
		paths  = []*haqqibctesting.Path{path, path2}
		method = s.precompile.Methods[authorization.AllowanceMethod]
	)
	// set channel, otherwise is "" and throws error
	path.EndpointA.ChannelID = "channel-0"
	path2.EndpointA.ChannelID = "channel-1"

	testCases := []struct {
		name        string
		malleate    func() []interface{}
		postCheck   func(bz []byte)
		gas         uint64
		expErr      bool
		errContains string
	}{
		{
			"fail - empty input args",
			func() []interface{} {
				return []interface{}{}
			},
			func([]byte) {},
			100000,
			true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 3, 1),
		},
		{
			"success - no allowance == empty array",
			func() []interface{} {
				return []interface{}{
					s.keyring.GetAddr(0),
					differentAddress,
				}
			},
			func(bz []byte) {
				var allocations []cmn.ICS20Allocation
				err := s.precompile.UnpackIntoInterface(&allocations, authorization.AllowanceMethod, bz)
				s.Require().NoError(err, "failed to unpack output")
				s.Require().Len(allocations, 0)
			},
			100000,
			false,
			"",
		},
		{
			"success - auth with one allocation",
			func() []interface{} {
				err := s.NewTransferAuthorization(
					s.network.GetContext(),
					s.network.App,
					differentAddress,
					s.keyring.GetAddr(0),
					path,
					defaultCoins,
					[]string{s.chainB.SenderAccount.GetAddress().String()},
					[]string{"memo"},
				)
				s.Require().NoError(err)

				return []interface{}{
					differentAddress,
					s.keyring.GetAddr(0),
				}
			},
			func(bz []byte) {
				expAllocs := []cmn.ICS20Allocation{
					{
						SourcePort:        path.EndpointA.ChannelConfig.PortID,
						SourceChannel:     path.EndpointA.ChannelID,
						SpendLimit:        defaultCmnCoins,
						AllowList:         []string{s.chainB.SenderAccount.GetAddress().String()},
						AllowedPacketData: []string{"memo"},
					},
				}

				var allocations []cmn.ICS20Allocation
				err := s.precompile.UnpackIntoInterface(&allocations, authorization.AllowanceMethod, bz)
				s.Require().NoError(err, "failed to unpack output")

				s.Require().Equal(expAllocs, allocations)
			},
			100000,
			false,
			"",
		},
		{
			"success - auth with multiple allocations",
			func() []interface{} {
				allocs := make([]types.Allocation, len(paths))
				for i, p := range paths {
					allocs[i] = types.Allocation{
						SourcePort:        p.EndpointA.ChannelConfig.PortID,
						SourceChannel:     p.EndpointA.ChannelID,
						SpendLimit:        mutliSpendLimit,
						AllowList:         []string{s.chainB.SenderAccount.GetAddress().String()},
						AllowedPacketData: []string{"memo"},
					}
				}

				err := s.NewTransferAuthorizationWithAllocations(
					s.network.GetContext(),
					s.network.App,
					differentAddress,
					s.keyring.GetAddr(0),
					allocs,
				)
				s.Require().NoError(err)

				return []interface{}{
					differentAddress,
					s.keyring.GetAddr(0),
				}
			},
			func(bz []byte) {
				expAllocs := make([]cmn.ICS20Allocation, len(paths))
				for i, p := range paths {
					expAllocs[i] = cmn.ICS20Allocation{
						SourcePort:        p.EndpointA.ChannelConfig.PortID,
						SourceChannel:     p.EndpointA.ChannelID,
						SpendLimit:        mutliCmnCoins,
						AllowList:         []string{s.chainB.SenderAccount.GetAddress().String()},
						AllowedPacketData: []string{"memo"},
					}
				}

				var allocations []cmn.ICS20Allocation
				err := s.precompile.UnpackIntoInterface(&allocations, authorization.AllowanceMethod, bz)
				s.Require().NoError(err, "failed to unpack output")

				s.Require().Equal(expAllocs, allocations)
			},
			100000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			args := tc.malleate()
			bz, err := s.precompile.Allowance(s.network.GetContext(), &method, args)

			if tc.expErr {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errContains)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(bz)
				tc.postCheck(bz)
			}
		})
	}
}
