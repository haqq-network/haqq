package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/haqq-network/haqq/x/ethiq/types"
)

// GetQueryCmd returns the parent command for all x/ethiq CLI query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the ethiq module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdQueryTotalBurned(),
		GetCmdQueryCalculate(),
		GetCmdQueryParams(),
	)

	return cmd
}

func GetCmdQueryTotalBurned() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total-burned",
		Short: "Query the total amount of burned aISLM coins",
		Args:  cobra.NoArgs,
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the total amount of burned aISLM coins.

Example:
  $ %s query %s total-burned
`,
				version.AppName, types.ModuleName,
			),
		),
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			ctx := cmd.Context()

			res, err := queryClient.TotalBurned(ctx, &types.QueryTotalBurnedRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func GetCmdQueryCalculate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calculate [ethiq-amount]",
		Short: "Calculate the required aISLM amount to mint aethiq coins",
		Args:  cobra.ExactArgs(1),
		Long: strings.TrimSpace(
			fmt.Sprintf(`Calculate the required aISLM amount to mint the specified amount of aethiq coins.

Example:
  $ %s query %s calculate 1000000000000000000
`,
				version.AppName, types.ModuleName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			ethiqAmount, ok := sdkmath.NewIntFromString(args[0])
			if !ok {
				return fmt.Errorf("invalid ethiq_amount: %s", args[0])
			}

			queryClient := types.NewQueryClient(clientCtx)
			ctx := cmd.Context()

			res, err := queryClient.Calculate(ctx, &types.QueryCalculateRequest{
				EthiqAmount: ethiqAmount,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the parameters of the ethiq module",
		Args:  cobra.NoArgs,
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the parameters of the ethiq module.

Example:
  $ %s query %s params
`,
				version.AppName, types.ModuleName,
			),
		),
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			ctx := cmd.Context()

			res, err := queryClient.Params(ctx, &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

