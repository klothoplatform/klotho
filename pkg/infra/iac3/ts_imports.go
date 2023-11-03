package iac3

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

func (tc *TemplatesCompiler) RenderImports(out io.Writer) error {
	resources, err := construct.ReverseTopologicalSort(tc.graph)
	if err != nil {
		return err
	}

	allImports := make(map[string]struct{})
	var errs error
	for _, r := range resources {
		t, err := tc.ResourceTemplate(r)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		for _, statement := range t.Imports {
			allImports[statement] = struct{}{}
		}
	}
	if errs != nil {
		return errs
	}

	sortedImports := make([]string, 0, len(allImports))
	for statement := range allImports {
		sortedImports = append(sortedImports, statement)
	}

	sort.Strings(sortedImports)

	_, err = fmt.Fprintf(
		out,
		"%s\n",
		strings.Join(sortedImports, "\n"),
	)
	if err != nil {
		return err
	}

	return nil
}
