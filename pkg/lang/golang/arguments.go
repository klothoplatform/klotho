package golang

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

type Argument struct {
	Content string
	Type    string
}

// GetArguements is passed a tree-sitter node, which is of type argument_list, and returns a list of in order Arguments
func GetArguements(args *sitter.Node) []Argument {

	arguments := []Argument{}
	nextMatch := doQuery(args, findArgs)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		arg := match["arg"]
		if arg == nil {
			continue
		}

		arguments = append(arguments, Argument{Content: arg.Content(), Type: arg.Type()})
	}
	return arguments
}

func ArgumentListToString(args []Argument) string {
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
	return a.Type == "interpreted_string_literal"
}
