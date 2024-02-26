package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/klothoplatform/klotho/pkg/auth"
	"github.com/klothoplatform/klotho/pkg/code"
	"github.com/klothoplatform/klotho/pkg/code/docker"
	"github.com/klothoplatform/klotho/pkg/code/python"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

type Args struct {
	Root            string `arg:"" help:"The root directory of the project." type:"existingdir"`
	ArchitectureId  string `short:"a" help:"If specified, the architecture id to upload."`
	Verbose         bool   `short:"v" help:"Enable verbose mode."`
	CategoryLogsDir string `arg:"log-dir" help:"The directory to write category logs to." default:"logs"`
}

var infracopilotUrl = os.Getenv("INFRACOPILOT_URL")

func main() {
	var args Args
	ctx := kong.Parse(&args)

	logOpts := logging.LogOpts{
		Verbose:         args.Verbose,
		CategoryLogsDir: args.CategoryLogsDir,
		DefaultLevels: map[string]zapcore.Level{
			"lsp":       zap.WarnLevel,
			"lsp/pylsp": zap.WarnLevel,
		},
		Encoding: "pretty_console",
	}

	zap.ReplaceGlobals(logOpts.NewLogger())
	defer zap.L().Sync()

	if err := ctx.Run(); err != nil {
		panic(err)
	}
}

func (a Args) Run(kctx *kong.Context) error {
	root, err := filepath.Abs(a.Root)
	if err != nil {
		return err
	}
	ctx := context.Background()

	files := os.DirFS(root)
	// For performance, filter some known irrelevant directories
	files = code.FilteredFS{
		FS: files,
		Exclude: func(path string) bool {
			switch filepath.Base(path) {
			case ".git", "node_modules", "__pycache__":
				return true
			}
			return false
		},
	}

	var c constraints.Constraints
	for _, f := range []func(context.Context, fs.FS) (constraints.Constraints, error){
		python.FindBoto3Constraints,
		docker.FindDockerConstraints,
	} {
		fc, err := f(ctx, files)
		if err != nil {
			return err
		}
		c.Append(fc)
	}

	if a.ArchitectureId != "" {
		return a.UploadArchitecture(ctx, c)
	}

	cy, _ := yaml.Marshal(c)
	fmt.Println("constraints:")
	fmt.Println(string(cy))

	return nil
}

func (a Args) UploadArchitecture(ctx context.Context, c constraints.Constraints) error {
	input := map[string]interface{}{
		"constraints": c.ToList(),
	}
	_, client, err := auth.GetAuthToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth token: %w", err)
	}

	buf := new(bytes.Buffer)

	log := zap.S().Named("infracopilot")
	log.Infof("Uploading architecture %s", a.ArchitectureId)

	if infracopilotUrl == "" {
		infracopilotUrl = "https://app.infracopilot.io"
	}

	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/architecture/%s/environment/default", infracopilotUrl, a.ArchitectureId),
		nil,
	)
	// req.Header.Add("Accept", "application/json")
	// for some reason, having accept json returns the json wrapped in a string, whereas octet-stream doesn't wrap it
	req.Header.Add("Accept", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get current version: %s", resp.Status)
	}

	buf.Reset()
	var current struct {
		Version int `json:"version"`
	}
	err = json.NewDecoder(io.TeeReader(resp.Body, buf)).Decode(&current)
	log.Debugf("Current version response: %s", strings.TrimSpace(buf.String()))
	if err != nil {
		return fmt.Errorf("failed to decode current version response: %w", err)
	}

	buf.Reset()
	err = json.NewEncoder(buf).Encode(input)
	if err != nil {
		return fmt.Errorf("failed to encode constraints: %w", err)
	}
	body := buf.Bytes()
	log.Debugf("Body: %s", strings.TrimSpace(string(body)))

	req, _ = http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/architecture/%s/environment/default/run?state=%d",
			infracopilotUrl, a.ArchitectureId, current.Version,
		),
		bytes.NewReader(body),
	)

	req.Header.Add("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post to architecture: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to post to architecture: %s", resp.Status)
	}

	buf.Reset()
	var result struct {
		Version int `json:"version"`
	}
	err = json.NewDecoder(io.TeeReader(resp.Body, buf)).Decode(&result)
	log.Debugf("Run response: %s", strings.TrimSpace(buf.String()))
	if err != nil {
		return fmt.Errorf("failed to decode run response: %w", err)
	}

	log.Infof("Successfully updated architecture %s to version %d", a.ArchitectureId, result.Version)
	return nil
}
