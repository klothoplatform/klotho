package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"gopkg.in/yaml.v3"
)

type Args struct {
	File string `arg:"" help:"The file to parse." type:"existingfile"`
}

func main() {
	var args Args
	ctx := kong.Parse(&args)

	if err := ctx.Run(); err != nil {
		panic(err)
	}
}

func (a Args) Run(kctx *kong.Context) error {
	p := sitter.NewParser()
	f, err := os.Open(a.File)
	if err != nil {
		return err
	}
	defer f.Close()

	switch filepath.Ext(a.File) {
	case ".py":
		p.SetLanguage(python.GetLanguage())

	case ".go":
		p.SetLanguage(golang.GetLanguage())

	case ".js":
		p.SetLanguage(javascript.GetLanguage())

	case ".ts":
		p.SetLanguage(typescript.GetLanguage())

	case ".Dockerfile":
		p.SetLanguage(dockerfile.GetLanguage())

	case "":
		switch filepath.Base(a.File) {
		case "Dockerfile":
			p.SetLanguage(dockerfile.GetLanguage())
		}
	}

	content, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	tree, err := p.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return err
	}

	err = yaml.NewEncoder(os.Stdout).Encode((*ast)(tree.RootNode()))
	if err != nil {
		return err
	}

	return nil
}

type ast sitter.Node

func (a ast) MarshalYAML() (interface{}, error) {
	return a.toNode(), nil
}

func (a ast) toNode() *yaml.Node {
	n := (*sitter.Node)(&a)
	y := &yaml.Node{}

	if n.NamedChildCount() > 0 {
		y.Kind = yaml.MappingNode
		for i := 0; i < int(n.ChildCount()); i++ {
			child := n.Child(i)
			if !child.IsNamed() {
				continue
			}
			key := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: n.FieldNameForChild(i),
			}
			if key.Value != "" {
				key.Value += ": "
			}
			key.Value += "(" + child.Type() + ")"

			value := (*ast)(child).toNode()

			c := child.Content()
			if !strings.Contains(c, "\n") {
				key.LineComment = c
			}

			y.Content = append(y.Content, key, value)
		}
	} else {
		y.Kind = yaml.ScalarNode
		y.Tag = "!!null"
		y.Value = ""

		c := n.Content()
		if !strings.Contains(c, "\n") {
			y.LineComment = c
		}
	}

	return y
}
