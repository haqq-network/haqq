// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package keeper

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	haqqtypes "github.com/haqq-network/haqq/types"
	"github.com/haqq-network/haqq/x/evm/core/vm"
	"github.com/haqq-network/haqq/x/evm/statedb"
	"github.com/haqq-network/haqq/x/evm/types"
)

// Keeper grants access to the EVM module state and implements the go-ethereum StateDB interface.
type Keeper struct {
	// Protobuf codec
	cdc codec.BinaryCodec
	// Store key required for the EVM Prefix KVStore. It is required by:
	// - storing account's Storage State
	// - storing account's Code
	// - storing transaction Logs
	// - storing Bloom filters by block height. Needed for the Web3 API.
	storeKey storetypes.StoreKey

	// key to access the transient store, which is reset on every block during Commit
	transientKey storetypes.StoreKey

	// the address capable of executing a MsgUpdateParams message. Typically, this should be the x/gov module account.
	authority sdk.AccAddress
	// access to account state
	accountKeeper types.AccountKeeper
	// update balance and accounting operations with coins
	bankKeeper types.BankKeeper
	// access historical headers for EVM state transition execution
	stakingKeeper types.StakingKeeper
	// fetch EIP1559 base fee and parameters
	feeMarketKeeper types.FeeMarketKeeper
	// erc20Keeper interface needed to instantiate erc20 precompiles
	erc20Keeper types.Erc20Keeper

	// chain ID number obtained from the context's chain id
	eip155ChainID *big.Int

	// Tracer used to collect execution traces from the EVM transaction execution
	tracer string

	// Legacy subspace
	ss paramstypes.Subspace

	// precompiles defines the map of all available precompiled smart contracts.
	// Some these precompiled contracts might not be active depending on the EVM
	// parameters.
	precompiles map[common.Address]vm.PrecompiledContract
}

// NewKeeper generates new evm module keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey, transientKey storetypes.StoreKey,
	authority sdk.AccAddress,
	ak types.AccountKeeper,
	bankKeeper types.BankKeeper,
	sk types.StakingKeeper,
	fmk types.FeeMarketKeeper,
	erc20Keeper types.Erc20Keeper,
	tracer string,
	ss paramstypes.Subspace,
) *Keeper {
	// ensure evm module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the EVM module account has not been set")
	}

	// ensure the authority account is correct
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(err)
	}

	// NOTE: we pass in the parameter space to the CommitStateDB in order to use custom denominations for the EVM operations
	return &Keeper{
		cdc:             cdc,
		authority:       authority,
		accountKeeper:   ak,
		bankKeeper:      bankKeeper,
		stakingKeeper:   sk,
		feeMarketKeeper: fmk,
		storeKey:        storeKey,
		transientKey:    transientKey,
		tracer:          tracer,
		erc20Keeper:     erc20Keeper,
		ss:              ss,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// WithChainID sets the chain id to the local variable in the keeper
func (k *Keeper) WithChainID(ctx sdk.Context) {
	chainID, err := haqqtypes.ParseChainID(ctx.ChainID())
	if err != nil {
		panic(err)
	}

	if k.eip155ChainID != nil && k.eip155ChainID.Cmp(chainID) != 0 {
		panic("chain id already set")
	}

	k.eip155ChainID = chainID
}

// ChainID returns the EIP155 chain ID for the EVM context
func (k Keeper) ChainID() *big.Int {
	return k.eip155ChainID
}

// ----------------------------------------------------------------------------
// Block Bloom
// Required by Web3 API.
// ----------------------------------------------------------------------------

// EmitBlockBloomEvent emit block bloom events
func (k Keeper) EmitBlockBloomEvent(ctx sdk.Context, bloom ethtypes.Bloom) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBlockBloom,
			sdk.NewAttribute(types.AttributeKeyEthereumBloom, string(bloom.Bytes())),
		),
	)
}

// GetAuthority returns the x/evm module authority address
func (k Keeper) GetAuthority() sdk.AccAddress {
	return k.authority
}

