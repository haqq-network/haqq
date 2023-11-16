package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/haqq-network/haqq/contracts"
	erc20keeper "github.com/haqq-network/haqq/x/erc20/keeper"
	erc20types "github.com/haqq-network/haqq/x/erc20/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// NewQuerier returns a new sdk.Keeper instance.
func NewQuerier(k bankkeeper.Keeper, erc20 erc20keeper.Keeper, legacyQuerierCdc *codec.LegacyAmino) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, error) {
		switch path[0] {
		case types.QueryBalance:
			return queryBalance(ctx, req, k, erc20, legacyQuerierCdc)

		case types.QueryAllBalances:
			return queryAllBalance(ctx, req, k, erc20, legacyQuerierCdc)

		case types.QueryTotalSupply:
			return queryTotalSupply(ctx, req, k, legacyQuerierCdc)

		case types.QuerySupplyOf:
			return querySupplyOf(ctx, req, k, legacyQuerierCdc)

		default:
			return nil, errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "unknown %s query endpoint: %s", types.ModuleName, path[0])
		}
	}
}

func queryBalance(ctx sdk.Context, req abci.RequestQuery, bk bankkeeper.Keeper, ek erc20keeper.Keeper, legacyQuerierCdc *codec.LegacyAmino) ([]byte, error) {
	var params types.QueryBalanceRequest

	if err := legacyQuerierCdc.UnmarshalJSON(req.Data, &params); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	address, err := sdk.AccAddressFromBech32(params.Address)
	if err != nil {
		return nil, err
	}

	balance := bk.GetBalance(ctx, address, params.Denom)

	if ek.IsERC20Enabled(ctx) {
		tokenPairs := ek.GetTokenPairs(ctx)
		evmAddr := common.BytesToAddress(address.Bytes())
		erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI

		for _, tokenPair := range tokenPairs {
			if !tokenPair.Enabled {
				continue
			}

			if tokenPair.Denom != params.Denom {
				continue
			}

			contract := tokenPair.GetERC20Contract()
			balanceToken := ek.BalanceOf(ctx, erc20, contract, evmAddr)
			if balanceToken == nil {
				return nil, errorsmod.Wrap(erc20types.ErrEVMCall, "failed to retrieve balance")
			}
			balanceTokenCoin := sdk.NewCoin(tokenPair.Denom, sdk.NewIntFromBigInt(balanceToken))

			balance = balance.Add(balanceTokenCoin)
		}
	}

	bz, err := codec.MarshalJSONIndent(legacyQuerierCdc, balance)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}

func queryAllBalance(ctx sdk.Context, req abci.RequestQuery, bk bankkeeper.Keeper, ek erc20keeper.Keeper, legacyQuerierCdc *codec.LegacyAmino) ([]byte, error) {
	var params types.QueryAllBalancesRequest

	if err := legacyQuerierCdc.UnmarshalJSON(req.Data, &params); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	address, err := sdk.AccAddressFromBech32(params.Address)
	if err != nil {
		return nil, err
	}

	balances := bk.GetAllBalances(ctx, address)

	if ek.IsERC20Enabled(ctx) {
		tokenPairs := ek.GetTokenPairs(ctx)
		evmAddr := common.BytesToAddress(address.Bytes())
		erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI

		for _, tokenPair := range tokenPairs {
			if !tokenPair.Enabled {
				continue
			}

			contract := tokenPair.GetERC20Contract()
			balanceToken := ek.BalanceOf(ctx, erc20, contract, evmAddr)
			if balanceToken == nil {
				return nil, errorsmod.Wrap(erc20types.ErrEVMCall, "failed to retrieve balance")
			}
			balanceTokenCoin := sdk.NewCoin(tokenPair.Denom, sdk.NewIntFromBigInt(balanceToken))

			balances = balances.Add(balanceTokenCoin)
		}
	}

	bz, err := codec.MarshalJSONIndent(legacyQuerierCdc, balances)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}

func queryTotalSupply(ctx sdk.Context, req abci.RequestQuery, k bankkeeper.Keeper, legacyQuerierCdc *codec.LegacyAmino) ([]byte, error) {
	var params types.QueryTotalSupplyRequest

	err := legacyQuerierCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	totalSupply, pageRes, err := k.GetPaginatedTotalSupply(ctx, params.Pagination)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	supplyRes := &types.QueryTotalSupplyResponse{
		Supply:     totalSupply,
		Pagination: pageRes,
	}

	res, err := codec.MarshalJSONIndent(legacyQuerierCdc, supplyRes)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}

func querySupplyOf(ctx sdk.Context, req abci.RequestQuery, k bankkeeper.Keeper, legacyQuerierCdc *codec.LegacyAmino) ([]byte, error) {
	var params types.QuerySupplyOfParams

	err := legacyQuerierCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	amount := k.GetSupply(ctx, params.Denom)
	supply := sdk.NewCoin(params.Denom, amount.Amount)

	bz, err := codec.MarshalJSONIndent(legacyQuerierCdc, supply)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}
