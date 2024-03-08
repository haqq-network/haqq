package main

import (
	"fmt"
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/haqq-network/haqq/app"
	"github.com/haqq-network/haqq/app/config"
)

func main() {
	config.SetupConfig()

	rootCmd, _ := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, app.Name, app.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err) // nolint: errcheck
		// Exit with default error code due possible exact error code overflow (max value is 125)
		os.Exit(1)
	}
}
