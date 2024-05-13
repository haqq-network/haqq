package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

// GetQueryCmd returns the parent command for all liquidvesting CLI query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the liquidvesting module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(GetDenomCmd())
	cmd.AddCommand(GetDenomsCmd())

	return cmd
}

// GetDenomCmd returns command for querying liquid denom
func GetDenomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denom DENOM",
		Short: "Gets information about denom of liquid vesting token",
		Long:  "Gets information about denom of liquid vesting token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryDenomRequest{
				Denom: args[0],
			}

			res, err := queryClient.Denom(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetDenomsCmd return command for querying all liquid denoms
func GetDenomsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denoms",
		Short: "Gets information about all denoms of liquid vesting tokens",
		Long:  "Gets information about all denoms of liquid vesting tokens",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			req := &types.QueryDenomsRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.Denoms(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
