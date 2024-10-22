package keeper

import (
	"bytes"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	channelkeeper "github.com/cosmos/ibc-go/v7/modules/core/04-channel/keeper"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/exp/maps"

	bankprecompile "github.com/haqq-network/haqq/precompiles/bank"
	"github.com/haqq-network/haqq/precompiles/bech32"
	distprecompile "github.com/haqq-network/haqq/precompiles/distribution"

	// erc20precompile "github.com/haqq-network/haqq/precompiles/erc20"
	ics20precompile "github.com/haqq-network/haqq/precompiles/ics20"
	// osmosisoutpost "github.com/haqq-network/haqq/precompiles/outposts/osmosis"
	// strideoutpost "github.com/haqq-network/haqq/precompiles/outposts/stride"
	"github.com/haqq-network/haqq/precompiles/p256"
	stakingprecompile "github.com/haqq-network/haqq/precompiles/staking"

	// vestingprecompile "github.com/haqq-network/haqq/precompiles/vesting"
	erc20Keeper "github.com/haqq-network/haqq/x/erc20/keeper"
	transferkeeper "github.com/haqq-network/haqq/x/ibc/transfer/keeper"
	stakingkeeper "github.com/haqq-network/haqq/x/staking/keeper"
	vestingkeeper "github.com/haqq-network/haqq/x/vesting/keeper"
)

// AvailablePrecompiles returns the list of all available precompiled contracts.
// NOTE: this should only be used during initialization of the Keeper.
func AvailablePrecompiles(
	chainID string,
	stakingKeeper stakingkeeper.Keeper,
	distributionKeeper distributionkeeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	erc20Keeper erc20Keeper.Keeper,
	vestingKeeper vestingkeeper.Keeper,
	authzKeeper authzkeeper.Keeper,
	transferKeeper transferkeeper.Keeper,
	channelKeeper channelkeeper.Keeper,
) map[common.Address]vm.PrecompiledContract {
	// Clone the mapping from the latest EVM fork.
	precompiles := maps.Clone(vm.PrecompiledContractsBerlin)

	// secp256r1 precompile as per EIP-7212
	p256Precompile := &p256.Precompile{}

	bech32Precompile, err := bech32.NewPrecompile(6000)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate bech32 precompile: %w", err))
	}

	stakingPrecompile, err := stakingprecompile.NewPrecompile(stakingKeeper, authzKeeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate staking precompile: %w", err))
	}

	distributionPrecompile, err := distprecompile.NewPrecompile(distributionKeeper, stakingKeeper, authzKeeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate distribution precompile: %w", err))
	}

	ibcTransferPrecompile, err := ics20precompile.NewPrecompile(stakingKeeper, transferKeeper, channelKeeper, authzKeeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate ICS20 precompile: %w", err))
	}

	/*
		vestingPrecompile, err := vestingprecompile.NewPrecompile(vestingKeeper, authzKeeper)
		if err != nil {
			panic(fmt.Errorf("failed to instantiate vesting precompile: %w", err))
		}
	*/
	bankPrecompile, err := bankprecompile.NewPrecompile(bankKeeper, erc20Keeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate bank precompile: %w", err))
	}

	/*
		var WEVMOSAddress common.Address
		if utils.IsMainNetwork(chainID) {
			WEVMOSAddress = common.HexToAddress(erc20precompile.WEVMOSContractMainnet)
		} else {
			WEVMOSAddress = common.HexToAddress(erc20precompile.WEVMOSContractTestnet)
		}

		strideOutpost, err := strideoutpost.NewPrecompile(
			WEVMOSAddress,
			transferKeeper,
			erc20Keeper,
			authzKeeper,
			stakingKeeper,
		)
		if err != nil {
			panic(fmt.Errorf("failed to instantiate stride outpost: %w", err))
		}

		osmosisOutpost, err := osmosisoutpost.NewPrecompile(
			WEVMOSAddress,
			authzKeeper,
			bankKeeper,
			transferKeeper,
			stakingKeeper,
			erc20Keeper,
			channelKeeper,
		)
		if err != nil {
			panic(fmt.Errorf("failed to instantiate osmosis outpost: %w", err))
		}
	*/

	// Stateless precompiles
	precompiles[bech32Precompile.Address()] = bech32Precompile
	precompiles[p256Precompile.Address()] = p256Precompile

	// Stateful precompiles
	precompiles[stakingPrecompile.Address()] = stakingPrecompile
	precompiles[distributionPrecompile.Address()] = distributionPrecompile
	// precompiles[vestingPrecompile.Address()] = vestingPrecompile
	precompiles[ibcTransferPrecompile.Address()] = ibcTransferPrecompile
	precompiles[bankPrecompile.Address()] = bankPrecompile

	/*	// Outposts
		precompiles[strideOutpost.Address()] = strideOutpost
		precompiles[osmosisOutpost.Address()] = osmosisOutpost
	*/
	return precompiles
}

