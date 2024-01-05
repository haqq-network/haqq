package cli_test

import (
	"fmt"
	"testing"

	tmcli "github.com/cometbft/cometbft/libs/cli"
	testcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/haqq-network/haqq/testutil/network"
	"github.com/haqq-network/haqq/x/erc20/client/cli"
	"github.com/haqq-network/haqq/x/erc20/types"
)

type ERC20QueryTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

func TestERC20QueryTestSuite(t *testing.T) {
	suite.Run(t, new(ERC20QueryTestSuite))
}

func (s *ERC20QueryTestSuite) SetupSuite() {
	cfg := network.DefaultConfig()
	cfg.NumValidators = 1
	erc20Genesis := types.DefaultGenesisState()
	erc20Genesis.TokenPairs = []types.TokenPair{
		types.NewTokenPair(
			common.HexToAddress("0x80b5a32E4F032B2a058b4F29EC95EEfEEB87aDcd"),
			"utesttoken",
			types.OWNER_MODULE,
		),
		types.NewTokenPair(
			common.HexToAddress("0xd567B3d7B8FE3C79a1AD8dA978812cfC4Fa05e75"),
			"utesttoken1",
			types.OWNER_EXTERNAL,
		),
	}
	g, err := cfg.Codec.MarshalJSON(erc20Genesis)
	s.Require().NoError(err)
	cfg.GenesisState[types.ModuleName] = g
	s.cfg = cfg

	basedir := s.T().TempDir()
	s.network, err = network.New(s.T(), basedir, cfg)
	s.Require().NoError(err)

	_, err = s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

func (s *ERC20QueryTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *ERC20QueryTestSuite) TestGetParams() {
	out, err := testcli.ExecTestCLICmd(s.network.Validators[0].ClientCtx, cli.GetParamsCmd(), []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	})
	s.Require().NoError(err)
	var resp types.QueryParamsResponse
	s.Require().NoError(s.network.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
	s.Require().True(resp.Params.GetEnableErc20())
	s.Require().True(resp.Params.GetEnableEVMHook())
}

func (s *ERC20QueryTestSuite) TestGetTokenPairs() {
	out, err := testcli.ExecTestCLICmd(s.network.Validators[0].ClientCtx, cli.GetTokenPairsCmd(), []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	})
	s.Require().NoError(err)
	var resp types.QueryTokenPairsResponse
	s.Require().NoError(s.network.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
	s.Require().True(resp.Pagination.Total == 2)
}

func (s *ERC20QueryTestSuite) TestGetTokenPair() {
	jsonFlag := fmt.Sprintf("--%s=json", tmcli.OutputFlag)
	for _, tc := range []struct {
		desc string
		args []string
		err  error
	}{
		{
			desc: "get token pair",
			args: []string{"utesttoken"},
			err:  nil,
		},
		{
			desc: "invalid token",
			args: []string{"-"},
			err:  status.Error(codes.InvalidArgument, "rpc error: code = InvalidArgument desc = invalid format for token -, should be either hex ('0x...') cosmos denom: invalid request"),
		},
		{
			desc: "token pair not found",
			args: []string{"notfound"},
			err:  status.Error(codes.NotFound, "rpc error: code = NotFound desc = token pair with token 'notfound': key not found"),
		},
	} {
		s.Run(tc.desc, func() {
			out, err := testcli.ExecTestCLICmd(
				s.network.Validators[0].ClientCtx,
				cli.GetTokenPairCmd(),
				append(tc.args, jsonFlag),
			)
			if tc.err != nil {
				se, ok := status.FromError(err)
				s.Require().True(ok)
				s.Require().ErrorIs(se.Err(), tc.err)
			} else {
				s.Require().NoError(err)
				var resp types.QueryTokenPairResponse
				s.Require().NoError(s.network.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
			}
		})
	}
}
