package bank

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/evmos/ethermint/testutil/network"
	"github.com/stretchr/testify/suite"

	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distcli "github.com/cosmos/cosmos-sdk/x/distribution/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	stakingcli "github.com/cosmos/cosmos-sdk/x/staking/client/cli"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	haqqnetwork "github.com/haqq-network/haqq/testutil/network"
)

type BurnCoinsTestSuite struct {
	suite.Suite

	cfg              network.Config
	network          *network.Network
	govModuleAddress sdk.AccAddress
}

func NewBurnCoinsTestSuite(cfg network.Config) *BurnCoinsTestSuite {
	return &BurnCoinsTestSuite{cfg: cfg}
}

func (s *BurnCoinsTestSuite) SetupSuite() {
	s.T().Log("setting up burn coins test suite")

	var err error
	baseDir := s.T().TempDir()
	s.network, err = network.New(s.T(), baseDir, s.cfg)
	s.NoError(err)

	_, err = s.network.WaitForHeight(1)
	s.NoError(err)

	GOVModuleHexAddress := "7B5FE22B5446F7C62EA27B8BD71CEF94E03F3DF2"
	s.govModuleAddress, err = sdk.AccAddressFromHex(GOVModuleHexAddress)
	s.NoError(err)
}

func (s *BurnCoinsTestSuite) TearDownSuite() {
	s.T().Log("tearing down burn coins test suite")
	s.network.Cleanup()
}

func (s *BurnCoinsTestSuite) TestCase1NoQuorum() {
	latestHeight, err := s.network.LatestHeight()
	s.NoError(err)
	val := s.network.Validators[0]

	// check communityPool state before burn
	communityPoolStateBefore := getCommunityPoolState(val)
	s.True(len(communityPoolStateBefore.Pool) == 0)

	// #################################################################
	// build proposal transaction
	// #################################################################

	proposal := upgradetypes.NewSoftwareUpgradeProposal("test", "test", upgradetypes.Plan{
		Name:   "test",
		Height: latestHeight + 10,
		Info:   "test",
	})

	depositAmount := sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10000000)))
	submitProposalMsg, err := govtypes.NewMsgSubmitProposal(proposal, depositAmount, val.Address)
	s.NoError(err)

	fee := sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(100)))
	txBuilder := s.cfg.TxConfig.NewTxBuilder()
	s.Require().NoError(txBuilder.SetMsgs(submitProposalMsg))
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetGasLimit(2000000)

	kb := val.ClientCtx.Keyring
	s.NoError(err)

	txFactory := tx.Factory{}
	txFactory = txFactory.
		WithChainID(s.cfg.ChainID).
		WithKeybase(kb).
		WithTxConfig(s.cfg.TxConfig).
		WithSequence(1)

	err = tx.Sign(txFactory, "node0", txBuilder, true)
	s.NoError(err)

	txBytes, err := s.cfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	s.NoError(err)

	// #################################################################
	// Check gov module balance before sending proposal is zero
	// #################################################################
	var govBalanceBefore banktypes.QueryAllBalancesResponse
	govBalanceBeforeResp, err := clitestutil.ExecTestCLICmd(val.ClientCtx, bankcli.GetBalancesCmd(), []string{string(s.govModuleAddress.String()), "--output", "json"})
	s.NoError(err)
	json.Unmarshal(govBalanceBeforeResp.Bytes(), &govBalanceBefore)

	s.Equal(len(govBalanceBefore.Balances), 0)

	// Get total supply before broadcast tx
	totalSupplyBefore, err := clitestutil.ExecTestCLICmd(val.ClientCtx, bankcli.GetCmdQueryTotalSupply(), []string{s.cfg.BondDenom, "--output", "json"})
	s.NoError(err)

	// #################################################################
	// Broadcast tx with new proposal
	// #################################################################

	result, err := val.RPCClient.BroadcastTxCommit(context.Background(), txBytes)
	s.NoError(err)

	// skip blocks
	currentHeight, err := s.network.LatestHeight()
	s.NoError(err)
	_, err = s.network.WaitForHeight(currentHeight + 3)
	s.NoError(err)

	// Check proposal submitted correctly
	txResp, err := val.RPCClient.Tx(context.Background(), result.Hash, false)
	s.NoError(err)
	s.Equal(txResp.TxResult.Code, uint32(0))

	// #################################################################
	// Check amounts
	// #################################################################

	// Check communityPool amount
	communityPoolStateAfter := getCommunityPoolState(val)
	cpAmount, err := strconv.ParseFloat(communityPoolStateAfter.Pool[0].Amount, 64)
	s.NoError(err)

	s.Equal(cpAmount, 10000002.0)

	// Check total supply
	totalSupplyAfter, err := clitestutil.ExecTestCLICmd(val.ClientCtx, bankcli.GetCmdQueryTotalSupply(), []string{s.cfg.BondDenom, "--output", "json"})
	s.NoError(err)

	s.Equal(totalSupplyBefore, totalSupplyAfter)
}

