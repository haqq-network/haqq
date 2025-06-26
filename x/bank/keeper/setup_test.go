package keeper_test

import (
	"testing"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibcgotesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/haqq-network/haqq/app"
	ibctesting "github.com/haqq-network/haqq/ibc/testing"
	"github.com/haqq-network/haqq/x/erc20/types"
	evm "github.com/haqq-network/haqq/x/evm/types"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	app            *app.Haqq
	queryClientEvm evm.QueryClient
	queryClient    types.QueryClient
	address        common.Address
	consAddress    sdk.ConsAddress
	clientCtx      client.Context //nolint:unused
	ethSigner      ethtypes.Signer
	priv           cryptotypes.PrivKey
	validator      stakingtypes.Validator
	signer         keyring.Signer

	coordinator *ibcgotesting.Coordinator

	// testing chains used for convenience and readability
	HaqqChain       *ibcgotesting.TestChain
	IBCOsmosisChain *ibcgotesting.TestChain
	IBCCosmosChain  *ibcgotesting.TestChain

	pathOsmosisHaqq   *ibctesting.Path
	pathCosmosHaqq    *ibctesting.Path
	pathOsmosisCosmos *ibctesting.Path

	suiteIBCTesting bool
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
