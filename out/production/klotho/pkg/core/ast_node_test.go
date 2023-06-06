package core

import (
	"fmt"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestNodeContent_Format(t *testing.T) {
	cases := []struct {
		name   string
		node   dummyNode
		format string
		want   string
	}{
		{
			name:   "single-line %s",
			node:   dummyNode{0, 0, "1+1"},
			format: "%s",
			want:   "0| 1+1",
		},
		{
			name:   "single-line %#v",
			node:   dummyNode{0, 0, "1+1"},
			format: "%#v",
			want:   "0| 1+1",
		},
		{
			name:   "single-line %#v on line 5",
			node:   dummyNode{5, 0, "1+1"},
			format: "%#v",
			want:   "5| 1+1",
		},
		{
			name:   "multi-line %s",
			node:   dummyNode{0, 0, "/*foo\nbar*/"},
			format: "%s",
			want:   "0-1| /*foo ...",
		},
		{
			name:   "multi-line %#v",
			node:   dummyNode{0, 0, "/*foo\nbar*/"},
			format: "%#v",
			want:   "0| /*foo\n1| bar*/",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			nodeContent := NodeContent{
				Endpoints: tt.node,
				Content:   tt.node.content,
			}

			actual := fmt.Sprintf(tt.format, nodeContent)

			assert.Equal(tt.want, actual)
		})
	}
}

type dummyNode struct {
	startRow uint32
	startCol uint32
	content  string
}

func (e dummyNode) StartPoint() sitter.Point {
	return sitter.Point{
		Row:    e.startRow,
		Column: e.startCol,
	}
}

func (e dummyNode) EndPoint() sitter.Point {
	lines := strings.Split(e.content, "\n")
	lastLine := lines[len(lines)-1]
	endCol := uint32(len(lastLine))
	if len(lines) == 1 {
		endCol += e.startCol
	}

	return sitter.Point{
		Row:    e.startRow + uint32(len(lines)) - 1,
		Column: endCol,
	}
}
