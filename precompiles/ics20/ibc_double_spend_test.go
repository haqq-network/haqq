package ics20_test

import (
	"math/big"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	"github.com/cosmos/cosmos-sdk/baseapp"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/ethereum/go-ethereum/common"

	haqqcontracts "github.com/haqq-network/haqq/contracts"
	"github.com/haqq-network/haqq/precompiles/testutil/contracts"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// Regression (PR #460): an ICS20 precompile transfer of a native ERC20 pair used to
// auto-convert the caller's ERC20 tokens when its bank coin balance fell short
// (x/ibc/transfer/keeper/msg_server.go -> x/erc20 ConvertERC20). The conversion is a
// nested EVM call that commits into the precompile cache context through a second
// StateDB, which the calling transaction's StateDB never reads. That StateDB kept
// serving the pre-conversion balance for the rest of the transaction and wrote it
// back on its final commit, so the caller could spend the escrowed tokens again -
// whether or not it had touched its balance slot before the call.
//
// Code reached from a precompile now runs on a marked context: the transfer keeper
// refuses to convert there, and x/evm refuses to commit any nested EVM call or to
// write EVM state through its keeper. An EVM caller has to hold the coins
// (converted beforehand through MsgConvertERC20) to transfer a native ERC20 over IBC.
//
// Run: go test ./precompiles/ics20/ -run TestPrecompileIntegrationTestSuite \
//
//	-ginkgo.focus "ICS20 native ERC20 auto-convert"
var _ = Describe("ICS20 native ERC20 auto-convert", func() {
	var (
		fixture        evmtypes.CompiledContract
		fixtureAddr    common.Address
		erc20Addr      common.Address
		tokenPairDenom string
		sink           common.Address

		mintAmount      = big.NewInt(1000)
		convertedAmount = big.NewInt(900)
		leftoverAmount  = big.NewInt(100) // == mintAmount - convertedAmount
	)

	erc20ABI := haqqcontracts.ERC20MinterBurnerDecimalsContract.ABI

	callArgs := func(args contracts.CallArgs) contracts.CallArgs {
		args.PrivKey = s.keyring.GetPrivKey(0)
		args.GasPrice = gasPrice
		if args.GasLimit == 0 {
			args.GasLimit = 5_000_000
		}
		return args
	}

	// call delivers an Ethereum tx from keyring account 0 and requires EVM success.
	call := func(args contracts.CallArgs) *evmtypes.MsgEthereumTxResponse {
		_, ethRes, err := contracts.Call(s.chainA.GetContext(), s.network.App, callArgs(args))
		Expect(err).To(BeNil(), "tx delivery failed: %v", err)
		Expect(ethRes.Failed()).To(BeFalse(), "evm execution failed: %s", ethRes.VmError)
		s.chainA.NextBlock()
		return ethRes
	}

	// callExpectRevert delivers an Ethereum tx that must revert as a whole.
	callExpectRevert := func(args contracts.CallArgs) {
		_, ethRes, err := contracts.Call(s.chainA.GetContext(), s.network.App, callArgs(args))
		Expect(err != nil || ethRes.Failed()).To(BeTrue(), "the transaction must revert")
		s.chainA.NextBlock()
	}

	tokenBalance := func(addr common.Address) *big.Int {
		bal := s.network.App.Erc20Keeper.BalanceOf(s.chainA.GetContext(), erc20ABI, erc20Addr, addr)
		Expect(bal).ToNot(BeNil())
		return bal
	}

	totalSupply := func() *big.Int {
		res, err := s.network.App.EvmKeeper.CallEVM(s.chainA.GetContext(), erc20ABI, erc20types.ModuleAddress, erc20Addr, false, "totalSupply")
		Expect(err).To(BeNil())
		out, err := erc20ABI.Unpack("totalSupply", res.Ret)
		Expect(err).To(BeNil())
		return out[0].(*big.Int)
	}

	// expectConserved checks that every ERC20 token is accounted for exactly once:
	// held by the fixture, by the sink, or by the erc20 module as conversion escrow.
	expectConserved := func() {
		Expect(totalSupply()).To(Equal(mintAmount), "no ERC20 mint occurred")
		sum := new(big.Int).Add(tokenBalance(fixtureAddr), tokenBalance(sink))
		sum.Add(sum, tokenBalance(erc20types.ModuleAddress))
		Expect(sum).To(Equal(mintAmount), "sum of ERC20 balances must equal totalSupply (no double count)")
	}

	respend := func(dirty, amount, again *big.Int) contracts.CallArgs {
		return contracts.CallArgs{
			ContractAddr: fixtureAddr,
			ContractABI:  fixture.ABI,
			MethodName:   "dirtyTransferThenRespend",
			Args:         []interface{}{erc20Addr, sink, dirty, amount, again, s.chainB.GetTimeoutHeight()},
		}
	}

	BeforeEach(func() {
		s = new(PrecompileTestSuite)
		s.SetT(ist)
		s.suiteIBCTesting = true
		s.SetupTest()

		var err error
		fixture, err = contracts.LoadIbcDoubleSpendContract()
		Expect(err).To(BeNil(), "error while loading the fixture: %v", err)

		cacheCtx, _ := s.chainA.GetContext().CacheContext()
		queryHelper := baseapp.NewQueryServerTestHelper(cacheCtx, s.network.App.InterfaceRegistry())
		evmtypes.RegisterQueryServer(queryHelper, s.network.App.EvmKeeper)
		fixtureAddr, err = DeployContract(
			s.chainA.GetContext(),
			s.network.App,
			s.keyring.GetPrivKey(0),
			gasPrice,
			evmtypes.NewQueryClient(queryHelper),
			fixture,
		)
		Expect(err).To(BeNil(), "error while deploying the fixture: %v", err)
		s.chainA.NextBlock()

		// native ERC20 token, registered through the governance authority as in the
		// existing "transfer ERC20" specs
		erc20Addr, err = s.DeployERC20Contract(s.chainA, testERC20.Name, testERC20.Symbol, testERC20.Decimals)
		Expect(err).To(BeNil(), "error while deploying ERC20 contract: %v", err)
		_, err = s.network.App.Erc20Keeper.RegisterERC20(s.chainA.GetContext(), &erc20types.MsgRegisterERC20{
			Authority:      authtypes.NewModuleAddress("gov").String(),
			Erc20Addresses: []string{erc20Addr.Hex()},
		})
		Expect(err).To(BeNil(), "error while registering the token pair: %v", err)
		tokenPairDenom = erc20types.CreateDenom(erc20Addr.Hex())

		pairID := s.network.App.Erc20Keeper.GetTokenPairID(s.chainA.GetContext(), tokenPairDenom)
		pair, found := s.network.App.Erc20Keeper.GetTokenPair(s.chainA.GetContext(), pairID)
		Expect(found).To(BeTrue())
		Expect(pair.IsNativeERC20()).To(BeTrue())
		Expect(pair.Enabled).To(BeTrue())

		// fund the fixture contract with ERC20 tokens only (no bank coins)
		call(contracts.CallArgs{
			ContractAddr: erc20Addr,
			ContractABI:  erc20ABI,
			MethodName:   "mint",
			Args:         []interface{}{fixtureAddr, mintAmount},
		})
		Expect(tokenBalance(fixtureAddr)).To(Equal(mintAmount))
		Expect(totalSupply()).To(Equal(mintAmount))
		Expect(s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), fixtureAddr.Bytes(), tokenPairDenom).Amount.IsZero()).To(BeTrue())

		sink = s.keyring.GetAddr(1)

		call(contracts.CallArgs{
			ContractAddr: fixtureAddr,
			ContractABI:  fixture.ABI,
			MethodName:   "configure",
			Args: []interface{}{
				s.transferPath.EndpointA.ChannelConfig.PortID,
				s.transferPath.EndpointA.ChannelID,
				tokenPairDenom,
				s.chainB.SenderAccount.GetAddress().String(),
			},
		})
	})

	expectPacketInFlight := func(seq uint64, amount *big.Int) {
		ctx := s.chainA.GetContext()
		port := s.transferPath.EndpointA.ChannelConfig.PortID
		channel := s.transferPath.EndpointA.ChannelID
		escrow := transfertypes.GetEscrowAddress(port, channel)

		Expect(s.network.App.IBCKeeper.ChannelKeeper.GetPacketCommitment(ctx, port, channel, seq)).ToNot(BeEmpty(),
			"packet commitment for the IBC transfer must exist")
		Expect(s.network.App.BankKeeper.GetBalance(ctx, escrow, tokenPairDenom).Amount.BigInt()).To(Equal(amount),
			"converted coins must be escrowed for the in-flight packet")
		Expect(s.network.App.TransferKeeper.GetTotalEscrowForDenom(ctx, tokenPairDenom).Amount.BigInt()).To(Equal(amount))
		Expect(s.network.App.BankKeeper.GetSupply(ctx, tokenPairDenom).Amount.BigInt()).To(Equal(amount),
			"bank supply of the pair denom equals the converted amount")
		Expect(tokenBalance(erc20types.ModuleAddress)).To(Equal(amount),
			"erc20 module escrows the converted ERC20 tokens backing the coins")
	}

	expectNothingMoved := func() {
		ctx := s.chainA.GetContext()
		port := s.transferPath.EndpointA.ChannelConfig.PortID
		channel := s.transferPath.EndpointA.ChannelID

		Expect(s.network.App.IBCKeeper.ChannelKeeper.GetPacketCommitment(ctx, port, channel, 1)).To(BeEmpty(),
			"no packet may be sent")
		Expect(s.network.App.BankKeeper.GetSupply(ctx, tokenPairDenom).Amount.IsZero()).To(BeTrue(),
			"no coins may be minted for the pair")
		Expect(tokenBalance(fixtureAddr)).To(Equal(mintAmount), "the fixture keeps all its tokens")
		Expect(tokenBalance(sink).Sign()).To(BeZero(), "the sink receives nothing")
		Expect(tokenBalance(erc20types.ModuleAddress).Sign()).To(BeZero(), "nothing is escrowed for conversion")
		expectConserved()
	}

	Context("when the caller holds only ERC20 tokens", func() {
		// Both shapes of the double spend: with the balance slot untouched before the
		// precompile call (the outer StateDB loads it from the pre-conversion state
		// afterwards) and with it dirtied first (the outer commit wrote the stale
		// value back over the conversion). Either way the re-spend after the transfer
		// would have moved tokens the conversion had already escrowed.
		for _, dirty := range []*big.Int{big.NewInt(0), leftoverAmount} {
			dirty := dirty
			amount := new(big.Int).Sub(mintAmount, dirty)

			It("refuses to auto-convert from inside the EVM (dirtied first: "+dirty.String()+")", func() {
				callExpectRevert(respend(dirty, amount, amount))
				expectNothingMoved()
			})
		}

		It("refuses a plain transfer as well", func() {
			callExpectRevert(contracts.CallArgs{
				ContractAddr: fixtureAddr,
				ContractABI:  fixture.ABI,
				MethodName:   "dirtyThenTransfer",
				Args:         []interface{}{erc20Addr, sink, big.NewInt(0), mintAmount, s.chainB.GetTimeoutHeight()},
			})
			expectNothingMoved()
		})
	})

	Context("when the caller converted its tokens beforehand", func() {
		BeforeEach(func() {
			s.convertERC20ToCoins(erc20Addr, fixtureAddr, convertedAmount)
			Expect(tokenBalance(fixtureAddr)).To(Equal(leftoverAmount))
			Expect(s.network.App.BankKeeper.GetBalance(s.chainA.GetContext(), fixtureAddr.Bytes(), tokenPairDenom).Amount.BigInt()).
				To(Equal(convertedAmount))
		})

		It("transfers the coins and leaves only the real ERC20 balance to spend", func() {
			ethRes := call(respend(big.NewInt(0), convertedAmount, leftoverAmount))
			out, err := fixture.ABI.Unpack("dirtyTransferThenRespend", ethRes.Ret)
			Expect(err).To(BeNil())

			Expect(out[0].(*big.Int)).To(Equal(leftoverAmount), "the fixture sees the ERC20 balance it really has")
			Expect(out[1].(bool)).To(BeTrue(), "spending the unconverted leftover succeeds")

			expectPacketInFlight(1, convertedAmount)
			Expect(tokenBalance(fixtureAddr).Sign()).To(BeZero())
			Expect(tokenBalance(sink)).To(Equal(leftoverAmount))
			expectConserved()
		})

		It("cannot spend more ERC20 than it holds after the transfer", func() {
			ethRes := call(respend(leftoverAmount, convertedAmount, convertedAmount))
			out, err := fixture.ABI.Unpack("dirtyTransferThenRespend", ethRes.Ret)
			Expect(err).To(BeNil())

			Expect(out[0].(*big.Int).Sign()).To(BeZero(), "the leftover went to the sink before the transfer")
			Expect(out[1].(bool)).To(BeFalse(), "re-spending the transferred amount must revert")

			expectPacketInFlight(1, convertedAmount)
			Expect(tokenBalance(fixtureAddr).Sign()).To(BeZero())
			Expect(tokenBalance(sink)).To(Equal(leftoverAmount), "the sink only received the real leftover")
			expectConserved()
		})
	})
})
