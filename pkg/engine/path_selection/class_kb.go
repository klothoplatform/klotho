package path_selection

import (
	"errors"
	"fmt"
	"slices"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/knowledgebase"
)

func ClassPaths(
	kb knowledgebase.Graph,
	start, end string,
	classification string,
	cb func([]string) error,
) error {
	adjacencyMap, err := kb.AdjacencyMap()
	if err != nil {
		return err
	}
	startTmpl, err := kb.Vertex(start)
	if err != nil {
		return fmt.Errorf("failed to find start template: %w", err)
	}
	return classPaths(
		kb,
		adjacencyMap,
		start, end,
		classification,
		cb,
		[]string{start},
		classification == "" || slices.Contains(startTmpl.Classification.Is, classification),
	)
}

var (
	SkipPathErr = errors.New("skip path")
)

func classPaths(
	kb knowledgebase.Graph,
	adjacencyMap map[string]map[string]graph.Edge[string],
	start, end string,
	classification string,
	cb func([]string) error,
	currentPath []string,
	classificationSatisfied bool,
) error {
	last := currentPath[len(currentPath)-1]
	frontier := adjacencyMap[last]
	if len(frontier) == 0 {
		return nil
	}
	var errs []error
	for next := range frontier {
		if slices.Contains(currentPath, next) {
			// Prevent infinite looping, since the knowledge base can be cyclic
			continue
		}
		nextClassificationSatisfied := classificationSatisfied
		edge, err := kb.Edge(last, next)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		edgeTmpl := edge.Properties.Data.(*knowledgebase.EdgeTemplate)
		if edgeTmpl.DirectEdgeOnly {
			continue
		}
		if !nextClassificationSatisfied && slices.Contains(edgeTmpl.Classification, classification) {
			nextClassificationSatisfied = true
		}
		tmpl, err := kb.Vertex(next)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if next != end {
			// ContainsUnneccessaryHopsInPath
			if fct := tmpl.GetFunctionality(); fct != knowledgebase.Unknown {
				continue
			}
		}
		if !nextClassificationSatisfied && slices.Contains(tmpl.Classification.Is, classification) {
			nextClassificationSatisfied = true
		}
		if classification != "" && slices.Contains(tmpl.PathSatisfaction.DenyClassifications, classification) {
			continue
		}

		// NOTE(gg): The old code let the end point satisfy the classification. Is this correct?
		if next == end && nextClassificationSatisfied {
			if err := cb(append(currentPath, end)); err != nil {
				errs = append(errs, err)
			}
			continue
		} else if next != end {
			err := classPaths(
				kb,
				adjacencyMap,
				start, end,
				classification,
				cb,
				// This append is okay because we're only doing one path at a time, in DFS.
				// Otherwise, we'd need to copy the slice. This is why we use DFS instead of BFS or a stack-based approach
				// (like used in [graph.AllPathsBetween]).
				append(currentPath, next),
				nextClassificationSatisfied,
			)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("failed to find paths from %s: %w", last, err)
	}
	return nil
}
