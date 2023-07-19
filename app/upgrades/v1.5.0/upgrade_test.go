package v150_test

import (
	"math"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/encoding"
	evmostypes "github.com/evmos/ethermint/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	epochstypes "github.com/evmos/evmos/v10/x/epochs/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/testutil"
	"github.com/haqq-network/haqq/testutil/contracts"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
)

var s *UpgradeTestSuite

type UpgradeTestSuite struct {
	suite.Suite

	ctx     sdk.Context
	app     *app.Haqq
	t       require.TestingT
	chainID string
	// account
	accPrivateKey *ethsecp256k1.PrivKey
	ethAddress    common.Address
	accAddress    sdk.AccAddress
	signer        keyring.Signer
	// validator
	valAddr   sdk.ValAddress
	validator stakingtypes.Validator
	// consensus
	consAddress sdk.ConsAddress
	// query client
	queryClientEvm evmtypes.QueryClient
	clientCtx      client.Context
	// signer
	ethSigner ethtypes.Signer
	// contract
	contractAddress common.Address
}

func TestKeeperTestSuite(t *testing.T) {
	s = new(UpgradeTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade v1.5.0 (Revesting) Suite")
}

func (suite *UpgradeTestSuite) SetupTest() {
	suite.DoSetupTest(suite.T())
}

func (suite *UpgradeTestSuite) DoSetupTest(t require.TestingT) {
	var err error
	checkTx := false
	suite.t = t
	suite.chainID = "haqq-121799-1"

	// account key
	suite.accPrivateKey, err = ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	suite.ethAddress = common.BytesToAddress(suite.accPrivateKey.PubKey().Address().Bytes())
	suite.accAddress = sdk.AccAddress(suite.ethAddress.Bytes())
	suite.signer = utiltx.NewSigner(suite.accPrivateKey)

	// consensus key
	consPrivateKey, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	suite.consAddress = sdk.ConsAddress(consPrivateKey.PubKey().Address())

	// Init app
	suite.app, _ = app.Setup(checkTx, nil)
	// Set Context
	header := testutil.NewHeader(1, time.Now().UTC(), suite.chainID, suite.consAddress, nil, nil)
	suite.ctx = suite.app.BaseApp.NewContext(checkTx, header)

	// Setup query helpers
	queryHelperEvm := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	evmtypes.RegisterQueryServer(queryHelperEvm, suite.app.EvmKeeper)
	suite.queryClientEvm = evmtypes.NewQueryClient(queryHelperEvm)

	// Set epoch start time and height for all epoch identifiers from the epoch
	// module
	identifiers := []string{epochstypes.WeekEpochID, epochstypes.DayEpochID}
	for _, identifier := range identifiers {
		epoch, found := suite.app.EpochsKeeper.GetEpochInfo(suite.ctx, identifier)
		require.True(t, found)
		epoch.StartTime = suite.ctx.BlockTime()
		epoch.CurrentEpochStartHeight = suite.ctx.BlockHeight()
		suite.app.EpochsKeeper.SetEpochInfo(suite.ctx, epoch)
	}

	acc := &evmostypes.EthAccount{
		BaseAccount: authtypes.NewBaseAccount(suite.accAddress, nil, 0, 0),
		CodeHash:    common.BytesToHash(crypto.Keccak256(nil)).String(),
	}

	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	// fund signer acc to pay for tx fees
	amt := sdk.NewInt(int64(math.Pow10(18) * 2))
	err = testutil.FundAccount(
		suite.ctx,
		suite.app.BankKeeper,
		suite.accAddress,
		sdk.NewCoins(sdk.NewCoin(suite.app.StakingKeeper.BondDenom(suite.ctx), amt)),
	)
	require.NoError(t, err)

	// Set Validator
	suite.valAddr = sdk.ValAddress(suite.accAddress)
	validator, err := stakingtypes.NewValidator(suite.valAddr, suite.accPrivateKey.PubKey(), stakingtypes.Description{})
	require.NoError(t, err)
	validator = stakingkeeper.TestingUpdateValidator(suite.app.StakingKeeper, suite.ctx, validator, true)
	err = suite.app.StakingKeeper.AfterValidatorCreated(suite.ctx, validator.GetOperator())
	require.NoError(t, err)
	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	require.NoError(t, err)
	validators := suite.app.StakingKeeper.GetValidators(suite.ctx, 1)
	suite.validator = validators[0]

	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	suite.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
	suite.ethSigner = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())

	// Deploy contracts
	suite.contractAddress, err = suite.DeployContract()
	require.NoError(t, err)
}

func (suite *UpgradeTestSuite) DeployContract() (common.Address, error) {
	suite.Commit()
	addr, err := testutil.DeployContract(
		suite.ctx,
		suite.app,
		suite.accPrivateKey,
		suite.queryClientEvm,
		contracts.HaqqTestingContract,
	)
	suite.Commit()

	return addr, err
}

// Commit commits and starts a new block with an updated context.
func (suite *UpgradeTestSuite) Commit() {
	suite.CommitAfter(time.Second * 0)
}

// Commit commits a block at a given time.
func (suite *UpgradeTestSuite) CommitAfter(t time.Duration) {
	var err error
	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, t, nil)
	require.NoError(suite.t, err)

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	evmtypes.RegisterQueryServer(queryHelper, suite.app.EvmKeeper)
	suite.queryClientEvm = evmtypes.NewQueryClient(queryHelper)
}

// MintFeeCollector mints coins with the bank modules and sends them to the fee
// collector.
func (suite *UpgradeTestSuite) MintFeeCollector(coins sdk.Coins) {
	err := suite.app.BankKeeper.MintCoins(suite.ctx, evmtypes.ModuleName, coins)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToModule(suite.ctx, evmtypes.ModuleName, authtypes.FeeCollectorName, coins)
	suite.Require().NoError(err)
}
