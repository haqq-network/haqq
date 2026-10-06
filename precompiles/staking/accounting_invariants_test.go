package staking_test

import (
	"fmt"
	"math/big"
	"math/rand"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	"github.com/haqq-network/haqq/testutil/integration/haqq/factory"
	testutils "github.com/haqq-network/haqq/testutil/integration/haqq/utils"
	utiltx "github.com/haqq-network/haqq/testutil/tx"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// chaosStep mirrors PrecompileChaos.Step for ABI encoding.
type chaosStep struct {
	Kind   uint8
	Target common.Address
	Value  *big.Int
	Data   []byte
	Gas    uint64
}

const (
	stepSend uint8 = iota
	stepCall
	stepNested
)

// abiPacker is what the static precompiles expose through their embedded ABI.
type abiPacker interface {
	Pack(name string, args ...interface{}) ([]byte, error)
}

// chaosEnv holds what the program generator draws from.
type chaosEnv struct {
	s         *PrecompileTestSuite
	fixture   evmtypes.CompiledContract
	chaos     common.Address
	validator string
	staking   common.Address
	distr     common.Address
	bank      common.Address
	packers   map[common.Address]abiPacker
	// value recipients: fresh accounts, a funded account, the fixture itself,
	// a precompile and module accounts the bank refuses to credit
	recipients []common.Address
}

func (s *PrecompileTestSuite) newChaosEnv(funding math.Int) *chaosEnv {
	chaos, fixture := s.deployFunded(funding, contracts.LoadPrecompileChaosContract)

	ctx := s.network.GetContext()
	params := s.network.App.EvmKeeper.GetParams(ctx)
	env := &chaosEnv{
		s:         s,
		fixture:   fixture,
		chaos:     chaos,
		validator: s.network.GetValidators()[0].OperatorAddress,
		staking:   common.HexToAddress(evmtypes.StakingPrecompileAddress),
		distr:     common.HexToAddress(evmtypes.DistributionPrecompileAddress),
		bank:      common.HexToAddress(evmtypes.BankPrecompileAddress),
		packers:   make(map[common.Address]abiPacker),
	}
	for _, addr := range []common.Address{env.staking, env.distr, env.bank} {
		precompile, found, err := s.network.App.EvmKeeper.GetStaticPrecompileInstance(&params, addr)
		s.Require().NoError(err)
		s.Require().True(found, "precompile %s must be active", addr)
		packer, ok := precompile.(abiPacker)
		s.Require().True(ok, "precompile %s must expose its ABI", addr)
		env.packers[addr] = packer
	}

	env.recipients = []common.Address{
		utiltx.GenerateAddress(),
		utiltx.GenerateAddress(),
		utiltx.GenerateAddress(),
		s.keyring.GetAddr(1),
		chaos,
		env.staking,
		common.BytesToAddress(authtypes.NewModuleAddress(stakingtypes.BondedPoolName)),
		common.BytesToAddress(authtypes.NewModuleAddress(distrtypes.ModuleName)),
	}
	return env
}

// deployFunded deploys a fixture, funds it and switches off coinomics so the
// aISLM supply only moves because of the transactions under test.
func (s *PrecompileTestSuite) deployFunded(
	funding math.Int,
	load func() (evmtypes.CompiledContract, error),
) (common.Address, evmtypes.CompiledContract) {
	ctx := s.network.GetContext()
	params := s.network.App.CoinomicsKeeper.GetParams(ctx)
	params.EnableCoinomics = false
	s.network.App.CoinomicsKeeper.SetParams(ctx, params)

	fixture, err := load()
	s.Require().NoError(err)

	signer := s.keyring.GetKey(0)
	addr, err := s.factory.DeployContract(signer.Priv, evmtypes.EvmTxArgs{}, factory.ContractDeploymentData{Contract: fixture})
	s.Require().NoError(err)
	s.Require().NoError(s.network.NextBlock())

	s.Require().NoError(testutils.FundAccountWithBaseDenom(s.factory, s.network, signer, addr.Bytes(), funding))
	s.Require().NoError(s.network.NextBlock())
	return addr, fixture
}

func (e *chaosEnv) pack(precompile common.Address, method string, args ...interface{}) []byte {
	input, err := e.packers[precompile].Pack(method, args...)
	e.s.Require().NoError(err)
	return input
}

func (e *chaosEnv) nested(steps []chaosStep, revert bool) chaosStep {
	data, err := e.fixture.ABI.Methods["run"].Inputs.Pack(steps, revert)
	e.s.Require().NoError(err)
	return chaosStep{Kind: stepNested, Target: e.chaos, Value: common.Big0, Data: data}
}

func send(to common.Address, amount *big.Int) chaosStep {
	return chaosStep{Kind: stepSend, Target: to, Value: amount, Data: []byte{}}
}

func call(to common.Address, value *big.Int, data []byte, gas uint64) chaosStep {
	return chaosStep{Kind: stepCall, Target: to, Value: value, Data: data, Gas: gas}
}

var chaosAmounts = []*big.Int{big.NewInt(1), big.NewInt(1e12), big.NewInt(1e15), big.NewInt(1e17)}

// precompileCall draws one call the fixture makes to a precompile: successful
// and failing transactions and queries, calls rejected on their calldata alone,
// value sent along, and gas limits low enough to run out inside the precompile.
func (e *chaosEnv) precompileCall(r *rand.Rand) chaosStep {
	amount := chaosAmounts[r.Intn(len(chaosAmounts))]
	gas := []uint64{0, 0, 40_000, 400_000}[r.Intn(4)]
	recipient := e.recipients[r.Intn(len(e.recipients))]

	switch r.Intn(9) {
	case 0:
		return call(e.staking, common.Big0, e.pack(e.staking, "delegate", e.chaos, e.validator, amount), gas)
	case 1: // more than the fixture holds: fails inside the precompile
		return call(e.staking, common.Big0, e.pack(e.staking, "delegate", e.chaos, e.validator, new(big.Int).Lsh(amount, 80)), gas)
	case 2:
		return call(e.staking, common.Big0, e.pack(e.staking, "undelegate", e.chaos, e.validator, amount), gas)
	case 3:
		return call(e.distr, common.Big0, e.pack(e.distr, "claimRewards", e.chaos, uint32(5)), gas)
	case 4:
		withdrawer := sdk.AccAddress(recipient.Bytes()).String()
		return call(e.distr, common.Big0, e.pack(e.distr, "setWithdrawAddress", e.chaos, withdrawer), gas)
	case 5:
		return call(e.bank, common.Big0, e.pack(e.bank, "balances", e.chaos), gas)
	case 6: // unknown selector: rejected before any state is touched
		return call(e.staking, common.Big0, []byte{0xde, 0xad, 0xbe, 0xef}, gas)
	case 7: // a valid call with value attached: the flush must fail as a unit
		return call(e.staking, amount, e.pack(e.staking, "delegate", e.chaos, e.validator, amount), gas)
	default: // empty calldata with value
		return call(e.staking, amount, []byte{}, gas)
	}
}

// program draws a random program of at most four steps, nesting sub-programs
// that may revert up to maxDepth levels deep.
func (e *chaosEnv) program(r *rand.Rand, depth int) []chaosStep {
	const maxDepth = 3
	steps := make([]chaosStep, 0, 4)
	for i := 1 + r.Intn(4); i > 0; i-- {
		switch k := r.Intn(10); {
		case k < 3:
			steps = append(steps, send(e.recipients[r.Intn(len(e.recipients))], chaosAmounts[r.Intn(len(chaosAmounts))]))
		case k < 8 || depth >= maxDepth:
			steps = append(steps, e.precompileCall(r))
		default:
			steps = append(steps, e.nested(e.program(r, depth+1), r.Intn(2) == 0))
		}
	}
	return steps
}

// run delivers one program in its own transaction and checks that it left the
// chain's accounting intact, whatever the transaction's own outcome.
func (e *chaosEnv) run(label string, steps []chaosStep, revert bool) bool {
	s := e.s
	supplyBefore := s.bankSupply()

	res, err := s.factory.ExecuteContractCall(
		s.keyring.GetPrivKey(0),
		evmtypes.EvmTxArgs{To: &e.chaos, GasLimit: 8_000_000},
		factory.CallArgs{ContractABI: e.fixture.ABI, MethodName: "run", Args: []interface{}{steps, revert}},
	)
	s.Require().NoError(s.network.NextBlock())

	s.Require().Equal(supplyBefore.String(), s.bankSupply().String(), "%s: aISLM supply moved", label)
	s.Require().NoError(s.network.CheckAccountingInvariants(), label)
	return err == nil && res.IsOK()
}

// TestAccountingInvariantsUnderRandomPrograms drives random interleavings of EVM
// value transfers, precompile calls and reverts at every call depth through a
// contract, and checks after each transaction that the aISLM supply has not
// moved (coinomics is off) and that the bank, staking and distribution module
// invariants still hold.
//
// Every accounting bug found around the StateDB and the precompiles so far -
// a reverted frame's mint surviving a failed precompile setup, a mint orphaned
// by a failed flush, a commit draining a staking pool - breaks one of these
// checks. The fixed scenarios replay those shapes; the random programs look
// for new ones.
func (s *PrecompileTestSuite) TestAccountingInvariantsUnderRandomPrograms() {
	s.SetupTest()
	env := s.newChaosEnv(math.NewInt(5e18))
	fresh := utiltx.GenerateAddress()

	scenarios := []struct {
		name  string
		steps []chaosStep
	}{
		{
			// a payment and a rejected precompile call in a reverted frame, then a
			// payment that leaves the payer dirty at commit
			"reverted payment around a rejected precompile call",
			[]chaosStep{
				env.nested([]chaosStep{
					send(utiltx.GenerateAddress(), big.NewInt(1e17)),
					call(env.staking, common.Big0, []byte{0xde, 0xad, 0xbe, 0xef}, 200_000),
				}, true),
				send(fresh, big.NewInt(1)),
			},
		},
		{
			// the same around a failing precompile body
			"reverted payment around a failing precompile call",
			[]chaosStep{
				env.nested([]chaosStep{
					send(utiltx.GenerateAddress(), big.NewInt(1e17)),
					call(env.staking, common.Big0, env.pack(env.staking, "delegate", env.chaos, env.validator, new(big.Int).Lsh(big.NewInt(1), 100)), 0),
				}, true),
				send(fresh, big.NewInt(1)),
			},
		},
		{
			// a successful delegation in a reverted frame
			"reverted successful delegation",
			[]chaosStep{
				env.nested([]chaosStep{
					call(env.staking, common.Big0, env.pack(env.staking, "delegate", env.chaos, env.validator, big.NewInt(1e15)), 0),
				}, true),
				send(fresh, big.NewInt(1)),
			},
		},
		{
			// value sent to a precompile along with a valid call: the flush has to
			// credit a blocked address and must fail without leaving a mint behind
			"value with a valid precompile call",
			[]chaosStep{
				call(env.staking, big.NewInt(1e15), env.pack(env.staking, "delegate", env.chaos, env.validator, big.NewInt(1e15)), 0),
			},
		},
		{
			// a delegation followed by a payment to the bonded pool
			"delegation then a payment to the bonded pool",
			[]chaosStep{
				call(env.staking, common.Big0, env.pack(env.staking, "delegate", env.chaos, env.validator, big.NewInt(1e15)), 0),
				send(common.BytesToAddress(authtypes.NewModuleAddress(stakingtypes.BondedPoolName)), big.NewInt(1)),
			},
		},
	}
	for _, sc := range scenarios {
		env.run(sc.name, sc.steps, false)
	}

	const programs = 60
	succeeded := 0
	for seed := int64(1); seed <= programs; seed++ {
		r := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic test input, not a secret
		if env.run(fmt.Sprintf("seed %d", seed), env.program(r, 0), r.Intn(5) == 0) {
			succeeded++
		}
	}

	// the generator must actually reach the precompiles' success paths
	s.T().Logf("%d of %d random program transactions succeeded", succeeded, programs)
	s.Require().Positive(succeeded, "no program transaction succeeded")
	delegation, err := s.network.App.StakingKeeper.GetDelegation(
		s.network.GetContext(), env.chaos.Bytes(), s.mustValAddr(env.validator),
	)
	s.Require().NoError(err, "no delegation through the precompile ever succeeded")
	s.Require().True(delegation.Shares.IsPositive())
}

func (s *PrecompileTestSuite) mustValAddr(bech32 string) sdk.ValAddress {
	addr, err := sdk.ValAddressFromBech32(bech32)
	s.Require().NoError(err)
	return addr
}
