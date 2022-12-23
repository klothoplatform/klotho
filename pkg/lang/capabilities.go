package lang

import (
	"fmt"
	"io"
	"regexp"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/query"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
)

type (
	commentBlock struct {
		comment string
		node    *sitter.Node
	}

	capabilityFinder struct {
		sitterQuery  string
		preprocessor CommentPreprocessor
	}

	// CommentPreprocessor edits a given comment string.
	//
	// Its input is the current comment string. Its output is what the comment string should be instead; this can just be the
	// input string, if no edits are necessary. Empty strings are *not* treated specially: they just look like an empty comment.
	CommentPreprocessor func(comment string) string
)

// RegexpRemovePreprocessor returns a preprocessor powered by a regexp that removes all matches.
//
// The comment will be amended via `regexp.MustCompile(pattern).ReplaceAllString(comment, "")`. The preprocessor will always
// combine comment blocks.
func RegexpRemovePreprocessor(pattern string) CommentPreprocessor {
	regexp := regexp.MustCompile(pattern)
	return func(comment string) string {
		result := regexp.ReplaceAllString(comment, "")
		return result
	}
}

func CompositePreprocessor(preprocessors ...CommentPreprocessor) CommentPreprocessor {
	return func(comment string) string {
		for _, pre := range preprocessors {
			comment = pre(comment)
		}
		return comment
	}
}

// NewCapabilityFinder creates a struct that you can use to find capabilities (annotations) within a source file.
//
// To do this, you provide a `sitterQuery` that looks for comments nodes that contain the `klotho::` annotations,
// as well as a preprocessor that runs over each of those nodes.
//
// If a source file contains multiple comment nodes in a row (as identified by having equal `.Type()`s), those comments
// will be preprocessed individually, but then merged into a single annotation.
func NewCapabilityFinder(sitterQuery string, preprocessor CommentPreprocessor) core.CapabilityFinder {
	return &capabilityFinder{
		sitterQuery:  sitterQuery,
		preprocessor: preprocessor,
	}
}

// FindAllCapabilities finds all of the annotations (ie, capabilities) in a SourceFile.
func (c *capabilityFinder) FindAllCapabilities(f *core.SourceFile) (core.AnnotationMap, error) {
	var merr multierr.Error
	capabilities := make(core.AnnotationMap)
	for _, block := range c.findAllCommentsBlocks(f) {
		cap, err := annotation.ParseCapability(block.comment)
		if cap == nil {
			continue
		}
		annotation := &core.Annotation{Capability: cap, Node: block.node}
		if err != nil {
			merr.Append(core.NewCompilerError(f, annotation, errors.Wrap(err, "error parsing annotation")))
			continue
		}
		capabilities.Add(annotation)
	}
	return capabilities, merr.ErrOrNil()
}

func (c *capabilityFinder) findAllCommentsBlocks(f *core.SourceFile) []*commentBlock {
	const fullCaptureName = "fullQueryCaptureForFindAllCommentsBlocks" // please don't use this in your query ;)
	queryString := fmt.Sprintf(`(%s) @%s`, c.sitterQuery, fullCaptureName)
	nextMatch := query.Exec(f.Language, f.Tree().RootNode(), queryString)

	blocks := []*commentBlock{}
	combineWithPrevious := false
	for {
		match, found := nextMatch()
		if !found || match == nil {
			break
		}
		capture := match[fullCaptureName]
		comment := capture.Content(f.Program())
		comment = c.preprocessor(comment)

		combineWithNext := capture.NextSibling() != nil && capture.NextSibling().Type() == capture.Type()
		node := capture.NextNamedSibling()
		if node == nil {
			continue // this is the last node in the AST, so it's effectively a break :)
		}
		if combineWithPrevious {
			prevBlock := blocks[len(blocks)-1]
			prevBlock.comment = prevBlock.comment + "\n" + comment
			prevBlock.node = node // The previous "node" was just this capture, so we want to basically push it forward
		} else {
			blocks = append(blocks, &commentBlock{comment: comment, node: node})
		}
		combineWithPrevious = combineWithNext
	}
	return blocks
}

func PrintCapabilities(program []byte, caps core.AnnotationMap, out io.Writer) error {
	for _, cap := range caps {
		fmt.Fprintln(out, cap.Capability, cap.Node.Content(program))
	}
	return nil
}
