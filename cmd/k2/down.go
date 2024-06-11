package main

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/k2/deployment"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestration"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
)

var downConfig struct {
	outputPath string
	project    string
	app        string
	env        string
}

func newDownCmd() *cobra.Command {
	downCommand := &cobra.Command{
		Use:   "down",
		Short: "Run the down command",
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
			downConfig.project = args[1]
			downConfig.app = args[2]
			downConfig.env = args[3]

			if downConfig.outputPath == "" {
				(&downConfig).outputPath = filepath.Join(filepath.Dir(absolutePath), ".k2")
			}

			downCmd(downConfig)
		},
	}
	flags := downCommand.Flags()
	flags.StringVarP(&upConfig.outputPath, "output", "o", "", "Output directory")
	return downCommand

}

func downCmd(args struct {
	outputPath string
	project    string
	app        string
	env        string
}) string {

	projectPath := filepath.Join(args.outputPath, args.project, args.app, args.env)

	entries, err := os.ReadDir(projectPath)
	if err != nil {
		log.Fatalf("failed to read directory: %v", err)
		return "failure"
	}

	var stackReferences []pulumi.StackReference
	for _, entry := range entries {
		if entry.IsDir() {
			constructPath := filepath.Join(projectPath, entry.Name())

			stackReference := pulumi.StackReference{
				ConstructURN: model.URN{},
				Name:         entry.Name(),
				IacDirectory: constructPath,
			}
			stackReferences = append(stackReferences, stackReference)
		}
	}

	var o orchestration.Orchestrator
	err = o.RunDownCommand(deployment.DownRequest{StackReferences: stackReferences, DryRun: commonCfg.dryRun})

	if err != nil {
		log.Fatalf("failed to run down command: %v", err)
		return "failure"
	}

	return "success"
}
