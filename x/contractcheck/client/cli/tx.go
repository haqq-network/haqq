package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/haqq-network/haqq/x/contractcheck/types"
	"github.com/spf13/cobra"
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

func NewMintNFTProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint-nft [CONTRACT_ADDRESS] [TO] [URI]",
		Short: "Submit a new mint nft proposal",
		Args:  cobra.RangeArgs(3, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(cli.FlagDescription)
			if err != nil {
				return err
			}

			depositStr, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			from := cliCtx.GetFromAddress()

			contractAddress := args[0]
			destinationAddress := args[1]
			nftURL := args[2]

			content := types.NewMintNFTProposal(title, description, contractAddress, destinationAddress, nftURL)

			msg, err := govv1beta1.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(cli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "1aISLM", "deposit of proposal")
	if err := cmd.MarkFlagRequired(cli.FlagTitle); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired(cli.FlagDescription); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired(cli.FlagDeposit); err != nil {
		panic(err)
	}
	return cmd
}
