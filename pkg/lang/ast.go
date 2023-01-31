package lang

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/klothoplatform/klotho/pkg/multierr"
	sitter "github.com/smacker/go-tree-sitter"
)

func WriteAST(node *sitter.Node, out io.Writer) error {
	w := ASTWriter{
		node: node,
		out:  out,
	}
	w.WriteAST()
	return w.Err()
}

type ASTWriter struct {
	out  io.Writer
	node *sitter.Node

	errors multierr.Error
}

func (w *ASTWriter) Err() error {
	return w.errors.ErrOrNil()
}

func (w *ASTWriter) indent(n *sitter.Node) {
	for i := 0; i < w.level(n); i++ {
		w.write("  ")
	}
}

func (w *ASTWriter) writeLine(n *sitter.Node, s string) {
	w.indent(n)
	w.write(s)
	if !strings.HasSuffix(s, "\n") {
		_, err := w.out.Write([]byte("\n"))
		if err != nil {
			w.errors.Append(err)
		}
	}
}

func (w *ASTWriter) write(s string) {
	var err error
	if out, ok := w.out.(io.StringWriter); ok {
		_, err = out.WriteString(s)
	} else {
		_, err = w.out.Write([]byte(s))
	}
	if err != nil {
		w.errors.Append(err)
	}
}

func (w *ASTWriter) nodeName(n *sitter.Node) string {
	if n.Parent() == nil {
		return ""
	}
	c := sitter.NewTreeCursor(n.Parent())
	defer c.Close()

	if !c.GoToFirstChild() {
		return ""
	}
	for c.CurrentNode() != n {
		if !c.GoToNextSibling() {
			return ""
		}
	}
	return c.CurrentFieldName()
}

func (w *ASTWriter) level(n *sitter.Node) int {
	l := 0
	for n.Parent() != nil && n.Parent() != w.node {
		l++
		n = n.Parent()
	}
	return l
}

func (w *ASTWriter) WriteAST() {
	iterator := sitter.NewIterator(w.node, sitter.DFSMode)

	w.writeLine(w.node, w.node.String())
	w.writeLine(w.node, "=====")
	//iterates over the entire AST in DFSMode
	err := iterator.ForEach(func(n *sitter.Node) error {
		name := w.nodeName(n)
		if n.NamedChildCount() > 0 {
			w.writeLine(n, fmt.Sprintf("%s: (%s)", name, n.Type()))
		} else {
			if n.Parent() != nil && n.Parent().NamedChildCount() > 0 {
				content := n.Content()
				if n.Type() == content {
					w.writeLine(n, fmt.Sprintf("%s = %s", name, content))
				} else {
					w.writeLine(n, fmt.Sprintf("%s: (%s) = %s", name, n.Type(), strconv.Quote(content)))
				}
			} else {
				return nil
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, io.EOF) {
		w.errors.Append(err)
	}
}
