package cli_test

import (
	"context"
	"fmt"
	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	testutilmod "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/x/erc20"
	"github.com/haqq-network/haqq/x/erc20/client/cli"
	"github.com/stretchr/testify/suite"
	"io"
	"testing"
)

type ERC20TxTestSuite struct {
	suite.Suite

	kr      keyring.Keyring
	encCfg  testutilmod.TestEncodingConfig
	baseCtx client.Context
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(ERC20TxTestSuite))
}

func (s *ERC20TxTestSuite) SetupSuite() {
	s.encCfg = testutilmod.MakeTestEncodingConfig(erc20.AppModuleBasic{})
	s.kr = keyring.NewInMemory(s.encCfg.Codec)

	s.baseCtx = client.Context{}.
		WithKeyring(s.kr).
		WithTxConfig(s.encCfg.TxConfig).
		WithCodec(s.encCfg.Codec).
		WithClient(clitestutil.MockTendermintRPC{Client: rpcclientmock.Client{}}).
		WithAccountRetriever(client.MockAccountRetriever{}).
		WithOutput(io.Discard)
}

func (s *ERC20TxTestSuite) TestConvertCoinCmd() {
	accounts := testutil.CreateKeyringAccounts(s.T(), s.kr, 2)
	cmd := cli.NewConvertCoinCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	extraArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
	}

	testCases := []struct {
		name      string
		ctxGen    func() client.Context
		args      []string
		extraArgs []string
		expectErr bool
		from      string
	}{
		{
			name: "valid transaction",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"1000testdenom",
			},
			extraArgs: extraArgs,
			expectErr: false,
			from:      accounts[0].Address.String(),
		},
		{
			name: "valid transaction with receiver",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"1000testdenom",
				common.BytesToAddress(accounts[1].Address).Hex(),
			},
			extraArgs: extraArgs,
			expectErr: false,
			from:      accounts[0].Address.String(),
		},
		{
			name: "invalid coin",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"invalidcoin1337",
			},
			extraArgs: extraArgs,
			expectErr: true,
			from:      accounts[0].Address.String(),
		},
		{
			name: "invalid receiver address",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"1000testdenom",
				"ivalid_receiver",
			},
			extraArgs: extraArgs,
			expectErr: true,
			from:      accounts[0].Address.String(),
		},
		{
			name: "invalid context",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"1000testdenom",
			},
			extraArgs: extraArgs,
			expectErr: true,
			from:      "invalid",
		},
		{
			name: "invalid sender",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"1000testdenom",
			},
			extraArgs: extraArgs,
			expectErr: true,
			from:      "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ctx := svrcmd.CreateExecuteContext(context.Background())
			cmd.SetContext(ctx)

			args := tc.extraArgs
			if tc.from != "" {
				args = append(tc.extraArgs, fmt.Sprintf("--from=%s", tc.from))
			}
			cmd.SetArgs(append(tc.args, args...))

			s.Require().NoError(client.SetCmdClientContextHandler(tc.ctxGen(), cmd))

			err := cmd.Execute()
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ERC20TxTestSuite) TestConvertERC20Cmd() {
	accounts := testutil.CreateKeyringAccounts(s.T(), s.kr, 2)
	cmd := cli.NewConvertERC20Cmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	extraArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
	}

	testCases := []struct {
		name      string
		ctxGen    func() client.Context
		args      []string
		extraArgs []string
		expectErr bool
		from      string
	}{
		{
			name: "valid transaction",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"0xdAC17F958D2ee523a2206206994597C13D831ec7",
				"1000",
			},
			extraArgs: extraArgs,
			expectErr: false,
			from:      accounts[0].Address.String(),
		},
		{
			name: "valid transaction with receiver",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"0xdAC17F958D2ee523a2206206994597C13D831ec7",
				"1000",
				accounts[1].Address.String(),
			},
			extraArgs: extraArgs,
			expectErr: false,
			from:      accounts[0].Address.String(),
		},
		{
			name: "invalid contract address",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"invalid_address",
				"1000",
			},
			extraArgs: extraArgs,
			expectErr: true,
			from:      accounts[0].Address.String(),
		},
		{
			name: "invalid amount",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"0xdAC17F958D2ee523a2206206994597C13D831ec7",
				"invalid_amount",
			},
			extraArgs: extraArgs,
			expectErr: true,
			from:      accounts[0].Address.String(),
		},
		{
			name: "invalid context",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"0xdAC17F958D2ee523a2206206994597C13D831ec7",
				"1000",
			},
			extraArgs: extraArgs,
			expectErr: true,
			from:      "invalid",
		},
		{
			name: "invalid default receiver",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"0xdAC17F958D2ee523a2206206994597C13D831ec7",
				"1000",
			},
			extraArgs: extraArgs,
			expectErr: true,
			from:      "",
		},
		{
			name: "invalid receiver",
			ctxGen: func() client.Context {
				return s.baseCtx
			},
			args: []string{
				"0xdAC17F958D2ee523a2206206994597C13D831ec7",
				"1000",
				"invalid",
			},
			extraArgs: extraArgs,
			expectErr: true,
			from:      accounts[0].Address.String(),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ctx := svrcmd.CreateExecuteContext(context.Background())
			cmd.SetContext(ctx)

			args := tc.extraArgs
			if tc.from != "" {
				args = append(tc.extraArgs, fmt.Sprintf("--from=%s", tc.from))
			}
			cmd.SetArgs(append(tc.args, args...))

			s.Require().NoError(client.SetCmdClientContextHandler(tc.ctxGen(), cmd))

			err := cmd.Execute()
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ERC20TxTestSuite) TestRegisterCoinProposalCmd() {

}
