package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/k2/deployment"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestration"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
	flags.StringVarP(&downConfig.outputPath, "output", "o", "", "Output directory")
	return downCommand

}

func downCmd(args struct {
	outputPath string
	project    string
	app        string
	env        string
}) string {

	projectPath := filepath.Join(args.outputPath, args.project, args.app, args.env)
	stateFile := filepath.Join(projectPath, "state.yaml")
	sm := model.NewStateManager(stateFile)

	if !sm.CheckStateFileExists() {
		zap.L().Error("state file does not exist", zap.String("stateFile", stateFile))
		return "failure"
	}

	err := sm.LoadState()
	if err != nil {
		zap.L().Error("error loading state", zap.Error(err))
		return "failure"
	}

	var stackReferences []pulumi.StackReference
	for name, construct := range sm.GetAllConstructs() {
		constructPath := filepath.Join(projectPath, name)
		stackReference := pulumi.StackReference{
			ConstructURN: *construct.URN,
			Name:         name,
			IacDirectory: constructPath,
		}
		stackReferences = append(stackReferences, stackReference)
	}

	o := orchestration.NewDownOrchestrator(sm, args.outputPath)
	err = o.RunDownCommand(deployment.DownRequest{StackReferences: stackReferences, DryRun: commonCfg.dryRun})
	if err != nil {
		zap.L().Error("error running down command", zap.Error(err))
		return "failure"
	}

	return "success"
}
