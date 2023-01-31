package python

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
)

type FunctionCallDetails struct {
	Name      string
	Arguments []FunctionArg
}

type FunctionArg struct {
	Name  string
	Value string
}

func (a FunctionArg) String() string {
	if a.Name != "" {
		return fmt.Sprintf("%s=%s", a.Name, a.Value)
	}
	return a.Value
}

func getNextCallDetails(node *sitter.Node) (callDetails FunctionCallDetails, found bool) {
	fnName := ""
	nextMatch := DoQuery(node, findFunctionCalls)
	callDetails = FunctionCallDetails{Arguments: []FunctionArg{}}
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		fn := match["function"]
		argName := match["argName"]
		arg := match["arg"]

		if fnName != "" && !query.NodeContentEquals(fn, fnName) {
			break
		}

		fnName = fn.Content()
		argNameContent := ""
		argContent := ""
		callDetails.Name = fnName

		if argName != nil {
			argNameContent = argName.Content()
		}
		if arg != nil {
			argContent = arg.Content()
		}

		// ignore overlapping pattern that captures keyword_argument nodes as standard aguments
		if argContent != "" && !strings.Contains(argContent, "=") {
			callDetails.Arguments = append(callDetails.Arguments, FunctionArg{Name: argNameContent, Value: argContent})
		}

	}
	if fnName != "" {
		found = true
	}
	return
}

func AddOrReplaceArg(arg FunctionArg, args []FunctionArg) []FunctionArg {
	for i, a := range args {
		if arg.Name == a.Name {
			args[i] = arg
			return args
		}
	}

	return append(args, arg)
}

func parentOfType(node *sitter.Node, parentType string) *sitter.Node {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Type() == parentType {
			return parent
		}
	}
	return nil
}