func (s *BurnCoinsTestSuite) TestCase2QuorumNoWithVeto() {
	val := s.network.Validators[0]

	// check communityPool balance before proposal
	communityPoolStateBefore := getCommunityPoolState(val)
	cpAmountBefore, err := strconv.ParseFloat(communityPoolStateBefore.Pool[0].Amount, 64)
	s.NoError(err)

	// check stacking params before proposal
	stakingModuleParams := getStackingModuleParams(val)
	maxValidatorsBeforeProposal := stakingModuleParams.MaxValidators

	s.Equal(maxValidatorsBeforeProposal, 100)

	// #################################################################
	// Build proposal tx
	// #################################################################

	changes := []paramsproposaltypes.ParamChange{
		paramsproposaltypes.NewParamChange(stakingtypes.ModuleName, string(stakingtypes.KeyMaxValidators), "10"),
	}
	proposal := paramsproposaltypes.NewParameterChangeProposal("title", "description", changes)

	depositAmount := sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10000000)))
	submitProposalMsg, err := govtypes.NewMsgSubmitProposal(proposal, depositAmount, val.Address)
	s.NoError(err)

	fee := sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(100)))
	txBuilder := s.cfg.TxConfig.NewTxBuilder()
	s.Require().NoError(txBuilder.SetMsgs(submitProposalMsg))
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetGasLimit(2000000)

	kb := val.ClientCtx.Keyring
	s.NoError(err)

	proposaltxFactory := tx.Factory{}
	proposaltxFactory = proposaltxFactory.
		WithChainID(s.cfg.ChainID).
		WithKeybase(kb).
		WithTxConfig(s.cfg.TxConfig).
		WithSequence(2)

	err = tx.Sign(proposaltxFactory, "node0", txBuilder, true)
	s.NoError(err)

	txBytes, err := s.cfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	s.NoError(err)

	// #################################################################
	// Get balances state before broadcast proposal tx and do boradcast
	// #################################################################

	// Check gov module balance before sending proposal is zero
	var govBalanceBefore banktypes.QueryAllBalancesResponse
	govBalanceBeforeResp, err := clitestutil.ExecTestCLICmd(
		val.ClientCtx, bankcli.GetBalancesCmd(),
		[]string{s.govModuleAddress.String(), "--output", "json"},
	)
	s.NoError(err)
	json.Unmarshal(govBalanceBeforeResp.Bytes(), &govBalanceBefore)

	s.Equal(len(govBalanceBefore.Balances), 0)

	// Get total supply before broadcast tx
	totalSupplyBefore, err := clitestutil.ExecTestCLICmd(
		val.ClientCtx, bankcli.GetCmdQueryTotalSupply(),
		[]string{s.cfg.BondDenom, "--output", "json"},
	)
	s.NoError(err)

	// Broadcast tx with new proposal
	proposalTxResult, err := val.RPCClient.BroadcastTxCommit(context.Background(), txBytes)
	s.NoError(err)

	// #######################################################################

	// Build vote transaction with option: NoWithVote
	voteMsg := govtypes.NewMsgVote(val.Address, 2, govtypes.OptionNoWithVeto)

	vote_fee := sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(100)))
	voteTxBuilder := s.cfg.TxConfig.NewTxBuilder()
	s.Require().NoError(voteTxBuilder.SetMsgs(voteMsg))
	voteTxBuilder.SetFeeAmount(vote_fee)
	voteTxBuilder.SetGasLimit(2000000)

	voteTxFactory := proposaltxFactory.WithSequence(3)

	err = tx.Sign(voteTxFactory, "node0", voteTxBuilder, true)
	s.NoError(err)

	voteTxBytes, err := s.cfg.TxConfig.TxEncoder()(voteTxBuilder.GetTx())
	s.NoError(err)

	// Broadcast tx with new proposal
	voteTxResult, err := val.RPCClient.BroadcastTxCommit(context.Background(), voteTxBytes)
	s.NoError(err)

	// ########################################################################

	// skip blocks
	currentHeight, err := s.network.LatestHeight()
	s.NoError(err)
	_, err = s.network.WaitForHeight(currentHeight + 3)
	s.NoError(err)

	// Check proposal and vote submitted correctly
	// -- proposal tx
	proposalTxResp, err := val.RPCClient.Tx(context.Background(), proposalTxResult.Hash, false)
	s.NoError(err)
	s.Equal(proposalTxResp.TxResult.Code, uint32(0))
	// -- vote tx
	voteTxResp, err := val.RPCClient.Tx(context.Background(), voteTxResult.Hash, false)
	s.NoError(err)
	s.Equal(voteTxResp.TxResult.Code, uint32(0))

	// ########################################################################

	// check stacking params before proposal
	stakingModuleParams = getStackingModuleParams(val)
	maxValidatorsAfterProposal := stakingModuleParams.MaxValidators

	s.Equal(maxValidatorsBeforeProposal, maxValidatorsAfterProposal)

	// check communityPool total amount
	communityPoolStateAfter := getCommunityPoolState(val)
	cpAmountAfter, err := strconv.ParseFloat(communityPoolStateAfter.Pool[0].Amount, 64)
	s.NoError(err)

	s.Equal(cpAmountAfter, cpAmountBefore+10000004.0)

	// Check total supply after burn
	totalSupplyAfter, err := clitestutil.ExecTestCLICmd(val.ClientCtx, bankcli.GetCmdQueryTotalSupply(), []string{s.cfg.BondDenom, "--output", "json"})
	s.NoError(err)

	s.Equal(totalSupplyBefore, totalSupplyAfter)
}

