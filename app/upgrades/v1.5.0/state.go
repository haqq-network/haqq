package v150

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/iavl"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
	dbm "github.com/tendermint/tm-db"
)

const commitInfoKeyFmt = "s/%d" // s/<version>

func (r *RevestingUpgradeHandler) loadStateOnHeight() error {
	r.ctx.Logger().Info("Loading history state")

	infos := make(map[string]storetypes.StoreInfo)
	cInfo, err := getCommitInfo(r.db, r.height)
	if err != nil {
		return err
	}

	// convert StoreInfos slice to map
	for _, storeInfo := range cInfo.StoreInfos {
		infos[storeInfo.Name] = storeInfo
	}

	var db dbm.DB
	for _, key := range r.keys {
		switch key.Name() {
		case authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey:
			prefix := "s/k:" + key.Name() + "/"
			db = dbm.NewPrefixDB(r.db, []byte(prefix))
			commitID := getCommitID(infos, key.Name())
			store, err := iavl.LoadStore(db, r.ctx.Logger(), key, commitID, false, iavl.DefaultIAVLCacheSize, true)
			if err != nil {
				return errors.Wrap(err, "failed to load store")
			}

			r.stores[key] = store
		}
	}

	if len(r.stores) < 3 {
		return errors.Wrap(err, "failed to load all necessary stores (acc, bank, staking)")
	}

	return nil
}

func (r *RevestingUpgradeHandler) getBondableBalanceFromHistory(acc authtypes.AccountI) (sdk.Coin, error) {
	bondDenom := r.StakingKeeper.BondDenom(r.ctx)
	bankStoreKey := r.keys[banktypes.StoreKey]
	bankStore := r.stores[bankStoreKey]
	store := prefix.NewStore(bankStore, banktypes.CreateAccountBalancesPrefix(acc.GetAddress().Bytes()))

	bz := store.Get([]byte(bondDenom))
	balance, err := bankkeeper.UnmarshalBalanceCompat(r.cdc, bz, bondDenom)
	if err != nil {
		return sdk.NewCoin(bondDenom, math.ZeroInt()), errors.Wrap(err, "failed to unmarshal balance from history store")
	}

	return balance, nil
}

func (r *RevestingUpgradeHandler) getValidatorFromHistory(addr sdk.ValAddress) (validator stakingtypes.Validator, found bool) {
	stakingStoreKey := r.keys[stakingtypes.StoreKey]
	stakingStore := r.stores[stakingStoreKey]

	value := stakingStore.Get(stakingtypes.GetValidatorKey(addr))
	if value == nil {
		return validator, false
	}

	validator = stakingtypes.MustUnmarshalValidator(r.cdc, value)
	return validator, true
}

func (r *RevestingUpgradeHandler) getDelegatedCoinsFromHistory(acc authtypes.AccountI) (sdk.Coin, error) {
	bondDenom := r.StakingKeeper.BondDenom(r.ctx)
	stakingStoreKey := r.keys[stakingtypes.StoreKey]
	stakingStore := r.stores[stakingStoreKey]

	delegatorPrefixKey := stakingtypes.GetDelegationsKey(acc.GetAddress())
	delegations := make([]stakingtypes.Delegation, 0)
	iterator := sdk.KVStorePrefixIterator(stakingStore, delegatorPrefixKey) // smallest to largest
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		delegation := stakingtypes.MustUnmarshalDelegation(r.cdc, iterator.Value())
		delegations = append(delegations, delegation)
	}

	amount := sdk.NewCoin(bondDenom, math.ZeroInt())
	for _, delegation := range delegations {
		val, found := r.getValidatorFromHistory(delegation.GetValidatorAddr())
		if !found {
			return amount, errors.Wrap(stakingtypes.ErrNoValidatorFound, "failed to get validator from history store")
		}

		amount = amount.Add(sdk.NewCoin(bondDenom, val.TokensFromShares(delegation.Shares).TruncateInt()))
	}

	return amount, nil
}

// Gets commitInfo from disk.
func getCommitInfo(db dbm.DB, ver int64) (*storetypes.CommitInfo, error) {
	cInfoKey := fmt.Sprintf(commitInfoKeyFmt, ver)

	bz, err := db.Get([]byte(cInfoKey))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get commit info")
	} else if bz == nil {
		return nil, errors.New("no commit info found")
	}

	cInfo := &storetypes.CommitInfo{}
	if err = cInfo.Unmarshal(bz); err != nil {
		return nil, errors.Wrap(err, "failed unmarshal commit info")
	}

	return cInfo, nil
}

func getCommitID(infos map[string]storetypes.StoreInfo, name string) storetypes.CommitID {
	info, ok := infos[name]
	if !ok {
		return storetypes.CommitID{}
	}

	return info.CommitId
}
