package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	engine_errs "github.com/klothoplatform/klotho/pkg/engine/errors"
	"github.com/r3labs/diff"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestEngine(t *testing.T) {
	os.Setenv("KLOTHO_DEBUG_DIR", "test_debug")
	if err := os.MkdirAll("test_debug", 0755); err != nil {
		t.Fatal(err)
	}

	tests, err := filepath.Glob(filepath.Join("testdata", "*.input.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range tests {
		tc := engineTestCase{inputPath: p}
		t.Run(filepath.Base(p), tc.Test)
	}
}

type engineTestCase struct {
	inputPath string
}

func (tc engineTestCase) readGraph(t *testing.T, f io.Reader) FileFormat {
	t.Helper()
	var ff FileFormat
	err := yaml.NewDecoder(f).Decode(&ff)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to read graph: %w", err))
	}
	return ff
}

func (tc engineTestCase) Test(t *testing.T) {
	t.Parallel()
	inputYaml, err := os.Open(tc.inputPath)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to open input file: %w", err))
	}
	defer inputYaml.Close()

	inputFile := tc.readGraph(t, inputYaml)

	main := EngineMain{}
	err = main.AddEngine()
	if err != nil {
		t.Fatal(fmt.Errorf("failed to add engine: %w", err))
	}
	context := &EngineContext{
		Constraints:  inputFile.Constraints,
		InitialState: inputFile.Graph,
	}
	returnCode, engineErrs := main.Run(context)
	// TODO find a convenient way to specify the return code in the testdata

	errDetails := new(bytes.Buffer)
	err = writeEngineErrsJson(engineErrs, errDetails)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to write engine errors: %w", err))
	}
	errDetailsFile, err := os.Open(strings.Replace(tc.inputPath, ".input.yaml", ".err.json", 1))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(fmt.Errorf("failed to open error details file: %w", err))
	}
	if errDetailsFile != nil {
		defer errDetailsFile.Close()
	}
	assertErrDetails(t, errDetailsFile, engineErrs)

	if returnCode == 1 {
		// Run resulted in a failure. After checking the error details, we're done.
		return
	}
	sol := context.Solutions[0]
	actualContent, err := yaml.Marshal(construct.YamlGraph{Graph: sol.DataflowGraph()})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to marshal actual output: %w", err))
	}

	expectYaml, err := os.Open(strings.Replace(tc.inputPath, ".input.yaml", ".expect.yaml", 1))
	if err != nil {
		t.Fatal(fmt.Errorf("failed to open expected output file: %w", err))
	}
	defer expectYaml.Close()
	expectContent, err := io.ReadAll(expectYaml)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to read expected output file: %w", err))
	}
	assertYamlMatches(t, string(expectContent), string(actualContent), "dataflow")

	// Always visualize views even if we're not testing them to make sure that it at least succeeds
	vizFiles, err := main.Engine.VisualizeViews(sol)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to generate views: %w", err))
	}

	iacVizFile, err := os.Open(strings.Replace(tc.inputPath, ".input.yaml", ".iac-viz.yaml", 1))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(fmt.Errorf("failed to open iac viz file: %w", err))
	}
	if iacVizFile != nil {
		defer iacVizFile.Close()
	}

	dataflowVizFile, err := os.Open(strings.Replace(tc.inputPath, ".input.yaml", ".dataflow-viz.yaml", 1))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(fmt.Errorf("failed to open dataflow viz file: %w", err))
	}
	if dataflowVizFile != nil {
		defer dataflowVizFile.Close()
	}
	if iacVizFile == nil && dataflowVizFile == nil {
		return
	}

	buf := new(bytes.Buffer)

	for _, f := range vizFiles {
		buf.Reset()
		_, err := f.WriteTo(buf)
		if err != nil {
			t.Fatal(fmt.Errorf("failed to write viz file %s: %w", f.Path(), err))
		}
		if strings.HasPrefix(f.Path(), "iac-") && iacVizFile != nil {
			iacExpect, err := io.ReadAll(iacVizFile)
			if err != nil {
				t.Error(fmt.Errorf("failed to read iac viz file: %w", err))
			} else {
				// assert.Equal(t, string(iacExpect), buf.String(), "IaC topology")
				assertYamlMatches(t, string(iacExpect), buf.String(), "IaC topology")
			}
		} else if strings.HasPrefix(f.Path(), "dataflow-") && dataflowVizFile != nil {
			dataflowExpect, err := io.ReadAll(dataflowVizFile)
			if err != nil {
				t.Error(fmt.Errorf("failed to read dataflow viz file: %w", err))
			} else {
				// assert.Equal(t, string(dataflowExpect), buf.String(), "dataflow topology")
				assertYamlMatches(t, string(dataflowExpect), buf.String(), "dataflow topology")
			}
		}
	}
}

func assertErrDetails(t *testing.T, expectR io.Reader, actual []engine_errs.EngineError) {
	var expectV []map[string]any
	err := json.NewDecoder(expectR).Decode(&expectV)
	if err != nil {
		t.Fatalf("failed to read expected error details: %v", err)
		return
	}
	actualBuf := new(bytes.Buffer)
	err = writeEngineErrsJson(actual, actualBuf)
	if err != nil {
		t.Fatalf("failed to write actual error details to buffer: %v", err)
		return
	}
	var actualV []map[string]any
	err = json.NewDecoder(actualBuf).Decode(&actualV)
	if err != nil {
		t.Fatalf("failed to read actual error details from buffer: %v", err)
		return
	}
	assert.Equal(t, expectV, actualV, "error details")
}

func assertYamlMatches(t *testing.T, expectStr, actualStr string, name string) {
	t.Helper()
	var expect, actual map[string]interface{}
	err := yaml.Unmarshal([]byte(expectStr), &expect)
	if err != nil {
		t.Errorf("failed to unmarshal expected %s graph: %v", name, err)
		return
	}
	err = yaml.Unmarshal([]byte(actualStr), &actual)
	if err != nil {
		t.Errorf("failed to unmarshal actual %s graph: %v", name, err)
		return
	}
	differ, err := diff.NewDiffer(diff.SliceOrdering(false))
	if err != nil {
		t.Errorf("failed to create differ for %s: %v", name, err)
		return
	}
	changes, err := differ.Diff(expect, actual)
	if err != nil {
		t.Errorf("failed to diff %s: %v", name, err)
		return
	}
	for _, c := range changes {
		path := strings.Join(c.Path, ".")
		switch c.Type {
		case diff.CREATE:
			t.Errorf("[%s] %s %s: %v", name, c.Type, path, c.To)
		case diff.DELETE:
			t.Errorf("[%s] %s %s: %v", name, c.Type, path, c.From)
		case diff.UPDATE:
			t.Errorf("[%s] %s %s: %v to %v", name, c.Type, path, c.From, c.To)
		}
	}
}