func (s *BurnCoinsTestSuite) TestCase3QuorumYes() {
	val := s.network.Validators[0]

	// Check communityPool balance before proposal
	communityPoolStateBefore := getCommunityPoolState(val)
	cpAmountBefore, err := strconv.ParseFloat(communityPoolStateBefore.Pool[0].Amount, 64)
	s.NoError(err)

	s.Equal(cpAmountBefore, 20000006.0)

	// Check stacking params before proposal
	stakingModuleParams := getStackingModuleParams(val)
	maxValidatorsBeforeProposal := stakingModuleParams.MaxValidators

	s.Equal(maxValidatorsBeforeProposal, 100)

	// #################################################################
	// Build proposal tx
	// #################################################################

	changes := []paramsproposaltypes.ParamChange{
		paramsproposaltypes.NewParamChange(stakingtypes.ModuleName, string(stakingtypes.KeyMaxValidators), "10"),
	}
	proposal := paramsproposaltypes.NewParameterChangeProposal("title", "description", changes)

	depositAmount := sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10000000)))
	submitProposalMsg, err := govtypes.NewMsgSubmitProposal(proposal, depositAmount, val.Address)
	s.NoError(err)

	fee := sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(100)))
	txBuilder := s.cfg.TxConfig.NewTxBuilder()
	s.Require().NoError(txBuilder.SetMsgs(submitProposalMsg))
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetGasLimit(2000000)

	kb := val.ClientCtx.Keyring
	s.NoError(err)

	proposaltxFactory := tx.Factory{}
	proposaltxFactory = proposaltxFactory.
		WithChainID(s.cfg.ChainID).
		WithKeybase(kb).
		WithTxConfig(s.cfg.TxConfig).
		WithSequence(4)

	err = tx.Sign(proposaltxFactory, "node0", txBuilder, true)
	s.NoError(err)

	txBytes, err := s.cfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	s.NoError(err)

	// #################################################################
	// Get balances state before broadcast proposal tx and do boradcast
	// #################################################################

	// Check gov module balance before sending proposal is zero
	var govBalanceBefore banktypes.QueryAllBalancesResponse
	govBalanceBeforeResp, err := clitestutil.ExecTestCLICmd(
		val.ClientCtx, bankcli.GetBalancesCmd(),
		[]string{s.govModuleAddress.String(), "--output", "json"},
	)
	s.NoError(err)
	json.Unmarshal(govBalanceBeforeResp.Bytes(), &govBalanceBefore)

	s.Equal(len(govBalanceBefore.Balances), 0)

	// Get total supply before broadcast tx
	totalSupplyBefore, err := clitestutil.ExecTestCLICmd(
		val.ClientCtx, bankcli.GetCmdQueryTotalSupply(),
		[]string{s.cfg.BondDenom, "--output", "json"},
	)
	s.NoError(err)

	// Broadcast tx with new proposal
	proposalTxResult, err := val.RPCClient.BroadcastTxCommit(context.Background(), txBytes)
	s.NoError(err)

	// #######################################################################

	// Build vote transaction with option: Yes
	voteMsg := govtypes.NewMsgVote(val.Address, 3, govtypes.OptionYes)

	vote_fee := sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(100)))
	voteTxBuilder := s.cfg.TxConfig.NewTxBuilder()
	s.Require().NoError(voteTxBuilder.SetMsgs(voteMsg))
	voteTxBuilder.SetFeeAmount(vote_fee)
	voteTxBuilder.SetGasLimit(2000000)

	voteTxFactory := proposaltxFactory.WithSequence(5)

	err = tx.Sign(voteTxFactory, "node0", voteTxBuilder, true)
	s.NoError(err)

	voteTxBytes, err := s.cfg.TxConfig.TxEncoder()(voteTxBuilder.GetTx())
	s.NoError(err)

	// Broadcast tx with new proposal
	voteTxResult, err := val.RPCClient.BroadcastTxCommit(context.Background(), voteTxBytes)
	s.NoError(err)

	// ########################################################################

	// Skip blocks
	currentHeight, err := s.network.LatestHeight()
	s.NoError(err)
	_, err = s.network.WaitForHeight(currentHeight + 4)
	s.NoError(err)

	// Check proposal and vote submitted correctly
	// -- proposal tx
	proposalTxResp, err := val.RPCClient.Tx(context.Background(), proposalTxResult.Hash, false)
	s.NoError(err)
	s.Equal(proposalTxResp.TxResult.Code, uint32(0))
	// -- vote tx
	voteTxResp, err := val.RPCClient.Tx(context.Background(), voteTxResult.Hash, false)
	s.NoError(err)
	s.Equal(voteTxResp.TxResult.Code, uint32(0))

	// ########################################################################

	// Check param changed in module after proposal
	stakingModuleParams = getStackingModuleParams(val)
	maxValidatorsAfterProposal := stakingModuleParams.MaxValidators
	s.Equal(maxValidatorsAfterProposal, 10)

	// Check communityPool total amount
	communityPoolStateAfter := getCommunityPoolState(val)
	cpAmountAfter, err := strconv.ParseFloat(communityPoolStateAfter.Pool[0].Amount, 64)
	s.NoError(err)

	s.Equal(cpAmountAfter, 20000010.0)

	// Check total supply after burn
	totalSupplyAfter, err := clitestutil.ExecTestCLICmd(val.ClientCtx, bankcli.GetCmdQueryTotalSupply(), []string{s.cfg.BondDenom, "--output", "json"})
	s.NoError(err)

	s.Equal(totalSupplyBefore, totalSupplyAfter)
}

