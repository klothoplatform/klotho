package engine2

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/r3labs/diff"
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
	expectYaml, err := os.Open(strings.Replace(tc.inputPath, ".input.yaml", ".expect.yaml", 1))
	if err != nil {
		t.Fatal(fmt.Errorf("failed to open expected output file: %w", err))
	}
	defer expectYaml.Close()

	inputFile := tc.readGraph(t, inputYaml)
	expectContent, err := io.ReadAll(expectYaml)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to read expected output file: %w", err))
	}

	main := EngineMain{}
	err = main.AddEngine()
	if err != nil {
		t.Fatal(fmt.Errorf("failed to add engine: %w", err))
	}
	context := &EngineContext{
		Constraints:  inputFile.Constraints,
		InitialState: inputFile.Graph,
	}
	err = main.Engine.Run(context)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to run engine: %w", err))
	}

	sol := context.Solutions[0]
	actualContent, err := yaml.Marshal(construct.YamlGraph{Graph: sol.DataflowGraph()})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to marshal actual output: %w", err))
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
			t.Errorf("[%s] %s %s: %v -> %v", name, c.Type, path, c.From, c.To)
		}
	}
}
