package cli

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/haqq-network/haqq/x/vesting/types"
)

// GetQueryCmd returns the parent command for all vesting CLI query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the vesting module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetBalancesCmd(),
		GetTotalLockedCmd(),
	)
	return cmd
}

// GetBalancesCmd queries the locked, unvested and vested tokens for a given vesting account.
func GetBalancesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balances ADDRESS",
		Short: "Gets locked, unvested and vested tokens for a vesting account",
		Long:  "Gets locked, unvested and vested tokens for a vesting account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryBalancesRequest{
				Address: args[0],
			}

			res, err := queryClient.Balances(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintString(
				fmt.Sprintf("Locked: %s\nUnvested: %s\nVested: %s\n", res.Locked, res.Unvested, res.Vested))
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetTotalLockedCmd queries summary of the locked, vested and unvested tokens for all accounts
func GetTotalLockedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total",
		Short: "Gets total summary of locked, unvested and vested tokens for all accounts",
		Long:  "Gets total summary of locked, unvested and vested tokens for all accounts",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryTotalLockedRequest{}

			res, err := queryClient.TotalLocked(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintString(
				fmt.Sprintf("Total locked: %s\nTotal unvested: %s\nTotal vested: %s\n", res.Locked, res.Unvested, res.Vested))
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
