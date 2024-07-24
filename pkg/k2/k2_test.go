package k2

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/k2/language_host"
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/orchestration"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/set"
	"github.com/r3labs/diff"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/semaphore"
	"gopkg.in/yaml.v3"
)

func TestK2(t *testing.T) {
	tests, err := filepath.Glob(filepath.Join("testdata", "*", "infra.py"))
	if err != nil {
		t.Fatal(err)
	}

	// use a shared semaphore to prevent too many parallel tests
	sem := semaphore.NewWeighted(int64(runtime.NumCPU()))

	log := logging.LogOpts{
		Verbose:  true,
		Encoding: "pretty_console",
		DefaultLevels: map[string]zapcore.Level{
			"engine":   zapcore.WarnLevel,
			"progress": zapcore.WarnLevel,
			"dataflow": zapcore.WarnLevel,
		},
	}.NewLogger()

	for _, p := range tests {
		dir := filepath.Dir(p)
		name := filepath.Base(dir)
		tc := testCase{inputPath: p, sem: sem, log: log.With(zap.String("test", name))}
		t.Run(name, tc.Test)
	}
}

type testCase struct {
	// input fields
	inputPath string
	sem       *semaphore.Weighted
	log       *zap.Logger

	// output/intermediate fields
	project, application string
	stateFs              afero.Fs
	outputFs             afero.Fs
}

func (tc testCase) Test(t *testing.T) {
	t.Parallel()

	log := tc.log.Sugar()

	ctx := context.Background()
	ctx = logging.WithLogger(ctx, tc.log)

	var langHost language_host.LanguageHost
	err := langHost.Start(ctx, language_host.DebugConfig{}, filepath.Dir(tc.inputPath))
	if err != nil {
		t.Fatalf("Failed to start language host: %v", err)
		return
	}
	defer func() {
		if err := langHost.Close(); err != nil {
			t.Fatalf("Failed to close language host: %v", err)
		}
	}()

	ir, err := langHost.GetIR(ctx, &pb.IRRequest{Filename: tc.inputPath})
	if err != nil {
		t.Fatalf("Failed to get IR: %v", err)
		return
	}

	expectedConstructs := tc.getAllExpectedConstructs(t)
	actualConstructs := make(set.Set[string])
	for c := range ir.Constructs {
		actualConstructs.Add(c)
	}
	missingConstructs := expectedConstructs.Difference(actualConstructs)
	extraConstructs := actualConstructs.Difference(expectedConstructs)
	if len(missingConstructs) > 0 || len(extraConstructs) > 0 {
		if len(missingConstructs) > 0 {
			t.Errorf("Missing constructs: %v", missingConstructs)
		}
		if len(extraConstructs) > 0 {
			t.Errorf("Extra constructs: %v", extraConstructs)
		}
		t.FailNow()
	}

	tc.project = ir.AppURN.Project
	tc.application = ir.AppURN.Application

	tc.stateFs = afero.NewMemMapFs()
	sm := model.NewStateManager(tc.stateFs, "state.yaml")
	sm.InitState(ir)

	tc.outputFs = afero.NewMemMapFs()

	o, err := orchestration.NewUpOrchestrator(sm, langHost.NewClient(), tc.outputFs, "/")
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
		return
	}

	err = o.RunUpCommand(ctx, ir, model.DryRunFileOnly, tc.sem)
	if err != nil {
		t.Fatalf("Failed to run up command: %v", err)
		return
	}

	// DEBUG
	_ = afero.Walk(tc.outputFs, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Errorf("failed to walk %s: %v", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		log.Infof("output file: %s (%d bytes)", path, info.Size())
		return nil
	})

	for c := range actualConstructs {
		t.Run(c, func(t *testing.T) {
			for _, f := range []string{"engine_input.yaml", "resources.yaml", "index.ts"} {
				// Evaluate the files in order they are created.
				// The processes that generate these files depend on the previous ones
				// thus, if one is not correct, then that will cascade to failures in subsequent
				// files. Because of that, stop the tests if it fails.
				if !tc.assertConstructFileEquals(t, c, f) {
					return
				}
			}
		})
	}
}

