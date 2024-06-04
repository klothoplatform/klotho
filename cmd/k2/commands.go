package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/k2/constructs"
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestrator"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
)

func initCmd() string {
	return "Initialization view"
}

func deployCmd(args struct {
	inputPath  string
	outputPath string
}) string {
	cmd := startPythonClient()
	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			log.Fatalf("failed to kill Python client: %v", err)
		}
	}()

	// Connect to the Python server
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewKlothoServiceClient(conn)

	// Wait for the server to be ready
	if err := waitForServer(client, 10, 1*time.Second); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	// Send IR Request
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &pb.IRRequest{Filename: args.inputPath}
	res, err := client.SendIR(ctx, req)
	if err != nil {
		log.Fatalf("could not execute script: %v", err)
	}
	programContext.IRYaml = res.YamlPayload

	for x := 0; x < 10; x++ {
		if programContext.IRYaml != "" {
			zap.S().Info("IR received")
			break
		}
		time.Sleep(1 * time.Second)
	}

	if programContext.IRYaml == "" {
		zap.S().Warn("No IR received")
		return "No IR received"
	}

	ir, err := model.ParseIRFile([]byte(programContext.IRYaml))
	if err != nil {
		return fmt.Sprintf("Error reading IR file: %s", err)
	}

	// Apply constraints
	for _, c := range ir.Constructs {
		var allConstraints constraints.ConstraintList
		var id constructs.ConstructId
		err = id.FromURN(c.URN)
		if err != nil {
			return fmt.Sprintf("Error parsing URN: %s", err)
		}
		constructOutDir := filepath.Join(args.outputPath, id.InstanceId)
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
		if ci == nil {
			return fmt.Sprintf("Error evaluating construct: %s", err)
		}
		marshaller := constructs.ConstructMarshaller{Context: ctx, Construct: ci}
		cs, err := marshaller.Marshal()
		if err != nil {
			return fmt.Sprintf("Error marshalling construct: %s", err)
		}
		allConstraints = append(allConstraints, cs...)

		// Marshal constructs to constraints
		marshalledConstraints, err := allConstraints.ToConstraints()
		if err != nil {
			return fmt.Sprintf("Error marshalling constraints: %s", err)
		}

		// Read existing state
		inputGraph, err := orchestrator.ReadInputGraph(constructOutDir)
		if err != nil {
			return fmt.Sprintf("Error reading input graph: %s", err)
		}

		// Run the engine
		var o orchestrator.Orchestrator
		engineContext, errs := o.RunEngine(orchestrator.EngineRequest{
			Provider:    "aws",
			InputGraph:  inputGraph,
			Constraints: marshalledConstraints,
			OutputDir:   constructOutDir,
			GlobalTag:   "k2",
		})
		if errs != nil {
			zap.S().Errorf("Engine returned with errors: %s", errs)
			return fmt.Sprintf("Engine returned with errors: %s", errs)
		}

		// GenerateIac
		err = o.GenerateIac(orchestrator.IacRequest{
			PulumiAppName: id.InstanceId,
			Context:       engineContext,
			OutputDir:     constructOutDir,
		})
		if err != nil {
			zap.S().Errorf("Error generating IaC: %s", err)
			return fmt.Sprintf("Error generating IaC: %s", err)
		}

	}

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
