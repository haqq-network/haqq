package bank_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/bank"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
	coinomicstypes "github.com/haqq-network/haqq/x/coinomics/types"
)

var s *PrecompileTestSuite

// PrecompileTestSuite is the implementation of the TestSuite interface for ERC20 precompile
// unit tests.
type PrecompileTestSuite struct {
	suite.Suite

	bondDenom, tokenDenom string
	evmosAddr, xmplAddr   common.Address

	// tokenDenom is the specific token denomination used in testing the ERC20 precompile.
	// This denomination is used to instantiate the precompile.
	network     *network.UnitTestNetwork
	factory     factory.TxFactory
	grpcHandler grpc.Handler
	keyring     testkeyring.Keyring

	precompile *bank.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(PrecompileTestSuite)
	suite.Run(t, s)
}

func (s *PrecompileTestSuite) SetupTest() {
	keyring := testkeyring.New(2)
	integrationNetwork := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	)
	grpcHandler := grpc.NewIntegrationHandler(integrationNetwork)
	txFactory := factory.New(integrationNetwork, grpcHandler)

	ctx := integrationNetwork.GetContext()
	sk := integrationNetwork.App.StakingKeeper
	bondDenom := sk.BondDenom(ctx)
	s.Require().NotEmpty(bondDenom, "bond denom cannot be empty")

	s.bondDenom = bondDenom
	s.tokenDenom = "xmpl"
	s.factory = txFactory
	s.grpcHandler = grpcHandler
	s.keyring = keyring
	s.network = integrationNetwork

	// Register ISLM
	evmosMetadata, found := s.network.App.BankKeeper.GetDenomMetaData(s.network.GetContext(), s.bondDenom)
	s.Require().True(found, "expected ISLM denom metadata")

	tokenPair, err := s.network.App.Erc20Keeper.RegisterCoin(s.network.GetContext(), evmosMetadata)
	s.Require().NoError(err, "failed to register coin")

	s.evmosAddr = common.HexToAddress(tokenPair.Erc20Address)

	// Mint and register a second coin for testing purposes
	err = s.network.App.BankKeeper.MintCoins(s.network.GetContext(), coinomicstypes.ModuleName, sdk.Coins{{Denom: "xmpl", Amount: math.NewInt(1e18)}})
	s.Require().NoError(err)

	xmplMetadata := banktypes.Metadata{
		Description: "An exemplary token",
		Base:        s.tokenDenom,
		// NOTE: Denom units MUST be increasing
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    s.tokenDenom,
				Exponent: 0,
				Aliases:  []string{s.tokenDenom},
			},
			{
				Denom:    s.tokenDenom,
				Exponent: 18,
			},
		},
		Name:    "Exemplary",
		Symbol:  "XMPL",
		Display: s.tokenDenom,
	}

	tokenPair, err = s.network.App.Erc20Keeper.RegisterCoin(s.network.GetContext(), xmplMetadata)
	s.Require().NoError(err, "failed to register coin")

	s.xmplAddr = common.HexToAddress(tokenPair.Erc20Address)

	s.precompile = s.setupBankPrecompile()
}
