package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/haqq-network/haqq/x/ethiq/types"
)

// NewTxCmd returns a root CLI command handler for all x/ethiq transaction commands.
func NewTxCmd() *cobra.Command {
	ethiqTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Ethiq transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	ethiqTxCmd.AddCommand(
		NewMintHaqqCmd(),
		NewMintHaqqByApplicationCmd(),
	)

	return ethiqTxCmd
}

// NewMintHaqqCmd returns a CLI command handler for creating a MsgMintHaqq transaction.
func NewMintHaqqCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint [islm_amount] [to_address]",
		Args:  cobra.ExactArgs(2),
		Short: "Mint aHAQQ coins in exchange for the given aISLM coins",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Mint aHAQQ coins in exchange for the given aISLM coins

Example:
$ %s tx %s mint 1000000000000000000 haqq1tjdjfavsy956d25hvhs3p0nw9a7pfghqm0up92 --from mykey
`,
				version.AppName, types.ModuleName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			islmAmount, ok := sdkmath.NewIntFromString(args[0])
			if !ok {
				return fmt.Errorf("invalid islm_amount: %s", args[0])
			}

			toAddress, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return fmt.Errorf("invalid to_address: %v", err)
			}

			fromAddress := clientCtx.GetFromAddress()

			msg := &types.MsgMintHaqq{
				FromAddress: fromAddress.String(),
				ToAddress:   toAddress.String(),
				IslmAmount:  islmAmount,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewMintHaqqByApplicationCmd returns a CLI command handler for creating a MsgMintHaqqByApplication transaction.
func NewMintHaqqByApplicationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint-by-app [app_id]",
		Args:  cobra.ExactArgs(1),
		Short: "Mint aHAQQ coins in exchange for the aISLM coins declared in the given application",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Mint aHAQQ coins in exchange for the aISLM coins declared in the given application

Example:
$ %s tx %s mint-by-app 1 --from mykey
`,
				version.AppName, types.ModuleName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			appID, ok := sdkmath.NewIntFromString(args[0])
			if !ok {
				return fmt.Errorf("invalid app_id: %s", args[0])
			}

			fromAddress := clientCtx.GetFromAddress()

			msg := &types.MsgMintHaqqByApplication{
				FromAddress:   fromAddress.String(),
				ApplicationId: appID.Uint64(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
