package main

import (
	"fmt"
	"os"

	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func initCmd() string {
	return "Initialization view"
}

func deployCmd() string {
	return "Deploy view"
}

func destroyCmd() string {
	return "Destroy view"
}

func planCmd() string {
	return "Plan view"
}

func irCmd(filePath string) string {
	ir, err := model.ReadIRFile(filePath)
	if err != nil {
		return fmt.Sprintf("Error reading IR file: %s", err)
	}
	res, err := yaml.Marshal(ir)
	if err != nil {
		return fmt.Sprintf("Error marshalling IR: %s", err)
	}
	return string(res)
}

func executeCommand(cmd func() string) {
	// Execute the command and print the view
	result := cmd()
	fmt.Println(result)
}

func executeIRCommand(filePath string) {
	result := irCmd(filePath)
	fmt.Println(result)
}

func main() {
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
