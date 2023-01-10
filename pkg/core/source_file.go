package core

import (
	"bytes"
	"context"
	"io"

	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
)

type SourceFile struct {
	Language SourceLanguage
	path     string
	parser   *sitter.Parser
	program  []byte
	tree     *sitter.Tree
	caps     AnnotationMap
}

type SourceLanguage struct {
	ID               LanguageId
	Sitter           *sitter.Language
	CapabilityFinder CapabilityFinder
	TurnIntoComment  Commenter
}

type LanguageId string

type CapabilityFinder interface {
	FindAllCapabilities(*SourceFile) (AnnotationMap, error)
}

type Commenter func(string) string

func (lid LanguageId) CastFile(f File) (*SourceFile, bool) {
	sourceFile, ok := f.(*SourceFile)
	if ok && sourceFile.Language.ID == lid {
		return sourceFile, true
	}
	return nil, false
}

func (f *SourceFile) Reparse(newProgram []byte) (err error) {
	f.program = newProgram
	f.tree, err = f.parser.ParseCtx(context.TODO(), nil, f.program)
	if err == nil {
		caps, err := f.Language.CapabilityFinder.FindAllCapabilities(f)
		if err != nil {
			return err
		}
		f.caps.Update(caps)
	} else {
		err = WrapErrf(err, "could not reparse %s", f.Path())
	}
	return
}

// CloneSourceFile implements the same behavior as `Clone()` (from the `File` interface), but
// returns the result as a `*SourceFile` so that you don't need to cast it.
func (f *SourceFile) CloneSourceFile() *SourceFile {
	nf := &SourceFile{
		Language: f.Language,
		path:     f.path,
		parser:   sitter.NewParser(),
		caps:     make(AnnotationMap),
	}
	nf.parser.SetLanguage(f.Language.Sitter)
	nf.program = make([]byte, len(f.program))
	copy(nf.program, f.program)
	err := nf.Reparse(nf.program)
	if err != nil {
		panic(errors.Wrap(err, "reparse during clone failed!"))
	}
	return nf
}

func (f *SourceFile) Clone() File {
	return f.CloneSourceFile()
}

func (f *SourceFile) Path() string {
	return f.path
}

func (f *SourceFile) WriteTo(out io.Writer) (int64, error) {
	n, err := out.Write([]byte(f.program))
	return int64(n), err
}

func (f *SourceFile) Tree() *sitter.Tree {
	return f.tree
}

func (f *SourceFile) Program() []byte {
	return f.program
}

func (f *SourceFile) Annotations() AnnotationMap {
	return f.caps
}

func (f *SourceFile) IsAnnotatedWith(capability string) bool {
	for _, annot := range f.Annotations() {
		if annot.Capability.Name == capability {
			return true
		}
	}
	return false
}

func (f *SourceFile) ReplaceNodeContent(node *sitter.Node, content string) error {
	buf := new(bytes.Buffer)
	buf.Write(f.program[:node.StartByte()])
	buf.WriteString(content)
	buf.Write(f.program[node.EndByte():])
	return f.Reparse(buf.Bytes())
}

func NewSourceFile(path string, content io.Reader, language SourceLanguage) (f *SourceFile, err error) {
	f = &SourceFile{
		Language: language,
		path:     path,
		caps:     make(AnnotationMap),
	}

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, content)
	if err != nil {
		err = WrapErrf(err, "error reading from %s", path)
		return
	}

	f.parser = sitter.NewParser()
	f.parser.SetLanguage(f.Language.Sitter)
	err = f.Reparse(buf.Bytes())

	return
}
