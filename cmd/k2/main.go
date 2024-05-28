package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func cli() {
	var rootCmd = &cobra.Command{Use: "app"}

	var initCommand = &cobra.Command{
		Use:   "init",
		Short: "Run the init command",
		Run: func(cmd *cobra.Command, args []string) {
			executeCommand(initCmd)
		},
	}

	var deployCommand = &cobra.Command{
		Use:   "deploy",
		Short: "Run the deploy command",
		Run: func(cmd *cobra.Command, args []string) {
			executeCommand(deployCmd)
		},
	}

	var destroyCommand = &cobra.Command{
		Use:   "destroy",
		Short: "Run the destroy command",
		Run: func(cmd *cobra.Command, args []string) {
			executeCommand(destroyCmd)
		},
	}

	var planCommand = &cobra.Command{
		Use:   "plan",
		Short: "Run the plan command",
		Run: func(cmd *cobra.Command, args []string) {
			executeCommand(planCmd)
		},
	}

	var irCommand = &cobra.Command{
		Use:   "ir [file path]",
		Short: "Run the IR command",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				fmt.Println("Invalid file path")
				os.Exit(1)
			}

			executeIRCommand(filePath)
		},
	}

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
	go startGRPCServer()

	// Wait for the server to be ready
	if err := waitForServer("localhost:50051", 10, 1*time.Second); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	startPythonClient("./pkg/k2/language_host/python/infra.py")
	select {}
	cli()

}
