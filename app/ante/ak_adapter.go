package ante

import (
	"context"
	"time"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	evmtypes "github.com/haqq-network/haqq/x/evm/types"
)

// accountKeeperAdapter adapts an evmtypes.AccountKeeper to satisfy auth/ante.AccountKeeper.
type accountKeeperAdapter struct {
	ak evmtypes.AccountKeeper
}

func NewAccountKeeperAdapter(ak evmtypes.AccountKeeper) authante.AccountKeeper {
	return &accountKeeperAdapter{ak: ak}
}

func (a *accountKeeperAdapter) GetParams(ctx context.Context) (params authtypes.Params) {
	return a.ak.GetParams(ctx)
}

func (a *accountKeeperAdapter) GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return a.ak.GetAccount(ctx, addr)
}

func (a *accountKeeperAdapter) SetAccount(ctx context.Context, acc sdk.AccountI) {
	a.ak.SetAccount(ctx, acc)
}

func (a *accountKeeperAdapter) GetModuleAddress(moduleName string) sdk.AccAddress {
	return a.ak.GetModuleAddress(moduleName)
}

func (a *accountKeeperAdapter) AddressCodec() address.Codec {
	return a.ak.AddressCodec()
}

func (a *accountKeeperAdapter) UnorderedTransactionsEnabled() bool {
	return false
}

func (a *accountKeeperAdapter) RemoveExpiredUnorderedNonces(ctx sdk.Context) error {
	return nil
}

func (a *accountKeeperAdapter) TryAddUnorderedNonce(ctx sdk.Context, sender []byte, timestamp time.Time) error {
	return nil
}
