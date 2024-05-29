package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/haqq-network/haqq/x/contractcheck/types"
)

// NewTxCmd returns a root CLI command handler for certain modules/contractcheck
// transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Contractcheck transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewMsgMintCmd(),
	)

	return txCmd
}

// NewMsgLiquidateCmd returns command for composing MsgLiquidate and sending it to blockchain
func NewMsgMintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint [CONTRACT_ADDRESS] [TO] [URI]",
		Short: "test mint call",
		Args:  cobra.RangeArgs(3, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			mintFrom := cliCtx.GetFromAddress().String()

			msg := types.NewMsgMint(args[0], args[1], mintFrom, args[2])

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}