package queries

import (
	_ "embed"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/dockerfile"
)

var (
	//go:embed from.scm
	fromStr []byte
	From    = query(fromStr)
)

var lang = dockerfile.GetLanguage()

func query(s []byte) *sitter.Query {
	q, err := sitter.NewQuery(s, lang)
	if err != nil {
		panic(err)
	}
	return q
}

var queryCache sync.Map // map[string]*sitter.Query

func MakeQuery(s string) *sitter.Query {
	if q, ok := queryCache.Load(s); ok {
		return q.(*sitter.Query)
	}
	q := query([]byte(s))
	queryCache.Store(s, q)
	return q
}
