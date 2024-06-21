package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/klothoplatform/klotho/pkg/k2/cleanup"
	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"

	clicommon "github.com/klothoplatform/klotho/pkg/cli_common"
	"github.com/spf13/cobra"
)

var commonCfg struct {
	clicommon.CommonConfig
	dryRun bool
}

func cli() {
	// Set up signal and panic handling to ensure cleanup is executed
	defer func() {
		if r := recover(); r != nil {
			_ = cleanup.Execute(syscall.SIGTERM)
			panic(r)
		}
	}()

	var rootCmd = &cobra.Command{
		Use: "app",
	}
	clean := clicommon.SetupRoot(rootCmd, &commonCfg.CommonConfig)
	defer clean()

	pre := rootCmd.PersistentPreRun
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cmd.SetContext(cleanup.InitializeHandler(cmd.Context()))
		cmd.SilenceErrors = true
		pre(cmd, args)
	}

	flags := rootCmd.PersistentFlags()
	flags.BoolVarP(&commonCfg.dryRun, "dry-run", "n", false, "Dry run")

	var initCommand = &cobra.Command{
		Use:   "init",
		Short: "Run the init command",
		Run: func(cmd *cobra.Command, args []string) {
			initCmd()
		},
	}

	deployCommand := newUpCmd()

	var downCommand = newDownCmd()

	var planCommand = &cobra.Command{
		Use:   "plan",
		Short: "Run the plan command",
		Run: func(cmd *cobra.Command, args []string) {
			planCmd()
		},
	}

	var irCommand = &cobra.Command{
		Use:   "ir [file path]",
		Short: "Run the IR command",
		//Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				fmt.Println("Invalid file path")
				os.Exit(1)
			}
			irCmd(filePath)
		},
	}

	rootCmd.AddCommand(initCommand)
	rootCmd.AddCommand(deployCommand)
	rootCmd.AddCommand(downCommand)
	rootCmd.AddCommand(planCommand)
	rootCmd.AddCommand(irCommand)

	if err := rootCmd.Execute(); err != nil {
		logging.GetLogger(rootCmd.Context()).With(zap.Error(err)).Error("Failed to execute command")
		logging.GetLogger(rootCmd.Context()).Error(fmt.Sprintf("Error: %v", err))
		clean()
		os.Exit(1)
	}
}

func main() {
	cli()
}
