package core

import (
	"fmt"
	"math"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type (
	NodeEndpoints interface {
		StartPoint() sitter.Point
		EndPoint() sitter.Point
	}

	NodeContent struct {
		Endpoints NodeEndpoints
		Content   string
	}
)

func (n NodeContent) Format(s fmt.State, verb rune) {
	start := n.Endpoints.StartPoint()
	end := n.Endpoints.EndPoint()
	if s.Flag('+') || s.Flag('#') {
		lineNum := start.Row

		lineNumWidth := 1
		if end.Row > 0 {
			lineNumWidth = int(math.Ceil(math.Log10(float64(end.Row))))
		}
		lineNumFmt := fmt.Sprintf("%%0%dd", lineNumWidth)

		content := lineIndentRE.ReplaceAllStringFunc(n.Content, func(s string) string {
			res := fmt.Sprintf(lineNumFmt+"| %s", lineNum, s)
			lineNum++
			return res
		})
		fmt.Fprint(s, content)
	} else {
		content := n.Content
		if firstNl := strings.Index(content, "\n"); firstNl >= 0 {
			content = content[:firstNl]
		}
		if end.Row > start.Row {
			fmt.Fprintf(s, "%d-%d| %s ...",
				start.Row,
				end.Row,
				content,
			)
		} else {
			fmt.Fprintf(s, "%d| %s", start.Row, content)
		}
	}
}
