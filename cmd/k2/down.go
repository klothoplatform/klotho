package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/engine/debug"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestration"
	"github.com/klothoplatform/klotho/pkg/k2/stack"
	"github.com/spf13/cobra"
)

var downConfig struct {
	outputPath string
}

func newDownCmd() *cobra.Command {
	downCommand := &cobra.Command{
		Use:   "down",
		Short: "Run the down command",
		RunE:  down,
	}
	flags := downCommand.Flags()
	flags.StringVarP(&downConfig.outputPath, "output", "o", "", "Output directory")
	return downCommand

}

func down(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return err
	}
	absolutePath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	project := args[1]
	app := args[2]
	env := args[3]

	if downConfig.outputPath == "" {
		downConfig.outputPath = filepath.Join(filepath.Dir(absolutePath), ".k2")
	}

	debugDir := debug.GetDebugDir(cmd.Context())
	if debugDir == "" {
		debugDir = upConfig.outputPath
		cmd.SetContext(debug.WithDebugDir(cmd.Context(), debugDir))
	}

	projectPath := filepath.Join(downConfig.outputPath, project, app, env)
	stateFile := filepath.Join(projectPath, "state.yaml")
	sm := model.NewStateManager(stateFile)

	if !sm.CheckStateFileExists() {
		return fmt.Errorf("state file does not exist: %s", stateFile)
	}

	err = sm.LoadState()
	if err != nil {
		return fmt.Errorf("error loading state: %w", err)
	}

	var stackReferences []stack.Reference
	for name, construct := range sm.GetAllConstructs() {
		constructPath := filepath.Join(projectPath, name)
		stackReference := stack.Reference{
			ConstructURN: *construct.URN,
			Name:         name,
			IacDirectory: constructPath,
		}
		stackReferences = append(stackReferences, stackReference)
	}

	o := orchestration.NewDownOrchestrator(sm, downConfig.outputPath)
	err = o.RunDownCommand(cmd.Context(), orchestration.DownRequest{StackReferences: stackReferences, DryRun: commonCfg.dryRun})
	if err != nil {
		return fmt.Errorf("error running down command: %w", err)
	}
	return nil
}