func (tc testCase) getAllExpectedConstructs(t *testing.T) set.Set[string] {
	t.Helper()

	inputDir := filepath.Dir(tc.inputPath)
	inputDirF, err := os.Open(inputDir)
	if err != nil {
		t.Fatalf("Failed to open input directory %s: %v", inputDir, err)
		return nil
	}
	defer inputDirF.Close()

	files, err := inputDirF.Readdirnames(-1)
	if err != nil {
		t.Fatalf("Failed to list input directory %s: %v", inputDir, err)
		return nil
	}

	constructs := make(set.Set[string])
	for _, f := range files {
		if c, ok := strings.CutSuffix(f, ".index.ts"); ok {
			constructs.Add(c)
		} else if c, ok := strings.CutSuffix(f, ".resources.yaml"); ok {
			constructs.Add(c)
		}
	}

	return constructs
}

func (tc testCase) assertConstructFileEquals(t *testing.T, construct, file string) bool {
	t.Helper()

	inputDir := filepath.Dir(tc.inputPath)

	expectedPath := filepath.Join(inputDir, construct+"."+file)
	expectedF, err := os.Open(expectedPath)
	if err != nil {
		t.Errorf("Failed to open expected file %s: %v", expectedPath, err)
		return false
	}
	defer expectedF.Close()

	actualPath := filepath.Join("/"+construct, file)
	actualF, err := tc.outputFs.Open(actualPath)
	if err != nil {
		t.Errorf("Failed to open actual file %s: %v", actualPath, err)
		return false
	}
	defer actualF.Close()

	switch ext := filepath.Ext(file); ext {
	case ".yaml", ".yml":
		return assertYamlEquals(t, file, expectedF, actualF)

	case ".ts":
		return assertContentEquals(t, file, expectedF, actualF)

	default:
		t.Errorf("Unsupported file extension %s", ext)
		return false
	}
}

func assertYamlEquals(t *testing.T, file string, expectedF, actualF io.Reader) bool {
	var expect, actual map[string]interface{}
	err := yaml.NewDecoder(expectedF).Decode(&expect)
	if err != nil {
		t.Errorf("failed to read expected yaml %s: %v", file, err)
		return false
	}

	err = yaml.NewDecoder(actualF).Decode(&actual)
	if err != nil {
		t.Errorf("failed to read actual yaml %s: %v", file, err)
		return false
	}
	differ, err := diff.NewDiffer(diff.SliceOrdering(false))
	if err != nil {
		t.Errorf("failed to create differ %v", err)
		return false
	}
	changes, err := differ.Diff(expect, actual)
	if err != nil {
		t.Errorf("failed to diff %s: %v", file, err)
		return false
	}
	for _, c := range changes {
		path := strings.Join(c.Path, ".")
		switch c.Type {
		case diff.CREATE:
			t.Errorf("[%s] %s %s: %v", path, c.Type, path, c.To)
		case diff.DELETE:
			t.Errorf("[%s] %s %s: %v", path, c.Type, path, c.From)
		case diff.UPDATE:
			t.Errorf("[%s] %s %s: %v to %v", path, c.Type, path, c.From, c.To)
		}
	}

	return len(changes) == 0
}

func assertContentEquals(t *testing.T, file string, expectedF, actualF io.Reader) bool {
	expect, err := io.ReadAll(expectedF)
	if err != nil {
		t.Errorf("failed to read expected content %s: %v", file, err)
		return false
	}

	actual, err := io.ReadAll(actualF)
	if err != nil {
		t.Errorf("failed to read actual content %s: %v", file, err)
		return false
	}

	return assert.Equal(t, string(expect), string(actual))
}
