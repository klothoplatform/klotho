package python

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/code"
	"github.com/klothoplatform/klotho/pkg/code/python/queries"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/set"
	sitter "github.com/smacker/go-tree-sitter"
	lsp "go.lsp.dev/protocol"
	"go.uber.org/zap"
)

type boto3 struct {
	Files fs.FS
	LSP   *code.LSP
}

func FindBoto3Constraints(ctx context.Context, files fs.FS) (constraints.Constraints, error) {
	var err error
	b3 := boto3{Files: files}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	b3.LSP, err = code.NewLSP(ctx, "pylsp", PyLSPLogger{zap.L().Named("lsp/pylsp")})
	if err != nil {
		return constraints.Constraints{}, err
	}

	err = b3.OpenFiles()
	if err != nil {
		return constraints.Constraints{}, fmt.Errorf("could not open files: %w", err)
	}

	return b3.FindAll()
}

func (b *boto3) FindAll() (constraints.Constraints, error) {
	type Result struct {
		c   constraints.Constraints
		err error
	}
	results := make(chan Result)
	numFiles := 0

	err := b.OnPythonFile(func(path string, d fs.DirEntry, file fs.File) error {
		if d.IsDir() {
			return nil
		}
		numFiles++
		go func() {
			c, err := b.FindInFile(file)
			results <- Result{c, err}
		}()
		return nil
	})
	if err != nil {
		return constraints.Constraints{}, err
	}

	var errs error
	var constraints constraints.Constraints
	for i := 0; i < numFiles; i++ {
		result := <-results
		if result.err != nil {
			errs = errors.Join(errs, result.err)
		}
		constraints.Append(result.c)
	}
	return constraints, errs
}

func findString(root *sitter.Node) string {
	// Use the first string. If `root` is not a string itself, use the first one in the subtree.
	for m := range sitter.QueryIterator(root, queries.MakeQuery(`(string) @string`)) {
		s := m["string"].Content()
		quotation := s[0]
		triple := strings.Repeat(string(quotation), 3)
		// Need to check for triple quoted form of string
		if strings.HasPrefix(s, triple) && strings.HasSuffix(s, triple) {
			return s[3 : len(s)-3]
		}
		return s[1 : len(s)-1]
	}
	// If there's no string, check for an identifier
	for m := range sitter.QueryIterator(root, queries.MakeQuery(`(identifier) @identifier`)) {
		return m["identifier"].Content()
	}
	// Finally, fallback to the entire content
	return root.Content()
}

func (b *boto3) FindInFile(file fs.File) (constraints.Constraints, error) {
	tree, err := ParseFile(context.Background(), file)
	if err != nil {
		return constraints.Constraints{}, err
	}
	info, err := file.Stat()
	if err != nil {
		return constraints.Constraints{}, err
	}
	log := zap.L().Named("boto3").With(zap.String("file", info.Name())).Sugar()

	// Use a set to dedupe cases where a bucket is instantiated multiple times
	createRes := make(set.Set[construct.ResourceId])
	var errs error
	for match := range sitter.QueryIterator(tree.RootNode(), queries.FuncCall) {
		name := match["name"]
		switch name.Content() {
		case "Bucket":
			for arg := range sitter.QueryIterator(match["statement"], queries.FuncCallArgs) {
				id := construct.ResourceId{
					Provider: "aws",
					Type:     "s3_bucket",
					Name:     strings.ToLower(findString(arg["arg.value"])),
				}
				log.Infof("found s3 bucket `%s` adding %s", arg["statement"].Content(), id)
				createRes.Add(id)
				// only use the first argument
				break
			}
		}
	}
	cs := constraints.Constraints{}
	for res := range createRes {
		cs.Application = append(cs.Application, constraints.ApplicationConstraint{
			Operator: constraints.MustExistConstraintOperator,
			Node:     res,
		})
	}
	return cs, errs
}

func (b *boto3) OpenFiles() error {
	return b.OnPythonFile(func(path string, d fs.DirEntry, file fs.File) error {
		text, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		return b.LSP.Server.DidOpen(
			b.LSP.Ctx,
			&lsp.DidOpenTextDocumentParams{
				TextDocument: lsp.TextDocumentItem{
					URI:        lsp.DocumentURI("file://" + path),
					LanguageID: lsp.PythonLanguage,
					Text:       string(text),
				},
			},
		)
	})
}

func (b *boto3) CloseFiles() error {
	return b.OnPythonFile(func(path string, d fs.DirEntry, file fs.File) error {
		return b.LSP.Server.DidClose(
			b.LSP.Ctx,
			&lsp.DidCloseTextDocumentParams{
				TextDocument: lsp.TextDocumentIdentifier{
					URI: lsp.DocumentURI("file://" + path),
				},
			},
		)
	})
}

func (b *boto3) OnPythonFile(f func(path string, d fs.DirEntry, file fs.File) error) error {
	return fs.WalkDir(b.Files, ".", func(path string, d fs.DirEntry, nerr error) error {
		if d.IsDir() {
			return nerr
		}
		if filepath.Ext(path) != ".py" {
			return nerr
		}
		file, err := b.Files.Open(path)
		if err != nil {
			return errors.Join(nerr, err)
		}
		err = f(path, d, file)
		if err == fs.SkipAll || err == fs.SkipDir {
			// return these as-is because WalkDir checks them using `==`
			return err
		}
		return errors.Join(nerr, err)
	})
}