// WithPrecompiles sets the available precompiled contracts.
func (k *Keeper) WithPrecompiles(precompiles map[common.Address]vm.PrecompiledContract) *Keeper {
	if k.precompiles != nil {
		panic("available precompiles map already set")
	}

	if len(precompiles) == 0 {
		panic("empty precompiled contract map")
	}

	k.precompiles = precompiles
	return k
}

// Precompiles returns the subset of the available precompiled contracts that
// are active given the current parameters.
func (k Keeper) Precompiles(
	activePrecompiles ...common.Address,
) map[common.Address]vm.PrecompiledContract {
	activePrecompileMap := make(map[common.Address]vm.PrecompiledContract)

	for _, address := range activePrecompiles {
		precompile, ok := k.precompiles[address]
		if !ok {
			panic(fmt.Sprintf("precompiled contract not initialized: %s", address))
		}

		activePrecompileMap[address] = precompile
	}

	return activePrecompileMap
}

// AddEVMExtensions adds the given precompiles to the list of active precompiles in the EVM parameters
// and to the available precompiles map in the Keeper. This function returns an error if
// the precompiles are invalid or duplicated.
func (k *Keeper) AddEVMExtensions(ctx sdk.Context, precompiles ...vm.PrecompiledContract) error {
	params := k.GetParams(ctx)

	addresses := make([]string, len(precompiles))
	precompilesMap := maps.Clone(k.precompiles)

	for i, precompile := range precompiles {
		// add to active precompiles
		address := precompile.Address()
		addresses[i] = address.String()

		// add to available precompiles, but check for duplicates
		if _, ok := precompilesMap[address]; ok {
			return fmt.Errorf("precompile already registered: %s", address)
		}
		precompilesMap[address] = precompile
	}

	params.ActivePrecompiles = append(params.ActivePrecompiles, addresses...)

	// NOTE: the active precompiles are sorted and validated before setting them
	// in the params
	if err := k.SetParams(ctx, params); err != nil {
		return err
	}

	// update the pointer to the map with the newly added EVM Extensions
	k.precompiles = precompilesMap
	return nil
}

// IsAvailablePrecompile returns true if the given precompile address is contained in the
// EVM keeper's available precompiles map.
func (k Keeper) IsAvailablePrecompile(address common.Address) bool {
	_, ok := k.precompiles[address]
	return ok
}

// GetAvailablePrecompileAddrs returns the list of available precompile addresses.
//
// NOTE: uses index based approach instead of append because it's supposed to be faster.
// Check https://stackoverflow.com/questions/21362950/getting-a-slice-of-keys-from-a-map.
func (k Keeper) GetAvailablePrecompileAddrs() []common.Address {
	addresses := make([]common.Address, len(k.precompiles))
	i := 0

	//#nosec G705 -- two operations in for loop here are fine
	for address := range k.precompiles {
		addresses[i] = address
		i++
	}

	sort.Slice(addresses, func(i, j int) bool {
		return bytes.Compare(addresses[i].Bytes(), addresses[j].Bytes()) == -1
	})

	return addresses
}
