package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/coinomics/types"
)

func (k Keeper) GetInflation(ctx sdk.Context) sdk.Dec {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrefixInflation)
	if len(bz) == 0 {
		return sdk.ZeroDec()
	}

	var inflationValue sdk.Dec
	err := inflationValue.Unmarshal(bz)
	if err != nil {
		panic(fmt.Errorf("unable to unmarshal inflationValue value: %w", err))
	}

	return inflationValue
}

func (k Keeper) SetInflation(ctx sdk.Context, inflation sdk.Dec) {
	binaryInfValue, err := inflation.Marshal()
	if err != nil {
		panic(fmt.Errorf("unable to marshal amount value: %w", err))
	}

	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyPrefixInflation, binaryInfValue)
}

// GetEra gets current era
func (k Keeper) GetEra(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrefixEra)
	if len(bz) == 0 {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// SetEra stores the current era
func (k Keeper) SetEra(ctx sdk.Context, era uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyPrefixEra, sdk.Uint64ToBigEndian(era))
}

// GetStartEraBlock gets current era start block number
func (k Keeper) GetEraStartedAtBlock(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrefixEraStartedAtBlock)
	if len(bz) == 0 {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// SetStartEraBlock stores the start era block number
func (k Keeper) SetEraStartedAtBlock(ctx sdk.Context, block uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyPrefixEraStartedAtBlock, sdk.Uint64ToBigEndian(block))
}

func (k Keeper) GetEraTargetMint(ctx sdk.Context) sdk.Coin {
	params := k.GetParams(ctx)

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KetPrefixEraTargetMint)
	if len(bz) == 0 {
		return sdk.NewCoin(params.MintDenom, sdk.ZeroInt())
	}

	var eraTargetMintValue sdk.Coin
	err := eraTargetMintValue.Unmarshal(bz)
	if err != nil {
		panic(fmt.Errorf("unable to unmarshal eraTargetMintValue value: %w", err))
	}

	return eraTargetMintValue
}

func (k Keeper) SetEraTargetMint(ctx sdk.Context, eraMint sdk.Coin) {
	binaryEraTargetMintValue, err := eraMint.Marshal()
	if err != nil {
		panic(fmt.Errorf("unable to marshal amount value: %w", err))
	}

	store := ctx.KVStore(k.storeKey)
	store.Set(types.KetPrefixEraTargetMint, binaryEraTargetMintValue)
}

func (k Keeper) GetEraClosingSupply(ctx sdk.Context) sdk.Coin {
	params := k.GetParams(ctx)

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrefixEraClosingSupply)
	if len(bz) == 0 {
		return sdk.NewCoin(params.MintDenom, sdk.ZeroInt())
	}

	var eraTarget sdk.Coin
	err := eraTarget.Unmarshal(bz)
	if err != nil {
		panic(fmt.Errorf("unable to unmarshal eraTarget value: %w", err))
	}

	return eraTarget
}

func (k Keeper) SetEraClosingSupply(ctx sdk.Context, eraClosingSupply sdk.Coin) {
	binaryEraClosingSupply, err := eraClosingSupply.Marshal()
	if err != nil {
		panic(fmt.Errorf("unable to marshal amount value: %w", err))
	}

	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyPrefixEraClosingSupply, binaryEraClosingSupply)
}

func (k Keeper) GetMaxSupply(ctx sdk.Context) sdk.Coin {
	params := k.GetParams(ctx)

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrefixMaxSupply)
	if len(bz) == 0 {
		return sdk.NewCoin(params.MintDenom, sdk.ZeroInt())
	}

	var maxSupply sdk.Coin
	err := maxSupply.Unmarshal(bz)
	if err != nil {
		panic(fmt.Errorf("unable to unmarshal maxSupply value: %w", err))
	}

	return maxSupply
}

func (k Keeper) SetMaxSupply(ctx sdk.Context, maxSupply sdk.Coin) {
	binaryMaxSupply, err := maxSupply.Marshal()
	if err != nil {
		panic(fmt.Errorf("unable to marshal amount value: %w", err))
	}

	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyPrefixMaxSupply, binaryMaxSupply)
}
