package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"

	sdkmath "cosmossdk.io/math"
	"github.com/haqq-network/haqq/x/ucdao/types"
)

// NewTxCmd returns a root CLI command handler for all x/distribution transaction commands.
func NewTxCmd() *cobra.Command {
	distTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "United Contributors DAO transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	distTxCmd.AddCommand(
		NewFundDAOCmd(),
		NewTransferOwnershipCmd(),
		NewConvertToEthiqCmd(),
	)

	return distTxCmd
}

// NewFundDAOCmd returns a CLI command handler for creating a MsgFund transaction.
func NewFundDAOCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund [amount]",
		Args:  cobra.ExactArgs(1),
		Short: "Funds the United Contributors DAO with the specified amount",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Funds the dao with the specified amount

Example:
$ %s tx %s fund 100aISLM --from mykey
`,
				version.AppName, types.ModuleName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			depositorAddr := clientCtx.GetFromAddress()
			amount, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgFund(amount, depositorAddr)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewTransferOwnershipCmd returns a CLI command handler for creating a MsgTransferOwnership transaction.
func NewTransferOwnershipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer-ownership [from_address] [to_address]",
		Args:  cobra.ExactArgs(2),
		Short: "Transfer all United Contributors DAO shares from one address to another",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Transfer all DAO shares from one address to another

Example:
$ %s tx %s transfer-ownership haqq1tjdjfavsy956d25hvhs3p0nw9a7pfghqm0up92 haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq --from mykey
`,
				version.AppName, types.ModuleName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			owner, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			newOwner, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return err
			}

			msg := types.NewMsgTransferOwnership(owner, newOwner)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewConvertToEthiqCmd returns a CLI command handler for creating a MsgConvertToEthiq transaction.
func NewConvertToEthiqCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert-to-haqq [from_address] [to_address] [islm_amount]",
		Args:  cobra.ExactArgs(3),
		Short: "Convert aISLM tokens to aHAQQ tokens",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Convert aISLM tokens to aHAQQ tokens. The sender must be a holder in the ucdao module.

Example:
$ %s tx %s convert-to-haqq haqq1hdr0lhv75vesvtndlh78ck4cez6esz8u2lk0hq 1000 --from mykey
`,
				version.AppName, types.ModuleName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			fromAddress, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			toAddr, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return err
			}

			islmAmount, ok := sdkmath.NewIntFromString(args[2])
			if !ok {
				return fmt.Errorf("invalid ethiq amount: %s", args[2])
			}

			msg := types.NewMsgConvertToHaqq(fromAddress, toAddr, islmAmount)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
