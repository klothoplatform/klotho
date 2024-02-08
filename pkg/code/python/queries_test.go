package python

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/code/python/queries"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestQueries(t *testing.T) {
	tests, err := filepath.Glob(filepath.Join("testfiles", "*.py"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range tests {
		tc := queryTestCase{inputPath: p}
		t.Run(filepath.Base(p), tc.Test)
	}
}

type queryTestCase struct {
	inputPath string
}

func (tc queryTestCase) Test(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile(tc.inputPath)
	if err != nil {
		t.Fatal(err)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	queries := map[string]*sitter.Query{
		"import":    queries.Import,
		"func_call": queries.FuncCall,
	}
	for name, query := range queries {
		t.Run(name, func(t *testing.T) { tc.TestQuery(t, tree, query, name) })
	}
}

func (tc queryTestCase) TestQuery(t *testing.T, tree *sitter.Tree, query *sitter.Query, name string) {
	var r []map[string]string
	for m := range sitter.QueryIterator(tree.RootNode(), query) {
		mr := make(map[string]string)
		for k, v := range m {
			mr[k] = v.Content()
		}
		r = append(r, mr)
	}

	var expect []map[string]string
	expectF, err := os.Open(strings.Replace(tc.inputPath, ".py", fmt.Sprintf("_%s.yaml", name), 1))
	if os.IsNotExist(err) {
		expect = nil
	} else if err != nil {
		t.Fatal(err)
	} else {
		err = yaml.NewDecoder(expectF).Decode(&expect)
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatal(fmt.Errorf("failed to decode %s: %w", expectF.Name(), err))
		}
	}

	if !assert.Equal(t, expect, r) {
		y, err := yaml.Marshal(r)
		if err == nil {
			fmt.Printf("%s actual:\n", name)
			fmt.Println(string(y))
		}
	}
}
