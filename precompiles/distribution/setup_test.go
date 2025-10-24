package distribution_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	"github.com/haqq-network/haqq/precompiles/distribution"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	"github.com/haqq-network/haqq/testutil/integration/haqq/grpc"
	testkeyring "github.com/haqq-network/haqq/testutil/integration/haqq/keyring"
	"github.com/haqq-network/haqq/testutil/integration/haqq/network"
)

type PrecompileTestSuite struct {
	suite.Suite

	network     *network.UnitTestNetwork
	factory     factory.TxFactory
	grpcHandler grpc.Handler
	keyring     testkeyring.Keyring

	precompile           *distribution.Precompile
	bondDenom            string
	validatorsKeys       []testkeyring.Key
	withValidatorSlashes bool
}

func TestPrecompileUnitTestSuite(t *testing.T) {
	suite.Run(t, new(PrecompileTestSuite))
}

func (s *PrecompileTestSuite) SetupTest() {
	keyring := testkeyring.New(2)
	s.validatorsKeys = generateKeys(3)
	customGen := network.CustomGenesisState{}

	operatorsAddr := make([]sdk.AccAddress, 3)
	for i, k := range s.validatorsKeys {
		operatorsAddr[i] = k.AccAddr
	}

	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
		network.WithCustomGenesis(customGen),
		network.WithValidatorOperators(operatorsAddr),
	)
	grpcHandler := grpc.NewIntegrationHandler(nw)
	txFactory := factory.New(nw, grpcHandler)

	ctx := nw.GetContext()
	sk := nw.App.StakingKeeper
	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		panic(err)
	}

	// set some slashing events for integration test
	if s.withValidatorSlashes {
		for i := 0; i < 2; i++ {
			valAddr := sdk.ValAddress(s.validatorsKeys[0].Addr.Bytes())
			val, err := sk.GetValidator(ctx, valAddr)
			if err != nil {
				panic(err)
			}

			// increment current period
			newPeriod, err := nw.App.DistrKeeper.IncrementValidatorPeriod(ctx, val)
			if err != nil {
				panic(err)
			}

			// increment reference count on period we need to track
			historical, err := nw.App.DistrKeeper.GetValidatorHistoricalRewards(ctx, valAddr, newPeriod)
			if err != nil {
				panic(err)
			}

			if historical.ReferenceCount > 2 {
				panic("reference count should never exceed 2")
			}
			historical.ReferenceCount++
			if err := nw.App.DistrKeeper.SetValidatorHistoricalRewards(ctx, valAddr, newPeriod, historical); err != nil {
				panic(err)
			}

			slashEvent := distrtypes.NewValidatorSlashEvent(newPeriod, math.LegacyNewDecWithPrec(5, 2))
			height := uint64(i)

			if err := nw.App.DistrKeeper.SetValidatorSlashEvent(ctx, valAddr, height, newPeriod, slashEvent); err != nil {
				panic(err)
			}
		}
	}

	s.bondDenom = bondDenom
	s.factory = txFactory
	s.grpcHandler = grpcHandler
	s.keyring = keyring
	s.network = nw
	s.precompile, err = distribution.NewPrecompile(
		s.network.App.DistrKeeper,
		s.network.App.StakingKeeper,
		s.network.App.AuthzKeeper,
	)
	if err != nil {
		panic(err)
	}
}
