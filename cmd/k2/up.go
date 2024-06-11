package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/k2/constructs"
	"github.com/klothoplatform/klotho/pkg/k2/deployment"
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestrator"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var upConfig struct {
	inputPath  string
	outputPath string
	region     string
}

func newUpCmd() *cobra.Command {
	var upCommand = &cobra.Command{
		Use:   "up",
		Short: "Run the up command",
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
			upConfig.inputPath = absolutePath

			if upConfig.outputPath == "" {
				upConfig.outputPath = filepath.Join(filepath.Dir(absolutePath), ".k2")
			}

			updCmd(upConfig)
		},
	}
	flags := upCommand.Flags()
	flags.StringVarP(&upConfig.outputPath, "output", "o", "", "Output directory")
	flags.StringVarP(&upConfig.region, "region", "r", "us-west-2", "AWS region")
	return upCommand
}

func updCmd(args struct {
	inputPath  string
	outputPath string
	region     string
}) string {
	cmd := startPythonClient()
	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			log.Fatalf("failed to kill Python client: %v", err)
		}
	}()

	// Connect to the Python server
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	ir, err := model.ParseIRFile([]byte(res.GetYamlPayload()))
	if err != nil {
		return fmt.Sprintf("Error reading IR file: %s", err)
	}

	// Take the IR -- generate and save a state file and stored in the
	// output directory, the path should include the environment name and
	// the project URN
	statefile := filepath.Join(args.outputPath, fmt.Sprintf("%s-%s-state.yaml", ir.ProjectURN, ir.Environment))

	// Create a new state manager
	sm := model.NewStateManager(statefile)

	// Initialize the state if it doesn't exist
	if !sm.CheckStateFileExists() {
		sm.InitState(ir)
		// Save the state
		if err = sm.SaveState(); err != nil {
			return fmt.Sprintf("Error saving state: %s", err)
		}
	} else {
		// Load the state
		if err = sm.LoadState(); err != nil {
			return fmt.Sprintf("Error loading state: %s", err)
		}
	}

	o := orchestrator.NewOrchestrator(sm, client)

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
		// TODO the engine currently assumes only 1 run globally, so the debug graphs and other files
		// will get overwritten with each run. We should fix this.
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

	var refs []pulumi.StackReference
	for _, c := range ir.Constructs {
		var id constructs.ConstructId
		err = id.FromURN(c.URN)
		if err != nil {
			return fmt.Sprintf("Error parsing URN: %s", err)
		}
		constructOutDir := filepath.Join(args.outputPath, id.InstanceId)
		refs = append(refs, pulumi.StackReference{
			ConstructURN: c.URN,
			Name:         id.InstanceId,
			IacDirectory: constructOutDir,
			AwsRegion:    args.region,
		})
	}

	upRequest := deployment.UpRequest{
		StackReferences: refs,
		DryRun:          commonCfg.dryRun,
	}

	err = o.RunUpCommand(upRequest)
	if err != nil {
		zap.S().Errorf("Error running up command: %s", err)
		return fmt.Sprintf("Error running up command: %s", err)
	}

	return "success"
}
