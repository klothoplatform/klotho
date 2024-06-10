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
