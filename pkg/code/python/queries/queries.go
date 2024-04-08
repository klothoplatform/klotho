package queries

import (
	_ "embed"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

var (
	//go:embed import.scm
	importStr []byte
	Import    = query(importStr)

	//go:embed func_call.scm
	funcCall []byte
	FuncCall = query(funcCall)

	//go:embed func_call_args.scm
	funcCallArgs []byte
	FuncCallArgs = query(funcCallArgs)

	//go:embed identifiers.scm
	identifiers []byte
	Identifiers = query(identifiers)

	//go:embed definitions.scm
	definitions []byte
	Definitions = query(definitions)

	//go:embed boto3_resource.scm
	boto3Resource []byte
	Boto3Resource = query(boto3Resource)
)

var lang = python.GetLanguage()

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
