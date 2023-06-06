package javascript

import (
	"regexp"
	"strings"
)

var commentRegex = regexp.MustCompile(`(?m)^(\s*)`)

// CommentNode
// TODO: this currently only uses line comments, which is not always the right
// behaviour.
func CommentNodes(oldFileContent string, nodesToComment ...string) string {
	if len(nodesToComment) == 0 {
		return oldFileContent
	}

	newFileContent := oldFileContent
	for _, oldNodeContent := range nodesToComment {
		newNodeContent := commentRegex.ReplaceAllString(oldNodeContent, "// $1")

		newFileContent = strings.ReplaceAll(
			newFileContent,
			oldNodeContent,
			newNodeContent,
		)
	}

	return newFileContent
}
