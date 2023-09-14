package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/haqq-network/haqq/x/coinomics/types"
)

// GetQueryCmd returns the cli query commands for the coinomics module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the coinomics module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetEra(),
		GetEraClosingSupply(),
		GetMaxSupply(),
		GetInflationRate(),
		GetParams(),
	)

	return cmd
}

func GetEra() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "era",
		Short: "Query the current era",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryEraRequest{}
			res, err := queryClient.Era(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintString(fmt.Sprintf("%v\n", res.Era))
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func GetEraClosingSupply() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "era-closing-supply",
		Short: "Query era closing supply",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryEraClosingSupplyRequest{}
			res, err := queryClient.EraClosingSupply(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintString(fmt.Sprintf("%v\n", res.EraClosingSupply))
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func GetMaxSupply() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "max-supply",
		Short: "Query max supply",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryMaxSupplyRequest{}
			res, err := queryClient.MaxSupply(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintString(fmt.Sprintf("%v\n", res.MaxSupply))
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func GetInflationRate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inflation-rate",
		Short: "Query current era inflation rate",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryInflationRateRequest{}
			res, err := queryClient.InflationRate(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintString(fmt.Sprintf("%v\n", res.InflationRate))
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func GetParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the current coinomics parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryParamsRequest{}
			res, err := queryClient.Params(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
