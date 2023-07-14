package tests

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/server/config"
	"github.com/evmos/ethermint/tests"
	evm "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"
	erc20types "github.com/evmos/evmos/v10/x/erc20/types"
	"github.com/haqq-network/haqq/app"
	haqqtypes "github.com/haqq-network/haqq/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	"github.com/tendermint/tendermint/version"
)

type TransferETHTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	app            *app.Haqq
	address        common.Address
	valAddr1       []byte
	signer         keyring.Signer
	queryClientEvm evm.QueryClient
	queryClient    erc20types.QueryClient
	consAddress    sdk.ConsAddress
	validator      stakingtypes.Validator
	denom          string
	privKey        *ethsecp256k1.PrivKey
}

func TestTransferETHSuite(t *testing.T) {
	s := new(TransferETHTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transfer ETH Suite")
}

func (suite *TransferETHTestSuite) SetupTest() {
	suite.DoSetupTest(suite.T())
}

func (suite *TransferETHTestSuite) DoSetupTest(t require.TestingT) {
	// account key
	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	suite.privKey = priv
	suite.address = common.BytesToAddress(priv.PubKey().Address().Bytes())
	suite.signer = tests.NewSigner(priv)

	// consensus key
	privCons, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	consAddress := sdk.ConsAddress(privCons.PubKey().Address())
	suite.consAddress = consAddress

	// denom
	suite.denom = "aISLM"

	// setup context
	haqqApp, valAddr1 := app.Setup(false, feemarkettypes.DefaultGenesisState())
	suite.valAddr1 = valAddr1
	suite.app = haqqApp
	suite.ctx = suite.app.BaseApp.NewContext(false, tmproto.Header{
		Height:          1,
		ChainID:         haqqtypes.LocalNetChainID + "-1",
		Time:            time.Now().UTC(),
		ProposerAddress: consAddress.Bytes(),

		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	})

	// setup query helpers
	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	erc20types.RegisterQueryServer(queryHelper, suite.app.Erc20Keeper)
	suite.queryClient = erc20types.NewQueryClient(queryHelper)

	// staking module - bond denom
	stakingParams := suite.app.StakingKeeper.GetParams(suite.ctx)
	stakingParams.BondDenom = suite.denom
	suite.app.StakingKeeper.SetParams(suite.ctx, stakingParams)

	// Set Validator
	valAddr := sdk.ValAddress(suite.address.Bytes())
	validator, err := stakingtypes.NewValidator(valAddr, privCons.PubKey(), stakingtypes.Description{})
	require.NoError(t, err)
	validator = stakingkeeper.TestingUpdateValidator(suite.app.StakingKeeper, suite.ctx, validator, true)
	err = suite.app.StakingKeeper.AfterValidatorCreated(suite.ctx, validator.GetOperator())
	require.NoError(t, err)
	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	require.NoError(t, err)

	// TODO change to setup with 1 validator
	validators := suite.app.StakingKeeper.GetValidators(suite.ctx, 2)

	// set a bonded validator that takes part in consensus
	if validators[0].Status == stakingtypes.Bonded {
		suite.validator = validators[0]
	} else {
		suite.validator = validators[1]
	}
}

func (suite *TransferETHTestSuite) Commit(numBlocks uint64) {
	for i := uint64(0); i < numBlocks; i++ {
		suite.CommitBlock()
	}
}

func (suite *TransferETHTestSuite) CommitBlock() {
	header := suite.ctx.BlockHeader()
	_ = suite.app.Commit()

	header.Height++

	// run begin block
	suite.app.BeginBlock(abci.RequestBeginBlock{
		Header: header,
	})

	// run end block
	suite.app.EndBlock(abci.RequestEndBlock{
		Height: header.Height,
	})

	// update ctx
	suite.ctx = suite.app.BaseApp.NewContext(false, header)

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	evm.RegisterQueryServer(queryHelper, suite.app.EvmKeeper)
	suite.queryClientEvm = evm.NewQueryClient(queryHelper)
}

func (suite *TransferETHTestSuite) MintFeeCollector(coins sdk.Coins) {
	err := suite.app.BankKeeper.MintCoins(suite.ctx, erc20types.ModuleName, coins)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToModule(suite.ctx, erc20types.ModuleName, authtypes.FeeCollectorName, coins)
	suite.Require().NoError(err)
}

func (suite *TransferETHTestSuite) MintToAccount(coins sdk.Coins) {
	err := suite.app.BankKeeper.MintCoins(suite.ctx, erc20types.ModuleName, coins)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, erc20types.ModuleName, suite.address.Bytes(), coins)
	suite.Require().NoError(err)
}

func (suite *TransferETHTestSuite) TestTransferETH() {
	// skip some blocks
	suite.Commit(3)
	ctx := sdk.WrapSDKContext(suite.ctx)

	evmDenom := suite.app.EvmKeeper.GetEVMDenom(suite.ctx)
	suite.MintToAccount(sdk.NewCoins(sdk.NewCoin(evmDenom, sdk.NewInt(100000))))

	suite.T().Log(suite.address.String())
	balanceBefore, err := suite.queryClientEvm.Balance(ctx, &evm.QueryBalanceRequest{Address: suite.address.String()})
	suite.NoError(err)
	suite.NotEmpty(balanceBefore.Balance)
	suite.Equal("100000", balanceBefore.Balance)

	account, err := suite.queryClientEvm.Account(ctx, &evm.QueryAccountRequest{Address: suite.address.String()})
	suite.NoError(err)

	nonce := account.Nonce

	chainID := suite.app.EvmKeeper.ChainID()
	var receiveAddr common.Address

	args, err := json.Marshal(&evm.TransactionArgs{To: &receiveAddr, From: &suite.address, Data: nil})
	suite.Require().NoError(err)
	res, err := suite.queryClientEvm.EstimateGas(ctx, &evm.EthCallRequest{
		Args:   args,
		GasCap: config.DefaultGasCap,
	})
	suite.Require().NoError(err)

	// Mint the max gas to the FeeCollector to ensure balance in case of refund
	suite.MintFeeCollector(sdk.NewCoins(sdk.NewCoin(evmDenom, sdk.NewInt(suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx).Int64()*int64(res.Gas)))))

	tx := evm.NewTx(
		chainID,
		nonce,
		&receiveAddr,
		big.NewInt(50000),
		uint64(22012),
		nil,
		suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
		big.NewInt(1),
		nil,
		&types.AccessList{},
	)

	tx.From = suite.address.Hex()
	err = tx.Sign(types.LatestSignerForChainID(chainID), suite.signer)
	suite.NoError(err)

	rsp, err := suite.app.EvmKeeper.EthereumTx(ctx, tx)
	suite.NoError(err)
	suite.Empty(rsp.VmError)

	// skip some blocks
	suite.Commit(5)

	balanceAfter, err := suite.queryClientEvm.Balance(ctx, &evm.QueryBalanceRequest{Address: suite.address.String()})
	suite.NoError(err)
	suite.T().Log("balance before", balanceBefore.Balance)
	suite.T().Log("balance after", balanceAfter.Balance)
	suite.NotEqual(balanceBefore.Balance, balanceAfter.Balance)
}
