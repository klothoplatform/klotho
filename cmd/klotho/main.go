package main

import (
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
	dryRun clicommon.LevelledFlag
}

func cli() int {
	// Set up signal and panic handling to ensure cleanup is executed
	defer func() {
		if r := recover(); r != nil {
			_ = cleanup.Execute(syscall.SIGTERM)
			panic(r)
		}
	}()

	var rootCmd = &cobra.Command{
		Use:          "app",
		SilenceUsage: true,
	}

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cmd.SetContext(cleanup.InitializeHandler(cmd.Context()))
		cmd.SilenceErrors = true
	}

	flags := rootCmd.PersistentFlags()
	dryRunFlag := flags.VarPF(&commonCfg.dryRun, "dry-run", "n", "Dry run (once for pulumi preview, twice for tsc)")
	dryRunFlag.NoOptDefVal = "true" // Allow -n to be used without a value

	var cleanupFuncs []func()
	defer func() {
		for _, f := range cleanupFuncs {
			f()
		}
	}()

	initCommand := newInitCommand()

	upCommand := newUpCmd()
	cleanupFuncs = append(cleanupFuncs, clicommon.SetupCoreCommand(upCommand, &commonCfg.CommonConfig))

	downCommand := newDownCmd()
	cleanupFuncs = append(cleanupFuncs, clicommon.SetupCoreCommand(downCommand, &commonCfg.CommonConfig))

	var irCommand = &cobra.Command{
		Use:   "ir [file path]",
		Short: "Run the IR command",
		//Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			if _, err := os.Stat(filePath); err != nil {
				return err
			}
			irCmd(filePath)
			return nil
		},
	}

	rootCmd.AddCommand(initCommand)
	rootCmd.AddCommand(upCommand)
	rootCmd.AddCommand(downCommand)
	rootCmd.AddCommand(irCommand)

	if err := rootCmd.Execute(); err != nil {
		logging.GetLogger(rootCmd.Context()).Error("Failed to execute command", zap.Error(err))
		return 1
	}
	return 0
}

func main() {
	os.Exit(cli())
}
