package keeper_test

import (
	"math"
	"testing"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	"github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
	feemarkettypes "github.com/haqq-network/haqq/x/feemarket/types"
)

type KeeperTestSuite struct {
	suite.Suite

	network *network.UnitTestNetwork
	handler grpc.Handler
	keyring keyring.Keyring
	factory factory.TxFactory

	enableFeemarket  bool
	enableLondonHF   bool
	mintFeeCollector bool
}

var s *KeeperTestSuite

func TestKeeperTestSuite(t *testing.T) {
	s = new(KeeperTestSuite)
	s.enableFeemarket = false
	s.enableLondonHF = true
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

func (suite *KeeperTestSuite) SetupTest() {
	keys := keyring.New(2)
	// Set custom balance based on test params
	customGenesis := network.CustomGenesisState{}
	feemarketGenesis := feemarkettypes.DefaultGenesisState()
	if suite.enableFeemarket {
		feemarketGenesis.Params.EnableHeight = 1
		feemarketGenesis.Params.NoBaseFee = false
	} else {
		feemarketGenesis.Params.NoBaseFee = true
	}
	customGenesis[feemarkettypes.ModuleName] = feemarketGenesis

	if !suite.enableLondonHF {
		evmGenesis := evmtypes.DefaultGenesisState()
		maxInt := sdkmath.NewInt(math.MaxInt64)
		evmGenesis.Params.ChainConfig.LondonBlock = &maxInt
		evmGenesis.Params.ChainConfig.ArrowGlacierBlock = &maxInt
		evmGenesis.Params.ChainConfig.GrayGlacierBlock = &maxInt
		evmGenesis.Params.ChainConfig.MergeNetsplitBlock = &maxInt
		evmGenesis.Params.ChainConfig.ShanghaiBlock = &maxInt
		evmGenesis.Params.ChainConfig.CancunBlock = &maxInt
		customGenesis[evmtypes.ModuleName] = evmGenesis
	}

	if suite.mintFeeCollector {
		// mint some coin to fee collector
		coins := sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(int64(params.TxGas)-1)))
		balances := []banktypes.Balance{
			{
				Address: authtypes.NewModuleAddress(authtypes.FeeCollectorName).String(),
				Coins:   coins,
			},
		}
		bankGenesis := banktypes.DefaultGenesisState()
		bankGenesis.Balances = balances
		customGenesis[banktypes.ModuleName] = bankGenesis
	}

	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keys.GetAllAccAddrs()...),
		network.WithCustomGenesis(customGenesis),
	)
	gh := grpc.NewIntegrationHandler(nw)
	tf := factory.New(nw, gh)

	suite.network = nw
	suite.factory = tf
	suite.handler = gh
	suite.keyring = keys
}
