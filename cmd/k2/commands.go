package main

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/k2/constructs"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestrator"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"log"
	"time"
)

func initCmd() string {
	return "Initialization view"
}

func deployCmd(filePath string) string {
	go startGRPCServer()
	if err := waitForServer("localhost:50051", 10, 1*time.Second); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	startPythonClient(filePath)
	time.Sleep(5 * time.Second)
	return "success"
}

func destroyCmd() string {
	return "Destroy view"
}

func planCmd() string {
	return "Plan view"
}

func irCmd(filePath string, outputPath string, outputConstraints bool) string {
	ir, err := model.ReadIRFile(filePath)
	if err != nil {
		return fmt.Sprintf("Error reading IR file: %s", err)
	}

	if !outputConstraints {
		res, err := yaml.Marshal(ir)
		if err != nil {
			return fmt.Sprintf("Error marshalling IR: %s", err)
		}
		return string(res)
	}

	// Apply constraints
	var allConstraints constraints.ConstraintList
	for _, c := range ir.Constructs {
		var id constructs.ConstructId
		err = id.FromURN(c.URN)
		if err != nil {
			return fmt.Sprintf("Error parsing URN: %s", err)
		}
		inputs := make(map[string]interface{})
		for k, v := range c.Inputs {
			if v.Status != "" && v.Status != model.Resolved {
				zap.S().Warnf("Input %s is not resolved", k)
				continue
			}

			inputs[k] = v.Value
		}
		ctx := constructs.NewContext(inputs, id)
		ci := ctx.EvaluateConstruct()
		if err != nil {
			return fmt.Sprintf("Error evaluating construct: %s", err)
		}
		marshaller := constructs.ConstructMarshaller{Context: ctx, Construct: ci}
		cs, err := marshaller.Marshal()
		if err != nil {
			return fmt.Sprintf("Error marshalling construct: %s", err)
		}
		allConstraints = append(allConstraints, cs...)
	}
	marshalledConstraints, err := allConstraints.ToConstraints()
	if err != nil {
		return fmt.Sprintf("Error marshalling constraints: %s", err)
	}
	out, err := yaml.Marshal(marshalledConstraints)
	if err != nil {
		return fmt.Sprintf("Error marshalling constraints2: %s", err)
	}

	inputGraph, err := orchestrator.ReadInputGraph(outputPath)
	if err != nil {
		return fmt.Sprintf("Error reading input graph: %s", err)
	}

	var o orchestrator.Orchestrator
	errs := o.RunEngine(orchestrator.EngineRequest{
		Provider:    "aws",
		InputGraph:  inputGraph,
		Constraints: marshalledConstraints,
		OutputDir:   "./k2-output",
		GlobalTag:   "k2",
	})
	if errs != nil {
		zap.S().Warnf("Engine returned with errors: %s", errs)
	}

	return string(out)
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
