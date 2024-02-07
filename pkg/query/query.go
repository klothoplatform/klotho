package query

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

type NextFunc[T any] func() (T, bool)

type MatchNodes = map[string]*sitter.Node

type NextMatchFunc = NextFunc[MatchNodes]

// Exec returns a function that acts as an iterator, each call will
// loop over the next match lazily and populate the results map with a mapping
// of field name as defined in the query to mapped node.
func Exec(lang *sitter.Language, c *sitter.Node, q string) NextMatchFunc {
	if c == nil {
		return func() (map[string]*sitter.Node, bool) {
			return nil, false
		}
	}

	query, err := sitter.NewQuery([]byte(q), lang)
	if err != nil {
		// Panic because this is a programmer error with the query string.
		panic(fmt.Errorf("Error constructing query for %s: %w", q, err))
	}

	cursor := sitter.NewQueryCursor()
	cursor.Exec(query, c)

	nextMatch := func() (map[string]*sitter.Node, bool) {
		match, found := cursor.NextMatch()
		if !found || match == nil {
			return nil, false
		}
		results := make(map[string]*sitter.Node)

		for _, capture := range match.Captures {
			results[query.CaptureNameForId(capture.Index)] = capture.Node
		}
		return results, true
	}

	return nextMatch
}

func Collect[T any](f NextFunc[T]) []T {
	var results []T
	for {
		if elem, found := f(); found {
			results = append(results, elem)
		} else {
			break
		}
	}
	return results
}
