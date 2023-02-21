package query

import (
	"github.com/klothoplatform/klotho/pkg/core"
	sitter "github.com/smacker/go-tree-sitter"
)

// queryCache stores references to all *sitter.Query instances created by invoking Exec for reuse in subsequent Exec invocations
var queryCache = Cache{}

type NextFunc[T any] func() (T, bool)

type MatchNodes = map[string]*sitter.Node

type NextMatchFunc = NextFunc[MatchNodes]

// Exec returns a function that acts as an iterator, each call will
// loop over the next match lazily and populate the results map with a mapping
// of field name as defined in the query to mapped node.
func Exec(lang core.SourceLanguage, c *sitter.Node, q string) NextMatchFunc {
	if c == nil {
		return func() (map[string]*sitter.Node, bool) {
			return nil, false
		}
	}

	query, ok := queryCache.GetQuery(lang.ID, q)
	if !ok {
		var err error
		query, err = sitter.NewQuery([]byte(q), lang.Sitter)
		if err != nil {
			// Panic because this is a programmer error with the query string.
			panic(core.WrapErrf(err, "Error constructing query for %s", q))
		}
		queryCache.AddQuery(lang.ID, q, query)
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

// Cache stores queries by language ID in a threadsafe manner
type Cache struct {
	queriesByLang core.ConcurrentMap[core.LanguageId, *core.ConcurrentMap[string, *sitter.Query]]
}

// AddQuery adds a new query to the cache
func (m *Cache) AddQuery(lang core.LanguageId, name string, query *sitter.Query) {
	m.queriesByLang.Compute(lang, func(k core.LanguageId, v *core.ConcurrentMap[string, *sitter.Query]) (*core.ConcurrentMap[string, *sitter.Query], bool) {
		if v == nil {
			v = &core.ConcurrentMap[string, *sitter.Query]{}
		}
		v.Set(name, query)
		return v, true
	})
}

// GetQuery gets the *sitter.Query instance associated with the provided language ID and name combination
func (m *Cache) GetQuery(lang core.LanguageId, name string) (*sitter.Query, bool) {
	if lCache, ok := m.queriesByLang.Get(lang); ok {
		return lCache.Get(name)
	}
	return nil, false
}
