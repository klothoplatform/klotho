package queries

import (
	_ "embed"

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
)

var lang = python.GetLanguage()

func query(s []byte) *sitter.Query {
	q, err := sitter.NewQuery(s, lang)
	if err != nil {
		panic(err)
	}
	return q
}
