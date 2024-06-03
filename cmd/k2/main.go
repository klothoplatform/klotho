package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var irConfig struct {
	constraints bool
	filePath    string
}

func cli() {
	var rootCmd = &cobra.Command{Use: "app"}

	var initCommand = &cobra.Command{
		Use:   "init",
		Short: "Run the init command",
		Run: func(cmd *cobra.Command, args []string) {
			initCmd()
		},
	}

	var deployCommand = &cobra.Command{
		Use:   "deploy",
		Short: "Run the deploy command",
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				fmt.Println("Invalid file path")
				os.Exit(1)
			}
			absolutePath, err := filepath.Abs(filePath)
			if err != nil {
				fmt.Println("couldn't convert to absolute path")
				os.Exit(1)
			}

			deployCmd(absolutePath)
		},
	}

	var destroyCommand = &cobra.Command{
		Use:   "destroy",
		Short: "Run the destroy command",
		Run: func(cmd *cobra.Command, args []string) {
			destroyCmd()
		},
	}

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
	flags := irCommand.Flags()
	flags.BoolVarP(&irConfig.constraints, "constraints", "c", false, "Print constraints")

	rootCmd.AddCommand(initCommand)
	rootCmd.AddCommand(deployCommand)
	rootCmd.AddCommand(destroyCommand)
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
