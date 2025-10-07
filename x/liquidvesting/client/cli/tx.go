package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/haqq-network/haqq/x/liquidvesting/types"
)

// NewTxCmd returns a root CLI command handler for certain modules/liquidvesting
// transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Liquidvesting transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewMsgLiquidateCmd(),
		NewMsgRedeemCmd(),
	)

	return txCmd
}

// NewMsgLiquidateCmd returns command for composing MsgLiquidate and sending it to blockchain
//
//nolint:dupl // false warning about duplicate code in NewMsgLiquidateCmd and NewMsgRedeemCmd methods
func NewMsgLiquidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "liquidate AMOUNT [RECEIVER]",
		Short: "Liquidate locked tokens from vesting account into erc20 token",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			coin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			var liquidateTo sdk.AccAddress
			liquidateFrom := cliCtx.GetFromAddress()

			if len(args) == 2 {
				liquidateTo, err = sdk.AccAddressFromBech32(args[1])
				if err != nil {
					return err
				}
			} else {
				liquidateTo = liquidateFrom
			}

			msg := types.NewMsgLiquidate(liquidateFrom, liquidateTo, coin)

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewMsgRedeemCmd returns command for composing MsgRedeem and sending it to blockchain
//
//nolint:dupl // false warning about duplicate code in NewMsgLiquidateCmd and NewMsgRedeemCmd methods
func NewMsgRedeemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redeem AMOUNT [RECEIVER]",
		Short: "Redeem liquid token into locked vesting tokens",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			coin, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			var redeemTo sdk.AccAddress
			redeemFrom := cliCtx.GetFromAddress()

			if len(args) == 2 {
				redeemTo, err = sdk.AccAddressFromBech32(args[1])
				if err != nil {
					return err
				}
			} else {
				redeemTo = redeemFrom
			}

			msg := types.NewMsgRedeem(redeemFrom, redeemTo, coin)

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
