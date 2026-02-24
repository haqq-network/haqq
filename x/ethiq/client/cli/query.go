package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
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
		GetCmdQueryCalculateForApplication(),
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
		Use:   "calculate [islm-amount]",
		Short: "Calculate the estimated aHAQQ amount to be minted for a given aISLM amount",
		Args:  cobra.ExactArgs(1),
		Long: strings.TrimSpace(
			fmt.Sprintf(`Calculate the estimated aHAQQ amount to be minted in exchange for the given amount of aISLM coins.

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

			islmAmount, ok := sdkmath.NewIntFromString(args[0])
			if !ok {
				return fmt.Errorf("invalid islm_amount: %s", args[0])
			}

			queryClient := types.NewQueryClient(clientCtx)
			ctx := cmd.Context()

			res, err := queryClient.Calculate(ctx, &types.QueryCalculateRequest{
				IslmAmount: islmAmount.String(),
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

func GetCmdQueryCalculateForApplication() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calculate-for-app [app_id]",
		Short: "Calculate the estimated aHAQQ amount to be minted by an execution of present application",
		Args:  cobra.ExactArgs(1),
		Long: strings.TrimSpace(
			fmt.Sprintf(`Calculate the estimated aHAQQ amount to be minted by an execution of present application.

Example:
  $ %s query %s calculate-for-app 1
`,
				version.AppName, types.ModuleName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			appID, ok := sdkmath.NewIntFromString(args[0])
			if !ok {
				return fmt.Errorf("invalid app_id: %s", args[0])
			}

			queryClient := types.NewQueryClient(clientCtx)
			ctx := cmd.Context()

			res, err := queryClient.CalculateForApplication(ctx, &types.QueryCalculateForApplicationRequest{
				ApplicationId: appID.Uint64(),
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

func GetCmdQueryGetApplications() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-applications",
		Short: "Get the paginated list of present applications",
		Args:  cobra.ExactArgs(1),
		Long: strings.TrimSpace(
			fmt.Sprintf(`Get the paginated list of present applications.

Example:
  $ %s query %s get-applications
`,
				version.AppName, types.ModuleName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			ctx := cmd.Context()

			// default
			res, err := queryClient.GetApplications(ctx, &types.QueryGetApplicationsRequest{
				Pagination: nil,
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
