package main

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"gopkg.in/yaml.v3"
)

func initCmd() string {
	return "Initialization view"
}

func planCmd() string {
	return "Plan view"
}

func irCmd(filePath string, outputPath string, outputConstraints bool) string {
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

func executeIRCommand(cfg struct {
	constraints bool
	filePath    string
	outputPath  string
}) {
	result := irCmd(cfg.filePath, cfg.outputPath, cfg.constraints)
	fmt.Println(result)
}