func (s *BurnCoinsTestSuite) TestCase4LowDeposit() {
	val := s.network.Validators[0]

	// Check communityPool balance before proposal
	communityPoolStateBefore := getCommunityPoolState(val)
	cpAmountBefore, err := strconv.ParseFloat(communityPoolStateBefore.Pool[0].Amount, 64)
	s.NoError(err)

	s.Equal(cpAmountBefore, 20000010.0)

	// #################################################################
	// Build proposal tx
	// #################################################################

	changes := []paramsproposaltypes.ParamChange{
		paramsproposaltypes.NewParamChange(stakingtypes.ModuleName, string(stakingtypes.KeyMaxValidators), "10"),
	}
	proposal := paramsproposaltypes.NewParameterChangeProposal("title", "description", changes)

	depositAmount := sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(1000000)))
	submitProposalMsg, err := govtypes.NewMsgSubmitProposal(proposal, depositAmount, val.Address)
	s.NoError(err)

	fee := sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(100)))
	txBuilder := s.cfg.TxConfig.NewTxBuilder()
	s.Require().NoError(txBuilder.SetMsgs(submitProposalMsg))
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetGasLimit(2000000)

	kb := val.ClientCtx.Keyring
	s.NoError(err)

	proposaltxFactory := tx.Factory{}
	proposaltxFactory = proposaltxFactory.
		WithChainID(s.cfg.ChainID).
		WithKeybase(kb).
		WithTxConfig(s.cfg.TxConfig).
		WithSequence(6)

	err = tx.Sign(proposaltxFactory, "node0", txBuilder, true)
	s.NoError(err)

	txBytes, err := s.cfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	s.NoError(err)

	// #################################################################
	// Get balances state before broadcast proposal tx and do boradcast
	// #################################################################

	// Check gov module balance before sending proposal is zero
	var govBalanceBefore banktypes.QueryAllBalancesResponse
	govBalanceBeforeResp, err := clitestutil.ExecTestCLICmd(
		val.ClientCtx, bankcli.GetBalancesCmd(),
		[]string{s.govModuleAddress.String(), "--output", "json"},
	)
	s.NoError(err)
	json.Unmarshal(govBalanceBeforeResp.Bytes(), &govBalanceBefore)

	s.Equal(len(govBalanceBefore.Balances), 0)

	// Get total supply before broadcast tx
	totalSupplyBefore, err := clitestutil.ExecTestCLICmd(
		val.ClientCtx, bankcli.GetCmdQueryTotalSupply(),
		[]string{s.cfg.BondDenom, "--output", "json"},
	)
	s.NoError(err)

	// Broadcast tx with new proposal
	proposalTxResult, err := val.RPCClient.BroadcastTxCommit(context.Background(), txBytes)
	s.NoError(err)

	// ########################################################################

	// Skip blocks
	currentHeight, err := s.network.LatestHeight()
	s.NoError(err)
	_, err = s.network.WaitForHeight(currentHeight + 4)
	s.NoError(err)

	// Check proposal and vote submitted correctly
	// -- proposal tx
	proposalTxResp, err := val.RPCClient.Tx(context.Background(), proposalTxResult.Hash, false)
	s.NoError(err)
	s.Equal(proposalTxResp.TxResult.Code, uint32(0))

	// ########################################################################

	// Check communityPool total amount
	communityPoolStateAfter := getCommunityPoolState(val)
	cpAmountAfter, err := strconv.ParseFloat(communityPoolStateAfter.Pool[0].Amount, 64)
	s.NoError(err)

	s.Equal(cpAmountAfter, 21000012.0)
	s.T().Log("cpAmountAfter: ", cpAmountAfter)

	// Check total supply after burn
	totalSupplyAfter, err := clitestutil.ExecTestCLICmd(val.ClientCtx, bankcli.GetCmdQueryTotalSupply(), []string{s.cfg.BondDenom, "--output", "json"})
	s.NoError(err)

	s.Equal(totalSupplyBefore, totalSupplyAfter)
}

