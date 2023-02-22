package golang

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
)

type Argument struct {
	Content string
	Type    string
}

// GetArguments is passed a tree-sitter node, which is of type argument_list, and returns a list of in order Arguments
func getArguments(args *sitter.Node) (arguments []Argument, found bool) {
	fnName := ""
	nextMatch := doQuery(args, findFunctionCall)
	for {
		match, found := nextMatch()
		if !found {
			break
		}
		fn := match["function"]
		arg := match["arg"]

		if fnName != "" && !query.NodeContentEquals(fn, fnName) {
			break
		}

		fnName = fn.Content()

		if arg == nil {
			continue
		}

		arguments = append(arguments, Argument{Content: arg.Content(), Type: arg.Type()})
	}
	if fnName != "" {
		found = true
	}
	return
}

func argumentListToString(args []Argument) string {
	result := "("
	for index, arg := range args {
		if index < len(args)-1 {
			result += fmt.Sprintf("%s, ", arg.Content)
		} else {
			result += arg.Content + ")"
		}
	}
	return result
}

func (a *Argument) IsString() bool {
	return a.Type == "interpreted_string_literal" || a.Type == "raw_string_literal"
}