// GetBlockBloomTransient returns bloom bytes for the current block height
func (k Keeper) GetBlockBloomTransient(ctx sdk.Context) *big.Int {
	store := prefix.NewStore(ctx.TransientStore(k.transientKey), types.KeyPrefixTransientBloom)
	//nolint: gosec // G115 blockHeight is positive int64 and can't overflow uint64
	heightBz := sdk.Uint64ToBigEndian(uint64(ctx.BlockHeight()))
	bz := store.Get(heightBz)
	if len(bz) == 0 {
		return big.NewInt(0)
	}

	return new(big.Int).SetBytes(bz)
}

// SetBlockBloomTransient sets the given bloom bytes to the transient store. This value is reset on
// every block.
func (k Keeper) SetBlockBloomTransient(ctx sdk.Context, bloom *big.Int) {
	store := prefix.NewStore(ctx.TransientStore(k.transientKey), types.KeyPrefixTransientBloom)
	//nolint: gosec // G115 blockHeight is positive int64 and can't overflow uint64
	heightBz := sdk.Uint64ToBigEndian(uint64(ctx.BlockHeight()))
	store.Set(heightBz, bloom.Bytes())
}

// ----------------------------------------------------------------------------
// Tx
// ----------------------------------------------------------------------------

// SetTxIndexTransient set the index of processing transaction
func (k Keeper) SetTxIndexTransient(ctx sdk.Context, index uint64) {
	store := ctx.TransientStore(k.transientKey)
	store.Set(types.KeyPrefixTransientTxIndex, sdk.Uint64ToBigEndian(index))
}

