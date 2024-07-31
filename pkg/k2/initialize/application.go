package initialize

import (
	"context"
	"embed"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/command"
	"github.com/klothoplatform/klotho/pkg/k2/cleanup"
	"github.com/klothoplatform/klotho/pkg/logging"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"
)

//go:embed templates/python/infra.py.tmpl
var files embed.FS

type ApplicationRequest struct {
	Context         context.Context
	ProjectName     string
	AppName         string
	Environment     string
	OutputDirectory string
	DefaultRegion   string
	Runtime         string
	ProgramFileName string
	SkipInstall     bool
}

func Application(request ApplicationRequest) error {
	outDir := request.OutputDirectory
	if outDir == "" {
		outDir = "."
	}

	if request.Runtime != "python" {
		return fmt.Errorf("unsupported runtime: %s", request.Runtime)
	}

	if !strings.HasSuffix(request.ProgramFileName, ".py") {
		request.ProgramFileName = fmt.Sprintf("%s.py", request.ProgramFileName)
	}

	if err := createOutputDirectory(outDir); err != nil {
		return err
	}

	fmt.Println("Creating Python program...")
	if err := createProgramFile(outDir, request.ProgramFileName, request); err != nil {
		return err
	}
	fmt.Printf("Created %s\n", filepath.Join(outDir, request.ProgramFileName))

	if !request.SkipInstall {
		if err := updatePipfile(request.Context, outDir); err != nil {
			return err
		}
	}
	return nil
}

func getPipCommand() (string, error) {
	if _, err := exec.LookPath("pip3"); err == nil {
		return "pip3", nil
	}
	if _, err := exec.LookPath("pip"); err == nil {
		return "pip", nil
	}
	return "", fmt.Errorf("pip not found")
}

// updatePipfile updates the Pipfile in the output directory with the necessary dependencies by invoking pipenv install
func updatePipfile(ctx context.Context, outDir string) error {
	// check if pipenv is installed and if not, install it
	if _, err := exec.LookPath("pipenv"); err != nil {
		fmt.Println("pipenv not found, installing pipenv")
		pip, err := getPipCommand()
		if err != nil {
			return err
		}
		if err = runCommand(ctx, outDir, pip, []string{"install", "pipenv"}); err != nil {
			return fmt.Errorf("failed to install pipenv: %w", err)
		}
		fmt.Println("pipenv installed successfully")
	}
	fmt.Println("Installing klotho python SDK...")
	// Install the necessary dependencies
	if err := runCommand(ctx, outDir, "pipenv", []string{"install", "-d", "klotho"}); err != nil {
		return fmt.Errorf("failed to install klotho python SDK: %w", err)
	}
	fmt.Println("klotho python SDK installed successfully")

	return nil
}

func runCommand(ctx context.Context, dir string, name string, args []string) error {
	log := logging.GetLogger(ctx).Sugar()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	command.SetProcAttr(cmd)

	cleanup.OnKill(func(signal syscall.Signal) error {
		cleanup.SignalProcessGroup(cmd.Process.Pid, syscall.SIGTERM)
		return nil
	})

	log.Debugf("Executing: %s for %v", cmd.Path, cmd.Args)
	err := cmd.Run()
	if err != nil {
		log.Errorf("%s process exited with error: %v", name, err)
	}
	log.Debugf("%s process exited successfully", name)
	return err
}

func createProgramFile(outDir, programFileName string, request ApplicationRequest) error {
	programTemplateContent, err := files.ReadFile("templates/python/infra.py.tmpl")
	if err != nil {
		return err
	}

	programFile, err := os.Create(filepath.Join(outDir, programFileName))
	if err != nil {
		return err
	}
	defer programFile.Close()

	program, err := template.New("program").Parse(string(programTemplateContent))
	if err != nil {
		return err
	}

	err = program.Execute(programFile, request)
	if err != nil {
		return err
	}
	return nil
}

func createOutputDirectory(outDir string) error {
	// Create the output directory if it doesn't exist
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		if err := os.Mkdir(outDir, 0755); err != nil {
			return err
		}
	}
	return nil
}
