package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/haqq-network/haqq/x/ucdao/types"
)

const (
	FlagDenom = "denom"
)

// GetQueryCmd returns the parent command for all x/dao CLi query commands. The
// provided clientCtx should have, at a minimum, a verifier, Tendermint RPC client,
// and marshaler set.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the dao module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetBalancesCmd(),
		GetCmdQueryTotalBalance(),
		GetCmdQueryEscrowAddress(),
		GetCmdQueryHolders(),
		GetCmdQueryParams(),
	)

	return cmd
}

func GetBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance [address] [denom]",
		Short: "Query the balance of a given address for a given denomination",
		Args:  cobra.ExactArgs(2),
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the balance of a given address for a given denomination.

Example:
  $ %s query %s balance [address] [denom]
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

			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			denom := args[1]

			res, err := queryClient.Balance(ctx, &types.QueryBalanceRequest{Address: addr.String(), Denom: denom})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res.Balance)
		},
	}

	return cmd
}

func GetBalancesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balances [address]",
		Short: "Query for account balances in the United Contributors DAO by address",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the total balance of an account or of a specific denomination.

Example:
  $ %s query %s balances [address]
  $ %s query %s balances [address] --denom=[denom]
`,
				version.AppName, types.ModuleName, version.AppName, types.ModuleName,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			denom, err := cmd.Flags().GetString(FlagDenom)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			if denom == "" {
				params := types.NewQueryAllBalancesRequest(addr, pageReq)

				res, err := queryClient.AllBalances(ctx, params)
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			params := types.NewQueryBalanceRequest(addr, denom)

			res, err := queryClient.Balance(ctx, params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res.Balance)
		},
	}

	cmd.Flags().String(FlagDenom, "", "The specific balance denomination to query for")
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "all balances")

	return cmd
}

func GetCmdQueryTotalBalance() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total-balance",
		Short: "Query the total balances of coins in the United Contributors DAO",
		Args:  cobra.NoArgs,
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query total balances of coins that are held by accounts in the DAO.

Example:
  $ %s query %s total-balance
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

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			res, err := queryClient.TotalBalance(ctx, &types.QueryTotalBalanceRequest{Pagination: pageReq})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "all supply totals")

	return cmd
}

func GetCmdQueryEscrowAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "escrow-address [address]",
		Short: "Query the escrow address for a given standard wallet address",
		Args:  cobra.ExactArgs(1),
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the escrow address for a given standard wallet address.

Example:
  $ %s query %s escrow-address [address]
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

			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			res, err := queryClient.EscrowAddress(ctx, &types.QueryEscrowAddressRequest{Address: addr.String()})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	return cmd
}

func GetCmdQueryHolders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "holders",
		Short: "Query the holders of the United Contributors DAO",
		Args:  cobra.NoArgs,
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the holders of the United Contributors DAO.

Example:
  $ %s query %s holders
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

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			res, err := queryClient.Holders(ctx, &types.QueryHoldersRequest{Pagination: pageReq})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "all holders")

	return cmd
}

func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the parameters of the United Contributors DAO",
		Args:  cobra.NoArgs,
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query the parameters of the United Contributors DAO.

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

	return cmd
}
