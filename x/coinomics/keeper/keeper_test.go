package keeper_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"
	"github.com/cometbft/cometbft/version"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/crypto/ethsecp256k1"
	"github.com/haqq-network/haqq/tests"
	haqqtypes "github.com/haqq-network/haqq/utils"
	"github.com/haqq-network/haqq/x/coinomics/types"
	evm "github.com/haqq-network/haqq/x/evm/types"
	feemarkettypes "github.com/haqq-network/haqq/x/feemarket/types"
)

var denomMint = types.DefaultMintDenom

type KeeperTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	app            *app.Haqq
	address        common.Address
	signer         keyring.Signer
	queryClientEvm evm.QueryClient
	queryClient    types.QueryClient
	consAddress    sdk.ConsAddress
	validator      stakingtypes.Validator
	denom          string
	privKey        *ethsecp256k1.PrivKey
}

var s *KeeperTestSuite

func TestKeeperTestSuite(t *testing.T) {
	s = new(KeeperTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.DoSetupTest(suite.T())
}

func (suite *KeeperTestSuite) DoSetupTest(t require.TestingT) {
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
	suite.denom = types.DefaultMintDenom

	// setup context
	chainID := haqqtypes.MainNetChainID + "-1"
	app, valAddr1 := app.Setup(false, feemarkettypes.DefaultGenesisState(), chainID)

	startTime := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	suite.app = app
	suite.ctx = suite.app.BaseApp.NewContext(false, tmproto.Header{
		Height:          1,
		ChainID:         chainID,
		Time:            startTime,
		ProposerAddress: valAddr1,

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
	types.RegisterQueryServer(queryHelper, suite.app.CoinomicsKeeper)
	suite.queryClient = types.NewQueryClient(queryHelper)

	// staking module - bond denom
	stakingParams := suite.app.StakingKeeper.GetParams(suite.ctx)
	stakingParams.BondDenom = suite.denom
	err = suite.app.StakingKeeper.SetParams(suite.ctx, stakingParams)
	require.NoError(t, err)

	// Set Validator
	valAddr := sdk.ValAddress(suite.address.Bytes())
	validator, err := stakingtypes.NewValidator(valAddr, privCons.PubKey(), stakingtypes.Description{})
	require.NoError(t, err)

	validator = stakingkeeper.TestingUpdateValidator(suite.app.StakingKeeper.Keeper, suite.ctx, validator, true)
	err = suite.app.StakingKeeper.Hooks().AfterValidatorCreated(suite.ctx, validator.GetOperator())
	require.NoError(t, err)
	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	require.NoError(t, err)

	// TODO change to setup with 1 validator
	validators := s.app.StakingKeeper.GetValidators(s.ctx, 2)

	// set a bonded validator that takes part in consensus
	if validators[0].Status == stakingtypes.Bonded {
		suite.validator = validators[0]
	} else {
		suite.validator = validators[1]
	}
}

func (suite *KeeperTestSuite) Commit(numBlocks uint64) {
	for i := uint64(0); i < numBlocks; i++ {
		suite.CommitBlock(6)
	}
}

func (suite *KeeperTestSuite) CommitWithShift(numBlocks uint64, shift uint64) {
	for i := uint64(0); i < numBlocks; i++ {
		suite.CommitBlock(shift)
	}
}

func (suite *KeeperTestSuite) CommitBlock(shift uint64) {
	header := suite.ctx.BlockHeader()
	_ = suite.app.Commit()

	header.Height++
	header.Time = header.Time.Add(time.Second * time.Duration(shift))

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

func (suite *KeeperTestSuite) CommitLeapYear() {
	header := suite.ctx.BlockHeader()
	_ = suite.app.Commit()

	leapYearTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	header.Height++
	header.Time = leapYearTime

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
