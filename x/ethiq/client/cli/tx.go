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
		NewMintEthiqCmd(),
	)

	return ethiqTxCmd
}

// NewMintEthiqCmd returns a CLI command handler for creating a MsgMintEthiq transaction.
func NewMintEthiqCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint [ethiq-amount] [max-islm-amount] [to-address]",
		Args:  cobra.ExactArgs(3),
		Short: "Mint aethiq coins in exchange for aISLM coins",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Mint aethiq coins in exchange for aISLM coins

Example:
$ %s tx %s mint 1000000000000000000 1000000000000000000 haqq1tjdjfavsy956d25hvhs3p0nw9a7pfghqm0up92 --from mykey
`,
				version.AppName, types.ModuleName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ethiqAmount, ok := sdkmath.NewIntFromString(args[0])
			if !ok {
				return fmt.Errorf("invalid ethiq_amount: %s", args[0])
			}

			maxISLMAmount, ok := sdkmath.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid max_islm_amount: %s", args[1])
			}

			toAddress, err := sdk.AccAddressFromBech32(args[2])
			if err != nil {
				return fmt.Errorf("invalid to_address: %v", err)
			}

			fromAddress := clientCtx.GetFromAddress()

			msg := &types.MsgMintEthiq{
				FromAddress:   fromAddress.String(),
				ToAddress:     toAddress.String(),
				EthiqAmount:   ethiqAmount,
				MaxIslmAmount: maxISLMAmount,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
