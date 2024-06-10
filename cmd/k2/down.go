package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestrator"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var downConfig struct {
	inputPath  string
	outputPath string
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
			downConfig.inputPath = absolutePath

			if downConfig.outputPath == "" {
				(&downConfig).outputPath = filepath.Join(filepath.Dir(absolutePath), ".k2")
			}

			downCmd(downConfig.outputPath)
		},
	}
	flags := downCommand.Flags()
	flags.StringVarP(&upConfig.outputPath, "output", "o", "", "Output directory")
	return downCommand

}

func downCmd(outputPath string) string {
	entries, err := os.ReadDir(outputPath)
	if err != nil {
		zap.L().Error("failed to read directory", zap.Error(err))
		return "failure"
	}

	var stackReferences []pulumi.StackReference
	for _, entry := range entries {
		if entry.IsDir() {
			stackReference := pulumi.StackReference{
				ConstructURN: &model.URN{},
				Name:         entry.Name(),
				IacDirectory: filepath.Join(outputPath, entry.Name()),
			}
			stackReferences = append(stackReferences, stackReference)
		}
	}

	var o orchestrator.Orchestrator
	err = o.RunDownCommand(orchestrator.DownRequest{StackReferences: stackReferences})

	if err != nil {
		log.Fatalf("failed to run down command: %v", err)
		return "failure"
	}

	return "success"
}
