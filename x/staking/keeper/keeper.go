package keeper

import (
	addresscodec "cosmossdk.io/core/address"
	storetypes "cosmossdk.io/core/store"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/haqq-network/haqq/utils"
)

// Keeper is a wrapper around the Cosmos SDK staking keeper.
type Keeper struct {
	*stakingkeeper.Keeper
	ak types.AccountKeeper
	bk types.BankKeeper
}

// NewKeeper creates a new staking Keeper wrapper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	authority string,
	validatorAddressCodec addresscodec.Codec,
	consensusAddressCodec addresscodec.Codec,
) *Keeper {
	return &Keeper{
		stakingkeeper.NewKeeper(cdc, storeService, ak, bk, authority, validatorAddressCodec, consensusAddressCodec),
		ak,
		bk,
	}
}

// BaseDenomBankBalance returns the bank balance of aISLM for addr (utils.BaseDenom).
// x/evm SetBalance reconciles native EVM accounts against this denom.
func (k Keeper) BaseDenomBankBalance(ctx sdk.Context, addr sdk.AccAddress) sdkmath.Int {
	return k.bk.GetBalance(ctx, addr, utils.BaseDenom).Amount
}
