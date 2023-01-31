package javascript

import (
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
)

func SpecificExportQuery(n *sitter.Node, wantName string) *sitter.Node {
	nextMatch := DoQuery(n, proxyExport)

	var last *sitter.Node
	for {
		match, found := nextMatch()
		if !found || match == nil {
			break
		}

		result, obj, name := match["result"], match["obj"], match["name"]

		if !query.NodeContentEquals(obj, "exports") {
			continue
		}

		if wantName == "" {
			last = result
		} else if query.NodeContentEquals(name, wantName) {
			last = result
		}
	}

	if last == nil {
		return nil
	}
	return last
}

func ImportUsageQuery(n *sitter.Node, importName string) []*sitter.Node {
	nodes := make([]*sitter.Node, 0)
	nextMatch := DoQuery(n, proxyUsage)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		obj, prop := match["obj"], match["prop"]

		if query.NodeContentEquals(obj, importName) {
			nodes = append(nodes, prop)
		}

	}
	return nodes
}

func SpecificAsyncFuncDecl(n *sitter.Node, funcName string) *sitter.Node {
	nextMatch := DoQuery(n, proxyAsync)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		function, name := match["function"], match["name"]

		// Because `async` is not a named child, we cannot get/select it from the query.
		checkAsyncNode := function.Child(0)
		if !query.NodeContentEquals(checkAsyncNode, "async") {
			continue
		}

		if query.NodeContentEquals(name, funcName) {
			return function
		}
	}
	return nil
}
