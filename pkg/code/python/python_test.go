package python

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/klothoplatform/klotho/pkg/code/python/queries"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestQueries(t *testing.T) {
	tests, err := filepath.Glob(filepath.Join("testfiles", "*.py"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range tests {
		tc := testCase{inputPath: p}
		t.Run(filepath.Base(p), tc.Test)
	}
}

type testCase struct {
	inputPath string
}

func (tc testCase) Test(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile(tc.inputPath)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := NewParser().ParseCtx(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	queries := map[string]*sitter.Query{
		"import":         queries.Import,
		"func_call_args": queries.FuncCallArgs,
		"func_call":      queries.FuncCall,
	}
	for name, query := range queries {
		t.Run(name, func(t *testing.T) { tc.TestQuery(t, tree, query, name) })
	}

	t.Run("constraints", func(t *testing.T) { tc.TestConstraints(t, input) })
}

func (tc testCase) readTestYaml(t *testing.T, suffix string, val any) {
	t.Helper()

	expectPath := strings.Replace(tc.inputPath, ".py", fmt.Sprintf("_%s.yaml", suffix), 1)
	expectF, err := os.Open(expectPath)
	if os.IsNotExist(err) {
		return
	} else if err != nil {
		t.Fatal(fmt.Errorf("failed to open test file %s: %w", expectPath, err))
	}
	defer expectF.Close()

	err = yaml.NewDecoder(expectF).Decode(val)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatal(fmt.Errorf("failed to decode %s: %w", expectPath, err))
	}
}

func (tc testCase) TestQuery(t *testing.T, tree *sitter.Tree, query *sitter.Query, name string) {
	var r []map[string]string
	for m := range sitter.QueryIterator(tree.RootNode(), query) {
		mr := make(map[string]string)
		for k, v := range m {
			mr[k] = v.Content()
		}
		r = append(r, mr)
	}

	var expect []map[string]string
	tc.readTestYaml(t, name, &expect)

	if !assert.Equal(t, expect, r) {
		y, err := yaml.Marshal(r)
		if err == nil {
			fmt.Printf("actual %s:\n", name)
			fmt.Println(string(y))
		}
	}
}

func (tc testCase) TestConstraints(t *testing.T, input []byte) {
	files := fstest.MapFS{
		"test.py": &fstest.MapFile{Data: input},
	}
	actual, err := FindBoto3Constraints(context.Background(), files)
	if errors.Is(err, exec.ErrNotFound) {
		t.Skip("pylsp not found")
	}
	if err != nil {
		t.Fatal(err)
	}
	var expect constraints.Constraints
	tc.readTestYaml(t, "constraints", &expect)

	if !assert.Equal(t, expect, actual) {
		y, err := yaml.Marshal(actual)
		if err == nil {
			fmt.Println("actual constraints:")
			fmt.Println(string(y))
		}
	}
}