// GetTxIndexTransient returns EVM transaction index on the current block.
func (k Keeper) GetTxIndexTransient(ctx sdk.Context) uint64 {
	store := ctx.TransientStore(k.transientKey)
	bz := store.Get(types.KeyPrefixTransientTxIndex)
	if len(bz) == 0 {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// ----------------------------------------------------------------------------
// Log
// ----------------------------------------------------------------------------

// GetLogSizeTransient returns EVM log index on the current block.
func (k Keeper) GetLogSizeTransient(ctx sdk.Context) uint64 {
	store := ctx.TransientStore(k.transientKey)
	bz := store.Get(types.KeyPrefixTransientLogSize)
	if len(bz) == 0 {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// SetLogSizeTransient fetches the current EVM log index from the transient store, increases its
// value by one and then sets the new index back to the transient store.
func (k Keeper) SetLogSizeTransient(ctx sdk.Context, logSize uint64) {
	store := ctx.TransientStore(k.transientKey)
	store.Set(types.KeyPrefixTransientLogSize, sdk.Uint64ToBigEndian(logSize))
}

// ----------------------------------------------------------------------------
// Storage
// ----------------------------------------------------------------------------

// GetAccountStorage return state storage associated with an account
func (k Keeper) GetAccountStorage(ctx sdk.Context, address common.Address) types.Storage {
	storage := types.Storage{}

	k.ForEachStorage(ctx, address, func(key, value common.Hash) bool {
		storage = append(storage, types.NewState(key, value))
		return true
	})

	return storage
}

// ----------------------------------------------------------------------------
// Account
// ----------------------------------------------------------------------------

// Tracer return a default vm.Tracer based on current keeper state
func (k Keeper) Tracer(ctx sdk.Context, msg core.Message, ethCfg *params.ChainConfig) vm.EVMLogger {
	return types.NewTracer(k.tracer, msg, ethCfg, ctx.BlockHeight())
}

// GetAccountWithoutBalance load nonce and codehash without balance,
// more efficient in cases where balance is not needed.
func (k *Keeper) GetAccountWithoutBalance(ctx sdk.Context, addr common.Address) *statedb.Account {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		return nil
	}

	codeHash := types.EmptyCodeHash
	ethAcct, ok := acct.(haqqtypes.EthAccountI)
	if ok {
		codeHash = ethAcct.GetCodeHash().Bytes()
	}

	return &statedb.Account{
		Nonce:    acct.GetSequence(),
		CodeHash: codeHash,
	}
}

// GetAccountOrEmpty returns empty account if not exist, returns error if it's not `EthAccount`
func (k *Keeper) GetAccountOrEmpty(ctx sdk.Context, addr common.Address) statedb.Account {
	acct := k.GetAccount(ctx, addr)
	if acct != nil {
		return *acct
	}

	// empty account
	return statedb.Account{
		Balance:  new(big.Int),
		CodeHash: types.EmptyCodeHash,
	}
}

// GetNonce returns the sequence number of an account, returns 0 if not exists.
func (k *Keeper) GetNonce(ctx sdk.Context, addr common.Address) uint64 {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		return 0
	}

	return acct.GetSequence()
}

// GetBalance load account's balance of gas token
func (k *Keeper) GetBalance(ctx sdk.Context, addr common.Address) *big.Int {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	evmParams := k.GetParams(ctx)
	evmDenom := evmParams.GetEvmDenom()
	// if node is pruned, params is empty. Return invalid value
	if evmDenom == "" {
		return big.NewInt(-1)
	}
	coin := k.bankKeeper.GetBalance(ctx, cosmosAddr, evmDenom)
	return coin.Amount.BigInt()
}

// GetBaseFee returns current base fee, return values:
// - `nil`: london hardfork not enabled.
// - `0`: london hardfork enabled but feemarket is not enabled.
// - `n`: both london hardfork and feemarket are enabled.
func (k Keeper) GetBaseFee(ctx sdk.Context, ethCfg *params.ChainConfig) *big.Int {
	return k.getBaseFee(ctx, types.IsLondon(ethCfg, ctx.BlockHeight()))
}

func (k Keeper) getBaseFee(ctx sdk.Context, london bool) *big.Int {
	if !london {
		return nil
	}
	baseFee := k.feeMarketKeeper.GetBaseFee(ctx)
	if baseFee == nil {
		// return 0 if feemarket not enabled.
		baseFee = big.NewInt(0)
	}
	return baseFee
}

// GetMinGasMultiplier returns the MinGasMultiplier param from the fee market module
func (k Keeper) GetMinGasMultiplier(ctx sdk.Context) math.LegacyDec {
	return k.feeMarketKeeper.GetParams(ctx).MinGasMultiplier
}

// ResetTransientGasUsed reset gas used to prepare for execution of current cosmos tx, called in ante handler.
func (k Keeper) ResetTransientGasUsed(ctx sdk.Context) {
	store := ctx.TransientStore(k.transientKey)
	store.Delete(types.KeyPrefixTransientGasUsed)
}

// GetTransientGasUsed returns the gas used by current cosmos tx.
func (k Keeper) GetTransientGasUsed(ctx sdk.Context) uint64 {
	store := ctx.TransientStore(k.transientKey)
	bz := store.Get(types.KeyPrefixTransientGasUsed)
	if len(bz) == 0 {
		return 0
	}
	return sdk.BigEndianToUint64(bz)
}

// SetTransientGasUsed sets the gas used by current cosmos tx.
func (k Keeper) SetTransientGasUsed(ctx sdk.Context, gasUsed uint64) {
	store := ctx.TransientStore(k.transientKey)
	bz := sdk.Uint64ToBigEndian(gasUsed)
	store.Set(types.KeyPrefixTransientGasUsed, bz)
}

// AddTransientGasUsed accumulate gas used by each eth msgs included in current cosmos tx.
func (k Keeper) AddTransientGasUsed(ctx sdk.Context, gasUsed uint64) (uint64, error) {
	result := k.GetTransientGasUsed(ctx) + gasUsed
	if result < gasUsed {
		return 0, errorsmod.Wrap(types.ErrGasOverflow, "transient gas used")
	}
	k.SetTransientGasUsed(ctx, result)
	return result, nil
}