func TestBurnCoins(t *testing.T) {
	cfg := network.DefaultConfig()
	encCfg := simapp.MakeTestEncodingConfig()
	cfg.AppConstructor = haqqnetwork.NewAppConstructor(encCfg)
	cfg.NumValidators = 1

	genesisGov := strings.Replace(
		string(cfg.GenesisState["gov"]), "\"voting_period\":\"172800s\"", "\"voting_period\":\"2s\"", 1,
	)
	genesisGov = strings.Replace(
		genesisGov, "\"max_deposit_period\":\"172800s\"", "\"max_deposit_period\":\"2s\"", 1,
	)

	cfg.GenesisState["gov"] = []byte(genesisGov)

	suite.Run(t, NewBurnCoinsTestSuite(cfg))
}

// -- helpers: types/funcs

type CommunityPoolData struct {
	Amount string `json:"amount"`
}

type CommunityPool struct {
	Pool []CommunityPoolData `json:"pool"`
}

type StackingModuleParams struct {
	MaxValidators int `json:"max_validators"`
}

func getCommunityPoolState(val *network.Validator) CommunityPool {
	communityPoolStateJson, _ := clitestutil.ExecTestCLICmd(val.ClientCtx, distcli.GetCmdQueryCommunityPool(), []string{"--output", "json"})

	var communityPoolState CommunityPool
	json.Unmarshal(communityPoolStateJson.Bytes(), &communityPoolState)

	return communityPoolState
}

func getStackingModuleParams(val *network.Validator) StackingModuleParams {
	stakingParamsJson, _ := clitestutil.ExecTestCLICmd(val.ClientCtx, stakingcli.GetCmdQueryParams(), []string{"--output", "json"})

	var stackingParams StackingModuleParams
	json.Unmarshal(stakingParamsJson.Bytes(), &stackingParams)

	return stackingParams
}
