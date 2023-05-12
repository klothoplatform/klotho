package testutil

import (
	"fmt"
	"strings"

	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

// UnIndent removes a level of indentation from the given string. The rules are very simple:
//   - first, drop any leading newlines from the string
//   - then, find the indentation in the first line, which is defined by the leading tabs-or-spaces in that line
//   - then, trim that prefix from all lines
//
// The prefix must match exactly, and this method removes it just by invoke [strings.CutPrefix] -- nothing fancier.
// In particular, this means that if you're mixing tabs and spaces, you may find yourself in for a bad time: if the
// first string uses "<space><tab>" and the second uses "<tab><space>", the second does not count as an indentation,
// and won't be affected.
//
// You can use this to embed yaml within test code as literal strings:
//
//	 ...
//	 SomeField: MyStruct {
//	 	foo: unIndent(`
//	 		hello: world
//	 		counts:
//	 		  - 1
//	 		  - 2
//	 		hello: world`
//	 	),
//	...
//
// The resulting string will be:
//
//	┌────────────┐ ◁─ no newline
//	│hello: world│ ◁─╮
//	│counts:     │ ◁─┤
//	│  - 1       │ ◁─┼─ no extra indentation
//	│  - 2       │ ◁─┤
//	│hello: world│ ◁─╯
//	└────────────┘
func UnIndent(y string) string {
	y = strings.TrimLeft(y, "\n")
	tabsCount := 0
	for ; tabsCount < len(y) && y[tabsCount] == '\t' || y[tabsCount] == ' '; tabsCount += 1 {
		// nothing; the tabsCount += 1 is the side effect we want
	}
	prefixTabs := y[:tabsCount]
	sb := strings.Builder{}
	sb.Grow(len(y))
	for _, line := range strings.Split(y, "\n") {
		line, _ = strings.CutPrefix(line, prefixTabs)
		sb.WriteString(line)
		sb.WriteRune('\n')
	}
	return sb.String()
}

// YamlPath returns a subset of the given yaml file, as specified by its path. It uses [yamlpath] under the hood.
// See the [yamlpath's github page] for details, though the package's godocs are easier to read.
//
// tldr: `$.path.to.your[0].subdocument` (the `$` is literally a dollar sign you should use to anchor the path).
//
// This function expects there to be a single node result. If you want a list, select the list's parent instead.
//
// If there are any errors along the way, this will return `// ERROR: ${msg}`.
//
// [yamlpath's github page]: https://github.com/vmware-labs/yaml-jsonpath
func SafeYamlPath(yamlStr string, path string) string {
	path_obj, err := yamlpath.NewPath(path)
	if err != nil {
		return fmt.Sprintf("// ERROR: %s", err)
	}

	var parsed_node yaml.Node
	err = yaml.Unmarshal([]byte(yamlStr), &parsed_node)
	if err != nil {
		return fmt.Sprintf("// ERROR: %s", err)
	}
	found_nodes, err := path_obj.Find(&parsed_node)
	if err != nil {
		return fmt.Sprintf("// ERROR: %s", err)
	}
	if len(found_nodes) != 1 {
		return fmt.Sprintf("// ERROR: expected exactly one match, but found %d", len(found_nodes))
	}
	result_bytes, err := yaml.Marshal(found_nodes[0])
	if err != nil {
		return fmt.Sprintf("// ERROR: %s", err)
	}
	return string(result_bytes)
}
