package main

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
)

var irConfig struct {
	constraints bool
	filePath    string
	outputPath  string
}

var commonCfg struct {
	verbose bool
	jsonLog bool
	logsDir string
}

func cli() {
	var rootCmd = &cobra.Command{Use: "app"}
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&commonCfg.logsDir, "logs-dir", "logs", "Logs directory (set to empty to disable folder logging)")
	flags.BoolVarP(&commonCfg.verbose, "verbose", "v", false, "Verbose flag")
	flags.BoolVar(&commonCfg.jsonLog, "json-log", false, "Output logs in JSON format.")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		logOpts := logging.LogOpts{
			Verbose:         commonCfg.verbose,
			CategoryLogsDir: commonCfg.logsDir,
		}
		if commonCfg.jsonLog {
			logOpts.Encoding = "json"
		}
		zap.ReplaceGlobals(logOpts.NewLogger())
	}
	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		_ = zap.L().Sync()
	}

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
			irConfig.filePath = filePath

			executeIRCommand(irConfig)
		},
	}
	flags = irCommand.Flags()
	flags.BoolVarP(&irConfig.constraints, "constraints", "c", false, "Print constraints")
	flags.StringVarP(&irConfig.outputPath, "output", "o", "", "Output file path")

	rootCmd.AddCommand(initCommand)
	rootCmd.AddCommand(deployCommand)
	rootCmd.AddCommand(downCommand)
	rootCmd.AddCommand(planCommand)
	rootCmd.AddCommand(irCommand)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	cli()
}
